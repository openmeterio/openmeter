# productcatalog

<!-- archie:ai-start -->

> TypeSpec definitions for the ProductCatalog domain: Plan (versioned phases with rate cards), Addon, PlanAddon associations, pricing models (free/flat/unit/graduated/volume), UnitConfig conversions, and CRUD+lifecycle operations compiled to v3 OpenAPI.

## Patterns

**Versioned resources with status lifecycle** — Plan and Addon use version (integer, Read-only, minValue(1)), effective_from/effective_to (Read-only), and a computed status enum (draft/active/archived/scheduled). Status is never set directly — derived from effective dates. (`@visibility(Lifecycle.Read) status: PlanStatus;
@visibility(Lifecycle.Read) effective_from?: Shared.DateTime;`)
**Lifecycle action sub-routes as @post operations** — State transitions (publish, archive) are modeled as POST sub-routes (/{id}/publish, /{id}/archive) rather than PATCH fields, returning Shared.UpdateResponse<T>. (`@post @route("/{planId}/publish") @operationId("publish-plan")
publishPlan(@path planId: Shared.ULID): Shared.UpdateResponse<Plan> | ...`)
**Discriminated union for Price polymorphism** — Price and PriceUsageBased are `@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })` unions. Every variant model carries a `type` field matching the PriceType enum. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Price { free: PriceFree, flat: PriceFlat, unit: PriceUnit, graduated: PriceGraduated, volume: PriceVolume }`)
**validation_errors as read-only list on resources** — Plan, Addon, and PlanAddon all carry `@visibility(Lifecycle.Read) validation_errors?: ProductCatalogValidationError[]` to surface draft-state validation issues through the API. (`@visibility(Lifecycle.Read)
validation_errors?: ProductCatalogValidationError[];`)
**Upsert operations use Shared.UpsertRequest<T> and Shared.UpsertResponse<T>** — PUT operations on versioned resources (update-plan, update-addon, update-plan-addon) use UpsertRequest/UpsertResponse wrappers, not CreateRequest/UpdateResponse. (`@put updatePlan(@path planId: Shared.ULID, @body plan: Shared.UpsertRequest<Plan>): Shared.UpsertResponse<Plan> | ...`)
**PlanPhase uses PickProperties spread to reuse ResourceWithKey fields** — PlanPhase spreads only selected fields from ResourceWithKey via `PickProperties<Shared.ResourceWithKey, "key" | "name" | "description" | "labels">` instead of the full spread. (`model PlanPhase {
  ...PickProperties<Shared.ResourceWithKey, "key" | "name" | "description" | "labels">;
  duration?: Shared.ISO8601Duration;
  rate_cards: RateCard[];
}`)

## Key Files

| File             | Role                                                                                                                                                           | Watch For                                                                                                                                                                                                             |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ratecard.tsp`   | RateCard model (key, feature ref, billing_cadence, price, payment_term, commitments, discounts, tax_config). Imports Features namespace for feature reference. | billing_cadence nullable means one-time charge (only valid for flat prices). settlement_mode enum is defined here but may not be referenced on RateCard directly — check for usage.                                   |
| `price.tsp`      | Price discriminated union and all variant models (PriceFree, PriceFlat, PriceUnit, PriceGraduated, PriceVolume) plus PriceTier and SpendCommitments.           | PriceUsageBased is a sub-union (unit/graduated/volume only) used when a usage-based-only price type is needed. PriceTier requires at least one of flat_price or unit_price set (docs only, not enforced by TypeSpec). |
| `unitconfig.tsp` | UnitConfig model for raw-to-billing-unit conversion and InvoiceUsageQuantityDetail for audit trail. Explains v1 DynamicPrice/PackagePrice equivalents.         | rounding defaults to None; precision only meaningful when rounding != none. applied_unit_config on InvoiceUsageQuantityDetail is a snapshot (Read-only) for historical correctness.                                   |
| `plan.tsp`       | Plan and PlanPhase models, PlanStatus enum, ProductCatalogValidationError model reused across all catalog entities.                                            | phases has @minItems(1) — at least one phase required. billing_cadence on Plan is Create+Read only (immutable).                                                                                                       |
| `operations.tsp` | PlanOperations, AddonOperations, PlanAddonOperations interfaces. PlanAddon sub-resource routes nest under /{planId}/addons/.                                   | All operations carry the three extension markers (Private/Unstable/Internal). PlanAddon operations require both @path planId and @path planAddonId.                                                                   |

## Anti-Patterns

- Setting status or effective_from/effective_to directly in Create/Update requests — they are server-computed Read-only fields
- Using CreateRequest/UpdateResponse for PUT plan/addon operations — use UpsertRequest/UpsertResponse
- Adding pricing logic (tier calculations, commitment enforcement) to TypeSpec models — belongs in Go billing service layer
- Omitting validation_errors from new catalog resource models — required for draft-state UX
- Using RateCard.price with PriceFree for usage-based rate cards — PriceUsageBased union restricts to unit/graduated/volume

## Decisions

- **UnitConfig is a separate model on RateCard rather than embedded in price types** — The same feature can be billed in different converted units across plans (bytes→GB, seconds→hours); keeping unit conversion at the rate card level decouples it from the pricing model and allows reuse.
- **Plan/Addon versioning is server-managed (version field Read-only, incremented on update)** — Prevents clients from accidentally creating version conflicts; the server controls the version chain ensuring subscription compatibility checks remain valid.

## Example: Adding a new lifecycle action (e.g. suspend-plan) to PlanOperations

```
// operations.tsp — inside interface PlanOperations
@post
@route("/{planId}/suspend")
@operationId("suspend-plan")
@summary("Suspend a plan version.")
@extension(Shared.PrivateExtension, true)
@extension(Shared.UnstableExtension, true)
@extension(Shared.InternalExtension, true)
suspendPlan(
  @path planId: Shared.ULID,
): Shared.UpdateResponse<Plan> | Common.ErrorResponses | Common.NotFound;
```

<!-- archie:ai-end -->
