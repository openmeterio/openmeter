# service

<!-- archie:ai-start -->

> Service implementation for meter data export: reads aggregated meter rows from ClickHouse via streaming.Connector, converts them to synthetic RawEvents, and streams results via Go iterators or dual channels. Supports only SUM and COUNT aggregations; AVG/UNIQUE_COUNT are rejected at validation.

## Patterns

**Config struct with validate()** — Constructor accepts a Config; Config.validate() collects missing-field errors via errors.Join. New() returns (*service, error). (`func New(config Config) (*service, error) { if err := config.validate(); err != nil { return nil, err }; return &service{Config: config}, nil }`)
**Interface compliance assertion** — var _ meterexport.Service = (*service)(nil) at the bottom of service.go enforces compile-time satisfaction. (`var _ meterexport.Service = (*service)(nil)`)
**Streaming via dual channels (resultCh + errCh)** — Streaming methods accept chan<- T and chan<- error; both are always closed by defer in the producer. Callers must drain both before inspecting results. (`defer func() { close(resultCh); close(errCh) }()`)
**errgroup for producer/consumer goroutines** — ExportSyntheticMeterData spawns funnel (producer) and row-to-event conversion (consumer) via errgroup.WithContext; sync.Once guards sending ctx-cancellation errors exactly once. (`var sendCtxErrOnce sync.Once; g, ctx := errgroup.WithContext(ctx); g.Go(producerFn); g.Go(consumerFn)`)
**iter.Seq2 wrapper for channel export** — ExportSyntheticMeterDataIter wraps the channel method in a Go 1.23 iter.Seq2[RawEvent, error], validating upfront and cancelling the child context when yield returns false. (`seq := func(yield func(streaming.RawEvent, error) bool) { ctx, cancel := context.WithCancel(ctx); defer cancel(); ... }; return seq, nil`)
**Windowed pagination in funnel** — funnel() advances queryFrom/queryTo in TARGET_ROWS_PER_QUERY (500) window steps via iterateQueryTime, checking ctx.Err() at the top of each loop. (`const TARGET_ROWS_PER_QUERY = 500; for { if ctx.Err() != nil { errCh <- ctx.Err(); return nil }; rows, err := s.StreamingConnector.QueryMeter(ctx, ...) }`)
**validateAndGetMeter reuse** — GetTargetMeterDescriptor and ExportSyntheticMeterData both delegate to validateAndGetMeter (config validation + meter fetch + aggregation whitelist), normalising output to SUM with SUM_VALUE_PROPERTY_KEY='value'. (`switch m.Aggregation { case meter.MeterAggregationSum, meter.MeterAggregationCount: default: return ..., fmt.Errorf("unsupported meter aggregation: %s", m.Aggregation) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, New() constructor, interface compliance assertion. Config is embedded in service for promoted field access. | Config embedding promotes all fields to service — avoid name collisions; update the interface assertion if meterexport.Service gains methods. |
| `syntheticdata.go` | Core export logic: validateAndGetMeter, ExportSyntheticMeterData, createEventFromMeterRow. Only SUM and COUNT produce events. | The aggregation whitelist in createEventFromMeterRow must match the switch in validateAndGetMeter; output events always use SUM_VALUE_PROPERTY_KEY='value'. |
| `funnel.go` | Windowed ClickHouse query loop; funnelParams, iterateQueryTime, TARGET_ROWS_PER_QUERY. | funnel always closes resultCh and errCh via defer — callers must never close them; ctx cancellation sends ctx.Err() once (consumer dedups via sync.Once). |
| `syntheticdata_iter.go` | iter.Seq2 wrapper starting ExportSyntheticMeterData in a goroutine, interleaving resultCh/errCh/startupErrCh via select. | startupErrCh is nilled after close to avoid repeated select hits; early yield(false) triggers ctx cancel via defer — never manually cancel before returning from yield. |
| `service_test.go` | Unit tests with local MockMeterService and testutils.NewMockStreamingConnector; no DB/Ent dependency. | New aggregation types must be tested for both whitelist acceptance and rejection. |

## Anti-Patterns

- Adding an aggregation type to createEventFromMeterRow without whitelisting it in validateAndGetMeter's switch — runtime rejection after validation
- Closing resultCh or errCh outside funnel/ExportSyntheticMeterData — double-close panics
- Skipping the sync.Once guard for ctx-cancellation errors — duplicate ctx.Err() entries on errCh
- Calling ExportSyntheticMeterData synchronously without draining both channels concurrently — deadlocks when buffers fill
- Adding Ent/DB dependencies — this package depends only on streaming.Connector and meter.Service interfaces

## Decisions

- **Dual-channel streaming instead of returning a slice** — Exports can span large time ranges; buffering all rows would OOM. Channels allow backpressure and early cancellation via context.
- **Always normalise output to SUM aggregation (SUM_VALUE_PROPERTY_KEY='value')** — Synthetic re-ingestion needs one canonical event shape; COUNT is representable as SUM=1 per occurrence while AVG/UNIQUE_COUNT cannot be safely decomposed, so they are rejected upfront.
- **iter.Seq2 wrapper as the preferred caller API** — Go 1.23 range-over-function is cleaner than manual channel management; the wrapper handles ctx cancellation on early break via deferred cancel.

## Example: Consume exported events via the iterator API with early-break safety

```
func consume(svc *meterexportservice.service, params meterexport.DataExportParams) error {
	seq, err := svc.ExportSyntheticMeterDataIter(context.Background(), params)
	if err != nil {
		return fmt.Errorf("init export: %w", err)
	}
	for event, err := range seq {
		if err != nil {
			return err
		}
		_ = event
	}
	return nil
}
```

<!-- archie:ai-end -->
