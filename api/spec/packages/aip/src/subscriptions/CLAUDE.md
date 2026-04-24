# subscriptions

<!-- archie:ai-start -->

> TypeSpec definitions for the Subscriptions v3 API domain: subscription lifecycle model, CRUD+action operations (create, list, get, cancel, unschedule-cancelation, change), and cross-domain reference type. All @friendlyName values are prefixed `Billing` to distinguish them in the generated Go/JS types.

## Patterns

**BillingXxx friendlyName prefix on all models and enums** — Every model, enum, and union in this namespace carries @friendlyName("BillingXxx") — the generated SDK type names are prefixed Billing to avoid collisions with identically-named types in other namespaces. (`@friendlyName("BillingSubscription") model Subscription { ... }`)
**Action operations use @route suffix + @post, not nested interfaces** — Non-CRUD actions (cancel, unschedule-cancelation, change) are declared as methods on the same interface with an explicit @route("/{subscriptionId}/action-name") decorator rather than nested interface hierarchies. (`@post @route("/{subscriptionId}/cancel") cancel(@path subscriptionId: Shared.ULID, @body body: SubscriptionCancel): ...`)
**OmitProperties<> to derive create/change request shapes from read model** — Request bodies are derived from the read model by omitting server-assigned fields (id, status, billing_anchor) with OmitProperties<>, then adding explicit Create/Update-only fields inline. (`model SubscriptionCreate { ...Shared.CreateRequest<OmitProperties<Subscription, "customer_id" | "plan_id" | "billing_anchor">>; customer: { id?: ULID; key?: ExternalResourceKey; }; }`)
**Union type for timing — enum or custom DateTime** — SubscriptionEditTiming is declared as a @oneOf union of the enum (immediate/next_billing_cycle) and a raw DateTime to allow both shorthand and explicit scheduling. (`@oneOf union SubscriptionEditTiming { Enum: SubscriptionEditTimingEnum, Custom: Shared.DateTime, }`)
**Dual-response for change operation** — The change operation returns a SubscriptionChangeResponse with both `current` and `next` Subscription fields — not just the resulting resource. (`model SubscriptionChangeResponse { current: Subscription; next: Subscription; }`)
**Namespace-scoped index.tsp barrel** — index.tsp imports subscription.tsp, then operations.tsp, then reference.tsp in dependency order. operations.tsp must come after subscription.tsp because it references its types. (`import "./subscription.tsp"; import "./operations.tsp"; import "./reference.tsp";`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscription.tsp` | Core domain model: Subscription read model, SubscriptionCreate, SubscriptionStatus enum, SubscriptionEditTiming union, SubscriptionEditTimingEnum enum. | Subscription spreads ...OmitProperties<Shared.Resource, "name" | "description"> — name/description are intentionally absent. Do not add them back. |
| `operations.tsp` | SubscriptionsOperations interface with all HTTP operations including action routes. Declares SubscriptionCancel and SubscriptionChange request models inline. | The change operation returns raw SubscriptionChangeResponse (not wrapped in Shared.UpdateResponse) because it has a two-subscription payload — do not wrap it. |
| `reference.tsp` | SubscriptionReference cross-domain reference type for use when other domains (billing lines) need to point to a subscription item. | Reference type is deeply nested (subscription → phase → item) — all inner structs are anonymous inline models, not named types. |

## Anti-Patterns

- Omitting @friendlyName("BillingXxx") from a new model/enum — generated SDK types would collide with same-named types in other namespaces
- Wrapping the change operation response in Shared.UpdateResponse<T> — its two-subscription payload is intentional and incompatible with the single-body envelope
- Adding name or description fields to the Subscription model — they are explicitly omitted via OmitProperties and not part of this domain resource
- Defining a new action endpoint as a separate interface instead of a @route-decorated method on SubscriptionsOperations

## Decisions

- **Billing-prefixed friendlyName on all types** — Multiple aip/ namespaces (subscriptions, billing, plans) expose similarly-named types; the Billing prefix makes generated Go and JS SDK types unambiguous without requiring Go package-qualified names.
- **Union type for SubscriptionEditTiming instead of separate fields** — Allows callers to use the convenient enum values (immediate, next_billing_cycle) or an explicit timestamp without an extra discriminator field or separate endpoint variants.

## Example: Adding a new action endpoint (e.g. pause) to SubscriptionsOperations following existing patterns

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
