# subscriptions

<!-- archie:ai-start -->

> TypeSpec definitions for the Subscriptions v3 API domain: subscription lifecycle model, CRUD+action operations (create, list, get, cancel, unschedule-cancelation, change), subscription addon model, and cross-domain SubscriptionReference type. All model and enum @friendlyName values are prefixed 'Billing' to avoid SDK type collisions.

## Patterns

**BillingXxx friendlyName prefix on all models and enums** — Every model, enum, and union carries @friendlyName("BillingXxx") — required to avoid SDK type name collisions with identically-named types in other namespaces (billing, productcatalog). Any new type added here must follow this prefix. (`@friendlyName("BillingSubscription") model Subscription { ... }
@friendlyName("BillingSubscriptionStatus") enum SubscriptionStatus { ... }`)
**Action operations as @route-decorated methods on the same interface** — Non-CRUD actions (cancel, unschedule-cancelation, change) are declared as methods on SubscriptionsOperations with an explicit @route("/{subscriptionId}/action-name") decorator, not as nested interfaces or separate interface extensions. (`@post @route("/{subscriptionId}/cancel") @operationId("cancel-subscription")
cancel(@path subscriptionId: Shared.ULID, @body body: SubscriptionCancel): Shared.UpdateResponse<Subscription> | ...`)
**OmitProperties<> to derive create/change shapes from the read model** — Request bodies are derived from the read model by omitting server-assigned fields (billing_anchor, status) with OmitProperties<Subscription, ...>, then adding explicit Create-only fields (customer, plan references) inline in the request model. (`model SubscriptionCreate {
  ...Shared.CreateRequest<OmitProperties<Subscription, "customer_id" | "plan_id" | "billing_anchor">>;
  customer: { id?: Shared.ULID; key?: Shared.ExternalResourceKey; };
}`)
**@oneOf union for flexible timing parameters** — SubscriptionEditTiming is a @oneOf union of an enum (immediate/next_billing_cycle) and a raw DateTime, allowing callers to use convenient shorthand or an explicit timestamp without a separate discriminator field. (`@oneOf @friendlyName("BillingSubscriptionEditTiming")
union SubscriptionEditTiming { Enum: SubscriptionEditTimingEnum, Custom: Shared.DateTime, }`)
**Dual-subscription response for change operation** — The change operation returns SubscriptionChangeResponse with both `current` and `next` Subscription fields — not wrapped in Shared.UpdateResponse<T> — because it has a two-subscription payload. (`@friendlyName("BillingSubscriptionChangeResponse")
model SubscriptionChangeResponse { current: Subscription; next: Subscription; }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscription.tsp` | Core domain model: Subscription (read model), SubscriptionCreate, SubscriptionStatus enum, SubscriptionEditTiming union, SubscriptionEditTimingEnum enum. | Subscription spreads ...OmitProperties<Shared.Resource, "name" | "description"> — name and description are intentionally absent. Do not add them back. |
| `operations.tsp` | SubscriptionsOperations interface with all HTTP operations including action routes. SubscriptionCancel and SubscriptionChange request models are declared inline in this file. | The change operation returns raw SubscriptionChangeResponse (not wrapped in Shared.UpdateResponse) — do not wrap it. The response has its own @Http.statusCode baked in. |
| `reference.tsp` | SubscriptionReference cross-domain reference type for when billing line items need to point to a specific subscription phase+item. | Reference type is deeply nested (subscription → phase → item) using anonymous inline structs, not named types. Do not extract the inner structs into separate named models. |
| `subscriptionaddon.tsp` | SubscriptionAddon model: addon reference, quantity, quantity_at, active_from/active_to lifecycle times. | quantity_at is Read-only (server resolves it). active_from/active_to are Read-only temporal fields. |

## Anti-Patterns

- Omitting @friendlyName("BillingXxx") from a new model or enum — generated SDK types would collide with same-named types in other namespaces
- Wrapping the change operation response in Shared.UpdateResponse<T> — its two-subscription payload is intentional and incompatible with the single-body envelope
- Adding name or description fields to the Subscription model — explicitly omitted via OmitProperties
- Defining a new action endpoint as a separate interface instead of a @route-decorated method on SubscriptionsOperations

## Decisions

- **Billing-prefixed @friendlyName on all types** — Multiple aip/ namespaces (subscriptions, billing, productcatalog) expose similarly-named types; the Billing prefix makes generated Go and JS SDK types unambiguous without requiring Go package-qualified names.
- **@oneOf union for SubscriptionEditTiming instead of separate fields** — Allows callers to use the convenient enum values (immediate, next_billing_cycle) or an explicit timestamp without an extra discriminator field or separate endpoint variants.

## Example: Adding a new action endpoint (e.g. pause) following existing patterns

```
// In operations.tsp, inside interface SubscriptionsOperations:
@post
@operationId("pause-subscription")
@summary("Pause subscription")
@route("/{subscriptionId}/pause")
pause(
  @path subscriptionId: Shared.ULID,
  @body body: SubscriptionPause,
):
  | Shared.UpdateResponse<Subscription>
  | Common.NotFound
  | Common.Conflict
  | Common.ErrorResponses;

// In subscription.tsp:
// ...
```

<!-- archie:ai-end -->
