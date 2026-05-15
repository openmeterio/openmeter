# meterevent

<!-- archie:ai-start -->

> Public API surface for listing metered usage events from ClickHouse; owns the Service interface, Event domain type, ListEventsParams/ListEventsV2Params with hard constraints (32-day lookback, 100-event limit), and cursor-based pagination. The adapter/ and httphandler/ sub-packages implement and expose the service without touching Postgres/Ent.

## Patterns

**Params.Validate() before delegation** — Both ListEventsParams and ListEventsV2Params expose Validate() using errors.Join for multi-error aggregation. Callers (adapter, httphandler) must invoke Validate() before delegating to streaming.Connector — limits and filter constraints are enforced here, not at the adapter. (`if err := params.Validate(); err != nil { return nil, err }`)
**SortBy-aware Cursor emission** — Event.Cursor() switches on the SortBy field (Time/IngestedAt/StoredAt) to pick the correct timestamp for pagination. StoreRowID is always the tiebreak. New sort fields require a matching case in Cursor() and a matching SortBy.Validate() entry. (`case streaming.EventSortFieldIngestedAt: return pagination.NewCursor(e.IngestedAt, e.StoreRowID)`)
**ValidationErrors as per-event field, not top-level error** — Individual event validation failures are attached to Event.ValidationErrors []error. The service method still returns a result slice — callers inspect per-event errors without a blanket failure. (`event.ValidationErrors = append(event.ValidationErrors, validationErr)`)
**CustomerID filter: only $in supported** — ListEventsV2Params.Validate() explicitly rejects CustomerID filters that are non-empty but do not use .In. This constraint is enforced in Validate(), not in the adapter or httphandler. (`if !p.CustomerID.IsEmpty() && p.CustomerID.In == nil { errs = append(errs, errors.New("customer id filter supports only in")) }`)
**NextCursor emitted only on full page** — The adapter must omit NextCursor when the result page is smaller than effectiveLimit — emitting a cursor on a partial page causes callers to make a redundant empty-result request. (`if len(events) == effectiveLimit { result.NextCursor = &lastCursor }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service interface, Event domain type (including Cursor() and SortBy field), ListEventsParams, ListEventsV2Params, MaximumFromDuration (32d), MaximumLimit (100). Source of truth for all validation logic. | Adding a new filter field requires a ValidateWithComplexity clause in Validate(); adding a sort field requires both a SortBy constant and a Cursor() case. Omitting either lets invalid input reach ClickHouse. |
| `adapter/adapter.go` | Single service implementation — delegates to streaming.Connector, enriches results with customer IDs (with per-call cache), attaches ValidationErrors per event. | Must not import openmeter/ent/db. Must not emit NextCursor when result page is smaller than effective limit. Subject→customerID cache is per-call, not process-wide. |
| `httphandler/mapping.go` | All API↔domain type conversions for both v1 and v2 handlers. Centralises field mappings. | Inlining conversions in event.go or event_v2.go violates the mapping.go convention and makes future API changes harder to audit. |
| `httphandler/handler.go` | Defines the composite Handler interface embedding per-endpoint sub-interfaces; wires sub-handlers for server registration. | New endpoints need a new sub-interface added here AND registered in openmeter/server/router. |

## Anti-Patterns

- Importing openmeter/ent/db in the adapter — this is a streaming-only (ClickHouse) package with no Postgres access
- Returning a top-level error for individual event validation failures instead of attaching to Event.ValidationErrors
- Emitting NextCursor when the result page is smaller than effectiveLimit
- Skipping params.Validate() before delegating to streaming.Connector — hard limits bypass
- Inlining API↔domain type conversions in event.go or event_v2.go instead of mapping.go

## Decisions

- **ValidationErrors are per-event fields, not top-level errors** — Allows partial results with inline error annotations rather than failing the entire listing for one invalid event — consumers can decide how to handle individual bad events.
- **Hard time-window and count limits (32 days, 100 events) enforced in Validate(), not in adapter or handler** — Centralising limits in the domain params type makes them enforceable regardless of which caller invokes the service, and enables unit tests without standing up adapters.

## Example: Adding a new sort field — requires changes in three places

```
// 1. Add constant to streaming package: EventSortFieldMyField
// 2. In service.go Event.Cursor():
case streaming.EventSortFieldMyField:
    return pagination.NewCursor(e.MyFieldTime, e.StoreRowID)
// 3. streaming.EventSortField.Validate() must accept the new value
// ListEventsV2Params.Validate() calls p.SortBy.Validate() automatically
```

<!-- archie:ai-end -->
