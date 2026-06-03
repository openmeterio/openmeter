# definitions

<!-- archie:ai-start -->

> Shared OpenAPI YAML component library for the v3 AIP spec: reusable filter schemas, RFC 7807 error responses, cursor pagination metadata, security schemes, and primitive property types referenced via $ref by all AIP routes. These are raw OpenAPI YAML fragments consumed by the TypeSpec build pipeline, not TypeSpec files.

## Patterns

**RFC 7807 error hierarchy via allOf BaseError** — Every error schema extends BaseError (status, title, instance, detail required) via allOf; BadRequestError adds invalid_parameters, others add example values only. Never add a top-level error schema without extending BaseError. (`BadRequestError: allOf: [$ref: BaseError, {required: [invalid_parameters], properties: {invalid_parameters: {$ref: InvalidParameters}}}]`)
**Filter composition via oneOf over named atomic sub-types** — Composite filters (StringFieldFilter, NumericFieldFilter, DateTimeFieldFilter) are oneOf over named sub-types each with additionalProperties:false. New variants must be a named sub-type first, then listed in the parent oneOf — never inline anonymous objects. (`NumericFieldFilter: oneOf: [{type: number}, {$ref: NumericFieldEqualsFilter}, {$ref: NumericFieldLTFilter}, ...]`)
**x-examples required on every filter schema** — Every filter schema carries x-examples covering all variant forms; missing them produce incomplete SDK examples. (`StringFieldFilter: x-examples: {example-1: 'value', example-2: {eq: 'value'}, example-3: {contains: 'value'}}`)
**Cursor pagination via CursorMeta family** — Paginated responses reference CursorMeta, CursorMetaWithTotal, or CursorMetaWithEstimatedTotal from metadatas.yaml; never inline page metadata. CursorMetaPage requires size, next, previous (nullable strings). (`meta: {$ref: '#/components/schemas/CursorMeta'}  # wraps CursorMetaPage with first/last/next/previous/size`)
**readOnly:true on server-generated fields; UUID_RW for writable IDs** — UUID, CreatedAt, UpdatedAt and all error fields are readOnly:true; UUID_RW is the writable ID variant. New response-only fields must carry readOnly:true. (`UUID: {format: uuid, readOnly: true}  vs  UUID_RW: {format: uuid}  # no readOnly`)
**x-flatten-allOf:true on specialised ID types** — ID types composed via allOf (UserId, TeamId, OrganizationId, NullableTimestamp) include x-flatten-allOf:true to collapse the allOf wrapper in SDK output. (`UserId: {x-flatten-allOf: true, allOf: [{$ref: '#/.../UUID'}, {description: '...'}]}`)
**Security schemes only in security.yaml** — All bearer/cookie security schemes live in security.yaml; internal ones carry x-internal:true. Never inline a securityScheme in a route file. (`clientToken: {x-internal: true, type: http, scheme: bearer, bearerFormat: Token}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `aip_filters.yaml` | All reusable query-filter schemas: String/Numeric/DateTime/Uuid/Boolean FieldFilter, SortQuery, Labels/PublicLabels/AttributesFieldFilter, and atomic sub-types. | UuidFieldFilter reuses StringFieldEquals/OEQ/NEQ sub-types — do not duplicate. Adding a variant without x-examples gives incomplete SDK docs. |
| `errors.yaml` | Full error tree: ErrorResponse status-discriminated union, per-status responses, BaseError, and InvalidParameter variants (Standard/Min/Max/Choice/Dependent). | Adding an HTTP error code requires updating both the oneOf list and discriminator.mapping. KonnectCPLegacy* is a parallel plain-JSON family — do not cross-reference with BaseError descendants. |
| `metadatas.yaml` | Cursor pagination schemas (CursorMetaPage, CursorMeta, CursorMetaWithTotal/EstimatedTotal/SizeAndTotal) and page query params (CursorPageQuery, PageBefore/After/Size/Number). | CursorMetaWithSizeAndTotal differs from CursorMetaWithTotal (total may be null on non-first pages). Do not remove x-speakeasy-terraform-ignore on PageNumber/PageSize/PaginatedMeta. |
| `properties.yaml` | Primitive property schemas (UUID, UUID_RW, CreatedAt, UpdatedAt, ExpiresAt, NullableTimestamp) and allOf+x-flatten-allOf ID types (UserId, PrincipalId, TeamId, OrganizationId). | UUID is readOnly; use UUID_RW for user-settable IDs. ID types need x-flatten-allOf:true. |
| `konnect_properties.yaml` | Konnect-specific schemas: NullableUUID, Labels/LabelsUpdate, PublicLabels/PublicLabelsUpdate, EntityType, path params (AuditLogDestinationId, Workspace). | LabelsUpdate/PublicLabelsUpdate are writeOnly nullable variants — do not collapse read and write forms into one schema. |
| `security.yaml` | Central registry of all securitySchemes (personalAccessToken, systemAccountAccessToken, konnectAccessToken, portalAccessToken, serviceAccessToken, clientToken). | clientToken carries x-internal:true — keep it to prevent public SDK exposure. Defining a scheme here is not enough; it must be referenced by routes. |

## Anti-Patterns

- Inline anonymous filter objects inside a oneOf instead of a named sub-type with additionalProperties:false
- Error schemas not extending BaseError via allOf — breaks RFC 7807 and the ErrorResponse discriminator
- Omitting x-examples on new filter schemas
- Adding security schemes in route files instead of security.yaml
- Mixing KonnectCPLegacy* schemas with standard BaseError descendants

## Decisions

- **Filter types use oneOf over named atomic sub-types** — Named sub-types let SDK generators produce distinct type names (e.g. NumericFieldLTFilter) and enable $ref reuse without duplication.
- **Pagination split into CursorMeta vs WithTotal vs WithEstimatedTotal** — Endpoints have different cost profiles for exact vs estimated counts; one schema would over-compute or mislead clients about accuracy.
- **KonnectCPLegacy* schemas coexist with RFC 7807 BaseError** — Legacy Kong gateway endpoints return plain {message} JSON while AIP endpoints return application/problem+json; both must be expressible without polluting each other.

<!-- archie:ai-end -->
