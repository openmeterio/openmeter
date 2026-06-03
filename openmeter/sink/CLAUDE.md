# sink

<!-- archie:ai-start -->

> High-throughput Kafka-to-ClickHouse sink worker: consumes raw CloudEvent messages from Kafka partitions, deduplicates (Redis or in-batch), batch-inserts into ClickHouse via streaming.Connector, then fires post-flush callbacks so downstream (balance-worker) can recalculate entitlements. Primary constraint: an exactly-once guarantee enforced by strict three-phase flush ordering.

## Patterns

**Three-phase flush ordering: storage -> offset commit -> Redis dedupe** — flush() always persists to ClickHouse (persistToStorage), then Consumer.StoreMessage for Kafka offsets, then dedupeSet to Redis. Reversing breaks exactly-once on consumer restart. The full sequence runs under s.mu, which also serialises partition pause/resume with Close. (`// sink.go: 1. persistToStorage -> 2. Consumer.StoreMessage -> 3. dedupeSet (under s.mu, partitions paused)`)
**FlushEventHandler dispatched in a goroutine, never blocking** — OnFlushSuccess runs in a goroutine bounded by FlushSuccessTimeout. Calling it synchronously inside flush() backpressures Kafka partitions. The flushhandler/ FlushEventHandlers multiplexer fans out to downstream handlers with two-phase drain. (`go func(){ ctx, cancel := context.WithTimeout(ctx, s.config.FlushSuccessTimeout); defer cancel(); s.config.FlushEventHandler.OnFlushSuccess(ctx, messages) }()`)
**SinkConfig.Validate() gates construction; Storage injected, not instantiated** — NewSink calls config.Validate() with explicit nil/zero checks for every required field before returning. Storage is the injected interface (ClickHouseStorage in prod, mock in tests) — never new ClickHouseStorage inside sink.go. (`type Storage interface { BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error }`)
**Meter definitions served from NamespacedMeterCache, never meter.Service in the hot path** — meters.go caches namespace->MetersByType behind an RWMutex with a periodic refetch goroutine started by Start(); the consume path calls GetAffectedMeters, never meter.Service directly. (`meters, err := s.meterCache.GetAffectedMeters(ctx, &message)`)
**SinkBuffer keyed by TopicPartition+Offset; revoke on rebalance** — buffer.go is a mutex-guarded map keyed by KafkaMessage.String() so Add is idempotent within a batch. RemoveByPartitions(revoked) must run before a rebalance completes to drop messages for revoked partitions. (`buffer.RemoveByPartitions(revokedPartitions)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sink.go` | Sink struct, SinkConfig + Validate, NewSink, flush() orchestration, Run() event loop. | s.mu must cover the entire ClickHouse+offset+Redis sequence; any reorder risks an exactly-once violation. |
| `storage.go` | Storage interface + ClickHouseStorage; maps SinkMessage to streaming.RawEvent, generates a ULID StoreRowID per event. | ingested_at/stored_at fall back to clock.Now(); a parse/insert error fails the whole batch (retry). |
| `buffer.go` | SinkBuffer in-batch dedup map; Add idempotent, Dequeue with transformers, RemoveByPartitions. | Call RemoveByPartitions before rebalance completes to avoid double processing. |
| `meters.go` | NamespacedMeterCache: RWMutex namespace->MetersByType, atomic isRunning, periodic refresh. | isRunning is set before the ticker starts; double-Start is a no-op but the cache won't refresh. |
| `models/models.go` | Pure types: SinkMessage pipeline carrier, ProcessingStatus/ProcessingState (OK/DROP). | No business logic; importing streaming/meter here creates a circular dependency. Use GetDedupeItem() rather than building dedupe.Item by hand. |
| `flushhandler/handler.go` | FlushEventHandler buffered async impl with two-phase shutdown (stopChan then drain) and FlushEventHandlers fan-out mux. | Call WaitForDrain() before process exit or buffered flush notifications are dropped. |

## Anti-Patterns

- Writing Redis dedupe before the Kafka offset commit — violates exactly-once on restart.
- Calling FlushEventHandler.OnFlushSuccess synchronously inside flush() — backpressures Kafka partitions.
- Querying meter.Service inside the hot consume path instead of NamespacedMeterCache — per-message DB round trips.
- Instantiating ClickHouseStorage inside sink.go instead of injecting via SinkConfig.Storage — breaks testability.
- Adding a ProcessingState iota without a String() case and persistToStorage() handling — 'unknown state type' at runtime.

## Decisions

- **Flush is protected by a mutex that also covers partition pause/resume.** — Prevents the sink closing mid-flush during a rebalance and keeps ClickHouse write + offset commit + Redis dedupe atomic from the consumer's view.
- **Storage is an injected interface, not instantiated inside Sink.** — Allows ClickHouse test doubles and aligns with the layered adapter pattern used across the codebase.
- **FlushEventHandler runs in a goroutine with FlushSuccessTimeout.** — Post-flush balance-recalculation notification must not block the ingest path; the timeout bounds the goroutine even if downstream is slow.

## Example: Implementing a new FlushEventHandler for post-flush side effects

```
import (
    "context"
    "github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
    sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

type myHandler struct{}

func (h *myHandler) OnFlushSuccess(ctx context.Context, msgs []sinkmodels.SinkMessage) error {
    // ctx carries a FlushSuccessTimeout deadline
    return nil
}

// Wire through the fan-out multiplexer, then WaitForDrain on shutdown:
mux := flushhandler.NewFlushEventHandlers()
```

<!-- archie:ai-end -->
