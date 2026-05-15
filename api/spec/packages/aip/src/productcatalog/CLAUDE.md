# productcatalog

<!-- archie:ai-start -->

> TypeSpec definitions for the ProductCatalog v3 API domain: Plan (versioned phases + rate cards), Addon, PlanAddon associations, Price polymorphism (free/flat/unit/graduated/volume), UnitConfig conversions, and CRUD+lifecycle operations. Primary constraint: Plan/Addon version and lifecycle status fields are server-managed Read-only — never set by clients.

## Patterns

**Versioned resources with server-computed status lifecycle** — Plan and Addon carry version (@visibility(Lifecycle.Read), @minValue(1)), effective_from/effective_to (Read-only), and status enum (draft/active/archived/scheduled). Status is never settable by clients — it is derived from effective dates on the server. (`@visibility(Lifecycle.Read) status: PlanStatus;
@visibility(Lifecycle.Read) effective_from?: Shared.DateTime;
@visibility(Lifecycle.Read) version: integer = 1;`)
**Lifecycle action sub-routes as @post operations on same interface** — State transitions (publish, archive) are modeled as POST sub-routes (/{id}/publish, /{id}/archive) on the same interface, returning Shared.UpdateResponse<T>. Not nested interfaces or PATCH fields. (`@post @route("/{planId}/publish") @operationId("publish-plan")
publishPlan(@path planId: Shared.ULID): Shared.UpdateResponse<Plan> | Common.ErrorResponses | Common.NotFound;`)
**Discriminated union for Price polymorphism** — Price and PriceUsageBased are @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) unions. Every variant model carries a `type` field matching the PriceType enum. PriceUsageBased is a sub-union (unit/graduated/volume only) for rate cards requiring usage-based pricing. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Price { free: PriceFree, flat: PriceFlat, unit: PriceUnit, graduated: PriceGraduated, volume: PriceVolume }`)
**validation_errors as Read-only list on all catalog resources** — Plan, Addon, and PlanAddon all carry @visibility(Lifecycle.Read) validation_errors?: ProductCatalogValidationError[] to surface draft-state validation issues through the API. Any new catalog resource must include this field. (`@visibility(Lifecycle.Read)
@summary("Validation errors")
validation_errors?: ProductCatalogValidationError[];`)
**UpsertRequest/UpsertResponse for PUT operations on versioned resources** — PUT operations (update-plan, update-addon, update-plan-addon) use Shared.UpsertRequest<T> body and Shared.UpsertResponse<T> return type — not CreateRequest/UpdateResponse. Responses also include Common.Gone for archived resources. (`@put updatePlan(@path planId: Shared.ULID, @body plan: Shared.UpsertRequest<Plan>):
  | Shared.UpsertResponse<Plan> | Common.ErrorResponses | Common.Gone | Common.NotFound;`)
**PickProperties spread for partial resource base models** — PlanPhase and RateCard use PickProperties<Shared.ResourceWithKey, "key" | "name" | "description" | "labels"> instead of the full ResourceWithKey spread, excluding id and timestamps because phases/rate cards are sub-resources without independent lifecycle. (`model PlanPhase {
  ...PickProperties<Shared.ResourceWithKey, "key" | "name" | "description" | "labels">;
  duration?: Shared.ISO8601Duration;
  rate_cards: RateCard[];
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `price.tsp` | Price discriminated union and all variant models. PriceUsageBased is a sub-union (unit/graduated/volume) used when a usage-based-only price type is needed on a rate card. | PriceTier requires at least one of flat_price or unit_price (documented semantics, not TypeSpec-enforced). PriceFlat embedded inside PriceTier does not carry a payment_term. |
| `ratecard.tsp` | RateCard model: key, feature reference, billing_cadence (null = one-time), price union, payment_term, commitments, discounts, tax_config. | billing_cadence nullable means one-time charge (only valid with flat prices). SettlementMode enum is defined here but verify usage before referencing. |
| `unitconfig.tsp` | UnitConfig model for raw-to-billing-unit conversion (divide/multiply + rounding) and InvoiceUsageQuantityDetail for audit trail on invoices. | rounding defaults to None; precision is only meaningful when rounding != none. applied_unit_config on InvoiceUsageQuantityDetail is a Read-only snapshot for historical correctness. |
| `plan.tsp` | Plan and PlanPhase models, PlanStatus enum, ProductCatalogValidationError model reused across all catalog entities. | phases has @minItems(1) — at least one phase required. billing_cadence on Plan is Create+Read only (immutable after creation). |
| `operations.tsp` | PlanOperations, AddonOperations, PlanAddonOperations interfaces. PlanAddon routes nest under /{planId}/addons/ and require both planId and planAddonId path params. | All operations carry UnstableExtension + InternalExtension. PlanAddon upsert uses both path params. |
| `addon.tsp` | Addon model mirroring Plan structure: key, version, instance_type (single/multiple), currency, effective dates, rate_cards, validation_errors. | AddonInstanceType affects max_quantity validation on PlanAddon — single instance addons must omit max_quantity. |

## Anti-Patterns

- Setting status, effective_from, or effective_to in Create/Update requests — they are server-computed Read-only fields
- Using Shared.CreateRequest/UpdateResponse for PUT plan/addon operations — must use UpsertRequest/UpsertResponse
- Omitting validation_errors from a new catalog resource model — required for draft-state UX across all catalog entities
- Using RateCard.price with PriceFree for usage-based rate cards — use PriceUsageBased union (unit/graduated/volume only)
- Adding a full ...Shared.Resource spread to PlanPhase or RateCard — these are sub-resources; use PickProperties to select only key/name/description/labels

## Decisions

- **UnitConfig is a separate model on RateCard rather than embedded in price types** — The same feature can be billed in different converted units across plans (bytes→GB, seconds→hours); keeping unit conversion at the rate card level decouples it from the pricing model and allows reuse.
- **Plan/Addon versioning is server-managed (version field Read-only, incremented on update)** — Prevents clients from accidentally creating version conflicts; the server controls the version chain ensuring subscription compatibility checks remain valid.
- **Lifecycle actions (publish, archive) as POST sub-routes, not PATCH status fields** — State transitions have server-side preconditions (e.g. publish requires no validation_errors); explicit action endpoints make the transition semantics discoverable and allow returning rich error responses.

## Example: Adding a new lifecycle action (e.g. suspend) to PlanOperations

```
// operations.tsp — inside interface PlanOperations:
@post
@route("/{planId}/suspend")
@operationId("suspend-plan")
@summary("Suspend a plan version.")
@extension(Shared.UnstableExtension, true)
@extension(Shared.InternalExtension, true)
suspendPlan(
  @path planId: Shared.ULID,
): Shared.UpdateResponse<Plan> | Common.ErrorResponses | Common.NotFound;
```

<!-- archie:ai-end -->
