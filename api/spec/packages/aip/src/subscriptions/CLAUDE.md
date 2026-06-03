# subscriptions

<!-- archie:ai-start -->

> TypeSpec definitions for the Subscriptions v3 API domain: subscription lifecycle model, CRUD+action operations (create, list, get, cancel, unschedule-cancelation, change), subscription addon model, and cross-domain SubscriptionReference. All @friendlyName values are 'Billing'-prefixed to avoid SDK type collisions.

## Patterns

**BillingXxx friendlyName prefix on all types** — Every model, enum, and union carries @friendlyName("BillingXxx") to avoid SDK type-name collisions with same-named types in billing/productcatalog namespaces. New types must follow the prefix. (`@friendlyName("BillingSubscription") model Subscription { ... }
@friendlyName("BillingSubscriptionStatus") enum SubscriptionStatus { ... }`)
**Action operations as @route-decorated methods on the same interface** — Non-CRUD actions (cancel, unschedule-cancelation, change) are methods on SubscriptionsOperations with an explicit @route("/{subscriptionId}/action-name"), not nested interfaces or extensions. (`@post @route("/{subscriptionId}/cancel") @operationId("cancel-subscription") cancel(@path subscriptionId: Shared.ULID, @body body: SubscriptionCancel): Shared.UpdateResponse<Subscription> | ...`)
**OmitProperties<> to derive create/change shapes from the read model** — Request bodies omit server-assigned fields (billing_anchor, status) from the read model via OmitProperties<Subscription, ...>, then add explicit Create-only fields (customer, plan references) inline. (`model SubscriptionCreate { ...Shared.CreateRequest<OmitProperties<Subscription, "customer_id" | "plan_id" | "billing_anchor">>; customer: { id?: Shared.ULID; key?: Shared.ExternalResourceKey; }; }`)
**@oneOf union for flexible timing parameters** — SubscriptionEditTiming is a @oneOf union of an enum (immediate/next_billing_cycle) and a raw DateTime, letting callers pass shorthand or an explicit timestamp without a discriminator field. (`@oneOf @friendlyName("BillingSubscriptionEditTiming") union SubscriptionEditTiming { Enum: SubscriptionEditTimingEnum, Custom: Shared.DateTime }`)
**Dual-subscription response for change operation** — The change operation returns SubscriptionChangeResponse with both current and next Subscription fields — not wrapped in Shared.UpdateResponse<T> — because of its two-subscription payload. (`@friendlyName("BillingSubscriptionChangeResponse") model SubscriptionChangeResponse { current: Subscription; next: Subscription; }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscription.tsp` | Core domain model: Subscription (read), SubscriptionCreate, SubscriptionStatus enum, SubscriptionEditTiming union, SubscriptionEditTimingEnum enum. | Subscription spreads ...OmitProperties<Shared.Resource, "name" | "description"> — name and description are intentionally absent. Do not add them back. |
| `operations.tsp` | SubscriptionsOperations interface with all HTTP operations including action routes; SubscriptionCancel and SubscriptionChange request models declared inline. | The change operation returns raw SubscriptionChangeResponse (not wrapped in Shared.UpdateResponse) with its own @Http.statusCode — do not wrap it. |
| `reference.tsp` | SubscriptionReference cross-domain type for billing line items pointing to a specific subscription phase+item. | Deeply nested (subscription -> phase -> item) using anonymous inline structs; do not extract inner structs into separate named models. |
| `subscriptionaddon.tsp` | SubscriptionAddon model: addon reference, quantity, quantity_at, active_from/active_to lifecycle times. | quantity_at is Read-only (server-resolved); active_from/active_to are Read-only temporal fields. |

## Anti-Patterns

- Omitting @friendlyName("BillingXxx") from a new model/enum — generated SDK types would collide with same-named types in other namespaces
- Wrapping the change operation response in Shared.UpdateResponse<T> — its two-subscription payload is intentional
- Adding name or description fields to the Subscription model — explicitly omitted via OmitProperties
- Defining a new action endpoint as a separate interface instead of a @route-decorated method on SubscriptionsOperations

## Decisions

- **Billing-prefixed @friendlyName on all types** — Multiple aip/ namespaces (subscriptions, billing, productcatalog) expose similarly-named types; the prefix makes generated Go/JS SDK types unambiguous without package-qualified names.
- **@oneOf union for SubscriptionEditTiming instead of separate fields** — Lets callers use convenient enum values (immediate, next_billing_cycle) or an explicit timestamp without an extra discriminator field or endpoint variants.

## Example: Adding a new action endpoint (e.g. pause)

```
// In operations.tsp, inside interface SubscriptionsOperations:
@post @operationId("pause-subscription") @summary("Pause subscription")
@route("/{subscriptionId}/pause")
pause(@path subscriptionId: Shared.ULID, @body body: SubscriptionPause):
  | Shared.UpdateResponse<Subscription>
  | Common.NotFound | Common.Conflict | Common.ErrorResponses;
```

<!-- archie:ai-end -->
