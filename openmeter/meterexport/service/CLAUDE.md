# service

<!-- archie:ai-start -->

> Service implementation for meter data export: reads aggregated meter rows from ClickHouse via streaming.Connector, converts them to synthetic RawEvents, and streams results to callers via Go iterators or dual channels. Supports only SUM and COUNT aggregations; AVG/UNIQUE_COUNT are rejected at validation time.

## Patterns

**Config struct with validate()** — Constructor accepts a Config struct; Config.validate() collects all missing-field errors via errors.Join before returning. New() returns (*service, error). (`func New(config Config) (*service, error) { if err := config.validate(); err != nil { return nil, err }; return &service{Config: config}, nil }`)
**Interface compliance assertion** — var _ meterexport.Service = (*service)(nil) at the bottom of service.go enforces compile-time interface satisfaction. (`var _ meterexport.Service = (*service)(nil)`)
**Streaming via dual channels (resultCh + errCh)** — Streaming methods accept chan<- T and chan<- error; both channels are always closed by defer in the producing function. Callers must drain both channels before inspecting results. (`func (s *service) ExportSyntheticMeterData(ctx context.Context, params meterexport.DataExportParams, resultCh chan<- streaming.RawEvent, errCh chan<- error) error { defer func() { close(resultCh); close(errCh) }(); ... }`)
**errgroup for producer/consumer goroutines** — ExportSyntheticMeterData spawns two goroutines via errgroup.WithContext: funnel (producer) and row-to-event conversion (consumer). sync.Once guards sending context-cancellation errors exactly once to prevent duplicates on errCh. (`var sendCtxErrOnce sync.Once; sendCtxErr := func() { sendCtxErrOnce.Do(func() { if err := ctx.Err(); err != nil { errCh <- err } }) }; g, ctx := errgroup.WithContext(ctx); g.Go(producerFn); g.Go(consumerFn)`)
**iter.Seq2 wrapper for channel-based export** — ExportSyntheticMeterDataIter wraps ExportSyntheticMeterData in a Go 1.23 iter.Seq2[RawEvent, error]. Performs upfront validation before returning the iterator. Cancels child context when yield returns false (early break). (`func (s *service) ExportSyntheticMeterDataIter(ctx context.Context, params meterexport.DataExportParams) (iter.Seq2[streaming.RawEvent, error], error) { if _, _, err := s.validateAndGetMeter(ctx, params.DataExportConfig); err != nil { return nil, err }; seq := func(yield func(streaming.RawEvent, error) bool) { ctx, cancel := context.WithCancel(ctx); defer cancel(); ... }; return seq, nil }`)
**Windowed pagination in funnel** — funnel() advances queryFrom/queryTo in TARGET_ROWS_PER_QUERY (500) window steps per iteration via iterateQueryTime. Checks ctx.Err() at the top of each loop iteration for early cancellation. (`const TARGET_ROWS_PER_QUERY = 500; for { if ctx.Err() != nil { errCh <- ctx.Err(); return nil }; rows, err := s.StreamingConnector.QueryMeter(ctx, ...); ... }`)
**validateAndGetMeter reuse** — Both GetTargetMeterDescriptor and ExportSyntheticMeterData delegate to validateAndGetMeter to avoid duplicating config validation + meter fetch + aggregation whitelist check. Always normalises output descriptor to SUM aggregation with SUM_VALUE_PROPERTY_KEY='value'. (`func (s *service) validateAndGetMeter(ctx context.Context, config meterexport.DataExportConfig) (meter.Meter, meterexport.TargetMeterDescriptor, error) { ... switch m.Aggregation { case meter.MeterAggregationSum, meter.MeterAggregationCount: default: return ..., fmt.Errorf("unsupported meter aggregation: %s", m.Aggregation) } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct definition, constructor New(), and interface compliance assertion. Config is embedded in service struct for promoted field access. | Config embedding promotes all Config fields to service; avoid name collisions when adding new fields. The interface assertion must be updated if the meterexport.Service interface gains new methods. |
| `syntheticdata.go` | Core export logic: validateAndGetMeter, ExportSyntheticMeterData, createEventFromMeterRow. Only SUM and COUNT aggregations produce output events. | Aggregation whitelist in createEventFromMeterRow must stay in sync with the switch in validateAndGetMeter. Output events always use SUM_VALUE_PROPERTY_KEY='value' regardless of source aggregation type. |
| `funnel.go` | Windowed ClickHouse query loop. Owns funnelParams, iterateQueryTime, and TARGET_ROWS_PER_QUERY constant. | funnel always closes resultCh and errCh via defer — callers must never close them. Context cancellation sends ctx.Err() to errCh before returning; consumer deduplicates via sync.Once. |
| `syntheticdata_iter.go` | iter.Seq2 wrapper that starts ExportSyntheticMeterData in a goroutine and interleaves resultCh, errCh, and startupErrCh via select. | startupErrCh is nilled after close to prevent repeated select hits on a closed channel. Early yield(false) triggers context cancel via defer — never manually cancel before returning from yield. |
| `service_test.go` | Unit tests using local MockMeterService and testutils.NewMockStreamingConnector. No database or Ent dependency. | Tests use context.Background() (acceptable for table-driven subtests with goroutines where t.Context() is not available). New aggregation types must be tested for both whitelist acceptance and rejection. |

## Anti-Patterns

- Adding a new aggregation type to createEventFromMeterRow without also whitelisting it in validateAndGetMeter's switch — causes runtime rejection after successful validation.
- Closing resultCh or errCh outside funnel/ExportSyntheticMeterData — ownership is exclusive to the producing function via defer close; double-close panics.
- Skipping the sync.Once guard for context-cancellation errors — causes duplicate ctx.Err() entries in errCh that callers cannot distinguish from distinct errors.
- Calling ExportSyntheticMeterData synchronously without draining both channels concurrently — will deadlock when buffered channels fill.
- Adding Ent/DB dependencies directly to this package — it depends only on streaming.Connector and meter.Service interfaces; persistence belongs in adapter layers.

## Decisions

- **Dual-channel streaming instead of returning a slice** — Meter exports can span large time ranges; buffering all rows in memory would OOM. Channels allow backpressure and early cancellation via context.
- **Always normalise output to SUM aggregation (SUM_VALUE_PROPERTY_KEY='value')** — Synthetic re-ingestion requires a single canonical event shape. COUNT meters are representable as SUM=1 per occurrence; AVG/UNIQUE_COUNT cannot be safely decomposed so they are rejected upfront.
- **iter.Seq2 wrapper as the preferred caller API** — Go 1.23 range-over-function syntax is cleaner than manual channel management; ExportSyntheticMeterDataIter handles context cancellation on early break automatically via deferred cancel.

## Example: Consuming exported events via the iterator API with early-break safety

```
import (
	"context"
	"fmt"

	meterexportservice "github.com/openmeterio/openmeter/openmeter/meterexport/service"
	"github.com/openmeterio/openmeter/openmeter/meterexport"
)

func consume(svc *meterexportservice.service, params meterexport.DataExportParams) error {
	seq, err := svc.ExportSyntheticMeterDataIter(context.Background(), params)
	if err != nil {
		return fmt.Errorf("init export: %w", err)
	}
	for event, err := range seq {
		if err != nil {
// ...
```

<!-- archie:ai-end -->
