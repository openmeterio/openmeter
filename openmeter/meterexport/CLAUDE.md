# meterexport

<!-- archie:ai-start -->

> Domain root for exporting/replaying meter data: defines the meterexport.Service interface that re-queries existing meter values and re-emits them as synthetic, pre-aggregated streaming.RawEvent records (one event per WindowSize). The single child (service) holds the channel/errgroup streaming implementation.

## Patterns

**Service interface + config/param types at root** — service.go declares the Service interface (GetTargetMeterDescriptor, ExportSyntheticMeterData, ExportSyntheticMeterDataIter), TargetMeterDescriptor, DataExportConfig, and DataExportParams. The concrete impl lives in package meterexportservice. (`type Service interface { GetTargetMeterDescriptor(...); ExportSyntheticMeterData(...); ExportSyntheticMeterDataIter(...) }`)
**Streaming export via channels, errors out-of-band** — ExportSyntheticMeterData(ctx, params, result chan<- streaming.RawEvent, err chan<- error) returns an error only if it fails to START; per-item failures go on the err channel and callers cancel ctx to stop. (`ExportSyntheticMeterData(ctx, params, result chan<- streaming.RawEvent, err chan<- error) error`)
**Iterator wrapper mirrors the channel API** — ExportSyntheticMeterDataIter returns iter.Seq2[streaming.RawEvent, error]; breaking the range loop auto-cancels the operation and errors are yielded inline with a zero-valued event. (`for event, err := range seq { if err != nil { ... } process(event) }`)
**Config/Params validate by accumulation** — DataExportConfig.Validate and DataExportParams.Validate collect into var errs []error and return errors.Join(errs...); Params embeds DataExportConfig and adds a timeutil.StartBoundedPeriod. (`if c.ExportWindowSize == "" { errs = append(errs, errors.New("export window size is required")) }`)
**Timezone-aware JSON via custom Marshal/Unmarshal** — DataExportConfig has explicit MarshalJSON/UnmarshalJSON that serialize *time.Location as its IANA name string (empty => nil) using the dataExportConfigJSON shadow struct. (`if raw.ExportWindowTimeZone != "" { loc, err := time.LoadLocation(raw.ExportWindowTimeZone); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, TargetMeterDescriptor, DataExportConfig/DataExportParams with Validate() and JSON marshalling. | ExportWindowTimeZone is a *time.Location; nil marshals to "" and does NOT round-trip back to a non-nil location. Only SUM and COUNT meters are supported; GroupBy/Customers not honored. |
| `service_test.go` | JSON round-trip and UnmarshalJSON tests for DataExportConfig, including the invalid-timezone error path. | Round-trip assertions skip the nil-timezone case on purpose; preserve that when changing marshalling. |

## Anti-Patterns

- Returning a runtime error from the export mid-stream instead of sending it to the err channel.
- Closing the result/err channels anywhere other than the producer's top-level defer.
- Adding GroupBy/Customer/ClientID query support without honoring the documented unsupported-param rejects.
- Assuming a nil ExportWindowTimeZone round-trips through JSON (it serializes to empty and decodes to nil).

## Decisions

- **Export is streamed via channels + errgroup rather than collecting a slice.** — Meters can hold huge event volumes; pre-aggregating one event per WindowSize and streaming avoids buffering the entire reconstruction in memory.
- **Both a channel API and an iter.Seq2 wrapper are exposed.** — The channel form gives callers explicit lifecycle/cancellation control; the iterator form is ergonomic for simple range-loop consumers with inline error handling.

## Example: Streaming export interface signature

```
// per-item errors go to the err channel; a returned error only means start failed
ExportSyntheticMeterData(
	ctx context.Context,
	params DataExportParams,
	result chan<- streaming.RawEvent,
	err chan<- error,
) error
```

<!-- archie:ai-end -->
