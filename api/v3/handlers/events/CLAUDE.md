# events

<!-- archie:ai-start -->

> HTTP handler for listing ingested events (v3 event query API) with rich filter support (type, time, ingested_at, stored_at, customer_id) and sort; includes unit tests for filter parsing.

## Patterns

**Filter parsing helpers are package-private and unit-tested** — fromAPICustomerIDFilter, fromAPIEventSort, and fromAPIEventFilter are unexported helpers in list.go with dedicated unit tests in list_test.go covering all supported and rejected filter variants. (`func fromAPICustomerIDFilter(ctx context.Context, f *api.ULIDFieldFilter) (*streaming.CustomerIDFilter, error)`)
**apierrors.NewBadRequestError with field path for filter errors** — Every unsupported filter operator returns apierrors.NewBadRequestError with the exact query-param field path as the InvalidParameter.Field value (e.g., 'filter[customer_id]', 'sort'). (`apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{{Field: "filter[customer_id]", ...}})`)
**Sort defaults to desc when no suffix provided** — fromAPIEventSort parses the SortQuery string; a field name without a suffix is treated as descending by default. (`sort := api.SortQuery("time") // → sortx.OrderDesc`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `list.go` | Main list events handler plus all filter/sort parsing helpers. fromAPICustomerIDFilter only supports Eq and Oeq; Neq/Contains/Ocontains/Exists are rejected. | Adding a new filter field requires adding both a parsing helper and a test case in list_test.go. |
| `list_test.go` | Unit tests for filter and sort parsing helpers using t.Context(). assertBadRequestField verifies the exact InvalidParameter.Field in error responses. | Tests use t.Context() not context.Background() — keep consistent. |

## Anti-Patterns

- Supporting filter operators beyond Eq/Oeq for customer_id without adding explicit rejection tests for unsupported operators
- Parsing filter defaults in the handler operation func instead of dedicated helper functions
- Returning raw errors from filter parsing instead of apierrors.NewBadRequestError with field path

## Decisions

- **Filter parsing helpers are unexported and unit-tested in list_test.go rather than integration-tested.** — Parser logic is deterministic and does not require a running service; unit tests give fast feedback on edge cases like malformed sort strings.

<!-- archie:ai-end -->
