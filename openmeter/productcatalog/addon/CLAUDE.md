# addon

<!-- archie:ai-start -->

> Domain package for add-on lifecycle management: defines the Addon aggregate (with RateCards), domain types (Plan, RateCard, RateCards), typed errors, domain events, and the Service + Repository interfaces. Primary constraint: domain types must always carry both productcatalog.* base types and managed fields (NamespacedID, ManagedModel).

## Patterns

**Addon aggregate embeds productcatalog.AddonMeta + RateCards** — Addon struct embeds models.NamespacedID, models.ManagedModel, productcatalog.AddonMeta, and addon.RateCards. RateCard wraps productcatalog.RateCard with RateCardManagedFields (AddonID field). (`type Addon struct { models.NamespacedID; models.ManagedModel; productcatalog.AddonMeta; RateCards RateCards }`)
**Typed domain errors wrapping models.NewGenericNotFoundError** — NotFoundError wraps models.NewGenericNotFoundError via Unwrap chain. Always use NewNotFoundError(NotFoundErrorParams{...}) and IsNotFound() — never return raw Ent not-found errors. (`return nil, addon.NewNotFoundError(addon.NotFoundErrorParams{Namespace: ns, ID: id})`)
**Domain events implement metadata.EventName / EventMetadata / Validate** — Each lifecycle event (Create, Update, Delete, Publish, Archive) is a struct with Addon *Addon + UserID *string, implementing EventName(), EventMetadata(), and Validate(). EventName uses metadata.GetEventName with subsystem + name + version. (`func (e AddonCreateEvent) EventName() string { return metadata.GetEventName(metadata.EventType{Subsystem: AddonEventSubsystem, Name: AddonCreateEventName, Version: "v1"}) }`)
**Input types implement models.Validator; IgnoreNonCriticalIssues for partial validation** — CreateAddonInput and UpdateAddonInput embed inputOptions{IgnoreNonCriticalIssues bool}. Validate() calls models.AsValidationIssues and filters by ErrorSeverityCritical when set. (`issues, err := models.AsValidationIssues(errors.Join(errs...)); if i.IgnoreNonCriticalIssues { issues = issues.WithSeverityOrHigher(models.ErrorSeverityCritical) }`)
**RateCards uses AsProductCatalogRateCards() for cross-type conversion** — addon.RateCards.AsProductCatalogRateCards() returns productcatalog.RateCards; used in assert helpers and Addon.AsProductCatalogAddon(). Never directly cast the slice. (`func (c RateCards) AsProductCatalogRateCards() productcatalog.RateCards { ... }`)
**ValidatorFunc[Addon] pattern for status/deletion guards** — validators.go exposes IsAddonDeleted and HasAddonStatus as models.ValidatorFunc[Addon] closures, composed via addon.Validate() via ValidateWith. (`func HasAddonStatus(statuses ...productcatalog.AddonStatus) models.ValidatorFunc[Addon] { return func(a Addon) error { ... } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service interface (ListAddons, CreateAddon, DeleteAddon, GetAddon, UpdateAddon, PublishAddon, ArchiveAddon, NextAddon), all input types, and their Validate() methods. | PublishAddonInput.Validate() enforces EffectiveFrom not in the past with a 30s jitter; clock.Now() is used — do not substitute time.Now(). |
| `addon.go` | Addon aggregate type with Validate(), ValidateWith(), and AsProductCatalogAddon(). | Plans field is *[]Plan — optional; only populated on expand. |
| `ratecard.go` | RateCard wraps productcatalog.RateCard with RateCardManagedFields; custom MarshalJSON/UnmarshalJSON dispatches on productcatalog.RateCardSerde.Type. | UnmarshalJSON must switch on FlatFeeRateCardType vs UsageBasedRateCardType — any new type needs a case here. |
| `errors.go` | NotFoundError type wrapping models.NewGenericNotFoundError; IsNotFound() for errors.As detection. | Never return raw entdb.IsNotFound — always wrap via NewNotFoundError. |
| `event.go` | All five lifecycle event structs; UserID sourced from session.GetSessionUserID(ctx). | AddonDeleteEvent.Validate() asserts Addon.DeletedAt != nil; ensure soft-delete is applied before publishing. |
| `repository.go` | Repository interface extends entutils.TxCreator — required for TransactingRepo in adapter. | TxCreator embedding is mandatory; adapters without it cannot participate in ctx-bound transactions. |
| `validators.go` | IsAddonDeleted and HasAddonStatus validator funcs for use in service-layer guards. | Use these funcs via Addon.ValidateWith() instead of inline status checks. |

## Anti-Patterns

- Returning raw entdb.IsNotFound errors — always wrap in addon.NewNotFoundError.
- Using context.Background() in service or adapter code — always propagate ctx.
- Calling RateCards adapter operations without eager-loading related rate cards (WithRatecards).
- Publishing domain events outside a transaction closure — event failure should roll back the DB write.
- Creating multiple non-deleted draft versions for the same key without checking HasDraft().

## Decisions

- **RateCard is a separate managed type wrapping productcatalog.RateCard with AddonID** — Each rate card needs its own DB identity (NamespacedID) and foreign-key reference (AddonID) separate from the base productcatalog.RateCard so Ent can persist and eager-load them independently.
- **inputOptions.IgnoreNonCriticalIssues enables partial validation for draft addons** — Draft addons can be created with non-critical issues (e.g. missing features not yet registered), so callers can opt into a relaxed validation that only fails on ErrorSeverityCritical.
- **Domain events carry the full Addon pointer + UserID from session context** — Downstream Watermill consumers need the full entity for side-effects; UserID supports audit trails without requiring a separate lookup.

## Example: Creating and publishing a domain event after a successful adapter write

```
import (
    "github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
    "github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

// inside service mutation:
created, err := s.adapter.CreateAddon(ctx, params)
if err != nil { return nil, err }
event := addon.NewAddonCreateEvent(ctx, created)
if err := s.publisher.Publish(ctx, eventbus.SystemTopic, event); err != nil {
    return nil, fmt.Errorf("publish addon created event: %w", err)
}
return created, nil
```

<!-- archie:ai-end -->
