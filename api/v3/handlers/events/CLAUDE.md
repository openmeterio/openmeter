# events

<!-- archie:ai-start -->

> HTTP handler for listing and ingesting CloudEvents in the v3 API; provides rich filter support (type, time, ingested_at, stored_at, customer_id) and sort with unit-tested filter parsing helpers.

## Patterns

**Filter parsing helpers are package-private and unit-tested** — fromAPICustomerIDFilter, fromAPIEventSort, and fromAPIEventFilter are unexported helpers in list.go with dedicated unit tests in list_test.go covering all supported and rejected filter variants. (`func fromAPICustomerIDFilter(ctx context.Context, f *api.ULIDFieldFilter) (*filter.FilterString, error)`)
**apierrors.NewBadRequestError with exact field path for filter errors** — Every unsupported filter operator returns apierrors.NewBadRequestError with the exact query-param field path as the InvalidParameter.Field value. (`apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{Field: "filter[customer_id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery}})`)
**Sort defaults to desc when no suffix provided** — fromAPIEventSort parses the SortQuery string; a field name without an asc/desc suffix is treated as descending by default so sort=time means most-recent first. (`sort := api.SortQuery("time") // → sortx.OrderDesc`)
**Customer ID filter supports only Eq and Oeq operators** — fromAPICustomerIDFilter explicitly rejects Neq, Contains, Ocontains, Exists operators with a descriptive error; only eq and oeq are forwarded as an IN set to the backend. (`if f.Neq != nil { err := errors.New("only eq and oeq operators are supported"); return nil, apierrors.NewBadRequestError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `list.go` | Main list events handler plus all filter/sort parsing helpers. fromAPICustomerIDFilter only supports Eq and Oeq; Neq/Contains/Ocontains/Exists are rejected. | Adding a new filter field requires adding both a parsing helper and a test case in list_test.go. |
| `list_test.go` | Unit tests for filter and sort parsing helpers using t.Context(). assertBadRequestField verifies the exact InvalidParameter.Field in error responses. | Tests use t.Context() not context.Background() — keep consistent. |
| `convert.go` | toAPIMeteringIngestedEvent converts meterevent.Event to API wire form; parses JSON data string to map[string]any; sets CloudEvents specversion=1.0. | e.Data is a raw JSON string — must unmarshal to map[string]any before assigning to nullable event.Data field. |
| `ingest.go` | IngestEvents handler for CloudEvent ingestion path. | Ingest path is separate from list path; do not mix streaming connector calls with ingest collector calls. |

## Anti-Patterns

- Supporting filter operators beyond Eq/Oeq for customer_id without adding explicit rejection tests for unsupported operators
- Parsing filter defaults in the handler operation func instead of dedicated unexported helper functions
- Returning raw errors from filter parsing instead of apierrors.NewBadRequestError with exact field path
- Using context.Background() in list_test.go tests instead of t.Context()

## Decisions

- **Filter parsing helpers are unexported and unit-tested in list_test.go rather than integration-tested.** — Parser logic is deterministic and does not require a running service; unit tests give fast feedback on edge cases like malformed sort strings.

## Example: Add a new filter field to the events list handler

```
// In list.go — add a parsing helper:
func fromAPISubjectFilter(ctx context.Context, f *api.StringFieldFilter) (*filter.FilterString, error) {
	if f == nil {
		return nil, nil
	}
	var values []string
	if f.Eq != nil {
		values = append(values, *f.Eq)
	}
	if len(values) == 0 {
		return nil, nil
	}
	return &filter.FilterString{In: &values}, nil
}
// Then call it in the decoder:
// ...
```

<!-- archie:ai-end -->
