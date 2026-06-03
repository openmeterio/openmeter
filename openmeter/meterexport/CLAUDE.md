# meterexport

<!-- archie:ai-start -->

> Streaming export of pre-aggregated synthetic meter events from ClickHouse; converts SUM/COUNT meter rows into RawEvents suitable for re-ingestion. The root package owns the Service interface, DataExportConfig, DataExportParams, and TargetMeterDescriptor; the service/ sub-package holds the dual-channel streaming implementation.

## Patterns

**Config/Params Validate() before delegation** — DataExportConfig and DataExportParams expose Validate() via errors.Join; the service calls Validate() at the start of every public method before touching ClickHouse. (`if err := params.Validate(); err != nil { return err }`)
**Dual-channel streaming result** — ExportSyntheticMeterData writes events to result chan<- streaming.RawEvent and errors to err chan<- error; channels are created by the caller and closed via defer only in the producing goroutine. (`func (s *svc) ExportSyntheticMeterData(ctx, params DataExportParams, result chan<- streaming.RawEvent, err chan<- error) error`)
**iter.Seq2 wrapper as the preferred caller API** — ExportSyntheticMeterDataIter wraps the dual-channel method in iter.Seq2[streaming.RawEvent, error]; early break from the range loop cancels the producer context. (`for event, err := range seq { if err != nil { handle(err) }; process(event) }`)
**Custom JSON marshal/unmarshal for time.Location** — DataExportConfig marshals ExportWindowTimeZone as an IANA name string; nil marshals to empty string and unmarshals back to nil. (`func (c DataExportConfig) MarshalJSON() ([]byte, error) { tzName := ""; if c.ExportWindowTimeZone != nil { tzName = c.ExportWindowTimeZone.String() }; ... }`)
**Aggregation whitelist in validateAndGetMeter** — Only SUM and COUNT are supported; validateAndGetMeter rejects all others before any export query, and must stay in sync with createEventFromMeterRow. (`switch meter.Aggregation { case meter.MeterAggregationSum, meter.MeterAggregationCount: /* ok */; default: return error }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (GetTargetMeterDescriptor, ExportSyntheticMeterData, ExportSyntheticMeterDataIter), DataExportConfig, DataExportParams, TargetMeterDescriptor and their Validate()/JSON methods. | Adding an aggregation type here requires whitelisting it in service/syntheticdata.go validateAndGetMeter AND createEventFromMeterRow — missing either causes runtime rejection after successful validation. |
| `service/funnel.go` | Windowed pagination loop driving the ClickHouse query and feeding the result channel. | Never close resultCh/errCh outside funnel — channel ownership is exclusive via defer close; double-close panics. |
| `service/syntheticdata_iter.go` | iter.Seq2 wrapper — the idiomatic range-based caller API. | Early break must cancel the producer context to stop the funnel goroutine; otherwise the goroutine leaks on partial iteration. |
| `service/syntheticdata.go` | validateAndGetMeter rejects unsupported aggregations; createEventFromMeterRow constructs RawEvents from ClickHouse rows. | The whitelist must stay in sync with createEventFromMeterRow — divergence causes silent data loss. |

## Anti-Patterns

- Adding a new aggregation type to createEventFromMeterRow without whitelisting it in validateAndGetMeter
- Closing resultCh or errCh outside the producing funnel goroutine — causes a double-close panic
- Calling ExportSyntheticMeterData synchronously without draining both channels concurrently — deadlocks when buffers fill
- Adding Ent/DB dependencies — depends only on streaming.Connector and meter.Service interfaces
- Skipping the sync.Once guard for context-cancellation errors in errCh — causes duplicate ctx.Err() entries

## Decisions

- **Dual-channel streaming rather than returning a slice** — Exports can span many ClickHouse window pages; buffering all results would OOM for large ranges. Channels allow backpressure and early cancellation.
- **Only SUM and COUNT aggregations supported** — Pre-aggregated synthetic events are only reversible for additive (SUM) and countable (COUNT) functions; MAX/UNIQUE_COUNT lose information when windowed.

<!-- archie:ai-end -->
