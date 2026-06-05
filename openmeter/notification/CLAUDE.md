# notification

<!-- archie:ai-start -->

> Root of the notification domain: declares the notification.Service and notification.Repository contracts, the channel/rule/event/eventpayload/deliverystatus domain models, the EventType catalog, and the EventHandler dispatch contract. Children implement persistence (adapter), Kafka consumption (consumer), HTTP transport (httpdriver), async delivery (eventhandler), and Svix webhooks (webhook).

## Patterns

**Every input is a models.Validator + CustomValidator pair** — Each *Input struct declares both `_ models.Validator` and `_ models.CustomValidator[T]` assertions, implements ValidateWith(...) delegating to models.Validate(i, validators...), and a Validate() that collects into `var errs []error` then returns models.NewNillableGenericValidationError(errors.Join(errs...)). (`func (i CreateChannelInput) Validate() error { var errs []error; if i.Namespace == "" { errs = append(errs, errors.New("namespace is required")) }; ...; return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Discriminated union types keyed by EventType/ChannelType** — ChannelConfig, RuleConfig, and EventPayload are union structs embedding a *Meta with a Type field; Validate() switches on Type and dispatches to the matching sub-config (BalanceThreshold/EntitlementReset/Invoice). (`switch c.Type { case EventTypeBalanceThreshold: if c.BalanceThreshold == nil { return models.NewGenericValidationError(...) }; return c.BalanceThreshold.Validate() ... }`)
**EventType registered in the central eventTypes slice** — New event types must be appended to the `eventTypes` var in event.go; EventType.Validate()/Values() derive from it via lo.Contains/lo.Map, and every union Validate() switch must gain a matching case. (`var eventTypes = []EventType{EventTypeBalanceThreshold, EventTypeEntitlementReset, EventTypeInvoiceCreated, EventTypeInvoiceUpdated}`)
**Repository composes entutils.TxCreator + sub-repos** — notification.Repository embeds entutils.TxCreator and the Channel/Rule/Event sub-repository interfaces; Service composes FeatureService + Channel/Rule/EventService. Keep CRUD method sets symmetric across Repository and Service. (`type Repository interface { entutils.TxCreator; ChannelRepository; RuleRepository; EventRepository }`)
**Typed domain errors, not raw ent errors** — Surface NotFoundError{NamespacedID} and UpdateAfterDeleteError from errors.go; these are mapped to HTTP codes by httpdriver. Returning untyped errors degrades to 500. (`func (e NotFoundError) Error() string { return fmt.Sprintf("resource with id %s not found in %s namespace", e.ID, e.Namespace) }`)
**Annotations carry cross-cutting routing/dedupe metadata** — annotations.go defines string keys (AnnotationEventFeatureKey, AnnotationBalanceEventDedupeHash, AnnotationEventInvoiceID, etc.); consumer/adapter store these on Event.Annotations and query them via JSONB. Add new keys here, never inline string literals. (`AnnotationBalanceEventDedupeHash = "event.balance.dedupe.hash"`)
**Channel-set diffing via NewChannelIDsDifference** — Rule channel-assignment changes are computed with NewChannelIDsDifference(new, old) (built on lo.Difference) exposing Additions()/Removals(); the service uses this to mirror changes to Svix. (`diff := NewChannelIDsDifference(newChannels, oldChannels); diff.Additions(); diff.Removals()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Declares Service/ChannelService/RuleService/EventService/FeatureService interfaces and OrderBy constants. Source of truth for the service contract. | Adding a method here means updating the service/ implementation and (usually) httpdriver. |
| `repository.go` | Declares Repository = TxCreator + Channel/Rule/Event sub-repos. | Method sets must mirror the corresponding Service methods; adapter must implement all. |
| `event.go` | Event domain model, the eventTypes registry, ListEventsInput with dedup/delivery-status filters. | Forgetting to register a new EventType in `eventTypes` makes Validate() reject it everywhere. |
| `channel.go / rule.go` | Channel/Rule models + ChannelConfig/RuleConfig unions + CRUD inputs. MaxChannelsPerRule=5 lives in rule.go. | Adding a config variant requires a new union field plus a Validate() switch case in both the config and its payload. |
| `eventpayload.go` | EventPayload union + EventPayloadVersionCurrent versioning + AsRawPayload marshaling. | Bump/handle EventPayloadVersionCurrent when changing payload shape; every payload type needs a Validate() switch case. |
| `deliverystatus.go` | EventDeliveryStatus state machine enum (PENDING/SENDING/RESENDING/SUCCESS/FAILED) + attempt records + Update/List/Get inputs. | New states must be added to Validate()/Values() and handled by eventhandler's reconcile switch. |
| `entitlements.go / invoice.go` | Entitlement (balance threshold, reset) and invoice event types, payloads, rule configs, plus ValidateRuleConfigWithFeatures (resolves features via FeatureService). | Threshold value bounds differ per type (BalanceValue allows 0, others must be >0). |
| `errors.go` | NotFoundError, UpdateAfterDeleteError domain errors consumed by httpdriver's error encoder. | Unregistered new error types fall through to HTTP 500. |

## Anti-Patterns

- Returning on the first invalid field instead of collecting into errs and joining with errors.Join + NewNillableGenericValidationError.
- Adding an EventType, ChannelType, or config variant without updating its registry/Values() and every discriminated-union Validate() switch.
- Inlining annotation key string literals instead of adding a constant to annotations.go.
- Returning raw ent/db errors instead of notification.NotFoundError / UpdateAfterDeleteError (breaks HTTP status mapping).
- Letting Service and Repository method sets drift apart, or skipping the Validator/CustomValidator assertion pair on a new *Input type.

## Decisions

- **Channel/Rule configs and payloads are discriminated unions keyed by Type rather than separate interfaces.** — A single struct serializes cleanly to JSON and to the DB while a Type-switch keeps validation and mapping centralized.
- **EventPayload carries an explicit integer version (EventPayloadVersionCurrent).** — Stored events outlive code; versioning lets the delivery/reconcile path evolve payload shape without breaking historical events.
- **Annotation-driven dedup/routing keys live in the domain root (annotations.go).** — Consumer dedup logic and adapter JSONB queries share the same key constants, avoiding drift between producer and storage layers.

## Example: Discriminated-union Validate with collected errors (the dominant pattern across this package)

```
func (c RuleConfig) Validate() error {
    switch c.Type {
    case EventTypeBalanceThreshold:
        if c.BalanceThreshold == nil {
            return models.NewGenericValidationError(errors.New("missing balance threshold rule config"))
        }
        return c.BalanceThreshold.Validate()
    case EventTypeInvoiceCreated, EventTypeInvoiceUpdated:
        if c.Invoice == nil {
            return models.NewGenericValidationError(errors.New("missing invoice rule config"))
        }
        return c.Invoice.Validate()
    default:
        return models.NewGenericValidationError(fmt.Errorf("unknown rule type: %s", c.Type))
    }
// ...
```

<!-- archie:ai-end -->
