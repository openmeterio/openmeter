# productcatalog

<!-- archie:ai-start -->

> TypeSpec source for the v1 product catalog, subscription, and add-on API contracts. Defines all models, enums, unions, and route interfaces that compile into api/openapi.yaml and drive Go server stubs — never edit generated outputs, only these .tsp files.

## Patterns

**Discriminated union with envelope:none** — All polymorphic types (Price, RateCard, RateCardEntitlement, SubscriptionEditOperation, FeatureUnitCost) use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" | "op" }) so the discriminator sits at the top level with no wrapper object. SubscriptionEditOperation uniquely uses 'op' as the discriminator property name, not 'type'. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union Price { flat: FlatPrice, unit: UnitPrice, tiered: TieredPrice, dynamic: DynamicPrice, package: PackagePrice }`)
**Visibility lifecycle annotation on every model field** — Every concrete model field carries @visibility(Lifecycle.Read), @visibility(Lifecycle.Read, Lifecycle.Create), or similar. Computed/server-assigned fields (status, effectiveFrom, effectiveTo, version, annotations) are Lifecycle.Read only. Only abstract base models like RateCardBase<T> may omit visibility on some fields. (`@visibility(Lifecycle.Read) status: PlanStatus; // server-computed, never writable`)
**Spread-based composition over TypeSpec extends** — Models are composed by spreading shared base models (...UniqueResource, ...ResourceTimestamps, ...RateCardBase<T>, ...SpendCommitments) rather than using TypeSpec extends. This gives explicit field ownership and enables OmitProperties<> overrides in create/update variants. (`model RateCardFlatFee { ...RateCardBase<RateCardType.flatFee>; billingCadence: duration | null; price: FlatPriceWithPaymentTerm | null; }`)
**Create/update variants via OmitProperties + withVisibility** — Write models (PlanAddonCreate, SubscriptionAddonCreate) are produced with @withVisibility(Lifecycle.Create) and OmitProperties<DefaultKeyVisibility<Model, Lifecycle.Read>, "field"> to strip read-only fields and replace embedded objects with just their ID. Never duplicate model fields from scratch. (`@withVisibility(Lifecycle.Create) model PlanAddonCreate { ...OmitProperties<DefaultKeyVisibility<PlanAddon, Lifecycle.Read>, "addon">; addonId: ULID; }`)
**All routes declared exclusively in routes.tsp** — Every HTTP interface (FeaturesEndpoints, PlansEndpoints, PlanAddonsEndpoints, AddonsEndpoints, SubscriptionsEndpoints) is declared in routes.tsp with @route, @tag, @friendlyName, @operationId, and @summary on every operation. No other .tsp file in this folder may declare route interfaces. (`@route("/api/v1/plans") @tag("Product Catalog") @friendlyName("Plans") interface PlansEndpoints { @get @operationId("listPlans") @summary("List plans") list(...): PaginatedResponse<Plan> | CommonErrors; }`)
**Domain-specific error alias for subscription mutations** — Subscription create/edit/change/cancel routes use CommonSubscriptionErrors (defined in errors.tsp) which extends the generic errors with SubscriptionBadRequestErrorResponse and SubscriptionConflictErrorResponse carrying SubscriptionErrorExtensions.validationErrors[]. Feature/plan/addon routes use the parent package's CommonErrors alias instead. (`alias CommonSubscriptionErrors = SubscriptionBadRequestErrorResponse | SubscriptionConflictErrorResponse | UnauthorizedError | ForbiddenError | InternalServerErrorError | ...;`)
**ISO8601 duration encoding on all duration fields** — Every duration field uses @encode(DurationKnownEncoding.ISO8601) and an @example(duration.fromISO("P1M")) annotation. Nullable billing cadence (one-time fee rate cards) is typed as duration | null. Omitting the encoding directive causes the TypeSpec compiler to default to seconds serialization. (`@encode(DurationKnownEncoding.ISO8601) @example(duration.fromISO("P1M")) billingCadence: duration | null;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routes.tsp` | Sole file declaring all HTTP interfaces; imports rest.tsp, subscriptionaddon.tsp, and errors.tsp. Uses 'using TypeSpec.Http; using TypeSpec.OpenAPI;'. | Missing @operationId or @friendlyName breaks Go codegen naming. Adding a route interface in any other file breaks @tag grouping. Subscription mutations must use CommonSubscriptionErrors, not CommonErrors. |
| `ratecards.tsp` | Defines RateCardBase<T> generic, RateCard union (flat_fee | usage_based), RateCardFlatFee, RateCardUsageBased, RateCardUsageBasedPrice union, and entitlement templates (RateCardMeteredEntitlement, RateCardStaticEntitlement, RateCardBooleanEntitlement). | Adding a new RateCard variant requires updating both the RateCard union AND RateCardUsageBasedPrice union. Entitlement templates must strip featureKey/featureId/usagePeriod via OmitProperties. |
| `prices.tsp` | Defines Price union and all price models (FlatPrice, UnitPrice, TieredPrice, DynamicPrice, PackagePrice) plus *WithCommitments variants that spread SpendCommitments. | New price type must be added to Price union, PriceType enum, and RateCardUsageBasedPrice union in ratecards.tsp. Forgetting any of the three leaves the type unreachable from rate cards. |
| `subscription.tsp` | Defines Subscription, SubscriptionExpanded, SubscriptionPhase/Item/Edit, SubscriptionCreate/@oneOf union (PlanSubscriptionCreate | CustomSubscriptionCreate), SubscriptionChange union, and all SubscriptionEditOperation variants. | SubscriptionCreate and SubscriptionChange are @oneOf unions — new variants need @summary. SubscriptionEditOperation uses discriminatorPropertyName: 'op', not 'type'. Expanding SubscriptionExpanded must not omit itemTimelines. |
| `plan.tsp` | Defines Plan, PlanPhase, PlanStatus, SettlementMode, PlanOrderBy, and PlanReferenceInput. Plan.phases has @minItems(1) and Plan.validationErrors is ValidationError[] | null (not optional). | New Lifecycle.Create fields on Plan must also work with TypeSpec.Rest.Resource.ResourceCreateModel<Plan> used in routes.tsp. Plan.validationErrors must remain | null, never optional. |
| `errors.tsp` | Defines CommonSubscriptionErrors alias and SubscriptionBadRequestErrorResponse/SubscriptionConflictErrorResponse models with SubscriptionErrorExtensions.validationErrors array. | This file imports ratecards.tsp — avoid circular imports if ratecards.tsp later needs error types. |
| `main.tsp` | Package entry point importing all sub-files in dependency order. features.tsp and subscriptionaddon.tsp are NOT imported here — features.tsp is used via the parent package; subscriptionaddon.tsp via the routes.tsp import chain. | New .tsp files must be imported here (in the correct dependency order) to be included in compilation. Importing features.tsp here would cause duplicate namespace conflicts. |
| `subscriptionaddon.tsp` | Defines SubscriptionAddon with timeline segments, SubscriptionAddonCreate (write model using OmitProperties + @withVisibility), and SubscriptionAddonRateCard. Imported by routes.tsp. | SubscriptionAddonCreate omits the 'addon' embedded object and replaces it with addon: { id: ULID } — the same pattern as PlanAddonCreate. QuantityAt is Lifecycle.Read only. |

## Anti-Patterns

- Defining a polymorphic type without @discriminated(#{ envelope: "none", ... }) — produces a wrapped discriminator that breaks client deserialization.
- Adding a route interface in any file other than routes.tsp — breaks @tag grouping and operationId consistency.
- Duplicating model fields for create/update variants instead of using OmitProperties<DefaultKeyVisibility<Model, Lifecycle.Read>, "..."> with @withVisibility.
- Using a duration field without @encode(DurationKnownEncoding.ISO8601) — the TypeSpec compiler defaults to seconds serialization, breaking ISO 8601 clients.
- Using CommonErrors (parent package) instead of CommonSubscriptionErrors for subscription mutation routes — the subscription-specific error extensions (validationErrors) will be missing from the OpenAPI schema.

## Decisions

- **SubscriptionEditOperation uses discriminatorPropertyName: 'op' instead of 'type'.** — Distinguishes patch commands (add_item, remove_item, add_phase, etc.) from resource type discriminators used in Price/RateCard, preventing naming collisions in generated code and making PATCH semantics explicit.
- **PlanAddonCreate and SubscriptionAddonCreate replace the embedded Addon/addon object with just { id: ULID } on write.** — On write operations only the identifier is needed; embedding the full Addon object (which is Lifecycle.Read) would require clients to supply server-computed fields like status, effectiveFrom, and version.
- **CommonSubscriptionErrors is a local alias in errors.tsp rather than reusing the parent CommonErrors.** — Subscription mutations return SubscriptionBadRequestErrorResponse and SubscriptionConflictErrorResponse with SubscriptionErrorExtensions (validationErrors array) not present in generic error types, requiring a domain-specific error alias.

## Example: Adding a new usage-based price type (e.g. 'stepped') to the product catalog

```
// 1. In prices.tsp — extend PriceType enum and add the model:
enum PriceType { ..., stepped: "stepped" }
model SteppedPrice {
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) type: PriceType.stepped;
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) amount: Money;
}
model SteppedPriceWithCommitments { ...SteppedPrice; ...SpendCommitments; }

// 2. In prices.tsp — add to Price union:
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Price { ..., stepped: SteppedPrice }

// 3. In ratecards.tsp — add to RateCardUsageBasedPrice union:
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union RateCardUsageBasedPrice { ..., stepped: SteppedPriceWithCommitments }
// ...
```

<!-- archie:ai-end -->
