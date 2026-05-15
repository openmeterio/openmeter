# common

<!-- archie:ai-start -->

> Organisational parent for shared OpenAPI YAML component fragments consumed by the v3 AIP TypeSpec build pipeline. Its single child (definitions/) centralises all reusable schemas, error responses, pagination metadata, filter types, and security schemes that every AIP route references via $ref.

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `CLAUDE.md` | Archie-generated intent layer for the common/ organisational folder; describes the role of definitions/ and the centralisation decision. | Do not add TypeSpec source files directly here — all component fragments belong in definitions/. |

## Anti-Patterns

- Adding OpenAPI YAML fragments directly in common/ instead of definitions/ — the build pipeline resolves $ref targets from definitions/ only.
- Duplicating schema definitions across route-specific files instead of $ref-ing the shared definitions — breaks the single-source guarantee.
- Adding route bindings or operation definitions here — this folder is shared infrastructure, not a route layer.

## Decisions

- **All shared components live in a single definitions/ child rather than being scattered alongside route files.** — Centralising $ref targets prevents duplicate schema definitions and ensures the TypeSpec build pipeline resolves all component references from one canonical location.

<!-- archie:ai-end -->
