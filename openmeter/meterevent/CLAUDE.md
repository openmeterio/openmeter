# meterevent

<!-- archie:ai-start -->

> Domain root for reading raw metering events back out of the streaming store (ClickHouse). Defines the meterevent.Service interface (ListEvents v1 and cursor-paginated ListEventsV2), the enriched Event value type, and the param/validation contracts; children split into adapter (streaming-backed query) and httphandler (HTTP transport).

## Patterns

**Service interface defined at package root** — service.go declares the Service interface with ListEvents/ListEventsV2; adapter and httphandler depend on this interface, never on each other's concrete types. (`type Service interface { ListEvents(...); ListEventsV2(...) }`)
**Params carry their own Validate()** — Each input struct (ListEventsParams, ListEventsV2Params) implements Validate() that accumulates into var errs []error and returns errors.Join(errs...); field errors wrapped with fmt.Errorf("field: %w", err). (`if i.Namespace == "" { errs = append(errs, errors.New("namespace is required")) }`)
**Event implements pagination.Item via Cursor()** — var _ pagination.Item = (*Event)(nil); Cursor() switches on Event.SortBy (EventSortFieldIngestedAt/StoredAt/Time) and pairs the matching timestamp with StoreRowID as the keyset tiebreak. (`case streaming.EventSortFieldIngestedAt: return pagination.NewCursor(e.IngestedAt, e.StoreRowID)`)
**Cursor column must match query ORDER BY** — SortBy on the Event must be the same column used to build the streaming query, or keyset pagination loses/duplicates rows. The adapter must propagate SortBy onto each returned Event. (`default: // EventSortFieldTime and zero value -> pagination.NewCursor(e.Time, e.StoreRowID)`)
**Time-window and limit bounds are hard constants** — MaximumFromDuration (32 days) and MaximumLimit (100) bound every query; From must be after now-MaximumFromDuration and Limit in [1, MaximumLimit]. (`minimumFrom := time.Now().Add(-MaximumFromDuration)`)
**CustomerID filter supports only $in** — ListEventsV2Params.Validate rejects a CustomerID FilterString that is non-empty but has In == nil. (`if !p.CustomerID.IsEmpty() && p.CustomerID.In == nil { errs = append(errs, errors.New("customer id filter supports only in")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, Event type, ListEventsParams/ListEventsV2Params with Validate(), MaximumFromDuration/MaximumLimit constants, and Event.Cursor(). | Event.Cursor() must stay in sync with the streaming ORDER BY; adding a new EventSortField requires a new switch case and SortBy.Validate() acceptance. |
| `service_test.go` | Table-driven tests for ListEventsV2Params.Validate and Event.Cursor across all SortBy values. | Cursor tests assert StoreRowID is always the tiebreak ID regardless of SortBy; keep that invariant when editing Cursor(). |

## Anti-Patterns

- Returning an error from list when a single event fails validation/customer-lookup instead of attaching to Event.ValidationErrors.
- Letting Event.SortBy diverge from the column the adapter sorted by — breaks keyset pagination.
- Allowing a CustomerID FilterString with Eq (anything other than In) through Validate.
- Skipping params.Validate() at the top of a Service method.
- Bypassing the 32-day / limit-100 bounds when adding new query paths.

## Decisions

- **Events are read from the streaming store and enriched (customer-ID, meter validation) in the adapter, not in the streaming layer.** — Keeps streaming.Connector domain-agnostic; meterevent owns the enriched Event shape and per-event validation semantics.
- **Two list methods (v1 slice, v2 cursor) coexist on one interface.** — v1 is the legacy ingested-event listing; v2 adds AIP-style FilterString/FilterTime filters and keyset pagination without breaking v1 callers.

## Example: Cursor selection keyed on SortBy

```
func (e Event) Cursor() pagination.Cursor {
	switch e.SortBy {
	case streaming.EventSortFieldIngestedAt:
		return pagination.NewCursor(e.IngestedAt, e.StoreRowID)
	case streaming.EventSortFieldStoredAt:
		return pagination.NewCursor(e.StoredAt, e.StoreRowID)
	default:
		return pagination.NewCursor(e.Time, e.StoreRowID)
	}
}
```

<!-- archie:ai-end -->
