# notification

<!-- archie:ai-start -->

> TypeSpec definitions for the Notifications API — channels (webhook), rules (balance threshold, entitlement reset, invoice events), and event/delivery-status tracking. Discriminated unions on NotificationRule and NotificationChannel allow type-safe multi-variant webhook rules.

## Patterns

**Discriminated union for polymorphic rule and channel types** — NotificationRule, NotificationRuleCreateRequest, NotificationChannel, and NotificationChannelCreateRequest are all `@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })` unions. Adding a rule type requires updating all four unions. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union NotificationRule { `entitlements.balance.threshold`: NotificationRuleBalanceThreshold, ... }`)
**NotificationRuleCommon<T> generic for rule base fields** — All rule models spread `...NotificationRuleCommon<NotificationEventType.X>` to inherit id, type (literal), name, disabled, channels, and metadata. Never duplicate these fields. (`model NotificationRuleBalanceThreshold { ...NotificationRuleCommon<NotificationEventType.entitlementsBalanceThreshold>; thresholds: Array<NotificationRuleBalanceThresholdValue>; }`)
**CreateRequest models use OmitProperties to strip read-only references** — Rule create request models use `OmitProperties<NotificationRuleXxx, "channels" | "features">` then re-add `channels: Array<ULID>` (IDs only, not full objects) and `features?: Array<ULIDOrKey>` for the writable form. (`@withVisibility(Lifecycle.Create, Lifecycle.Update) model NotificationRuleBalanceThresholdCreateRequest { ...OmitProperties<NotificationRuleBalanceThreshold, "channels" | "features">; channels: Array<ULID>; features?: Array<ULIDOrKey>; }`)
**Payload models carry the full denormalized context** — NotificationEventBalanceThresholdPayload and NotificationEventResetPayload spread `...NotificationEventEntitlementValuePayloadBase` to embed entitlement, feature, subject, value, and customer inline — payloads are self-contained for webhook consumers. (`model NotificationEventBalanceThresholdPayloadData { ...NotificationEventEntitlementValuePayloadBase; threshold: NotificationRuleBalanceThresholdValue; }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `channel.tsp` | NotificationChannel/NotificationChannelWebhook models, NotificationChannelsEndpoints interface, NotificationChannelType enum. Includes CRUD for channels. | NotificationChannelType.webhook uses uppercase value `"WEBHOOK"` with a legacy `#suppress` annotation — preserve this for backward compat. |
| `rule.tsp` | NotificationRulesEndpoints interface, NotificationRule union, all rule variant models, and NotificationRuleCreateRequest union. Also has a `/{ruleId}/test` POST endpoint. | Rule update uses PUT with NotificationRuleCreateRequest (same schema as create) — not PATCH. |
| `entitlements.tsp` | Entitlement-specific rule and event payload models: NotificationRuleBalanceThreshold, NotificationRuleEntitlementReset, and their payload models. Also defines FeatureMeta. | NotificationRuleBalanceThresholdValueType has deprecated PERCENT/NUMBER variants — keep them for backward compat, add new variants alongside. |
| `invoice.tsp` | Invoice event payload models (NotificationEventInvoicePayload) and invoice notification rules (NotificationRuleInvoiceCreated, NotificationRuleInvoiceUpdated). | Invoice notification rules reference billing Invoice models — ensure billing/ imports are present. |
| `event.tsp` | NotificationEventType enum, NotificationEventDeliveryStatus/State models, and delivery attempt models. | All NotificationEventType values use dotted lowercase (entitlements.balance.threshold) with `#suppress` for casing — maintain this pattern for new event types. |
| `main.tsp` | Import manifest only. No definitions. | Add new .tsp files here when adding notification event families. |

## Anti-Patterns

- Adding a new rule type only to NotificationRule union without also updating NotificationRuleCreateRequest.
- Putting delivery status logic into channel.tsp — delivery status models belong in event.tsp.
- Using PATCH for rule updates — the API uses PUT with the full create-request body.
- Hardcoding feature/channel lists inline in rule models instead of using FeatureMeta and NotificationChannelMeta spread models.

## Decisions

- **OmitProperties + re-add writable-form fields for create requests** — Prevents accidental write of server-assigned fields (channel full objects, feature full objects) while keeping the shared model as the source of truth for read responses.
- **Payload models embed full denormalized context (entitlement, feature, subject, customer)** — Webhook consumers should not need a second API call to act on a notification — payloads are designed to be self-contained.

## Example: Adding a new notification rule type (e.g. subscription.cancelled)

```
// 1. Add to NotificationEventType enum in event.tsp
enum NotificationEventType { ..., `subscription.cancelled`: "subscription.cancelled" }

// 2. Create a new file subscription.tsp with rule + payload models
model NotificationRuleSubscriptionCancelled {
  ...NotificationRuleCommon<NotificationEventType.`subscription.cancelled`>;
}
model NotificationRuleSubscriptionCancelledCreateRequest {
  ...OmitProperties<NotificationRuleSubscriptionCancelled, "channels">;
  channels: Array<ULID>;
}

// 3. Add variants to unions in rule.tsp
union NotificationRule { ..., `subscription.cancelled`: NotificationRuleSubscriptionCancelled }
union NotificationRuleCreateRequest { ..., `subscription.cancelled`: NotificationRuleSubscriptionCancelledCreateRequest }
// ...
```

<!-- archie:ai-end -->
