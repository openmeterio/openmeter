# sink

<!-- archie:ai-start -->

> Sink-worker core (driven by cmd/sink-worker): consumes usage events from Kafka, buffers them per (topic,partition,offset), dedupes, validates against cached meters, batch-inserts to ClickHouse, then commits offsets and writes dedupe keys — preserving an at-least-once / exactly-once-on-storage guarantee. Children: models (DTOs), flushhandler (post-flush callbacks).

## Patterns

**Config struct with Validate() then New constructor** — Every component (SinkConfig, ClickHouseStorageConfig, NamespacedMeterCacheConfig) has a Validate() returning errors.New(...) per missing dep, and a New* that calls config.Validate() first and wraps failure with fmt.Errorf. (`func NewSink(config SinkConfig) (*Sink, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid sink configuration: %w", err) } ... }`)
**Ordered flush: storage -> store offset -> dedupe set** — Sink.flush() runs under s.mu.Lock with partitions paused: (1) persistToStorage, (2) StoreMessage offsets sorted ascending so the largest commits last, (3) dedupeSet to Redis. The ordering is the consistency contract. (`// 1. Persist to storage; 2. Store Offset (sort by Offset asc); 3. Sink to Redis`)
**Buffer keyed by Kafka coordinate, mutex-guarded** — SinkBuffer.Add keys messages by KafkaMessage.String() (topic+partition+offset) so duplicates within a batch collapse; Dequeue applies MessageTransformerFunc transformers (e.g. withStoredAt) while draining under the lock. (`key := message.KafkaMessage.String(); b.data[key] = message`)
**Pause/resume partitions around flush** — flush() calls s.pause() before draining and defers s.resume(), preventing new messages from being processed mid-flush to keep storage and offset state consistent. (`err := s.pause(); ...; defer func() { err = s.resume() ... }()`)
**Periodic background cache refetch** — NamespacedMeterCache.Start guards with atomic.Bool, seeds the cache, then a goroutine ticks on periodicRefetchInterval calling fetchMeters; GetAffectedMeters reads under RWMutex and returns nil for dropped messages. (`if n.isRunning.Swap(true) { return errors.New("namespaced meter cache is already running") }`)
**FlushEventHandler invoked in a goroutine with timeout** — After a successful flush, if FlushEventHandler is set it runs OnFlushSuccess in a `go func()` under context.WithTimeout(ctx, FlushSuccessTimeout); errors are logged, never propagated to the flush path. (`go func() { ctx, cancel := context.WithTimeout(ctx, s.config.FlushSuccessTimeout); defer cancel(); err := s.config.FlushEventHandler.OnFlushSuccess(ctx, messages); ... }()`)
**OTel metrics + tracing wrap every phase** — Sink holds Int64Counter/Int64Histogram instruments created in NewSink and starts tracer spans (flush-lock, flush, persist, storage-batch-insert); reportFlushMetrics records ingest delay from IngestedAt/StoredAt. (`ctx, flushSpan := s.config.Tracer.Start(ctx, "flush", trace.WithSpanKind(trace.SpanKindConsumer), ...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sink.go` | Sink struct, SinkConfig + Validate, NewSink, and the flush()/persistToStorage()/reportFlushMetrics() pipeline. The heart of the worker. | The storage->offset->dedupe ordering and the s.mu.Lock + pause/resume invariants protect the exactly-once-on-storage guarantee; do not reorder. |
| `buffer.go` | SinkBuffer (mutex map keyed by KafkaMessage.String()), Dequeue with transformers, RemoveByPartitions on revoke. | All access must hold b.mu; key collisions are intentional dedup, not a bug. |
| `storage.go` | Storage interface + ClickHouseStorage.BatchInsert mapping SinkMessage -> streaming.RawEvent with a fresh ulid StoreRowID and IngestedAt/StoredAt fallback to clock.Now(). | An error from BatchInsert retries the whole batch; mapping reads message.Serialized.* which must be non-nil. |
| `meters.go` | NamespacedMeterCache (meter cache keyed by namespace->eventType) with periodic refetch and GetAffectedMeters. | GetAffectedMeters returns nil for messages with Status.DropError set or unknown namespace; reads guarded by RWMutex. |
| `partition.go` | prettyPartitions helper for logging topic-partitions. | Purely diagnostic; handles nil Topic pointer. |

## Anti-Patterns

- Reordering the flush phases (storage, then offset store, then dedupe set) or removing the pause/resume + s.mu.Lock guard — breaks the at-least-once / exactly-once-on-storage guarantee.
- Running OnFlushSuccess synchronously inside flush() or letting its error fail the flush, causing re-processing of already-committed ClickHouse writes.
- Constructing Sink/Storage/MeterCache without calling config.Validate() first.
- Reading SinkBuffer.data or NamespacedMeterCache.namespaces without the corresponding mutex.
- Inserting into ClickHouse without a fresh StoreRowID (ulid.Make()) per RawEvent, or without falling back IngestedAt/StoredAt to clock.Now().

## Decisions

- **Flush persists to ClickHouse before committing Kafka offsets and before writing Redis dedupe keys.** — If offset commit or dedupe write fails after storage, messages are reprocessed (at-least-once) rather than lost; the comments in flush() document this 'exactly once'-on-storage reasoning.
- **Meters are cached per-namespace in-process and refetched on an interval rather than queried per event.** — Event validation runs on the hot path; a cached MetersByType keyed by namespace/eventType avoids a DB round-trip per message.
- **Post-flush side effects go through an optional FlushEventHandler run on a detached, time-boxed goroutine.** — Keeps ingest-notification and other side effects off the latency-critical flush path and prevents them from failing committed writes.

## Example: Constructor validates config then wires OTel + cache (the standard New* shape in this package)

```
func NewSink(config SinkConfig) (*Sink, error) {
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("invalid sink configuration: %w", err)
    }
    messageCounter, err := config.MetricMeter.Int64Counter("sink.kafka.messages", metric.WithUnit("{message}"))
    if err != nil {
        return nil, fmt.Errorf("failed to create messages counter: %w", err)
    }
    meterCache, err := NewNamespaceStore(NamespacedMeterCacheConfig{
        PeriodicRefetchInterval: config.MeterRefetchInterval,
        Logger:                  config.Logger,
        MeterService:            config.MeterService,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create namespace meter cache: %w", err)
// ...
```

<!-- archie:ai-end -->
