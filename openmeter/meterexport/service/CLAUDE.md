# service

<!-- archie:ai-start -->

> Service implementation for meter data export. Reads aggregated meter rows from ClickHouse via streaming.Connector, converts them to synthetic RawEvents, and streams results to callers via channels or Go iterators. Supports only SUM and COUNT aggregations.

## Patterns

**Config struct with validate()** — Constructor accepts a Config struct; Config.validate() uses errors.Join to collect all missing-field errors before returning. New() returns (*service, error). (`func New(config Config) (*service, error) { if err := config.validate(); err != nil { return nil, err } ... }`)
**Interface compliance assertion** — var _ meterexport.Service = (*service)(nil) at the bottom of service.go enforces compile-time interface satisfaction. (`var _ meterexport.Service = (*service)(nil)`)
**Streaming via dual channels (resultCh + errCh)** — Streaming methods accept chan<- T and chan<- error; both channels are always closed by defer in the producing goroutine. Callers must drain both channels. (`func (s *service) ExportSyntheticMeterData(ctx context.Context, params ..., resultCh chan<- streaming.RawEvent, errCh chan<- error) error`)
**errgroup for producer/consumer goroutines** — ExportSyntheticMeterData spawns two goroutines via errgroup.WithContext: one for funnel (producer) and one for row-to-event conversion (consumer). sync.Once guards sending ctx cancellation errors exactly once. (`g, ctx := errgroup.WithContext(ctx); g.Go(func() error { return s.funnel(...) }); g.Go(func() error { /* consumer */ })`)
**iter.Seq2 wrapper for channel-based export** — ExportSyntheticMeterDataIter wraps ExportSyntheticMeterData in a Go 1.23 iter.Seq2[RawEvent, error]. Creates a child context; cancels it when caller breaks early. Validates upfront before returning the iterator. (`func (s *service) ExportSyntheticMeterDataIter(ctx context.Context, params ...) (iter.Seq2[streaming.RawEvent, error], error)`)
**Windowed pagination in funnel** — funnel() advances queryFrom/queryTo by TARGET_ROWS_PER_QUERY (500) windows per iteration via iterateQueryTime. Checks ctx.Err() at the top of each loop iteration for cancellation. (`const TARGET_ROWS_PER_QUERY = 500; for { if ctx.Err() != nil { errCh <- ctx.Err(); return nil }; ... }`)
**validateAndGetMeter reuse** — Both GetTargetMeterDescriptor and ExportSyntheticMeterData delegate to validateAndGetMeter to avoid duplicating config validation + meter fetch + aggregation check logic. (`func (s *service) validateAndGetMeter(ctx context.Context, config meterexport.DataExportConfig) (meter.Meter, meterexport.TargetMeterDescriptor, error)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct definition, constructor New(), interface assertion. Embeds Config in service struct for direct field access. | Config embedding means all Config fields are promoted to service; avoid name collisions when adding new fields. |
| `syntheticdata.go` | Core export logic: validateAndGetMeter, ExportSyntheticMeterData, createEventFromMeterRow. Only SUM and COUNT aggregations are handled; all others return an error. | Aggregation whitelist in createEventFromMeterRow must stay in sync with the switch in validateAndGetMeter. Output events always use SUM_VALUE_PROPERTY_KEY='value' regardless of source aggregation. |
| `funnel.go` | Windowed ClickHouse query loop. Owns funnelParams, iterateQueryTime, TARGET_ROWS_PER_QUERY constant. | funnel always closes resultCh and errCh via defer — callers must not close them. Context cancellation sends ctx.Err() to errCh before returning; consumer must deduplicate with sync.Once. |
| `syntheticdata_iter.go` | iter.Seq2 wrapper. Starts ExportSyntheticMeterData in a goroutine; uses select across resultCh, errCh, and startupErrCh. Cancels child context when yield returns false. | startupErrCh is nilled after close to prevent repeated select hits on a closed channel. |
| `service_test.go` | Unit tests. Uses MockMeterService (local) and testutils.NewMockStreamingConnector for all tests. No database needed. | Tests use context.Background() — acceptable here since t.Context() is not always available in table-driven subtests with goroutines. |

## Anti-Patterns

- Adding a new aggregation type to createEventFromMeterRow without also whitelisting it in validateAndGetMeter's switch.
- Closing resultCh or errCh outside of funnel/ExportSyntheticMeterData — ownership is exclusive to the producing function via defer close.
- Skipping the sync.Once guard for context-cancellation errors, causing duplicate ctx.Err() entries in errCh.
- Calling ExportSyntheticMeterData synchronously without draining both channels — will deadlock on buffered-channel overflow.
- Adding DB/Ent dependencies directly to this package — it depends only on streaming.Connector and meter.Service interfaces.

## Decisions

- **Dual-channel streaming instead of returning a slice** — Meter exports can span large time ranges; buffering all rows in memory would OOM. Channels allow backpressure and early cancellation.
- **Always normalize output to SUM aggregation (SUM_VALUE_PROPERTY_KEY)** — Synthetic re-ingestion requires a single canonical event shape. COUNT meters are representable as SUM=1 per occurrence; AVG/UNIQUE_COUNT cannot be safely decomposed so they are rejected.
- **iter.Seq2 wrapper as the preferred caller API** — Go 1.23 range-over-function syntax is cleaner than manual channel management; ExportSyntheticMeterDataIter handles context cancellation on early break automatically.

## Example: Consuming exported events via the iterator API

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/meterexport"
	meterexportservice "github.com/openmeterio/openmeter/openmeter/meterexport/service"
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
