# productcatalog

<!-- archie:ai-start -->

> TypeSpec source for v1 product catalog, subscription, and add-on API contracts. Defines all models, enums, unions, and route interfaces that compile into api/openapi.yaml and drive Go server stubs — never edit generated outputs, only these .tsp files.

## Patterns

**Discriminated union with envelope:none** — Polymorphic types (Price, RateCard, RateCardEntitlement, SubscriptionEditOperation, SubscriptionCreate) use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" | "op" }) so the discriminator field sits at the top level with no wrapper. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union Price { flat: FlatPrice, unit: UnitPrice, tiered: TieredPrice }`)
**Visibility lifecycle annotation on every field** — Every model field carries @visibility(Lifecycle.Read), @visibility(Lifecycle.Read, Lifecycle.Create), or similar. Read-only computed fields (status, effectiveFrom, effectiveTo, version) are Lifecycle.Read only. Omit visibility only on structurally shared base models like RateCardBase<T>. (`@visibility(Lifecycle.Read) status: PlanStatus;`)
**Spread-based model composition over inheritance** — Models are composed by spreading shared base models (e.g. ...UniqueResource, ...ResourceTimestamps, ...RateCardBase<T>, ...SpendCommitments) rather than using TypeSpec extends, giving explicit field ownership and enabling OmitProperties<> overrides in create/update variants. (`model RateCardFlatFee { ...RateCardBase<RateCardType.flatFee>; billingCadence: duration | null; price: FlatPriceWithPaymentTerm | null; }`)
**Create/update variants via OmitProperties + withVisibility** — Write models (PlanAddonCreate, SubscriptionAddonCreate) are produced with @withVisibility(Lifecycle.Create) and OmitProperties<DefaultKeyVisibility<Model, Lifecycle.Read>, "field">, not by defining duplicate models from scratch. (`@withVisibility(Lifecycle.Create) model PlanAddonCreate { ...OmitProperties<DefaultKeyVisibility<PlanAddon, Lifecycle.Read>, "addon">; addonId: ULID; }`)
**Route interfaces tagged with @tag("Product Catalog") or @tag("Subscriptions")** — All route interfaces in routes.tsp use @route, @tag, @friendlyName, and per-operation @operationId + @summary. Operations return typed union responses (Model | NotFoundError | CommonErrors), never raw status codes alone. (`@route("/api/v1/plans") @tag("Product Catalog") interface PlansEndpoints { @get @operationId("listPlans") list(...): PaginatedResponse<Plan> | CommonErrors; }`)
**Shared error aliases from errors.tsp** — Subscription routes use CommonSubscriptionErrors alias (defined in errors.tsp) that extends SubscriptionBadRequestErrorResponse and SubscriptionConflictErrorResponse with SubscriptionErrorExtensions. Feature/plan routes use CommonErrors from the parent package. (`alias CommonSubscriptionErrors = SubscriptionBadRequestErrorResponse | SubscriptionConflictErrorResponse | UnauthorizedError | ...;`)
**ISO8601 duration encoding for all duration fields** — Every duration field uses @encode(DurationKnownEncoding.ISO8601) and an @example(duration.fromISO("P1M")) annotation. Null-able billing cadence (one-time fees) is typed as duration | null. (`@encode(DurationKnownEncoding.ISO8601) @example(duration.fromISO("P1M")) billingCadence: duration | null;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `routes.tsp` | Sole file that declares HTTP interfaces (FeaturesEndpoints, PlansEndpoints, PlanAddonsEndpoints, AddonsEndpoints, SubscriptionsEndpoints). Must import rest.tsp and errors.tsp; uses TypeSpec.Http and TypeSpec.OpenAPI. | Adding a new route interface without @operationId or @friendlyName breaks Go codegen naming. Missing 'using TypeSpec.Http;' causes @query/@route to fail. |
| `ratecards.tsp` | Defines RateCardBase<T>, RateCard union, RateCardFlatFee, RateCardUsageBased, RateCardUsageBasedPrice union, and entitlement templates. Central dependency imported by plan.tsp, addon.tsp, subscription.tsp. | Adding a new RateCard variant requires updating both the RateCard union and RateCardUsageBasedPrice union. Entitlement templates omit featureKey/featureId/usagePeriod via OmitProperties. |
| `prices.tsp` | Defines Price union and all price models (FlatPrice, UnitPrice, TieredPrice, DynamicPrice, PackagePrice) plus *WithCommitments variants. SpendCommitments is spread into usage-based price variants. | New price type must be added to both the Price union and RateCardUsageBasedPrice union in ratecards.tsp. PriceType enum must also be extended. |
| `subscription.tsp` | Defines Subscription, SubscriptionExpanded, SubscriptionPhase, SubscriptionItem, SubscriptionCreate/Change unions, SubscriptionEdit, SubscriptionEditOperation union, and SubscriptionTiming. Imports entitlements/main.tsp. | SubscriptionCreate and SubscriptionChange are @oneOf unions — new variants need explicit @summary. SubscriptionEditOperation uses discriminatorPropertyName: "op" not "type". |
| `plan.tsp` | Defines Plan, PlanPhase, PlanStatus, PlanOrderBy, PlanReferenceInput, SettlementMode. Plan.phases has @minItems(1). Imports prorating.tsp. | Plan.validationErrors is typed ValidationError[] | null (not optional). New plan-level fields with Lifecycle.Create visibility must also appear in the ResourceCreateModel used by the create route. |
| `addon.tsp` | Defines Addon, AddonStatus (draft/active/archived), AddonInstanceType (single/multiple), AddonOrderBy. Mirrors Plan structure but without phases. | AddonStatus lifecycle parallels PlanStatus but lacks 'scheduled'. Status is @visibility(Lifecycle.Read) only — never writable. |
| `main.tsp` | Package entry point — imports all sub-files in dependency order. New .tsp files must be imported here to be included in compilation. | features.tsp and subscriptionaddon.tsp are NOT imported here — features.tsp is used via the parent package, subscriptionaddon.tsp via routes.tsp import chain. |

## Anti-Patterns

- Defining a new polymorphic type without @discriminated(#{ envelope: "none", ... }) — omitting this produces a wrapped discriminator that breaks client deserialization.
- Adding a route in any file other than routes.tsp — interfaces must be co-located there for @tag grouping and operationId consistency.
- Duplicating model fields for create/update variants instead of using OmitProperties<DefaultKeyVisibility<Model, Lifecycle.Read>, "..."> with @withVisibility.
- Using a duration field without @encode(DurationKnownEncoding.ISO8601) — the TypeSpec compiler defaults to seconds serialization, breaking ISO 8601 clients.
- Hand-editing the generated api/openapi.yaml or api/api.gen.go — always regenerate via 'make gen-api' then 'make generate' after .tsp changes.

## Decisions

- **Subscription edit operations use a separate 'op' discriminator (not 'type') in SubscriptionEditOperation union.** — Distinguishes patch commands (add_item, remove_item, add_phase, etc.) from resource type discriminators used in Price/RateCard, preventing naming collisions in generated code and making PATCH semantics explicit.
- **PlanAddonCreate and SubscriptionAddonCreate flatten the addonId into the body instead of embedding the full Addon.** — On write operations only the add-on identifier is needed; embedding the full Addon object (which is read-only) would require clients to supply server-computed fields like status and effectiveFrom.
- **CommonSubscriptionErrors is a local alias in errors.tsp rather than reusing CommonErrors from the parent.** — Subscription mutations return SubscriptionBadRequestErrorResponse and SubscriptionConflictErrorResponse with SubscriptionErrorExtensions (validationErrors array) not present in the generic error types, requiring a domain-specific error alias.

## Example: Adding a new usage-based price type (e.g. 'stepped') to the product catalog

```
// 1. In prices.tsp — add the model and extend PriceType:
enum PriceType { ..., stepped: "stepped" }
model SteppedPrice {
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  type: PriceType.stepped;
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  amount: Money;
}
model SteppedPriceWithCommitments { ...SteppedPrice; ...SpendCommitments; }

// 2. In prices.tsp — add to Price union:
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union Price { ..., stepped: SteppedPrice }

// 3. In ratecards.tsp — add to RateCardUsageBasedPrice union:
// ...
```

<!-- archie:ai-end -->
