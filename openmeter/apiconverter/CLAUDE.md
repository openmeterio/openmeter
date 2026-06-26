# apiconverter

<!-- archie:ai-start -->

> Boundary converters that map v1 API filter/pagination types (api package) to internal pkg/filter and pkg/pagination types and back. Its primary constraint: the actual field-mapping logic is goverter-generated and must never be hand-edited.

## Patterns

**goverter variable converters** — Converter funcs are declared as package-level var func signatures in filter.go under a `// goverter:variables` block; the bodies are generated into filter.gen.go. Add a new conversion by declaring its signature here, not by writing the body. (`ConvertString func(api.FilterString) filter.FilterString`)
**Ptr / Map variants per type** — Each base converter has paired Ptr and Map/MapPtr variants (e.g. ConvertString, ConvertStringPtr, ConvertStringMap, ConvertStringMapPtr) that nil-guard and delegate to the base converter. (`ConvertStringPtr returns nil when source is nil, else &ConvertString(*source)`)
**goverter:ignore for API/internal field drift** — Fields present on internal filter types but absent from v1 api types are silenced with `// goverter:ignore <Field>` directives above the var. When v1 grows the field, remove the ignore so it copies through instead of being silently dropped. (`// goverter:ignore Exists Contains Ncontains above ConvertString`)
**Hand-written non-goverter converters** — Pagination cursor conversion (cursor.go) is hand-written and delegates to pagination.DecodeCursor; it is NOT under goverter generation. (`func ConvertCursor(s api.CursorPaginationCursor) (*pagination.Cursor, error) { return pagination.DecodeCursor(s) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | Source of truth: goverter var declarations and ignore/skip directives; has `//go:generate ... goverter gen ./` and `// goverter:skipCopySameType`. | Editing converter bodies here does nothing — only signatures and goverter comments matter. Adding a field to internal filter without an ignore entry breaks generation. |
| `filter.gen.go` | Generated converter bodies wired in init(); also contains the private filterFilterStringToApiFilterString helper for the reverse direction. | Marked DO NOT EDIT and `//go:build !goverter`. Regenerate via go generate / make generate after changing filter.go. |
| `cursor.go` | Hand-written cursor converters wrapping pagination/v2 DecodeCursor. | Imports pkg/pagination/v2 (not v1). Keep nil-guarding in the Ptr variant. |

## Anti-Patterns

- Hand-editing filter.gen.go instead of regenerating from filter.go.
- Adding a filter operator to internal types without removing the matching goverter:ignore (silently dropped at the boundary).
- Writing bespoke field-copy logic in filter.go function declarations — bodies belong to goverter.

## Decisions

- **Use goverter codegen for API<->internal filter mapping.** — Filter types have many operators (Eq/Ne/Gt/In/Like/And/Or...); generation avoids error-prone manual copies and makes drift between API and internal types explicit via ignore directives.

## Example: Declare a new generated converter with field-drift ignores

```
// goverter:variables
// goverter:skipCopySameType
var (
	// goverter:ignore Exists Contains Ncontains
	ConvertString    func(api.FilterString) filter.FilterString
	ConvertStringPtr func(*api.FilterString) *filter.FilterString
)
```

<!-- archie:ai-end -->
