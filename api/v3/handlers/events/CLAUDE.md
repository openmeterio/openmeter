# events

<!-- archie:ai-start -->

> v3 HTTP handlers for listing (cursor-paginated) and ingesting CloudEvents, with rich filter support (id, source, subject, type, time, ingested_at, stored_at, customer_id) and sort, backed by unit-tested filter/sort parsing helpers.

## Patterns

**Package-private, unit-tested filter/sort helpers** — fromAPICustomerIDFilter, fromAPIEventSort, and applyFilters are unexported helpers in list.go/convert.go with dedicated tests in list_test.go covering supported and rejected variants. (`func fromAPICustomerIDFilter(ctx context.Context, f *api.ULIDFieldFilter) (*filter.FilterString, error)`)
**apierrors.NewBadRequestError with exact field path** — Every filter/sort error returns apierrors.NewBadRequestError with the exact bracket query-param path as InvalidParameter.Field and Source: apierrors.InvalidParamSourceQuery. (`apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{Field: "filter[customer_id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery}})`)
**Sort defaults to desc with no suffix** — fromAPIEventSort parses SortQuery; a single bare field (no asc/desc) defaults to sortx.OrderDesc so sort=time means most-recent-first. Only time, ingested_at, stored_at are accepted. (`sort := api.SortQuery("time") // -> EventSortFieldTime, sortx.OrderDesc`)
**customer_id filter supports only eq/oeq** — fromAPICustomerIDFilter rejects Neq (others fall through) and forwards eq/oeq as a concrete IN set because ListEventsV2Params requires it. (`if f.Neq != nil { /* return BadRequest 'only eq and oeq operators are supported' */ }`)
**Content-type dispatch on ingest** — IngestEvents parses Content-Type via mime.ParseMediaType and dispatches: application/json (single AsEvent or batch AsIngestEventsBody1), application/cloudevents+json, application/cloudevents-batch+json; empty event set is a 400. (`switch contentType { case "application/json": ...; case "application/cloudevents+json": ... }`)
**Forward-only cursor pagination** — List rejects page[before] (backward pagination unsupported), decodes page[after] via pagination/v2.DecodeCursor, and enforces 1 <= page[size] <= meterevent.MaximumLimit. (`cursor, err := pagination.DecodeCursor(*params.Page.After)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `list.go` | ListMeteringEvents handler + applyFilters wiring each filter field through filters.FromAPIFilterString/DateTime and fromAPICustomerIDFilter; builds response.NewCursorPaginationResponse. | Adding a filter field requires both an applyFilters branch and a list_test.go case. page[before] is rejected; page[size] bounded by meterevent.MaximumLimit. |
| `convert.go` | toAPIMeteringIngestedEvent (meterevent.Event -> wire), fromAPICustomerIDFilter, fromAPIEventSort. Sets CloudEvents specversion=1.0. | e.Data is a raw JSON string — unmarshal to map[string]any before assigning to the nullable Data field, and set Datacontenttype=application/json. |
| `ingest.go` | IngestEvents handler routing CloudEvent payloads to ingest.Service.IngestEvents; returns 202 Accepted with empty body. | Ingest path is separate from list path — do not mix streaming connector calls with the ingest collector. Imports the v1 api package (not api/v3) for the body types. |
| `list_test.go` | Unit tests for fromAPICustomerIDFilter and fromAPIEventSort; assertBadRequestField asserts the exact InvalidParameter.Field via errors.As(*apierrors.BaseAPIError). | Tests use t.Context(), never context.Background(). |

## Anti-Patterns

- Supporting customer_id operators beyond eq/oeq without explicit rejection tests
- Parsing filter defaults in the operation func instead of dedicated unexported helpers
- Returning raw filter-parse errors instead of apierrors.NewBadRequestError with exact field path
- Using context.Background() in list_test.go instead of t.Context()
- Supporting page[before] backward pagination on the list endpoint

## Decisions

- **Filter/sort helpers are unexported and unit-tested rather than integration-tested** — Parser logic is deterministic and needs no running service; unit tests give fast feedback on edge cases like malformed sort strings.

## Example: Add a new filter field to the events list handler

```
// In list.go applyFilters — add a parsing branch:
subject, err := filters.FromAPIFilterString(f.Subject)
if err != nil {
    return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
        {Field: "filter[subject]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
    })
}
req.Subject = subject
```

<!-- archie:ai-end -->
