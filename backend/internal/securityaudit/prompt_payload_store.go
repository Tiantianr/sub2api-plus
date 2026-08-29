package securityaudit

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type PayloadStore interface {
	Set(ctx context.Context, jobID int64, scanText string, ttl time.Duration) error
	Get(ctx context.Context, jobID int64) (string, error)
	Delete(ctx context.Context, jobID int64) error
	Ping(ctx context.Context) error
}

type DeepReviewStateStore interface {
	Required(ctx context.Context, userID int64) (token string, required bool, err error)
	Require(ctx context.Context, userID int64, token string) error
	Replace(ctx context.Context, userID int64, oldToken, newToken string) (bool, error)
	Clear(ctx context.Context, userID int64, token string) (bool, error)
}

type RedisPayloadStore struct {
	client *redis.Client
}

func NewRedisPayloadStore(client *redis.Client) *RedisPayloadStore {
	return &RedisPayloadStore{client: client}
}

func (s *RedisPayloadStore) Set(ctx context.Context, jobID int64, scanText string, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("prompt audit payload store unavailable")
	}
	if jobID <= 0 || scanText == "" {
		return fmt.Errorf("prompt audit payload input invalid")
	}
	if ttl <= 0 || ttl > DefaultPayloadTTL {
		ttl = DefaultPayloadTTL
	}
	return s.client.Set(ctx, payloadKey(jobID), scanText, ttl).Err()
}

func (s *RedisPayloadStore) Get(ctx context.Context, jobID int64) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("prompt audit payload store unavailable")
	}
	return s.client.Get(ctx, payloadKey(jobID)).Result()
}

func (s *RedisPayloadStore) Delete(ctx context.Context, jobID int64) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("prompt audit payload store unavailable")
	}
	return s.client.Del(ctx, payloadKey(jobID)).Err()
}

func (s *RedisPayloadStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("prompt audit payload store unavailable")
	}
	return s.client.Ping(ctx).Err()
}

func (s *RedisPayloadStore) Required(ctx context.Context, userID int64) (string, bool, error) {
	if s == nil || s.client == nil {
		return "", false, fmt.Errorf("prompt audit deep review state unavailable")
	}
	if userID <= 0 {
		return "", false, nil
	}
	token, err := s.client.Get(ctx, deepReviewStateKey(userID)).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return token, strings.TrimSpace(token) != "", nil
}

func (s *RedisPayloadStore) Require(ctx context.Context, userID int64, token string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("prompt audit deep review state unavailable")
	}
	if userID <= 0 || strings.TrimSpace(token) == "" {
		return fmt.Errorf("prompt audit deep review state input invalid")
	}
	return s.client.Set(ctx, deepReviewStateKey(userID), token, 0).Err()
}

func (s *RedisPayloadStore) Replace(ctx context.Context, userID int64, oldToken, newToken string) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("prompt audit deep review state unavailable")
	}
	if userID <= 0 || strings.TrimSpace(oldToken) == "" || strings.TrimSpace(newToken) == "" {
		return false, fmt.Errorf("prompt audit deep review state input invalid")
	}
	result, err := s.client.Eval(ctx, `
		if redis.call('GET', KEYS[1]) == ARGV[1] then
			redis.call('SET', KEYS[1], ARGV[2])
			return 1
		end
		return 0`, []string{deepReviewStateKey(userID)}, oldToken, newToken).Int64()
	return result == 1, err
}

func (s *RedisPayloadStore) Clear(ctx context.Context, userID int64, token string) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("prompt audit deep review state unavailable")
	}
	if userID <= 0 || strings.TrimSpace(token) == "" {
		return false, fmt.Errorf("prompt audit deep review state input invalid")
	}
	result, err := s.client.Eval(ctx, `
		if redis.call('GET', KEYS[1]) == ARGV[1] then
			return redis.call('DEL', KEYS[1])
		end
		return 0`, []string{deepReviewStateKey(userID)}, token).Int64()
	return result == 1, err
}

func payloadKey(jobID int64) string {
	return PayloadKeyPrefix + strconv.FormatInt(jobID, 10)
}

func deepReviewStateKey(userID int64) string {
	return DeepReviewStateKeyPrefix + strconv.FormatInt(userID, 10)
}
