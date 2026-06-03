# productcatalog

<!-- archie:ai-start -->

> TypeSpec source for the v1 product-catalog, subscription, and add-on API contracts. Defines all models, enums, unions and route interfaces compiled into api/openapi.yaml and Go server stubs — only edit these .tsp files, never the generated outputs.

## Patterns

**Discriminated unions with envelope:none** — All polymorphic types (Price, RateCard, RateCardUsageBasedPrice, FeatureUnitCost, SubscriptionEditOperation) use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }). SubscriptionEditOperation uniquely uses "op" as the discriminator, not "type". (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union Price { flat: FlatPrice, unit: UnitPrice, tiered: TieredPrice, dynamic: DynamicPrice, package: PackagePrice }`)
**Per-field @visibility lifecycle annotation** — Every concrete model field carries an explicit @visibility(...). Server-computed fields (status, effectiveFrom, effectiveTo, version, validationErrors, annotations) are Lifecycle.Read only. Only abstract base models (RateCardBase<T>) may omit visibility on some fields. (`@visibility(Lifecycle.Read) status: PlanStatus;`)
**Spread-based composition over extends** — Models compose by spreading shared bases (...UniqueResource, ...ResourceTimestamps, ...RateCardBase<T>, ...SpendCommitments, ...FlatPrice) instead of TypeSpec extends — enabling OmitProperties overrides in create variants. (`model RateCardFlatFee { ...RateCardBase<RateCardType.flatFee>; billingCadence: duration | null; price: FlatPriceWithPaymentTerm | null; }`)
**Create/update variants via OmitProperties + withVisibility** — Write models (PlanAddonCreate, SubscriptionAddonCreate) use @withVisibility(Lifecycle.Create) and OmitProperties<DefaultKeyVisibility<Model, Lifecycle.Read>, "field"> to strip read-only fields and replace embedded objects with just an ID. Never duplicate fields from scratch. (`@withVisibility(Lifecycle.Create) model PlanAddonCreate { ...OmitProperties<DefaultKeyVisibility<PlanAddon, Lifecycle.Read>, "addon">; addonId: ULID; }`)
**All routes declared exclusively in routes.tsp** — Every HTTP interface (FeaturesEndpoints, PlansEndpoints, PlanAddonsEndpoints, AddonsEndpoints, SubscriptionsEndpoints) is declared only in routes.tsp with @route/@tag/@friendlyName/@operationId/@summary. No other file declares route interfaces. (`@route("/api/v1/plans") @tag("Product Catalog") interface PlansEndpoints { @get @operationId("listPlans") list(...): PaginatedResponse<Plan> | CommonErrors; }`)
**Subscription mutations use CommonSubscriptionErrors** — Subscription create/edit/change/cancel routes use CommonSubscriptionErrors (errors.tsp) carrying SubscriptionBadRequestErrorResponse/SubscriptionConflictErrorResponse with validationErrors[]. Feature/plan/addon routes use the parent CommonErrors alias. (`alias CommonSubscriptionErrors = SubscriptionBadRequestErrorResponse | SubscriptionConflictErrorResponse | ...;`)
**ISO8601 encoding on all duration fields** — Every duration field uses @encode(DurationKnownEncoding.ISO8601) with an @example(duration.fromISO("P1M")). One-time-fee cadence is typed duration | null. Omitting the encoding defaults to seconds serialization. (`@encode(DurationKnownEncoding.ISO8601) @example(duration.fromISO("P1M")) billingCadence: duration | null;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routes.tsp` | Sole file declaring all HTTP interfaces; imports rest.tsp, subscriptionaddon.tsp, errors.tsp. | Missing @operationId/@friendlyName breaks Go codegen naming. Subscription mutations must use CommonSubscriptionErrors, not CommonErrors. |
| `ratecards.tsp` | RateCardBase<T> generic, RateCard union (flat_fee | usage_based), flat/usage rate cards, RateCardUsageBasedPrice union, and entitlement templates. | A new RateCard variant requires updating both the RateCard union and RateCardUsageBasedPrice union. Entitlement templates strip featureKey/featureId/usagePeriod via OmitProperties. |
| `prices.tsp` | Price union, PriceType enum, all price models (Flat/Unit/Tiered/Dynamic/Package) and their *WithCommitments variants spreading SpendCommitments. | A new price type must be added to PriceType enum, the Price union, AND RateCardUsageBasedPrice in ratecards.tsp — missing any one leaves the type unreachable from rate cards. |
| `subscription.tsp` | Subscription/Expanded, phase/item/edit models, SubscriptionCreate and SubscriptionChange @oneOf unions, and all SubscriptionEditOperation variants. | SubscriptionEditOperation uses discriminatorPropertyName: "op", not "type". @oneOf union variants need @summary. |
| `plan.tsp` | Plan, PlanPhase, PlanStatus, SettlementMode, PlanOrderBy, ProRatingConfig usage, PlanReference(Input). | Plan.phases has @minItems(1); Plan.validationErrors is ValidationError[] | null (never optional). billingCadence/duration require ISO8601 encoding. |
| `errors.tsp` | CommonSubscriptionErrors alias and the subscription-specific error response models with SubscriptionErrorExtensions.validationErrors. | Imports ratecards.tsp — avoid creating a circular import if ratecards.tsp later needs error types. |
| `main.tsp` | Package entry point importing sub-files in dependency order. | features.tsp and subscriptionaddon.tsp are intentionally NOT imported here (used via parent package / routes.tsp chain). Importing features.tsp here causes duplicate-namespace conflicts. |
| `features.tsp` | Feature/FeatureCreateInputs models plus the FeatureUnitCost discriminated union (manual/llm) and resolved LLM pricing. | meterGroupByFilters is deprecated in favor of advancedMeterGroupByFilters. FeatureLLMUnitCost.pricing is Lifecycle.Read only. |

## Anti-Patterns

- Defining a polymorphic type without @discriminated(#{ envelope: "none", ... }) — produces a wrapped discriminator that breaks client deserialization.
- Adding a route interface in any file other than routes.tsp — breaks @tag grouping and operationId consistency.
- Duplicating fields for create/update variants instead of OmitProperties<DefaultKeyVisibility<Model, Lifecycle.Read>, "..."> + @withVisibility.
- Using a duration field without @encode(DurationKnownEncoding.ISO8601) — defaults to seconds serialization.
- Using CommonErrors instead of CommonSubscriptionErrors for subscription mutation routes — drops validationErrors from the schema.

## Decisions

- **SubscriptionEditOperation uses discriminatorPropertyName: "op" instead of "type".** — Distinguishes PATCH command operations from resource-type discriminators used in Price/RateCard, avoiding naming collisions in generated code.
- **PlanAddonCreate / SubscriptionAddonCreate replace the embedded Addon object with just { id } / addonId on write.** — Write operations only need the identifier; embedding the read-only Addon would force clients to supply server-computed fields like status and version.
- **CommonSubscriptionErrors is a local alias rather than reusing parent CommonErrors.** — Subscription mutations return error bodies with a validationErrors array (SubscriptionErrorExtensions) not present in generic error types.

## Example: Adding a new usage-based price type to the catalog

```
// prices.tsp: enum PriceType { ..., stepped: "stepped" }
model SteppedPrice { @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) type: PriceType.stepped; amount: Money; }
model SteppedPriceWithCommitments { ...SteppedPrice; ...SpendCommitments; }
// add stepped to Price union (prices.tsp) AND RateCardUsageBasedPrice union (ratecards.tsp)
```

<!-- archie:ai-end -->
