# filters

<!-- archie:ai-start -->

> Provides AIP-style filter[field][op]=value query-parameter parsing and conversion into the pkg/filter domain filter types consumed by service List methods. Three concerns: filter.go defines API-layer filter structs, parse.go reflection-parses url.Values into them, convert.go maps them to pkg/filter.

## Patterns

**filters.Parse for URL to typed filter struct** — Call filters.Parse(r.URL.Query(), &myFilter) in the request decoder. Target must be a non-nil pointer to a struct (or **struct for lazy allocation) whose fields are *FilterString, *FilterStringExact, *FilterULID, *FilterNumeric, *FilterDateTime, *FilterBoolean, FilterLabels, *string, or *NamedStringType. Field names come from json struct tags. (`var f MyFilters
if err := filters.Parse(r.URL.Query(), &f); err != nil { return err }`)
**FromAPI* converters for API filter to pkg/filter** — After parsing, convert each field via FromAPIFilterString/ULID/StringExact/Numeric/DateTime/Boolean/Labels. Each returns (nil, nil) when input is nil — callers must treat nil as 'no filter applied'. Multi-operator combinations (e.g. Gt+Lte) are normalized to And-of-single-op leaves. (`pf, err := filters.FromAPIFilterString(f.Name)
if err != nil { return err }
input.NameFilter = pf`)
**filter[field][op]=value bracket convention** — All filter params use bracket syntax (filter[name][eq]=foo, filter[created_at][gte]=2024-01-01T00:00:00Z). Bare filter[field]=value equals filter[field][eq]=value for string fields. Label fields use dot-notation filter[labels.env][eq]=prod or nested filter[labels][env][eq]=prod. (`filter[status][oeq]=active,pending&filter[labels.region][contains]=us`)
**Unknown filter fields rejected by Parse** — checkUnknownFilterKeys validates every filter[x] key against a json-tagged field of the target struct; unrecognised names return an error, enforcing a strict API contract. (`// filter[typo][eq]=x -> error: unknown filter field(s): typo`)
**FromAPIStatusFilter for enum status fields** — Generic helper FromAPIStatusFilter[T validator](ctx, *FilterStringExact) accepts only eq and oeq, builds a []T, and validates each value via T.Validate(); neq returns an error. (`statuses, err := filters.FromAPIStatusFilter[plan.Status](ctx, f.Status)`)
**Pointer-to-pointer target lazily allocates** — If target is **MyFilters, Parse allocates the inner struct only when at least one filter key is present (hasFilterKeys); otherwise the pointer stays nil. Use for optional filter parameters. (`var f *MyFilters
if err := filters.Parse(r.URL.Query(), &f); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | API-layer filter struct definitions (FilterString, FilterStringExact, FilterULID, FilterNumeric, FilterDateTime, FilterBoolean, FilterLabel, FilterLabels). Add a new filter type here when pkg/filter gains a new base type. | FilterLabels is a type alias (map[string]FilterLabel), NOT a pointer-to-struct; it is special-cased in parseFiltersValue via filterLabelsType/filterLabelsPtrType reflection checks. |
| `parse.go` | Reflection-driven URL query parser dispatching on field type via reflect.TypeFor constants; handles *string, *NamedStringType (parseStringPtrTyped), and *T implementing encoding.TextUnmarshaler (parseTextUnmarshalerPtr). | FilterULIDType is exported (capital F) while sibling type vars are unexported — external callers reference it in switch statements. maxCommaSeparatedItems=50 and maxFilterValueLength=1024 are hard limits. *string/typed fields reject operator-style keys. |
| `convert.go` | Converts each API filter struct to its pkg/filter equivalent, canonicalizing multi-operator inputs into And-of-single-op leaf nodes. | Every FromAPI* returns (nil, nil) on nil input. FilterStringExact maps to a single FilterString (no And split). FromAPIStatusFilter wraps validation errors in models.NewNillableGenericValidationError. |

## Anti-Patterns

- Parsing filter query params manually with r.URL.Query().Get() instead of filters.Parse
- Passing pkg/filter types directly as API response/wire fields — they are internal types
- Adding a new operator string in parse.go without matching handling in convert.go
- Treating a non-nil FromAPI* return for an empty filter as set — empty input returns nil
- Using filterLabelsType reflection for anything other than the FilterLabels field

## Decisions

- **Reflection-driven Parse dispatching on field type rather than per-field generated code** — A single Parse handles all filter structs uniformly; adding a filterable field needs only a json-tagged struct field, no new parsing code.
- **Multi-operator combinations normalized to And-of-leaves in FromAPI* converters** — pkg/filter requires each node to carry at most one operator, so the converter canonicalizes ranges (Gte+Lte -> And[{Gte},{Lte}]) to always emit a valid filter tree that passes Validate().

## Example: Parsing and converting filters in a v3 list handler decoder

```
import "github.com/openmeterio/openmeter/api/v3/filters"

type MyListFilters struct {
    Name   *filters.FilterString      `json:"name,omitempty"`
    Status *filters.FilterStringExact `json:"status,omitempty"`
}

func decode(r *http.Request) (MyListInput, error) {
    var f MyListFilters
    if err := filters.Parse(r.URL.Query(), &f); err != nil {
        return MyListInput{}, err
    }
    name, err := filters.FromAPIFilterString(f.Name)
    if err != nil { return MyListInput{}, err }
    return MyListInput{NameFilter: name}, nil
// ...
```

<!-- archie:ai-end -->
