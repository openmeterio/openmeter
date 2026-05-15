# meterexport

<!-- archie:ai-start -->

> Streaming export of pre-aggregated synthetic meter events from ClickHouse; converts SUM/COUNT meter rows into RawEvents suitable for re-ingestion. The root package owns the Service interface, DataExportConfig, DataExportParams, and TargetMeterDescriptor; the service/ sub-package holds the dual-channel streaming implementation.

## Patterns

**Config/Params Validate() before delegation** — DataExportConfig and DataExportParams both expose Validate() using errors.Join. The service/ sub-package calls Validate() at the start of every public method before touching ClickHouse. (`if err := params.Validate(); err != nil { return err }`)
**Dual-channel streaming result** — ExportSyntheticMeterData writes events to result chan<- streaming.RawEvent and errors to err chan<- error. Channel ownership is exclusive to the producing goroutine — channels are created by the caller and closed via defer in the producer. Never close channels outside the producing goroutine. (`func (s *svc) ExportSyntheticMeterData(ctx context.Context, params DataExportParams, result chan<- streaming.RawEvent, err chan<- error) error`)
**iter.Seq2 wrapper as preferred caller API** — ExportSyntheticMeterDataIter wraps the dual-channel method in iter.Seq2[streaming.RawEvent, error] — handles channel management and context cancellation internally. Early break from the range loop cancels the producer context. (`for event, err := range seq { if err != nil { handle(err) }; process(event) }`)
**Custom JSON marshal/unmarshal for time.Location** — DataExportConfig implements MarshalJSON/UnmarshalJSON to serialise ExportWindowTimeZone as an IANA name string. Nil timezone marshals to empty string and unmarshals back to nil. (`func (c DataExportConfig) MarshalJSON() ([]byte, error) { tzName := ""; if c.ExportWindowTimeZone != nil { tzName = c.ExportWindowTimeZone.String() }; ... }`)
**Aggregation whitelist in validateAndGetMeter** — Only SUM and COUNT aggregations are supported. validateAndGetMeter in service/syntheticdata.go rejects all other aggregation types before any export query is issued. Adding a new aggregation to createEventFromMeterRow requires a matching whitelist entry in validateAndGetMeter. (`switch meter.Aggregation { case meter.MeterAggregationSum, meter.MeterAggregationCount: // ok; default: return error }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service interface (GetTargetMeterDescriptor, ExportSyntheticMeterData, ExportSyntheticMeterDataIter), DataExportConfig, DataExportParams, TargetMeterDescriptor, and their Validate()/JSON methods. | Adding a new aggregation type here requires whitelisting it in service/syntheticdata.go validateAndGetMeter switch AND createEventFromMeterRow — missing either causes runtime rejection after successful validation. |
| `service/funnel.go` | Windowed pagination loop that drives the ClickHouse query and feeds the result channel. | Never close resultCh or errCh outside funnel — channel ownership is exclusive to this function via defer close. Double-close panics. |
| `service/syntheticdata_iter.go` | iter.Seq2 wrapper — the idiomatic range-based API for callers. | Early break from the range loop must cancel the producer context to stop the funnel goroutine; otherwise goroutine leaks on partial iteration. |
| `service/syntheticdata.go` | validateAndGetMeter rejects unsupported aggregation types; createEventFromMeterRow constructs RawEvents from ClickHouse rows. | Aggregation whitelist must be kept in sync with createEventFromMeterRow — whitelist validates, createEvent converts; divergence causes silent data loss. |

## Anti-Patterns

- Adding a new aggregation type to createEventFromMeterRow without whitelisting it in validateAndGetMeter
- Closing resultCh or errCh outside the producing funnel goroutine — causes double-close panic
- Calling ExportSyntheticMeterData synchronously without draining both channels concurrently — deadlocks when buffered channels fill
- Adding Ent/DB dependencies to this package — depends only on streaming.Connector and meter.Service interfaces
- Skipping sync.Once guard for context-cancellation errors in errCh — causes duplicate ctx.Err() entries callers cannot distinguish

## Decisions

- **Dual-channel streaming rather than returning a slice** — Export can span many ClickHouse window pages; buffering all results in memory before returning would OOM for large time ranges. Channels allow backpressure and early cancellation.
- **Only SUM and COUNT aggregations supported** — Pre-aggregated synthetic events are only reversible for additive (SUM) and countable (COUNT) functions. MAX and UNIQUE_COUNT lose information when windowed and cannot be safely reconstructed.

## Example: Consuming exports via the iter.Seq2 API with early-exit safety

```
seq, err := svc.ExportSyntheticMeterDataIter(ctx, params)
if err != nil {
    return fmt.Errorf("start export: %w", err)
}
for event, iterErr := range seq {
    if iterErr != nil {
        // non-fatal: log and continue, or cancel ctx to abort
        log.Warn("export error", "err", iterErr)
        continue
    }
    if err := ingest(ctx, event); err != nil {
        return err // early break cancels producer goroutine automatically
    }
}
```

<!-- archie:ai-end -->
