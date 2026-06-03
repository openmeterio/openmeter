# common

<!-- archie:ai-start -->

> Shared TypeSpec primitives for the v3 AIP spec: error response @useRef stubs, cursor/page pagination query and meta models, AIP-style filter union types (String/ULID/DateTime/Boolean/Numeric/Labels) with x-go-type extensions, SortQuery, and Labels property references. All other aip namespaces import from here via 'Common.*'.

## Patterns

**Error responses via @useRef to external YAML** — All error models are empty stubs with @useRef pointing to common/definitions/errors.yaml and a single @statusCode; never define inline error schemas. The ErrorResponses alias groups the three universal errors (BadRequest, Unauthorized, Forbidden). (`@useRef("../../../../common/definitions/errors.yaml#/components/responses/NotFound") @friendlyName("NotFound") model NotFound { @statusCode _: 404; }`)
**Filter types carry x-go-type extensions for Go codegen** — Every filter model/union carries @extension("x-go-type", "filters.FilterXxx") and @extension("x-go-type-import", #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }) so oapi-codegen emits the hand-written Go filter type instead of a generated struct. (`@extension("x-go-type", "filters.FilterString") @extension("x-go-type-import", #{ path: "github.com/openmeterio/openmeter/api/v3/filters" }) union StringFieldFilter { ... }`)
**Pagination models split by strategy** — CursorPaginationQuery and PagePaginationQuery are separate models; an operation picks exactly one and must return the matching paginated response type. Never mix cursor and page parameters. (`// cursor: list(...Common.CursorPaginationQuery): Shared.CursorPaginatedResponse<T>
// page:   list(...Common.PagePaginationQuery): Shared.PagePaginatedResponse<T>`)
**Filter unions support shorthand scalar and object form** — String/ULID/DateTime/Boolean/Numeric filters are unions of (scalar | object-with-operators) so clients can pass filter[x]=value or filter[x][eq]=value; array operators use @encode(ArrayEncoding.commaDelimited). (`union StringFieldFilter { equals: string, object: { eq?: string, neq?: string, contains?: string, oeq?: string[], exists?: boolean } }`)
**SortQuery and Labels delegate to YAML $ref** — SortQuery, Labels, and PublicLabels are empty TypeSpec models with @useRef into aip_filters.yaml / konnect_properties.yaml; do not expand sort or label logic inline in TypeSpec. (`@useRef("../../../../common/definitions/aip_filters.yaml#/components/schemas/SortQuery") @friendlyName("SortQuery") model SortQuery {}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `error.tsp` | Canonical error response stubs (400/401/403/404/409/410/413/415/422/429/500/501/503) and the ErrorResponses alias (400|401|403). | ErrorResponses only covers 400/401/403. Operations that can 404 must add '| Common.NotFound' explicitly; 409 needs '| Common.Conflict'. |
| `parameters.tsp` | All AIP filter unions (Boolean/Numeric/String/StringExact/ULID/DateTime/LabelsFieldFilter) and SortQuery. New filter types follow the union-of-(scalar | object) pattern with both x-go-type extensions. | StringFieldFilterExact / ULIDFieldFilter are exact-only (eq/neq/oeq) — don't add fuzzy operators. LabelsFieldFilter is `Record<StringFieldFilter>`, not a union. |
| `pagination.tsp` | CursorPaginationQuery (page.after/before/size), PagePaginationQuery (page.number/size), and the CursorMeta/PageMeta metadata models (delegated to metadatas.yaml). | Cursor pagination uses page.after/before; page pagination uses page.number. Both use @query(#{ explode: true, style: "deepObject" }). Picking the wrong query model breaks SDK pagination helpers. |
| `properties.tsp` | Labels and PublicLabels models delegating to konnect_properties.yaml via @useRef. | Label key constraints (1-63 chars, reserved-prefix exclusions) live in the YAML, not in TypeSpec. |

## Anti-Patterns

- Defining inline error response bodies instead of @useRef to errors.yaml
- Adding a new filter type without both x-go-type and x-go-type-import extensions — breaks Go codegen
- Using Common.PagePaginationQuery on an operation that returns CursorPaginatedResponse (or vice versa)
- Declaring non-empty bodies on Common error models — they are stubs that reference YAML
- Mixing sort/filter logic inline in operations.tsp instead of referencing Common.SortQuery and Common.*FieldFilter

## Decisions

- **Filter types carry x-go-type extensions pointing to hand-written Go filter structs in api/v3/filters/.** — AIP filter semantics (eq/neq/contains/oeq/ocontains/exists) need parsing logic that is hard to generate; keeping the Go implementation hand-written and referencing it avoids duplicating that logic in generated code.
- **Error response models delegate to external YAML via @useRef.** — The YAML definitions are shared with the v1 spec and carry OpenAPI extensions; a TypeSpec-native redefinition would diverge and break shared client error handling.

## Example: Add a new filter type for an enum field

```
@friendlyName("EnumFieldFilter")
@extension("x-go-type", "filters.FilterEnum")
@extension("x-go-type-import", #{ path: "github.com/openmeterio/openmeter/api/v3/filters" })
union EnumFieldFilter {
  string: string,
  object: {
    eq?: string,
    neq?: string,
    @encode(ArrayEncoding.commaDelimited)
    oeq?: string[],
  },
}
```

<!-- archie:ai-end -->
