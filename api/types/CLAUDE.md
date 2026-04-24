# types

<!-- archie:ai-start -->

> Holds manually-authored Go types that fill gaps where oapi-codegen cannot generate correct types from the OpenAPI spec (e.g. oneOf/anyOf schemas). Types defined here are referenced from the OpenAPI spec via `x-go-type` so the generated Go client uses them correctly.

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `doc.go` | Package declaration and authoritative comment explaining when and why to add types here — read before adding any new type | Adding types here for anything oapi-codegen CAN handle; this package is strictly for unsupported spec constructs |

## Anti-Patterns

- Adding types that oapi-codegen can generate automatically
- Adding business logic or helper functions — this is a pure type definition package
- Duplicating types already present in api/api.gen.go or api/v3/api.gen.go

## Decisions

- **Types live in a separate package rather than inline in api.gen.go** — Generated files must not be edited; a separate package allows manual type definitions to survive regeneration cycles while still being importable by generated code via x-go-type references

<!-- archie:ai-end -->
