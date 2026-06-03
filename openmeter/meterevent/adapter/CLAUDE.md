# adapter

<!-- archie:ai-start -->

> Implements meterevent.Service by composing streaming.Connector (ClickHouse), customer.Service, and meter.Service — no Ent/Postgres access. Resolves customer filters early, delegates raw queries to the streaming layer, then runs a fixed post-process pipeline (map → enrich → validate).

## Patterns

**Interface compliance assertion** — Declare var _ meterevent.Service = (*adapter)(nil) in adapter.go to enforce compile-time conformance. (`var _ meterevent.Service = (*adapter)(nil)`)
**Validate-before-delegate** — ListEvents and ListEventsV2 call params.Validate() and wrap any error in models.NewGenericValidationError before reaching the streaming connector. (`if err := params.Validate(); err != nil { return nil, models.NewGenericValidationError(fmt.Errorf("validate input: %w", err)) }`)
**Fixed post-process pipeline** — After fetching raw events always call eventPostProcess which runs mapEventsToMeterEvents → enrichEventsWithCustomerID → validateEvents in that order; never skip or reorder steps. (`meterEvents, err = a.eventPostProcess(ctx, params.Namespace, rawEvents)`)
**Early-empty-return on customer filter miss** — If a customerIDs filter is supplied but listCustomers returns zero results, return an empty slice immediately without querying the streaming layer. (`if len(customers) == 0 { return []meterevent.Event{}, nil }`)
**Subject→customerID per-call cache** — enrichEventsWithCustomerID builds a map[string]string cache keyed by subject to avoid repeated DB lookups within a single call; new enrichment helpers must replicate this. (`cache := make(map[string]string); if customerID, ok := cache[event.Subject]; ok { ... }`)
**Cursor emitted only on full page** — In ListEventsV2, set result.NextCursor only when len(meterEvents) == effectiveLimit; leave it nil for partial pages. (`if len(meterEvents) > 0 && len(meterEvents) == effectiveLimit { cursor := meterEvents[len(meterEvents)-1].Cursor(); result.NextCursor = &cursor }`)
**ValidationErrors attached per-event, not returned** — validateEvents appends errors to event.ValidationErrors for each invalid event and still appends the event to the output; it never returns a top-level error for individual event failures. (`event.ValidationErrors = validationErrors; validatedEvents = append(validatedEvents, event)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor New() and struct holding streamingConnector, customerService, meterService only. | Never add an Ent/DB client here — this adapter is streaming-only; persistence imports break domain isolation. |
| `event.go` | All method implementations: ListEvents, ListEventsV2, listCustomers, eventPostProcess, mapEventsToMeterEvents, validateEvents, enrichEventsWithCustomerID. | enrichEventsWithCustomerID has a FIXME: one DB call per event; the cache mitigates N+1 within one call but batching is a TODO — do not add more per-event service calls without a cache. |

## Anti-Patterns

- Importing openmeter/ent/db or any Ent-generated package — this adapter never touches Postgres.
- Returning a top-level error from validateEvents for individual event failures — use Event.ValidationErrors.
- Skipping params.Validate() before delegating to the streaming connector.
- Emitting NextCursor when the result page is smaller than effectiveLimit.
- Adding additional per-event service calls inside eventPostProcess without a per-call cache.

## Decisions

- **Implemented as adapter/ rather than a service/ sub-package** — meterevent has no independent business logic; it purely composes streaming queries with customer/meter lookups, so a distinct service layer would be empty indirection.
- **ValidationErrors are per-event fields, not top-level errors** — Event listing is a query API; callers need all events including partially-invalid ones, so failing the whole request for one bad event would break observability workflows.

## Example: Resolving a domain-service filter before querying the streaming layer

```
func (a *adapter) ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) (pagination.Result[meterevent.Event], error) {
	if err := params.Validate(); err != nil {
		return pagination.Result[meterevent.Event]{}, models.NewGenericValidationError(fmt.Errorf("validate input: %w", err))
	}
	listParams := streaming.ListEventsV2Params{Namespace: params.Namespace}
	if params.CustomerID != nil && params.CustomerID.In != nil && len(*params.CustomerID.In) > 0 {
		customers, err := a.listCustomers(ctx, params.Namespace, *params.CustomerID.In)
		if err != nil { return pagination.Result[meterevent.Event]{}, err }
		if len(customers) == 0 { return pagination.Result[meterevent.Event]{}, nil }
		listParams.Customers = &customers
	}
	// ... query + eventPostProcess
}
```

<!-- archie:ai-end -->
