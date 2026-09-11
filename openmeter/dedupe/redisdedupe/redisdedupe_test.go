package redisdedupe

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

type lookupHook struct {
	keys    map[string]bool
	calls   int
	err     error
	latency time.Duration
}

func (h *lookupHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *lookupHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *lookupHook) ProcessHook(_ redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		h.calls++
		if h.latency > 0 {
			time.Sleep(h.latency)
		}
		if h.err != nil {
			return h.err
		}
		switch cmd := cmd.(type) {
		case *redis.SliceCmd:
			values := make([]any, len(cmd.Args())-1)
			for i, arg := range cmd.Args()[1:] {
				if h.keys[arg.(string)] {
					values[i] = ""
				}
			}
			cmd.SetVal(values)
		case *redis.IntCmd:
			var count int64
			for _, arg := range cmd.Args()[1:] {
				if h.keys[arg.(string)] {
					count++
				}
			}
			cmd.SetVal(count)
		default:
			return errors.New("unexpected Redis command")
		}
		return nil
	}
}

func TestCheckUniqueBatch(t *testing.T) {
	for _, mode := range []DedupeMode{DedupeModeRawKey, DedupeModeKeyHash, DedupeModeKeyHashMigration} {
		t.Run(string(mode), func(t *testing.T) {
			items := []dedupe.Item{
				{Namespace: "ns", Source: "source", ID: "new"},
				{Namespace: "ns", Source: "source", ID: "raw"},
				{Namespace: "ns", Source: "source", ID: "hashed"},
				{Namespace: "ns", Source: "source", ID: "both"},
				{Namespace: "other", Source: "source", ID: "both"},
			}
			hook := &lookupHook{keys: map[string]bool{
				items[1].Key():             true,
				GetKeyHash(items[2].Key()): true,
				items[3].Key():             true,
				GetKeyHash(items[3].Key()): true,
			}}
			client := redis.NewClient(&redis.Options{})
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			client.AddHook(hook)
			d := Deduplicator{Redis: client, Mode: mode}

			result, err := d.CheckUniqueBatch(t.Context(), items)
			require.NoError(t, err)
			require.Equal(t, 1, hook.calls)
			require.Contains(t, result.UniqueItems, items[0])
			require.Contains(t, result.UniqueItems, items[4])
			require.Contains(t, result.AlreadyProcessedItems, items[3])
			for _, item := range items {
				unique, err := d.CheckUnique(t.Context(), item)
				require.NoError(t, err)
				_, batchUnique := result.UniqueItems[item]
				require.Equal(t, unique, batchUnique, item.ID)
				_, processed := result.AlreadyProcessedItems[item]
				require.NotEqual(t, batchUnique, processed)
			}

			hook.err = errors.New("redis unavailable")
			_, err = d.CheckUniqueBatch(t.Context(), items)
			require.ErrorIs(t, err, hook.err)
			_, err = d.CheckUniqueBatch(t.Context(), nil)
			require.ErrorIs(t, err, ErrNoDedupItems)
		})
	}
}

func BenchmarkUniquenessChecks(b *testing.B) {
	items := make([]dedupe.Item, 1000)
	for i := range items {
		items[i] = dedupe.Item{Namespace: "ns", Source: "source", ID: strconv.Itoa(i)}
	}
	for _, batched := range []bool{false, true} {
		name := "per-message-plus-batch"
		if batched {
			name = "batch-only"
		}
		b.Run(name, func(b *testing.B) {
			client := redis.NewClient(&redis.Options{})
			b.Cleanup(func() { require.NoError(b, client.Close()) })
			hook := &lookupHook{latency: time.Millisecond}
			client.AddHook(hook)
			d := Deduplicator{Redis: client, Mode: DedupeModeRawKey}
			b.ResetTimer()
			for b.Loop() {
				if !batched {
					for _, item := range items {
						_, err := d.CheckUnique(b.Context(), item)
						if err != nil {
							b.Fatal(err)
						}
					}
				}
				_, err := d.CheckUniqueBatch(b.Context(), items)
				if err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(hook.calls)/float64(b.N), "redis-calls/batch")
		})
	}
}

func TestCheckUniqueBatchInvalidMode(t *testing.T) {
	client := redis.NewClient(&redis.Options{})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	client.AddHook(&lookupHook{})
	d := Deduplicator{Redis: client, Mode: DedupeMode("invalid")}
	_, err := d.CheckUniqueBatch(t.Context(), []dedupe.Item{{ID: "id"}})
	require.Error(t, err)
}
