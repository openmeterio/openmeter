# request

<!-- archie:ai-start -->

> HTTP request parsing and query parameter conversion utilities for the v3 API layer. Translates raw HTTP inputs (JSON bodies, sort strings, API filter types) into internal domain types before they reach handlers.

## Patterns

**Return *apierrors.BaseAPIError not error** — All parse functions return *apierrors.BaseAPIError (nilable pointer), not the Go error interface. Callers check for nil to detect parse failure. (`func ParseBody(r *http.Request, payload any) *apierrors.BaseAPIError`)
**Contains operator wraps to SQL LIKE pattern** — api.QueryFilterString.Contains/Ncontains are converted to LIKE/NLIKE via filter.ContainsPattern, which escapes SQL metacharacters (%, _, \). Never pass Contains values directly to filter.FilterString.Like. (`Like: convertContainsOperator(source.Contains)`)
**Recursive filter conversion** — And/Or slices in QueryFilterString are converted recursively via convertQueryFilterStringList so nested boolean expressions are fully translated. (`And: convertQueryFilterStringList(source.And)`)
**SortBy text format: 'field [asc|desc]'** — ParseSortBy parses a string with UnmarshalText: one or two whitespace-separated tokens. First token is Field, second (optional) is Order; default order is asc. Anything else returns ErrSortByInvalid. (`parts := strings.Fields(string(text)); s.Field = parts[0]`)
**Nil-safe pointer converters** — ConvertQueryFilterStringPtr returns nil when given nil, avoiding nil-dereference in optional filter fields. (`func ConvertQueryFilterStringPtr(source *api.QueryFilterString) *filter.FilterString`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `body.go` | JSON body decoder; ParseOptionalBody silently ignores io.EOF (empty body) to allow PATCH semantics. | ParseOptionalBody leaves payload unchanged on empty body — callers must treat unchanged payload as 'no update', not 'zero value'. |
| `filter.go` | Converts api.QueryFilterString / QueryFilterStringMapItem to pkg/filter.FilterString. QueryFilterStringMapItem adds an Exists field not present in QueryFilterString. | Neq maps to Ne (field name differs between API and internal type). Contains maps to Like with wrapping %; do not set Like directly from Contains. |
| `sort.go` | SortBy value type with UnmarshalText; ToSortxOrder converts to pkg/sortx.Order. | Validate() is called inside UnmarshalText — callers must not skip validation. ErrSortFieldRequired fires when Field is empty after parsing. |

## Anti-Patterns

- Returning the Go error interface instead of *apierrors.BaseAPIError from a parse function
- Setting filter.FilterString.Like directly from an API Contains value without calling filter.ContainsPattern
- Calling Validate() separately after ParseSortBy — it is already called inside UnmarshalText
- Adding business logic or domain service calls in this package — it is pure translation/parsing only

## Decisions

- **Parse functions return *apierrors.BaseAPIError instead of error** — Allows handlers to directly propagate structured API errors with field path and source metadata without wrapping or type-asserting.
- **Contains wraps to SQL LIKE pattern inside this package** — Centralises metacharacter escaping so handlers never need to know about SQL LIKE syntax; all filter construction uses filter.ContainsPattern consistently.

## Example: Handler converts API filter query param to internal FilterString

```
import (
    api "github.com/openmeterio/openmeter/api/v3"
    "github.com/openmeterio/openmeter/api/v3/request"
    "github.com/openmeterio/openmeter/pkg/filter"
)

var f *filter.FilterString
if params.Filter != nil {
    f = request.ConvertQueryFilterStringPtr(params.Filter)
}
```

<!-- archie:ai-end -->
