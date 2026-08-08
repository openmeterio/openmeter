package redisdedupe

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

type redisError string

func (e redisError) Error() string {
	return string(e)
}

func (redisError) RedisError() {}

type pipelineErrorHook struct {
	err error
}

func (h pipelineErrorHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h pipelineErrorHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return next
}

func (h pipelineErrorHook) ProcessPipelineHook(redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error {
		return h.err
	}
}

func TestSetReconnectsOnFailoverShapedWriteError(t *testing.T) {
	firstClient := newHookedClient(redisError("READONLY You can't write against a read only replica"))
	secondClient := newHookedClient(nil)

	var reconnects int
	deduplicator := NewDeduplicator(firstClient, time.Hour, DedupeModeRawKey, func() (*redis.Client, error) {
		reconnects++
		return secondClient, nil
	})

	_, err := deduplicator.Set(context.Background(), dedupe.Item{
		Namespace: "namespace",
		Source:    "source",
		ID:        "id",
	})

	require.Error(t, err)
	require.Equal(t, 1, reconnects)
	require.Same(t, secondClient, deduplicator.Redis)
}

func TestSetDoesNotReconnectOnApplicationWriteError(t *testing.T) {
	firstClient := newHookedClient(redisError("WRONGTYPE Operation against a key holding the wrong kind of value"))
	secondClient := newHookedClient(nil)

	var reconnects int
	deduplicator := NewDeduplicator(firstClient, time.Hour, DedupeModeRawKey, func() (*redis.Client, error) {
		reconnects++
		return secondClient, nil
	})

	_, err := deduplicator.Set(context.Background(), dedupe.Item{
		Namespace: "namespace",
		Source:    "source",
		ID:        "id",
	})

	require.Error(t, err)
	require.Zero(t, reconnects)
	require.Same(t, firstClient, deduplicator.Redis)
}

func TestSetDoesNotReconnectAfterContextCancellation(t *testing.T) {
	firstClient := newHookedClient(redisError("READONLY You can't write against a read only replica"))
	secondClient := newHookedClient(nil)

	var reconnects int
	deduplicator := NewDeduplicator(firstClient, time.Hour, DedupeModeRawKey, func() (*redis.Client, error) {
		reconnects++
		return secondClient, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := deduplicator.Set(ctx, dedupe.Item{
		Namespace: "namespace",
		Source:    "source",
		ID:        "id",
	})

	require.Error(t, err)
	require.Zero(t, reconnects)
	require.Same(t, firstClient, deduplicator.Redis)
}

func newHookedClient(err error) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:       "127.0.0.1:0",
		MaxRetries: -1,
	})

	if err != nil {
		client.AddHook(pipelineErrorHook{err: err})
	}

	return client
}
