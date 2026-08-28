package securityaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	conversationKeyPrefix       = "sub2api:prompt_audit:conversation:"
	conversationLockKeyPrefix   = "sub2api:prompt_audit:conversation_lock:"
	conversationParentKeyPrefix = "sub2api:prompt_audit:conversation_parent:"

	conversationStatusClean        = "clean"
	conversationStatusFullRequired = "full_required"

	defaultConversationStateTTL = 2 * time.Hour
	// ponytail: fixed 2h lease covers the current 1h Live and 15m WS limits;
	// add renewal only if a supported transport can outlive this bound.
	defaultConversationLockTTL = 2 * time.Hour
)

var (
	errConversationBusy       = errors.New("prompt audit conversation already has an active turn")
	errConversationLeaseLost  = errors.New("prompt audit conversation lease lost")
	errConversationStoreDown  = errors.New("prompt audit conversation store unavailable")
	errConversationKeyInvalid = errors.New("prompt audit conversation key invalid")
)

var beginConversationScript = redis.NewScript(`
if redis.call('exists', KEYS[2]) == 1 then
  return {0}
end
local prior = redis.call('hmget', KEYS[1], 'status', 'config_version', 'context_hash', 'output_ciphertext', 'output_ready', 'response_id_hash', 'input_hash', 'input_count', 'output_hash', 'output_count')
redis.call('psetex', KEYS[2], ARGV[2], ARGV[1])
redis.call('hset', KEYS[1], 'status', 'full_required', 'updated_at', ARGV[4])
redis.call('hdel', KEYS[1], 'output_ciphertext', 'output_ready', 'response_id_hash')
redis.call('expire', KEYS[1], ARGV[3])
return {1, prior[1], prior[2], prior[3], prior[4], prior[5], prior[6], prior[7], prior[8], prior[9], prior[10]}
`)

var commitConversationScript = redis.NewScript(`
if redis.call('get', KEYS[2]) ~= ARGV[1] then
  return 0
end
redis.call('hset', KEYS[1],
  'status', 'clean',
  'config_version', ARGV[2],
  'context_hash', ARGV[3],
	  'output_ciphertext', ARGV[4],
  'output_ready', '1',
	  'input_hash', ARGV[7],
	  'input_count', ARGV[8],
	  'output_hash', ARGV[9],
	  'output_count', ARGV[10],
	  'updated_at', ARGV[11])
if ARGV[5] == '' then
  redis.call('hdel', KEYS[1], 'response_id_hash')
else
	  redis.call('hset', KEYS[1], 'response_id_hash', ARGV[5])
	  redis.call('set', KEYS[3], ARGV[12], 'EX', ARGV[6])
end
redis.call('expire', KEYS[1], ARGV[6])
redis.call('del', KEYS[2])
return 1
`)

var failConversationScript = redis.NewScript(`
if redis.call('get', KEYS[2]) ~= ARGV[1] then
  return 0
end
redis.call('hset', KEYS[1], 'status', 'full_required', 'updated_at', ARGV[3])
redis.call('hdel', KEYS[1], 'output_ciphertext', 'output_ready', 'response_id_hash')
redis.call('expire', KEYS[1], ARGV[2])
redis.call('del', KEYS[2])
return 1
`)

type conversationCheckpoint struct {
	Status           string
	ConfigVersion    int64
	ContextHash      string
	OutputCiphertext string
	OutputReady      bool
	ResponseIDHash   string
	Input            conversationFingerprint
	OutputDigest     conversationFingerprint
}

type conversationLease struct {
	key        string
	token      string
	apiKeyID   int64
	checkpoint conversationCheckpoint
}

type RedisConversationStore struct {
	client   *redis.Client
	stateTTL time.Duration
	lockTTL  time.Duration
}

func NewRedisConversationStore(client *redis.Client) *RedisConversationStore {
	return &RedisConversationStore{
		client: client, stateTTL: defaultConversationStateTTL, lockTTL: defaultConversationLockTTL,
	}
}

// ConversationKey hashes a tenant-isolated stable client identity. Raw
// session identifiers never enter Redis keys, values, logs, or audit events.
func ConversationKey(apiKeyID int64, rawIdentity string) string {
	rawIdentity = strings.TrimSpace(rawIdentity)
	if apiKeyID <= 0 || rawIdentity == "" {
		return ""
	}
	digest := sha256.Sum256([]byte("prompt-audit-conversation:v1|api_key=" + strconv.FormatInt(apiKeyID, 10) + "|identity=" + rawIdentity))
	return hex.EncodeToString(digest[:])
}

func NewConversationKey(apiKeyID int64) string {
	return ConversationKey(apiKeyID, "generated:"+uuid.NewString())
}

func (s *RedisConversationStore) ResolveParent(ctx context.Context, apiKeyID int64, parentID string) (string, bool, error) {
	if s == nil || s.client == nil {
		return "", false, errConversationStoreDown
	}
	key := conversationParentKey(apiKeyID, parentID)
	if key == "" {
		return "", false, nil
	}
	value, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !validConversationKey(value) {
		return "", false, errConversationKeyInvalid
	}
	return value, true, nil
}

