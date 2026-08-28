package securityaudit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisConversationStoreConsumesAndCommitsCheckpointAtomically(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisConversationStore(client)
	store.stateTTL = time.Hour
	store.lockTTL = time.Minute
	key := ConversationKey(7, "session-1")

	first, err := store.Begin(context.Background(), 7, key)
	require.NoError(t, err)
	require.Empty(t, first.checkpoint.Status)
	_, err = store.Begin(context.Background(), 7, key)
	require.ErrorIs(t, err, errConversationBusy)

	inputDigest := fingerprintConversationTexts([]string{"user input"})
	outputDigest := fingerprintConversationTexts([]string{"assistant output"})
	require.NoError(t, store.Commit(context.Background(), first, 4, "ctx-1", "assistant output", "resp_1", inputDigest, outputDigest))
	resolved, found, err := store.ResolveParent(context.Background(), 7, "resp_1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, key, resolved)

	second, err := store.Begin(context.Background(), 7, key)
	require.NoError(t, err)
	require.Equal(t, conversationStatusClean, second.checkpoint.Status)
	require.Equal(t, int64(4), second.checkpoint.ConfigVersion)
	require.Equal(t, "ctx-1", second.checkpoint.ContextHash)
	require.Equal(t, "assistant output", second.checkpoint.OutputCiphertext)
	require.True(t, second.checkpoint.OutputReady)
	require.Equal(t, conversationParentHash(7, "resp_1"), second.checkpoint.ResponseIDHash)
	require.NotEqual(t, "resp_1", second.checkpoint.ResponseIDHash)
	require.Equal(t, inputDigest, second.checkpoint.Input)
	require.Equal(t, outputDigest, second.checkpoint.OutputDigest)
	require.NoError(t, store.Fail(context.Background(), second))

	third, err := store.Begin(context.Background(), 7, key)
	require.NoError(t, err)
	require.Equal(t, conversationStatusFullRequired, third.checkpoint.Status)
	require.False(t, third.checkpoint.OutputReady)
	require.Empty(t, third.checkpoint.OutputCiphertext)
	require.NoError(t, store.Fail(context.Background(), third))
}

func TestRedisConversationStoreRejectsStaleLease(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisConversationStore(client)
	store.lockTTL = time.Second
	key := ConversationKey(9, "session-2")

	lease, err := store.Begin(context.Background(), 9, key)
	require.NoError(t, err)
	server.FastForward(2 * time.Second)
	require.ErrorIs(t, store.Commit(context.Background(), lease, 1, "ctx", "output", "resp_2", fingerprintConversationTexts(nil), fingerprintConversationTexts(nil)), errConversationLeaseLost)
}
