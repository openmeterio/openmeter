# types

<!-- archie:ai-start -->

> Holds manually-authored Go types that fill gaps where oapi-codegen cannot generate correct types from the OpenAPI spec (e.g. oneOf/anyOf schemas). Types here are referenced from the OpenAPI spec via `x-go-type` so generated clients use them correctly — this package must never contain business logic.

## Patterns

**x-go-type reference only** — New types are added here only when oapi-codegen cannot generate them (oneOf/anyOf). The type is then referenced in the OpenAPI spec via the `x-go-type` extension, not imported directly by hand-written code. (`In openapi.yaml: schema: x-go-type: types.MyUnionType`)
**Pure type definitions** — Files in this package contain only type declarations (struct, interface, type alias). No functions, no methods, no init(), no business logic. (`type MyUnionType struct { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `doc.go` | Package declaration and authoritative comment explaining when and why to add types here — read before adding any new type. | The comment is the contract: only add types for spec constructs oapi-codegen cannot handle (oneOf/anyOf). If oapi-codegen can handle it, do not add it here. |

## Anti-Patterns

- Adding types that oapi-codegen can generate automatically from the OpenAPI spec
- Adding business logic, helper functions, or methods to types in this package
- Duplicating types already present in api/api.gen.go or api/v3/api.gen.go
- Importing this package from domain code (openmeter/*) — it is consumed via x-go-type by generated code only
- Editing api/api.gen.go or api/v3/api.gen.go to inline types instead of using this package

## Decisions

- **Types live in a separate package rather than inline in generated files** — Generated files (api/api.gen.go, api/v3/api.gen.go) must not be hand-edited; a separate package allows manual type definitions to survive regeneration cycles while remaining importable by generated code via x-go-type references.

<!-- archie:ai-end -->
