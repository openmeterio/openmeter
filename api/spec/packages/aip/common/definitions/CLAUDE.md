# definitions

<!-- archie:ai-start -->

> Shared OpenAPI YAML component library for the v3 AIP spec — defines reusable filter schemas, error responses, pagination metadata, security schemes, and common property types that all AIP resource routes reference via $ref. These files are not TypeSpec; they are raw OpenAPI YAML fragments consumed by the TypeSpec build pipeline.

## Patterns

**RFC 7807 error schema hierarchy** — All error schemas extend BaseError (status, title, instance, detail required) via allOf. BadRequestError adds invalid_parameters; specific errors add example values only. Never add a new top-level error schema without extending BaseError. (`BadRequestError: allOf: [$ref: BaseError, {type: object, required: [invalid_parameters], properties: {invalid_parameters: {$ref: InvalidParameters}}}]`)
**Filter type composition via oneOf** — Composite filter types (StringFieldFilter, NumericFieldFilter, DateTimeFieldFilter) are expressed as oneOf over atomic sub-types (StringFieldEqualsFilter, NumericFieldGTFilter, etc.) each with additionalProperties: false. New filter variants must be added as a new named sub-type and then listed in the parent oneOf — never inline anonymous objects in the parent. (`NumericFieldFilter: oneOf: [{type: number}, {$ref: NumericFieldEqualsFilter}, {$ref: NumericFieldLTFilter}, ...]`)
**x-examples on every filter schema** — Every filter schema must carry x-examples showing all supported variant forms (plain value, eq, lt, lte, gt, gte, contains, oeq, neq). Missing x-examples causes SDK generation to produce incomplete examples. (`StringFieldFilter: x-examples: {example-1: 'value', example-2: {eq: 'value'}, example-3: {contains: 'value'}, ...}`)
**Cursor-based pagination via CursorMetaPage** — Paginated list responses reference CursorMeta, CursorMetaWithTotal, or CursorMetaWithEstimatedTotal from metadatas.yaml — never define inline page metadata. CursorMetaPage requires next and previous (nullable strings) plus size. (`response meta: {$ref: '#/components/schemas/CursorMeta'} — wraps CursorMetaPage with first/last/next/previous/size`)
**readOnly: true on server-generated fields** — UUID, CreatedAt, UpdatedAt, and all error schema fields are marked readOnly: true. Writable UUID variant is UUID_RW. New response-only fields must carry readOnly: true to prevent SDK clients from attempting to set them. (`UUID: {type: string, format: uuid, readOnly: true} vs UUID_RW: {type: string, format: uuid} (no readOnly)`)
**Label schemas enforce pattern and maxProperties** — Labels and PublicLabels additionalProperties enforce pattern '^[a-z0-9A-Z]{1}([a-z0-9A-Z-._]*[a-z0-9A-Z]+)?$', maxLength: 63, maxProperties: 50. Update variants (LabelsUpdate, PublicLabelsUpdate) add nullable: true on values and writeOnly: true on the schema. (`Labels: {type: object, maxProperties: 50, additionalProperties: {type: string, pattern: '^[a-z0-9A-Z]...', maxLength: 63}}`)
**Security schemes defined centrally in security.yaml** — All bearer/cookie security schemes (personalAccessToken, systemAccountAccessToken, konnectAccessToken, portalAccessToken, serviceAccessToken, clientToken) live only in security.yaml. New auth mechanisms must be added here and referenced via $ref — never inline a securityScheme in a route file. (`personalAccessToken: {type: http, scheme: bearer, bearerFormat: Token}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `aip_filters.yaml` | Defines all reusable query-filter component schemas: StringFieldFilter, NumericFieldFilter, DateTimeFieldFilter, UuidFieldFilter, BooleanFieldFilter, SortQuery, LabelsFieldFilter, and their atomic sub-types. | Adding a new filter variant without also adding an x-examples entry causes incomplete SDK docs. UuidFieldFilter reuses StringField sub-types (StringFieldEqualsFilter, StringFieldOEQFilter, StringFieldNEQFilter) — do not duplicate those schemas. |
| `errors.yaml` | Defines the full error response component tree: ErrorResponse discriminated union, per-status responses (BadRequest…NotAvailable), BaseError, and all InvalidParameter variants (Standard, MinimumLength, MaximumLength, ChoiceItem, DependentItem). | ErrorResponse uses a status-keyed discriminator mapping — adding a new HTTP error code requires updating both the oneOf list and the discriminator.mapping. KonnectCPLegacy* variants are a parallel schema family for Kong-legacy JSON error format; do not mix them with RFC 7807 BaseError descendants. |
| `metadatas.yaml` | Defines cursor-based pagination schemas (CursorMetaPage, CursorMeta, CursorMetaWithTotal, CursorMetaWithEstimatedTotal, CursorPaginatedMetaWithSizeAndTotal) and query parameters (CursorPageQuery, PageBefore, PageAfter, PageSize). | CursorMetaWithSizeAndTotal is a separate schema from CursorMetaWithTotal — the former is used when total may be null on non-first pages. x-speakeasy-terraform-ignore: true on PageNumber/PageSize/PaginatedMeta must not be removed. |
| `properties.yaml` | Defines primitive property schemas (UUID, UUID_RW, CreatedAt, UpdatedAt, ExpiresAt, NullableTimestamp) and specialised ID types composed via allOf+x-flatten-allOf (UserId, TeamId, OrganizationId). | UUID is readOnly; use UUID_RW for user-settable ID fields. Specialised ID types use x-flatten-allOf: true to collapse allOf wrappers in generated SDK output — omitting this causes verbose nested types. |
| `konnect_properties.yaml` | Konnect-specific reusable schemas: NullableUUID, Labels/LabelsUpdate, PublicLabels/PublicLabelsUpdate, EntityType, and path parameters (AuditLogDestinationId, Workspace). | LabelsUpdate and PublicLabelsUpdate are writeOnly; Labels and PublicLabels are the read form. Do not collapse them into a single schema — the write variant must allow nullable values on additionalProperties. |
| `security.yaml` | Central registry of all OpenAPI securitySchemes used across the AIP spec. | clientToken carries x-internal: true — internal schemes must keep this flag to prevent public SDK exposure. Adding a new scheme here is not sufficient; it must also be referenced in the routes that require it. |

## Anti-Patterns

- Defining inline anonymous filter objects inside a oneOf — always create a named sub-type with a title and additionalProperties: false
- Adding error schemas that do not extend BaseError via allOf — breaks RFC 7807 contract and the ErrorResponse discriminator
- Omitting x-examples on new filter schemas — SDK generators use these to produce client-side examples
- Adding new security schemes directly in route files instead of security.yaml — breaks the central auth registry
- Mixing KonnectCPLegacy* schemas with standard BaseError descendants — they are separate error-format families and must not cross-reference

## Decisions

- **Filter types use oneOf over named atomic sub-types rather than inline objects** — Named sub-types allow SDK generators to produce distinct type names (e.g. NumericFieldLTFilter) and enable $ref reuse across resource schemas without duplicating definitions.
- **Pagination metadata is separated into CursorMeta vs CursorMetaWithTotal vs CursorMetaWithEstimatedTotal** — Different list endpoints have different cost profiles for computing exact vs estimated counts; forcing a single schema would either over-compute or mislead clients about total accuracy.
- **KonnectCPLegacy* schemas coexist alongside RFC 7807 BaseError schemas** — Legacy Kong gateway endpoints return plain JSON {message} errors; the AIP endpoints return application/problem+json. Both must be expressible in the same spec without one polluting the other.

## Example: Add a new string filter variant (e.g. prefix match) to StringFieldFilter

```
# 1. Define the atomic sub-type in aip_filters.yaml:
StringFieldPrefixFilter:
  title: StringFieldPrefixFilter
  description: Filters by prefix match on the string field.
  type: object
  additionalProperties: false
  properties:
    prefix:
      type: string
  required: [prefix]
  x-examples:
    example-1:
      prefix: 'acme-'

# 2. Add the $ref to StringFieldFilter's oneOf list:
// ...
```

<!-- archie:ai-end -->
