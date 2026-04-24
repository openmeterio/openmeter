# meterevent

<!-- archie:ai-start -->

> Public API surface for listing metered usage events; owns Service interface, domain types (Event, ListEventsParams, ListEventsV2Params), and hard constraints (32-day lookback, 100-event limit). Adapter and HTTP handler sub-packages implement and expose the service.

## Patterns

**Params.Validate() before delegation** — Both ListEventsParams and ListEventsV2Params expose a Validate() method using errors.Join for multi-error aggregation. Callers (adapter, httphandler) must call Validate() before delegating to streaming.Connector. (`if err := params.Validate(); err != nil { return nil, err }`)
**SortBy-aware Cursor emission** — Event.Cursor() switches on SortBy (Time/IngestedAt/StoredAt) to pick the correct timestamp field. StoreRowID is always used as tiebreak. New sort fields require a matching case in Cursor(). (`case streaming.EventSortFieldIngestedAt: return pagination.NewCursor(e.IngestedAt, e.StoreRowID)`)
**ValidationErrors as per-event field** — Individual event validation failures are attached to Event.ValidationErrors []error, not returned as a top-level error from the service method. (`event.ValidationErrors = append(event.ValidationErrors, validationErr)`)
**CustomerID filter: only $in supported** — ListEventsV2Params.Validate() rejects any CustomerID filter that is not empty and does not use .In — enforced by explicit check in Validate(). (`if !p.CustomerID.IsEmpty() && p.CustomerID.In == nil { errs = append(errs, errors.New("customer id filter supports only in")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service interface, Event domain type, ListEventsParams, ListEventsV2Params, and the two const limits (MaximumFromDuration, MaximumLimit). Source of truth for validation logic. | Adding new filter fields requires a matching Validate() clause using ValidateWithComplexity; omitting it lets invalid filters reach ClickHouse. |
| `adapter/adapter.go` | Single service implementation — delegates to streaming.Connector, enriches results with customer IDs (with caching), and attaches ValidationErrors per event. | Do not import openmeter/ent/db here. Do not emit NextCursor when the result page is smaller than effective limit. |
| `httphandler/mapping.go` | All API↔domain type conversions. Must be the only place where field mappings live. | Inlining conversions in event.go or event_v2.go is an anti-pattern. |

## Anti-Patterns

- Importing openmeter/ent/db in adapter — this is a streaming-only package
- Returning top-level error for individual event validation failures instead of attaching to Event.ValidationErrors
- Emitting NextCursor when result page is smaller than the effective limit
- Skipping params.Validate() before calling streaming.Connector
- Inlining API↔domain conversions in handler files instead of mapping.go

## Decisions

- **ValidationErrors are per-event fields, not returned as top-level errors** — Allows partial results to be returned with inline error annotations rather than failing the entire listing operation for one bad event.
- **Hard time-window and count limits (32 days, 100 events) enforced in Validate(), not in adapter or handler** — Centralising limits in the domain params type makes them enforceable regardless of which caller invokes the service.

## Example: Adding a new sort field: requires both SortField constant and a Cursor() case

```
// In streaming package: add EventSortFieldMyField
// In service.go Event.Cursor():
case streaming.EventSortFieldMyField:
    return pagination.NewCursor(e.MyFieldTime, e.StoreRowID)
// In ListEventsV2Params.Validate(): SortBy.Validate() will reject unknown values automatically
```

<!-- archie:ai-end -->
