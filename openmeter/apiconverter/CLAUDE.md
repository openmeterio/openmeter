# apiconverter

<!-- archie:ai-start -->

> Thin translation layer converting v1 API filter/pagination types (api.Filter*) into internal pkg/filter and pkg/pagination/v2 types. All converter functions are Goverter-generated package-level vars; manual code is limited to cursor decoding and the goverter directive file.

## Patterns

**Goverter variable declarations in filter.go** — Converter functions are package-level var of function type with goverter directive comments. Never implement conversion by hand — add a new var with goverter:ignore annotations and let go generate produce filter.gen.go. (`// goverter:ignore Exists Contains Ncontains
var ConvertString func(api.FilterString) filter.FilterString`)
**goverter:ignore for API-internal type mismatches** — When the internal filter type has fields not present in the v1 API type (Exists, Contains, Ncontains), add a // goverter:ignore <Field> directive above the var. Remove when the v1 API grows the field. (`// goverter:ignore Eq
// goverter:ignore Exists
ConvertTime func(api.FilterTime) filter.FilterTime`)
**Nil-safe pointer variants for every converter** — Every converter has a *Ptr variant (ConvertStringPtr) handling nil input, so callers never need inline nil-checks. (`ConvertStringPtr = func(source *api.FilterString) *filter.FilterString { if source != nil { v := ConvertString(*source); return &v }; return nil }`)
**Cursor conversion delegates to pagination.DecodeCursor** — ConvertCursor and ConvertCursorPtr in cursor.go are hand-written thin wrappers over pagination.DecodeCursor; keep cursor logic here, not in httpdriver packages. (`func ConvertCursor(s api.CursorPaginationCursor) (*pagination.Cursor, error) { return pagination.DecodeCursor(s) }`)
**init() block initializes all generated vars** — filter.gen.go uses a single init() to assign all converter vars. The generated file carries the goverter DO NOT EDIT header. (`func init() { ConvertBoolean = func(source api.FilterBoolean) filter.FilterBoolean { ... } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `filter.go` | Source of truth for all converter declarations; only var declarations and goverter directives. Run go generate ./ here to regenerate filter.gen.go. | Adding hand-written conversion logic here — all logic must live in the generated filter.gen.go. |
| `filter.gen.go` | Goverter-generated converter implementations. | DO NOT EDIT — manual edits are overwritten on next go generate. |
| `cursor.go` | Hand-written cursor conversion using pagination.DecodeCursor from pkg/pagination/v2. | Duplicating cursor decoding elsewhere in httpdriver packages instead of calling ConvertCursor/ConvertCursorPtr. |

## Anti-Patterns

- Hand-writing conversion logic in filter.go instead of declaring a goverter var and regenerating
- Editing filter.gen.go directly — it is fully generated and will be overwritten
- Adding domain business logic here; this package is a pure type-translation boundary
- Forgetting goverter:ignore for fields that exist on internal types but not on the v1 API type
- Adding v3 API types here — v3 has its own converter layer

## Decisions

- **Use Goverter variable-style converters instead of methods or standalone functions** — Variable-style converters allow recursive self-references (ConvertString calling itself for And/Or slices) and initialize in one init() block, keeping the generated code self-contained.
- **Keep cursor conversion hand-written rather than Goverter-generated** — pagination.DecodeCursor performs base64 decoding that cannot be mechanically derived by field mapping, so it must stay hand-written in cursor.go.

## Example: Adding a new filter type converter for a hypothetical FilterEnum API type

```
// In filter.go — add a new var with directives:
// goverter:ignoreMissing
var ConvertEnum    func(api.FilterEnum) filter.FilterEnum
var ConvertEnumPtr func(*api.FilterEnum) *filter.FilterEnum

// Then run: go generate ./
// This regenerates filter.gen.go with the new ConvertEnum and ConvertEnumPtr implementations.
```

<!-- archie:ai-end -->
