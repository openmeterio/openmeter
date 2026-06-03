# notification

<!-- archie:ai-start -->

> TypeSpec source for the v1 Notifications API — webhook channels, rules (balance threshold, entitlement reset, invoice created/updated), and event/delivery-status tracking. Discriminated unions on NotificationRule and NotificationChannel give type-safe multi-variant webhook rules.

## Patterns

**Discriminated union (envelope:none, discriminator type) for polymorphic types** — NotificationRule, NotificationRuleCreateRequest, NotificationChannel, NotificationChannelCreateRequest, and NotificationEventPayload are all @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }). Adding a rule type requires updating NotificationRule, NotificationRuleCreateRequest, and NotificationEventPayload together. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union NotificationRule { `entitlements.balance.threshold`: NotificationRuleBalanceThreshold, ... }`)
**NotificationRuleCommon<T> / NotificationChannelCommon<T> generic base spread** — Every rule model spreads ...NotificationRuleCommon<NotificationEventType.X> to inherit id, type literal, name, disabled, channels, annotations, metadata. Channel models spread ...NotificationChannelCommon<NotificationChannelType.X>. Never duplicate these fields. (`model NotificationRuleBalanceThreshold { ...NotificationRuleCommon<NotificationEventType.entitlementsBalanceThreshold>; thresholds: Array<NotificationRuleBalanceThresholdValue>; }`)
**CreateRequest via OmitProperties + re-added writable refs** — Create-request models use @withVisibility(Lifecycle.Create, Lifecycle.Update) and OmitProperties<Rule, "channels" | "features">, then re-add channels: Array<ULID> and features?: Array<ULIDOrKey> as ID-only writable forms. (`@withVisibility(Lifecycle.Create, Lifecycle.Update) model NotificationRuleBalanceThresholdCreateRequest { ...OmitProperties<NotificationRuleBalanceThreshold, "channels" | "features">; channels: Array<ULID>; features?: Array<ULIDOrKey>; }`)
**Payload models carry full denormalized context** — Entitlement payloads spread ...NotificationEventEntitlementValuePayloadBase to embed entitlement, feature, subject, value and optional customer inline. Invoice payloads embed the full Invoice. Webhook consumers need no second API call. (`model NotificationEventBalanceThresholdPayloadData { ...NotificationEventEntitlementValuePayloadBase; threshold: NotificationRuleBalanceThresholdValue; }`)
**Backward-compat enum casing with #suppress** — NotificationEventType values are dotted lowercase (entitlements.balance.threshold), NotificationChannelType.webhook maps to "WEBHOOK", and order-by enums use camelCase values — all carry #suppress "@openmeter/api-spec-legacy/casing" for backward compatibility. (`#suppress "@openmeter/api-spec-legacy/casing" "Ignore due to backward compatibility" webhook: "WEBHOOK",`)
**PUT (not PATCH) updates with the create-request body** — updateNotificationRule and updateNotificationChannel use @put and accept the full NotificationRuleCreateRequest / NotificationChannelCreateRequest schema — there is no partial-update body. (`@put @operationId("updateNotificationRule") update(@path ruleId: ULID, @body request: NotificationRuleCreateRequest)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `channel.tsp` | NotificationChannelType enum, NotificationChannelMeta/Common/Webhook models, NotificationChannel union, and NotificationChannelsEndpoints CRUD interface. | webhook value is uppercase "WEBHOOK" with a #suppress casing annotation — preserve for backward compat. signingSecret has a strict @pattern. |
| `rule.tsp` | NotificationRuleMeta/Common, the NotificationRule and NotificationRuleCreateRequest unions, and NotificationRulesEndpoints (CRUD plus a /{ruleId}/test POST). | Updates use PUT with the create-request body, not PATCH. A new rule type must be added to both unions here plus NotificationEventPayload in event.tsp. |
| `entitlements.tsp` | Entitlement rule/payload models: NotificationRuleBalanceThreshold, NotificationRuleEntitlementReset, payload bases, FeatureMeta, and NotificationRuleBalanceThresholdValueType. | NotificationRuleBalanceThresholdValueType has deprecated PERCENT/NUMBER variants — keep them, add new variants alongside (balance_value/usage_percentage/usage_value). |
| `invoice.tsp` | Generic NotificationEventInvoicePayload<T>, the created/updated payload aliases, and NotificationRuleInvoiceCreated/Updated rule models plus their create requests. | Invoice payloads embed the full billing Invoice model — ensure billing imports resolve. Create requests only re-add channels: Array<ULID> (no features). |
| `event.tsp` | NotificationEventType enum, NotificationEvent model, delivery-status/attempt models, NotificationEventPayload union, NotificationEventResendRequest, and NotificationEventsEndpoints (list/get/resend). | All event-type values are dotted lowercase with #suppress for casing — maintain for new types. Delivery-status models live here, not in channel.tsp. |
| `main.tsp` | Import manifest only — imports channel/rule/event/entitlements/invoice. | Add new event-family .tsp files here. |

## Anti-Patterns

- Adding a rule type to NotificationRule without also updating NotificationRuleCreateRequest and NotificationEventPayload.
- Putting delivery-status logic in channel.tsp — those models belong in event.tsp.
- Using PATCH for updates — the API uses PUT with the full create-request body.
- Inlining feature/channel lists instead of spreading FeatureMeta and NotificationChannelMeta.
- Dropping the #suppress casing annotations on existing enum values — breaks backward compatibility.

## Decisions

- **Create requests use OmitProperties to strip embedded objects and re-add ID-only writable fields.** — Prevents clients from supplying server-assigned full channel/feature objects while keeping the read model as the single source of truth.
- **Payload models embed full denormalized context (entitlement, feature, subject, customer, invoice).** — Webhook consumers should act on a notification without a second API call — payloads are self-contained.

## Example: Adding a new notification rule type end-to-end

```
// 1. event.tsp: add to NotificationEventType enum and NotificationEventPayload union
// 2. new .tsp: model NotificationRuleX { ...NotificationRuleCommon<NotificationEventType.x>; }
//    @withVisibility(Lifecycle.Create, Lifecycle.Update) model NotificationRuleXCreateRequest { ...OmitProperties<NotificationRuleX, "channels">; channels: Array<ULID>; }
// 3. rule.tsp: add to NotificationRule and NotificationRuleCreateRequest unions
```

<!-- archie:ai-end -->