// Begin atomically consumes the prior CLEAN checkpoint and marks the
// conversation FULL_REQUIRED before any Guard or upstream work starts. A
// crash, cancellation, or missed completion therefore cannot leave a stale
// CLEAN checkpoint behind.
func (s *RedisConversationStore) Begin(ctx context.Context, apiKeyID int64, key string) (*conversationLease, error) {
	if s == nil || s.client == nil {
		return nil, errConversationStoreDown
	}
	if apiKeyID <= 0 || !validConversationKey(key) {
		return nil, errConversationKeyInvalid
	}
	token := uuid.NewString()
	result, err := beginConversationScript.Run(ctx, s.client,
		[]string{conversationStateKey(key), conversationLockKey(key)},
		token,
		s.lockTTL.Milliseconds(),
		int64(s.stateTTL/time.Second),
		time.Now().UTC().Unix(),
	).Slice()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 || redisString(result[0]) != "1" {
		return nil, errConversationBusy
	}
	checkpoint := conversationCheckpoint{}
	if len(result) > 1 {
		checkpoint.Status = redisString(result[1])
	}
	if len(result) > 2 {
		checkpoint.ConfigVersion, _ = strconv.ParseInt(redisString(result[2]), 10, 64)
	}
	if len(result) > 3 {
		checkpoint.ContextHash = redisString(result[3])
	}
	if len(result) > 4 {
		checkpoint.OutputCiphertext = redisString(result[4])
	}
	if len(result) > 5 {
		checkpoint.OutputReady = redisString(result[5]) == "1"
	}
	if len(result) > 6 {
		checkpoint.ResponseIDHash = redisString(result[6])
	}
	if len(result) > 7 {
		checkpoint.Input.Hash = redisString(result[7])
	}
	if len(result) > 8 {
		checkpoint.Input.Count, _ = strconv.Atoi(redisString(result[8]))
	}
	if len(result) > 9 {
		checkpoint.OutputDigest.Hash = redisString(result[9])
	}
	if len(result) > 10 {
		checkpoint.OutputDigest.Count, _ = strconv.Atoi(redisString(result[10]))
	}
	return &conversationLease{key: key, token: token, apiKeyID: apiKeyID, checkpoint: checkpoint}, nil
}

func (s *RedisConversationStore) Commit(ctx context.Context, lease *conversationLease, configVersion int64, contextHash, output, responseID string, input, outputDigest conversationFingerprint) error {
	if s == nil || s.client == nil {
		return errConversationStoreDown
	}
	if lease == nil || !validConversationKey(lease.key) || strings.TrimSpace(lease.token) == "" {
		return errConversationKeyInvalid
	}
	parentKey := conversationStateKey(lease.key)
	responseIDHash := ""
	if strings.TrimSpace(responseID) != "" {
		parentKey = conversationParentKey(lease.apiKeyID, responseID)
		responseIDHash = conversationParentHash(lease.apiKeyID, responseID)
	}
	result, err := commitConversationScript.Run(ctx, s.client,
		[]string{conversationStateKey(lease.key), conversationLockKey(lease.key), parentKey},
		lease.token,
		configVersion,
		contextHash,
		output,
		responseIDHash,
		int64(s.stateTTL/time.Second),
		input.Hash,
		input.Count,
		outputDigest.Hash,
		outputDigest.Count,
		time.Now().UTC().Unix(),
		lease.key,
	).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return errConversationLeaseLost
	}
	return nil
}

func (s *RedisConversationStore) Fail(ctx context.Context, lease *conversationLease) error {
	if s == nil || s.client == nil {
		return errConversationStoreDown
	}
	if lease == nil || !validConversationKey(lease.key) || strings.TrimSpace(lease.token) == "" {
		return errConversationKeyInvalid
	}
	result, err := failConversationScript.Run(ctx, s.client,
		[]string{conversationStateKey(lease.key), conversationLockKey(lease.key)},
		lease.token,
		int64(s.stateTTL/time.Second),
		time.Now().UTC().Unix(),
	).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return errConversationLeaseLost
	}
	return nil
}

func validConversationKey(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func conversationStateKey(key string) string { return conversationKeyPrefix + key }
func conversationLockKey(key string) string  { return conversationLockKeyPrefix + key }

func conversationParentKey(apiKeyID int64, parentID string) string {
	hash := conversationParentHash(apiKeyID, parentID)
	if hash == "" {
		return ""
	}
	return conversationParentKeyPrefix + hash
}

func conversationParentHash(apiKeyID int64, parentID string) string {
	parentID = strings.TrimSpace(parentID)
	if apiKeyID <= 0 || parentID == "" {
		return ""
	}
	digest := sha256.Sum256([]byte("prompt-audit-parent:v1|api_key=" + strconv.FormatInt(apiKeyID, 10) + "|parent=" + parentID))
	return hex.EncodeToString(digest[:])
}

func redisString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprint(typed)
	}
}
