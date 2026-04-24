# common

<!-- archie:ai-start -->

> Organisational parent for shared OpenAPI YAML component fragments consumed by the v3 AIP TypeSpec build pipeline. Its single child (definitions/) contains all reusable schemas, error responses, pagination metadata, filter types, and security schemes that every AIP route references via $ref.

## Decisions

- **All shared components live in a single definitions/ child rather than being scattered alongside route files.** — Centralising $ref targets prevents duplicate schema definitions and ensures the TypeSpec build pipeline resolves all component references from one canonical location.

<!-- archie:ai-end -->
