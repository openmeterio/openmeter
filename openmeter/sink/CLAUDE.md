# sink

<!-- archie:ai-start -->

> High-throughput Kafka-to-ClickHouse sink worker: consumes raw CloudEvent messages from Kafka partitions, deduplicates via Redis or in-memory, batch-inserts into ClickHouse via streaming.Connector, then fires post-flush callbacks so the balance-worker can recalculate entitlements. Primary constraint: exactly-once guarantee requires strict flush ordering (ClickHouse insert → Kafka offset commit → Redis dedupe).

## Patterns

**Three-phase flush ordering: storage → offset commit → Redis dedupe** — flush() always writes ClickHouse first (persistToStorage), then stores Kafka offsets (Consumer.StoreMessage), then sets Redis dedupe keys. Reversing this order breaks the exactly-once guarantee on consumer restart. (`// sink.go: 1. persistToStorage → 2. Consumer.StoreMessage → 3. dedupeSet
// Never reorder these three phases.`)
**FlushEventHandler called in a goroutine after flush, never blocking** — OnFlushSuccess is dispatched in a separate goroutine with a FlushSuccessTimeout-bounded context. Never call FlushEventHandler synchronously from flush() — it blocks the main sink loop and causes backpressure on Kafka partitions. (`go func() {
    ctx, cancel := context.WithTimeout(ctx, s.config.FlushSuccessTimeout)
    defer cancel()
    s.config.FlushEventHandler.OnFlushSuccess(ctx, messages)
}()`)
**SinkConfig.Validate() before any Sink construction** — All Sink configuration is validated in SinkConfig.Validate() before NewSink returns. Every required field has an explicit nil/zero check; callers must not pass partially-configured SinkConfig. (`func NewSink(config SinkConfig) (*Sink, error) {
    if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid config: %w", err) }
}`)
**NamespacedMeterCache for meter definitions — never query meter.Service in the hot path** — Meter definitions are cached in NamespacedMeterCache and periodically re-fetched at MeterRefetchInterval. Access meter definitions via meterCache.GetMetersByType(), never query meter.Service directly in the Kafka consume path. (`meters, err := s.meterCache.GetMetersByType(message)`)
**Storage interface injected via SinkConfig — never instantiated inside sink.go** — ClickHouseStorage implements Storage. Tests inject a mock. Never instantiate ClickHouseStorage directly inside sink.go. (`type Storage interface { BatchInsert(ctx context.Context, messages []sinkmodels.SinkMessage) error }`)
**SinkBuffer.RemoveByPartitions on Kafka partition revocation** — On rebalance, SinkBuffer.RemoveByPartitions(revokedPartitions) must be called before the rebalance completes to drop buffered messages for revoked partitions and prevent double processing. (`buffer.RemoveByPartitions(revokedPartitions)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sink.go` | Main Sink struct, SinkConfig, NewSink constructor, flush() orchestration, Run() event loop. | The flush mutex (s.mu) serialises partition pause/resume with close — any change to flush ordering risks exactly-once violation; the mu.Lock must cover the full ClickHouse+offset+Redis sequence. |
| `storage.go` | Storage interface and ClickHouseStorage implementation. Reads ingested_at Kafka header; generates a ULID StoreRowID per event. | ingested_at header parse failure returns an error stopping the entire batch — validate header format at ingest time. |
| `buffer.go` | SinkBuffer: mutex-guarded map keyed by TopicPartition for deduplication within a batch. Add is idempotent on same TopicPartition+Offset. | RemoveByPartitions used on partition revocation — callers must call this before rebalance completes. |
| `meters.go` | NamespacedMeterCache: RWMutex-guarded namespace→MetersByType map with atomic isRunning flag. Periodic refresh goroutine started by Start(). | isRunning must be set before starting the ticker goroutine; double-start is a no-op but the cache will not refresh. |
| `models/models.go` | SinkMessage (pipeline carrier), ProcessingStatus/ProcessingState (OK/DROP). Pure types package. | Do not add business logic here; importing streaming or meter from models creates a circular dependency. |
| `flushhandler/handler.go` | FlushEventHandler interface + buffered async impl with two-phase shutdown (stopChan signal then drain loop). | WaitForDrain() must be called before process exit or buffered messages will be dropped. |

## Anti-Patterns

- Changing flush() to write Redis dedupe before Kafka offset commit — violates exactly-once guarantee on consumer restart.
- Calling FlushEventHandler.OnFlushSuccess synchronously inside flush() — blocks the main sink loop and causes backpressure on Kafka partitions.
- Querying meter.Service directly inside the hot Kafka consume path instead of NamespacedMeterCache — causes per-message DB round trips.
- Constructing ClickHouseStorage inside sink.go instead of injecting via SinkConfig.Storage — breaks testability.
- Adding new ProcessingState iota values without implementing ProcessingState.String() and handling them in persistToStorage() — results in an 'unknown state type' error at runtime.

## Decisions

- **Flush is protected by a mutex that also covers partition pause/resume.** — Prevents the sink from closing mid-flush when a rebalance arrives; ensures ClickHouse write + offset commit + Redis dedupe are atomic from the consumer's perspective.
- **Storage is an interface injected via SinkConfig, not instantiated inside Sink.** — Allows test doubles for ClickHouse without spinning up a real ClickHouse instance; aligns with the layered adapter pattern used across the codebase.
- **FlushEventHandler dispatched in a goroutine with a FlushSuccessTimeout.** — Post-flush notification (for balance recalculation) must not block the main ingest path; a timeout bounds the goroutine lifetime even if downstream is slow.

## Example: Implementing a new FlushEventHandler for post-flush side effects

```
import (
    "context"
    "github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
    sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

type myHandler struct{}

func (h *myHandler) OnFlushSuccess(ctx context.Context, msgs []sinkmodels.SinkMessage) error {
    // ctx has FlushSuccessTimeout deadline; process msgs here
    return nil
}

// Wire via FlushEventHandlers multiplexer:
mux := flushhandler.NewFlushEventHandlers()
// ...
```

<!-- archie:ai-end -->
