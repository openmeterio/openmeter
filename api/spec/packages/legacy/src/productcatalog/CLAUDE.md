# productcatalog

<!-- archie:ai-start -->

> TypeSpec source of truth for the v1 product-catalog API surface: features, plans, addons, plan-addon assignments, rate cards, prices, discounts, tax, pro-rating, and subscriptions. Models defined here compile to api/openapi.yaml and downstream Go/JS SDKs via `make gen-api` — edit the .tsp, never the generated code.

## Patterns

**namespace OpenMeter for every model file** — Every .tsp declares `namespace OpenMeter;` after imports so all models join the single OpenMeter namespace; routes.tsp additionally does `using TypeSpec.Http;` and `using TypeSpec.OpenAPI;`. (`import "@typespec/http";
import "../types.tsp";
import "./ratecards.tsp";
namespace OpenMeter;`)
**@friendlyName on every exposed model/enum/union** — Each generated schema name comes from @friendlyName, not the TypeSpec identifier — e.g. `@friendlyName("RateCardFlatFee")`, `@friendlyName("BillingSettlementMode") enum SettlementMode`. Adding a model without @friendlyName produces an unstable auto-generated OpenAPI name. (`@friendlyName("FlatPrice")
model FlatPrice { type: PriceType.flat; amount: Money; }`)
**Discriminated unions for polymorphic types** — Price, RateCard, RateCardUsageBasedPrice, RateCardEntitlement, FeatureUnitCost, SubscriptionEditOperation all use `@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })` (or "op") with a per-variant literal discriminator field (e.g. `type: PriceType.flat`). (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union RateCard { flat_fee: RateCardFlatFee, usage_based: RateCardUsageBased }`)
**Lifecycle @visibility on every field** — Fields annotate `@visibility(Lifecycle.Read)`, `(Lifecycle.Read, Lifecycle.Create)`, or `(Read, Create, Update)` to drive Create/Replace model generation. Server-computed fields (version, status, effectiveFrom/To, annotations, validationErrors) are Read-only. (`@visibility(Lifecycle.Read)
status: PlanStatus;`)
**Compose, don't duplicate, via spread + OmitProperties/Resource mixins** — Models reuse shared shapes with `...UniqueResource`, `...ResourceTimestamps`, `...Archiveable`, `...global.Resource`, `...global.CadencedResource`, and derive create bodies with `OmitProperties<...>` / `ResourceCreateModel<Plan>` / `@withVisibility(Lifecycle.Create)` rather than redefining fields. (`model Feature { ...ResourceTimestamps; ...Archiveable; ...FeatureCreateInputs; @visibility(Lifecycle.Read) id: ULID; }`)
**ISO8601 durations and Money/Numeric scalars** — Durations use `duration` with `@encode(DurationKnownEncoding.ISO8601)`; monetary/quantity values use the shared `Money`, `Numeric`, `Percentage`, `CurrencyCode`, `ULID`, `Key` scalars from ../types.tsp — not raw number/string. (`@encode(DurationKnownEncoding.ISO8601)
billingCadence: duration | null;`)
**Route interfaces carry @route/@tag/@operationId per operation** — routes.tsp groups endpoints into interfaces (FeaturesEndpoints, PlansEndpoints, AddonsEndpoints, SubscriptionsEndpoints) under `@route("/api/v1/...")` + `@tag`, each op tagged `@operationId` (the SDK method name) and `@summary`, returning `Model | NotFoundError | CommonErrors`. (`@get @operationId("getFeature") @summary("Get feature")
get(@path featureId: string): Feature | NotFoundError | CommonErrors;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Package barrel — imports `..` then every sibling .tsp (discounts, plan, prices, ratecards, routes, tax, subscription, alignment, addon, planaddon). New files must be added here or they are not compiled. | Adding a new .tsp without an import line here silently drops it from the OpenAPI output. |
| `routes.tsp` | All HTTP operations for features/plans/plan-addons/addons/subscriptions. Only file using @route/@get/@post and the `using TypeSpec.Http` directive. | @operationId is the public SDK method name — renaming is a breaking SDK change. Subscription ops return CommonSubscriptionErrors (with validation extensions), not plain CommonErrors. |
| `ratecards.tsp` | RateCardBase<T> generic + RateCardFlatFee/RateCardUsageBased + RateCardEntitlement union deriving from EntitlementMetered/Static/BooleanCreateInputs via OmitProperties. | Entitlement templates omit featureKey/featureId/usagePeriod because the rate card supplies them; do not re-add those fields. |
| `prices.tsp` | Price union (flat/unit/tiered/dynamic/package) and the *WithCommitments variants that spread SpendCommitments. | RateCardUsageBasedPrice (in ratecards.tsp) references the WithCommitments variants; flat-fee rate cards use FlatPriceWithPaymentTerm only. |
| `plan.tsp` | Plan, PlanPhase, PlanStatus, SettlementMode (@friendlyName BillingSettlementMode), PlanReference/Input. proRatingConfig and settlementMode carry default object literals. | PlanStatus is Read-only/computed from effectiveFrom/effectiveTo; phases requires @minItems(1). |
| `subscription.tsp` | Largest model file: Subscription, SubscriptionExpanded, phases/items, Create/Change unions (Plan vs Custom), SubscriptionTiming, and the SubscriptionEditOperation discriminated union (op-based). | Imports ../entitlements/main.tsp and uses `global.` prefixed shared types; SubscriptionEdit.customizations capped at @maxItems(100). |
| `errors.tsp` | Subscription-specific error envelope: CommonSubscriptionErrors alias and Subscription{BadRequest,Conflict}ErrorResponse overriding `extensions` with SubscriptionErrorExtensions (validationErrors). | These wrap the base BadRequestError/ConflictError via OmitProperties<..., "extensions">; keep them in sync with ../errors.tsp. |
| `features.tsp` | Feature/FeatureCreateInputs plus FeatureUnitCost union (manual vs llm) and FeatureLLMUnitCost pricing lookup. Uses `using TypeSpec.OpenAPI`. | meterGroupByFilters is #deprecated in favor of advancedMeterGroupByFilters; LLM cost provider/model/tokenType each have a *Property variant that is mutually exclusive with the static value. |

## Anti-Patterns

- Editing api/openapi.yaml or generated Go/JS SDK files instead of these .tsp sources — they are overwritten by `make gen-api`.
- Adding a model/enum without @friendlyName, or a new .tsp without importing it in main.tsp.
- Adding @query/@route/@get decorators in a file that lacks `import "@typespec/http"` + `using TypeSpec.Http;` (compile error: Unknown decorator).
- Making a server-computed field (version, status, effective dates, validationErrors, annotations) Create/Update-writable instead of @visibility(Lifecycle.Read).
- Redefining fields by hand instead of composing with spread/OmitProperties/ResourceCreateModel, drifting create bodies out of sync with the read model.

## Decisions

- **TypeSpec is the single source of truth; OpenAPI and all SDKs are generated.** — One contract definition keeps v1 server code, Go client, and JS client provably in sync; per AGENTS.md the workflow is edit .tsp -> make gen-api -> make generate.
- **Envelope-less discriminated unions keyed on `type`/`op`.** — Produces flat OpenAPI discriminator schemas that map cleanly to Go interface implementations for prices, rate cards, and subscription edit operations without an extra wrapper object.
- **Subscriptions get a dedicated error family (CommonSubscriptionErrors with validation extensions).** — Subscription create/edit/change need structured validationErrors in the 400/409 body that the generic CommonErrors do not carry.

## Example: Defining a polymorphic catalog type with a friendly name, an envelope-less discriminator, and lifecycle visibility.

```
import "@typespec/http";
import "../types.tsp";

namespace OpenMeter;

@friendlyName("FlatPrice")
model FlatPrice {
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.flat;

  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  amount: Money;
}

@friendlyName("Price")
// ...
```

<!-- archie:ai-end -->
