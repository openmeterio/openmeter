# common

<!-- archie:ai-start -->

> Shared TypeSpec primitives for the v3 AIP spec: error response @useRef stubs, cursor/page pagination query models, AIP-style filter union types (String/ULID/DateTime/Boolean/Numeric/Labels) with x-go-type extensions, SortQuery, and Labels property references. All other namespaces import from here via 'Common.*'.

## Patterns

**Error responses via @useRef to external YAML** — All error models use @useRef pointing to common/definitions/errors.yaml; never define inline error schemas. The ErrorResponses alias groups the three universal errors (BadRequest, Unauthorized, Forbidden). (`@useRef("../../../../common/definitions/errors.yaml#/components/responses/NotFound") @friendlyName("NotFound") model NotFound { @statusCode _: 404; }`)
**Filter types carry x-go-type extensions for Go codegen** — Every filter model/union carries @extension("x-go-type", "filters.FilterXxx") and @extension("x-go-type-import", #{path: "github.com/openmeterio/openmeter/api/v3/filters"}) so oapi-codegen emits the hand-written Go filter type. (`@extension("x-go-type", "filters.FilterString") @extension("x-go-type-import", #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }) union StringFieldFilter { ... }`)
**Pagination models split by strategy** — CursorPaginationQuery and PagePaginationQuery are separate models; operations choose exactly one. Never mix cursor and page parameters. (`// Cursor: interface Ops { @get list(...Common.CursorPaginationQuery, ...): Shared.CursorPaginatedResponse<T> }
// Page: @get list(...Common.PagePaginationQuery, ...): Shared.PagePaginatedResponse<T>`)
**Filter unions support both shorthand scalar and object form** — StringFieldFilter, ULIDFieldFilter, DateTimeFieldFilter, LabelsFieldFilter are unions of (scalar | object-with-operators) so clients can pass filter[x]=value or filter[x][eq]=value. (`union StringFieldFilter { equals: string, object: { eq?: string, neq?: string, contains?: string, oeq?: string[], exists?: boolean } }`)
**SortQuery delegates to YAML $ref** — SortQuery is an empty TypeSpec model with @useRef. Do not expand sort logic inline in TypeSpec. (`@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/SortQuery") @friendlyName("SortQuery") model SortQuery {}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `error.tsp` | Canonical error response models and the ErrorResponses alias. Import Common.ErrorResponses at the end of every operation's return union. | ErrorResponses only covers 400/401/403. Ops that can 404 must add '| Common.NotFound' explicitly; 409 needs '| Common.Conflict'. |
| `parameters.tsp` | All AIP filter types and SortQuery. New filter types must follow the union-of-(scalar | object) pattern and carry both x-go-type extensions. | StringFieldFilterExact is a model (not union for shorthand scalar) — don't accidentally make exact-only types a union with a bare scalar. |
| `pagination.tsp` | CursorPaginationQuery, PagePaginationQuery, and their associated meta models. CursorMeta and PageMeta delegate to YAML $refs. | Cursor pagination uses page.after/before; page pagination uses page.number. Using the wrong query model breaks SDK pagination helpers. |
| `properties.tsp` | Labels and PublicLabels models delegating to konnect_properties.yaml via @useRef. | Labels are constrained to 1-63 char keys excluding reserved prefixes — constraints are in the YAML, not in TypeSpec. |

## Anti-Patterns

- Defining inline error response bodies instead of using @useRef to errors.yaml
- Adding a new filter type without both x-go-type and x-go-type-import extensions — breaks Go codegen
- Using Common.PagePaginationQuery on an operation that returns CursorPaginatedResponse (or vice versa)
- Declaring non-empty bodies on Common error models — they are stubs that reference YAML
- Mixing sort and filter logic inline in operations.tsp instead of referencing Common.SortQuery and Common.*FieldFilter

## Decisions

- **Filter types carry x-go-type extensions pointing to hand-written Go filter structs** — AIP filter semantics (eq/neq/contains/oeq/ocontains/exists) require parsing logic that is hard to generate; keeping the Go implementation in api/v3/filters/ and referencing it from TypeSpec avoids duplicating that logic in generated code.
- **Error response models delegate to external YAML via @useRef** — The YAML definitions are shared with the v1 spec and contain OpenAPI extensions; a TypeSpec-native redefinition would diverge and break shared client error handling.

## Example: Add a new filter type for an enum field

```
// In parameters.tsp:
@friendlyName("EnumFieldFilter")
@extension("x-go-type", "filters.FilterEnum")
@extension(
  "x-go-type-import",
  #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }
)
union EnumFieldFilter {
  string: string,
  object: {
    eq?: string,
    neq?: string,
    @encode(ArrayEncoding.commaDelimited)
    oeq?: string[],
  },
// ...
```

<!-- archie:ai-end -->
