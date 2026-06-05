# adapter

<!-- archie:ai-start -->

> Adapter implementation of meterevent.Service: reads raw events from the streaming.Connector (ClickHouse) and post-processes them into enriched meterevent.Event values (customer-ID resolution + meter validation). This is a read-only query layer, not an Ent/Postgres adapter.

## Patterns

**Constructor returns the domain Service interface** — New(streamingConnector, customerService, meterService) returns meterevent.Service; the concrete *adapter is unexported and asserted with `var _ meterevent.Service = (*adapter)(nil)`. (`func New(streamingConnector streaming.Connector, customerService customer.Service, meterService meter.Service) meterevent.Service`)
**Validate params at the top of every Service method** — Each public method calls params.Validate() first and wraps failures in models.NewGenericValidationError(fmt.Errorf("validate input: %w", err)) before any I/O. (`if err := params.Validate(); err != nil { return ..., models.NewGenericValidationError(fmt.Errorf("validate input: %w", err)) }`)
**Translate domain params to streaming params, never query streaming directly with domain types** — ListEvents/ListEventsV2 build streaming.ListEventsParams / streaming.ListEventsV2Params field-by-field, then call a.streamingConnector. CustomerID filters are resolved to []streaming.Customer via listCustomers before passing down. (`listParams := streaming.ListEventsV2Params{Namespace: params.Namespace, Cursor: params.Cursor, ...}`)
**Shared eventPostProcess pipeline** — Both list methods funnel raw events through eventPostProcess: mapEventsToMeterEvents -> enrichEventsWithCustomerID -> validateEvents. Add new per-event enrichment as a step here, not inline in list methods. (`meterEvents := mapEventsToMeterEvents(rawEvents); meterEvents, err = a.enrichEventsWithCustomerID(...); meterEvents, err = a.validateEvents(...)`)
**Validation errors are collected on the event, not returned** — validateEvents appends to event.ValidationErrors (no meter match, parse failure, missing customer) instead of failing the whole list; the event is still returned so the API can surface per-event ValidationError. (`validationErrors = append(validationErrors, fmt.Errorf("no meter found for event type: %s", event.Type)); event.ValidationErrors = validationErrors`)
**Cursor pagination only emits NextCursor on a full page** — ListEventsV2 propagates listParams.SortBy onto each Event so Event.Cursor() matches the ORDER BY, and only sets result.NextCursor when len(meterEvents) == effectiveLimit (lo.FromPtrOr(params.Limit, meterevent.MaximumLimit)). (`if len(meterEvents) > 0 && len(meterEvents) == effectiveLimit { cursor := meterEvents[len(meterEvents)-1].Cursor(); result.NextCursor = &cursor }`)
**Missing-customer lookups are tolerated, not fatal** — enrichEventsWithCustomerID treats models.IsGenericNotFoundError from GetCustomerByUsageAttribution as 'leave CustomerID nil' (later flagged in validateEvents); only non-not-found errors abort. (`if models.IsGenericNotFoundError(err) { eventsWithCustomerID = append(eventsWithCustomerID, event); continue }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines the unexported adapter struct, its three injected dependencies, the New constructor, and the meterevent.Service compile-time assertion. | Return meterevent.Service (not *adapter) from New; keep the `var _ meterevent.Service = (*adapter)(nil)` assertion when adding methods. |
| `event.go` | All Service method bodies: ListEvents (v1), ListEventsV2 (cursor-paginated), plus helpers listCustomers, eventPostProcess, mapEventsToMeterEvents, validateEvents, enrichEventsWithCustomerID. | enrichEventsWithCustomerID has a FIXME: it queries the customer service per-event (only cached by subject) — do not amplify N+1 lookups; meter validation iterates all meters per event. |

## Anti-Patterns

- Querying streaming.Connector with domain (meterevent) param types instead of building streaming.ListEventsParams / ListEventsV2Params.
- Returning an error from validateEvents/enrich when a single event fails validation or has no customer — failures belong in event.ValidationErrors.
- Setting NextCursor unconditionally or forgetting to propagate SortBy onto events, which desynchronizes Event.Cursor() from the query ORDER BY.
- Skipping params.Validate() or not wrapping its error in models.NewGenericValidationError.
- Treating a not-found customer in enrichEventsWithCustomerID as a hard error instead of leaving CustomerID nil.

## Decisions

- **Customer-ID resolution and meter validation are done in the adapter post-process rather than the streaming layer.** — Streaming (ClickHouse) only stores raw CloudEvents; customer attribution and meter schema validation require the customer and meter services, keeping the streaming connector domain-agnostic.
- **Validation issues are attached per-event (ValidationErrors) instead of short-circuiting the list.** — Event listing is a debugging/observability surface; users need to see invalid events alongside valid ones with their specific failure reasons.

## Example: Adapter list method: validate, map domain->streaming params, query, post-process

```
func (a *adapter) ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) (pagination.Result[meterevent.Event], error) {
	if err := params.Validate(); err != nil {
		return pagination.Result[meterevent.Event]{}, models.NewGenericValidationError(fmt.Errorf("validate input: %w", err))
	}
	listParams := streaming.ListEventsV2Params{Namespace: params.Namespace, Cursor: params.Cursor, Limit: params.Limit, /* ... */}
	events, err := a.streamingConnector.ListEventsV2(ctx, listParams)
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("query events: %w", err)
	}
	meterEvents, err := a.eventPostProcess(ctx, params.Namespace, events)
	if err != nil {
		return pagination.Result[meterevent.Event]{}, fmt.Errorf("post process events: %w", err)
	}
	return pagination.Result[meterevent.Event]{Items: meterEvents}, nil
}
```

<!-- archie:ai-end -->
