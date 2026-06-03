# types

<!-- archie:ai-start -->

> Manually-authored Go types that fill gaps where oapi-codegen cannot generate correct types from the OpenAPI spec (notably oneOf/anyOf schemas). Referenced from the spec via the `x-go-type` extension so generated clients use them; this package must never contain business logic.

## Patterns

**x-go-type reference only** — Add a type here only when oapi-codegen cannot generate it (oneOf/anyOf), then reference it in the OpenAPI spec via x-go-type rather than importing it by hand. (`In openapi.yaml: schema: x-go-type: types.MyUnionType`)
**Pure type definitions** — Files contain only type declarations (struct, interface, alias). No functions, methods, init(), or business logic. (`type MyUnionType struct { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `doc.go` | Package declaration plus the authoritative comment explaining when/why to add types here. | The comment is the contract: only add types for spec constructs oapi-codegen cannot handle (oneOf/anyOf). If oapi-codegen can handle it, do not add it here. |

## Anti-Patterns

- Adding types that oapi-codegen can generate automatically from the spec
- Adding business logic, helper functions, or methods to types here
- Duplicating types already present in api/api.gen.go or api/v3/api.gen.go
- Importing this package from domain code (openmeter/*) — it is consumed via x-go-type by generated code only
- Hand-editing api/api.gen.go to inline types instead of using this package

## Decisions

- **Types live in a separate package rather than inline in generated files** — Generated files must not be hand-edited; a separate package lets manual type definitions survive regeneration while remaining importable via x-go-type references.

<!-- archie:ai-end -->
