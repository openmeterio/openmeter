# request

<!-- archie:ai-start -->

> Pure HTTP request parsing and query-parameter conversion utilities for the v3 API: translates raw HTTP inputs (JSON bodies, sort strings, API filter types) into typed internal domain types before handlers. Contains no business logic.

## Patterns

**Return *apierrors.BaseAPIError not error** — Parse functions return *apierrors.BaseAPIError (nilable pointer), not the Go error interface; callers check nil and propagate structured API errors directly. (`func ParseBody(r *http.Request, payload any) *apierrors.BaseAPIError`)
**Contains operator wraps to SQL LIKE pattern** — QueryFilterString.Contains/Ncontains are converted to Like/Nlike via filter.ContainsPattern, which escapes SQL metacharacters (%, _, \). Never set filter.FilterString.Like directly from Contains. (`Like: convertContainsOperator(source.Contains) // wraps to %value% with escaping`)
**Recursive filter conversion for And/Or** — And/Or slices in QueryFilterString are converted recursively via convertQueryFilterStringList so nested boolean expressions are fully translated. (`And: convertQueryFilterStringList(source.And)`)
**SortBy text format 'field [asc|desc]'** — ParseSortBy uses UnmarshalText: one or two whitespace-separated tokens; first is Field, second optional Order (default asc). Validate() is called inside UnmarshalText. (`parts := strings.Fields(string(text)); s.Field = parts[0]`)
**API Neq maps to internal Ne** — The API field Neq maps to the internal filter field Ne (names differ); QueryFilterStringMapItem additionally has an Exists field not present in QueryFilterString. (`Ne: source.Neq // API uses Neq, internal type uses Ne`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `body.go` | JSON body decoders. ParseOptionalBody silently ignores io.EOF (empty body) for PATCH semantics. | ParseOptionalBody leaves payload unchanged on empty body — callers must treat that as 'no update', not 'zero value'. |
| `filter.go` | Converts api.QueryFilterString and QueryFilterStringMapItem to pkg/filter.FilterString; handles Contains->Like wrapping and recursive And/Or. | Neq maps to Ne; Contains maps to Like with % wrapping — never set Like directly from Contains. |
| `sort.go` | SortBy value type with UnmarshalText; ToSortxOrder converts to pkg/sortx.Order. | Validate() runs inside UnmarshalText — don't skip or duplicate. ErrSortFieldRequired fires when Field is empty after parsing. |

## Anti-Patterns

- Returning the Go error interface instead of *apierrors.BaseAPIError from a parse function
- Setting filter.FilterString.Like directly from an API Contains value without filter.ContainsPattern
- Calling Validate() separately after ParseSortBy — already called inside UnmarshalText
- Adding business logic or domain service calls — this package is pure translation/parsing
- Passing a Contains string directly to internal filter without % wrapping via ContainsPattern

## Decisions

- **Parse functions return *apierrors.BaseAPIError instead of error** — Lets handlers directly propagate structured API errors with field path and source metadata without wrapping or type-asserting.
- **Contains wraps to SQL LIKE pattern inside this package** — Centralises metacharacter escaping so handlers never deal with SQL LIKE syntax; all filter construction uses filter.ContainsPattern.

## Example: Handler converts an API filter query param to internal FilterString

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
