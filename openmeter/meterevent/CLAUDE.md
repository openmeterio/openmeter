# meterevent

<!-- archie:ai-start -->

> Public API surface for listing metered usage events from ClickHouse; owns the Service interface, the Event domain type, ListEventsParams/ListEventsV2Params with hard constraints (32-day lookback, 100-event limit) and cursor-based pagination. The adapter/ and httphandler/ sub-packages implement and expose the service without touching Postgres/Ent.

## Patterns

**Params.Validate() before delegation** — Both ListEventsParams and ListEventsV2Params expose Validate() via errors.Join; callers (adapter, httphandler) must call Validate() before delegating to streaming.Connector — limits and filter constraints are enforced here, not at the adapter. (`if err := params.Validate(); err != nil { return nil, err }`)
**SortBy-aware Cursor emission** — Event.Cursor() switches on SortBy (Time/IngestedAt/StoredAt) to pick the pagination timestamp; StoreRowID is always the tiebreak. (`case streaming.EventSortFieldIngestedAt: return pagination.NewCursor(e.IngestedAt, e.StoreRowID)`)
**ValidationErrors as a per-event field** — Individual event validation failures are attached to Event.ValidationErrors []error; the method still returns the result slice so callers inspect per-event errors without a blanket failure. (`event.ValidationErrors = append(event.ValidationErrors, validationErr)`)
**CustomerID filter: only $in supported** — ListEventsV2Params.Validate() rejects a non-empty CustomerID filter that does not use .In — enforced in Validate(), not the adapter or handler. (`if !p.CustomerID.IsEmpty() && p.CustomerID.In == nil { errs = append(errs, errors.New("customer id filter supports only in")) }`)
**NextCursor emitted only on a full page** — The adapter omits NextCursor when the result page is smaller than effectiveLimit, avoiding a redundant empty-result follow-up request. (`if len(events) == effectiveLimit { result.NextCursor = &lastCursor }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, Event domain type (Cursor() + SortBy), ListEventsParams, ListEventsV2Params, MaximumFromDuration (32d), MaximumLimit (100). Source of truth for all validation. | Adding a filter field needs a ValidateWithComplexity clause; adding a sort field needs both a SortBy constant and a Cursor() case — omitting either lets invalid input reach ClickHouse. |
| `adapter/adapter.go` | Single service implementation: delegates to streaming.Connector, enriches with customer IDs (per-call cache), attaches ValidationErrors per event. | Must not import openmeter/ent/db. Must not emit NextCursor on a partial page. Subject→customerID cache is per-call, not process-wide. |
| `httphandler/mapping.go` | All API↔domain conversions for v1 and v2 handlers, centralised. | Inlining conversions in event.go/event_v2.go violates the mapping.go convention and makes future API changes harder to audit. |
| `httphandler/handler.go` | Composite Handler interface embedding per-endpoint sub-interfaces; wires sub-handlers for server registration. | New endpoints need a sub-interface here AND registration in openmeter/server/router. |

## Anti-Patterns

- Importing openmeter/ent/db in the adapter — this is a streaming-only (ClickHouse) package with no Postgres access
- Returning a top-level error for individual event validation failures instead of attaching to Event.ValidationErrors
- Emitting NextCursor when the result page is smaller than effectiveLimit
- Skipping params.Validate() before delegating to streaming.Connector — hard limits bypass
- Inlining API↔domain conversions in event.go or event_v2.go instead of mapping.go

## Decisions

- **ValidationErrors are per-event fields, not top-level errors** — Allows partial results with inline error annotations rather than failing the entire listing for one bad event.
- **Hard time-window and count limits (32 days, 100 events) enforced in Validate()** — Centralising limits in the domain params type makes them enforceable regardless of caller and unit-testable without adapters.

<!-- archie:ai-end -->
