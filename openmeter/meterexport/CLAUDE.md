# meterexport

<!-- archie:ai-start -->

> Streaming export of pre-aggregated synthetic meter events; converts ClickHouse SUM/COUNT meter rows into RawEvents suitable for re-ingestion. The root package owns the Service interface and config/params types; implementation lives in the service/ sub-package.

## Patterns

**Config.Validate() before construction** — DataExportConfig and DataExportParams both expose Validate() using errors.Join. The service/ sub-package calls Validate() at the start of every public method. (`if err := params.Validate(); err != nil { return err }`)
**Dual-channel streaming result** — ExportSyntheticMeterData writes events to result chan<- streaming.RawEvent and errors to err chan<- error. Channels are owned and closed exclusively by the producing goroutine via defer close. (`func (s *svc) ExportSyntheticMeterData(ctx context.Context, params DataExportParams, result chan<- streaming.RawEvent, err chan<- error) error`)
**iter.Seq2 wrapper as preferred caller API** — ExportSyntheticMeterDataIter wraps the dual-channel method in an iter.Seq2[streaming.RawEvent, error] that handles channel management and context cancellation internally. (`for event, err := range seq { if err != nil { handle(err) }; process(event) }`)
**Custom JSON marshal/unmarshal for time.Location** — DataExportConfig implements MarshalJSON/UnmarshalJSON to serialize ExportWindowTimeZone as a string (IANA name). Nil timezone marshals to empty string and unmarshals back to nil. (`func (c DataExportConfig) MarshalJSON() ([]byte, error) { tzName := ""; if c.ExportWindowTimeZone != nil { tzName = c.ExportWindowTimeZone.String() } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service interface, TargetMeterDescriptor, DataExportConfig, DataExportParams, and their Validate()/JSON methods. Source of truth for supported aggregations constraint (SUM, COUNT only). | Adding a new aggregation type requires whitelisting it in service/syntheticdata.go validateAndGetMeter switch AND createEventFromMeterRow. |
| `service/funnel.go` | Windowed pagination logic that drives the ClickHouse query loop and feeds the result channel. | Never close resultCh or errCh outside of funnel — channel ownership is exclusive. |
| `service/syntheticdata_iter.go` | iter.Seq2 wrapper — the idiomatic API for callers that want range-based iteration. | Early break from range loop must cancel context to stop the producer goroutine. |

## Anti-Patterns

- Adding a new aggregation type to createEventFromMeterRow without whitelisting in validateAndGetMeter
- Closing resultCh or errCh outside the producing funnel goroutine
- Calling ExportSyntheticMeterData synchronously without draining both channels — deadlocks on buffer overflow
- Adding Ent/DB dependencies — this package depends only on streaming.Connector and meter.Service
- Skipping sync.Once guard for context-cancellation errors in errCh — causes duplicate entries

## Decisions

- **Dual-channel streaming rather than returning a slice** — Export can span many ClickHouse window pages; buffering all results in memory before returning would OOM for large time ranges.
- **Only SUM and COUNT aggregations supported** — Pre-aggregated synthetic events are only reversible/reconstructable for additive (SUM) and countable (COUNT) aggregation functions; other types (MAX, UNIQUE_COUNT) lose information when windowed.

<!-- archie:ai-end -->
