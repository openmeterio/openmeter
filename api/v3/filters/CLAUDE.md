# filters

<!-- archie:ai-start -->

> Provides AIP-style filter[field][op]=value query-parameter parsing and conversion to the pkg/filter domain filter types used by service List methods. Two concerns: filter.go defines the API-layer filter structs; parse.go parses url.Values into them; convert.go converts them to pkg/filter types.

## Patterns

**filters.Parse for URL → API filter struct** — Call filters.Parse(r.URL.Query(), &myFilter) in the request decoder to populate a typed filter struct. The target must be a pointer to a struct whose fields are *FilterString, *FilterStringExact, *FilterULID, *FilterNumeric, *FilterDateTime, *FilterBoolean, FilterLabels, or *string — field names come from json struct tags. (`var f MyFilters
if err := filters.Parse(r.URL.Query(), &f); err != nil { return err }`)
**FromAPI* converters for API filter → pkg/filter** — After parsing, convert each API filter field to the pkg/filter equivalent using the FromAPI* functions (FromAPIFilterString, FromAPIFilterULID, FromAPIFilterNumeric, FromAPIFilterDateTime, FromAPIFilterBoolean, FromAPIFilterLabels). These normalize multi-operator combinations into And nodes automatically. (`pf, err := filters.FromAPIFilterString(f.Name)
if err != nil { return err }
input.NameFilter = pf`)
**filter[field][op]=value URL parameter convention** — All filter params use the bracket syntax: filter[name][eq]=foo, filter[created_at][gte]=2024-01-01T00:00:00Z. Label fields use dot-notation: filter[labels.env][eq]=prod. A bare filter[field]=value is equivalent to filter[field][eq]=value. (`filter[status][oeq]=active,pending&filter[labels.region][contains]=us`)
**Pointer-to-pointer target allocates lazily** — If the target is **MyFilters, Parse allocates the inner struct only when at least one filter key is present in the query string. Use this pattern in handlers where filters are optional. (`var f *MyFilters
if err := filters.Parse(r.URL.Query(), &f); err != nil { return err }
// f is nil if no filter keys were provided`)
**Unknown filter fields are rejected** — Parse validates that every filter[x] key matches a json-tagged field in the target struct. Unrecognised field names return an error. This enforces strict API contracts and prevents silent misuse. (`// filter[typo][eq]=x → error: unknown filter field(s): typo`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | API-layer filter struct definitions. Add new filter types here if pkg/filter gains new base types. | FilterLabels is defined as a type alias (map[string]FilterLabel), not a pointer-to-struct — it is handled separately in parseFiltersValue via filterLabelsType reflection check. |
| `parse.go` | Reflection-driven URL query parser. Dispatches on field types via reflect.TypeFor constants. maxCommaSeparatedItems=50 and maxFilterValueLength=1024 are the hard limits. | FilterULIDType is exported (capital F) while other type vars are unexported — required for external callers using the reflection type in switch statements. |
| `convert.go` | Converts API filter structs to pkg/filter equivalents. Multi-operator combinations (e.g. Gt+Lte) are normalized into And-of-single-op leaf nodes. | Each FromAPI* function returns (nil, nil) when the input pointer is nil — callers must handle nil output as 'no filter applied'. |

## Anti-Patterns

- Parsing filter query params manually with r.URL.Query().Get() instead of filters.Parse
- Passing the pkg/filter types directly as API response fields — they are internal types, not wire types
- Adding new operator strings to parse.go without corresponding handling in convert.go
- Using filterLabelsType for anything other than the FilterLabels field — dot-notation parsing is special-cased

## Decisions

- **Reflection-driven parse dispatching on field type rather than per-field generated code** — A single Parse function handles all filter structs uniformly; adding a new filterable field requires only declaring the json-tagged struct field — no new parsing code.
- **Multi-operator combinations normalized to And-of-leaves in FromAPI* converters** — pkg/filter requires each node to carry at most one operator; the converter canonicalizes multi-bound inputs (e.g. Gte+Lte → And[{Gte},{Lte}]) so domain adapters always receive valid filter trees.

## Example: Parsing and converting filters in a v3 list handler decoder

```
import (
    "github.com/openmeterio/openmeter/api/v3/filters"
    "github.com/openmeterio/openmeter/pkg/filter"
)

type MyListFilters struct {
    Name   *filters.FilterString `json:"name,omitempty"`
    Status *filters.FilterStringExact `json:"status,omitempty"`
}

func decodeRequest(r *http.Request) (MyListInput, error) {
    var f MyListFilters
    if err := filters.Parse(r.URL.Query(), &f); err != nil {
        return MyListInput{}, err
    }
// ...
```

<!-- archie:ai-end -->
