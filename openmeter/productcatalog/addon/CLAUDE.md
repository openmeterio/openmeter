# addon

<!-- archie:ai-start -->

> Domain package for add-on lifecycle management: defines the Addon aggregate (with RateCards), typed errors, domain events, and the Service + Repository interfaces. Primary constraint: domain types always carry both productcatalog.* base types and managed identity fields (NamespacedID, ManagedModel).

## Patterns

**Addon aggregate embedding pattern** — Addon embeds models.NamespacedID, models.ManagedModel, productcatalog.AddonMeta, and RateCards. RateCard wraps productcatalog.RateCard with RateCardManagedFields (AddonID field). Never use struct literals missing these fields. (`type Addon struct { models.NamespacedID; models.ManagedModel; productcatalog.AddonMeta; RateCards RateCards }`)
**Typed NotFoundError wrapping models.NewGenericNotFoundError** — Always return addon.NewNotFoundError(NotFoundErrorParams{...}) from adapters. Never surface raw entdb.IsNotFound. Use IsNotFound() for errors.As detection at service boundaries. (`return nil, addon.NewNotFoundError(addon.NotFoundErrorParams{Namespace: ns, ID: id})`)
**Domain events implement EventName / EventMetadata / Validate** — Each lifecycle event (Create, Update, Delete, Publish, Archive) is a struct with Addon *Addon + UserID *string sourced from session.GetSessionUserID(ctx). EventName uses metadata.GetEventName with subsystem='addon', name, version='v1'. (`func (e AddonCreateEvent) EventName() string { return metadata.GetEventName(metadata.EventType{Subsystem: AddonEventSubsystem, Name: AddonCreateEventName, Version: "v1"}) }`)
**inputOptions.IgnoreNonCriticalIssues for draft validation** — CreateAddonInput and UpdateAddonInput embed inputOptions{IgnoreNonCriticalIssues bool}. Validate() calls models.AsValidationIssues and filters with WithSeverityOrHigher(ErrorSeverityCritical) when set. Allows draft addons with non-critical issues. (`issues, err := models.AsValidationIssues(errors.Join(errs...)); if i.IgnoreNonCriticalIssues { issues = issues.WithSeverityOrHigher(models.ErrorSeverityCritical) }`)
**RateCards.AsProductCatalogRateCards() for cross-type conversion** — Never directly cast addon.RateCards to productcatalog.RateCards. Always call RateCards.AsProductCatalogRateCards() which iterates and extracts the inner productcatalog.RateCard from each RateCard wrapper. (`func (c RateCards) AsProductCatalogRateCards() productcatalog.RateCards { var rcs productcatalog.RateCards; for _, rc := range c { rcs = append(rcs, rc.RateCard) }; return rcs }`)
**ValidatorFunc[Addon] for status/deletion guards** — validators.go exposes IsAddonDeleted and HasAddonStatus as models.ValidatorFunc[Addon] closures. Compose them via Addon.ValidateWith() instead of inline status checks in service methods. (`if err := addon.ValidateWith(addon.HasAddonStatus(productcatalog.AddonStatusDraft)); err != nil { return nil, err }`)
**RateCard custom MarshalJSON/UnmarshalJSON on type discriminator** — RateCard.UnmarshalJSON reads productcatalog.RateCardSerde.Type first, then switches on FlatFeeRateCardType vs UsageBasedRateCardType to pick the concrete struct. Any new rate card type requires a new case here. (`switch s.Type { case productcatalog.FlatFeeRateCardType: serde.RateCard = &productcatalog.FlatFeeRateCard{} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (ListAddons, CreateAddon, DeleteAddon, GetAddon, UpdateAddon, PublishAddon, ArchiveAddon, NextAddon) and all input types with Validate(). | PublishAddonInput.Validate() uses a 30s jitter via clock.Now() for EffectiveFrom — do not substitute time.Now(). |
| `addon.go` | Addon aggregate type with Validate(), ValidateWith(), and AsProductCatalogAddon(). | Plans field is *[]Plan — only populated on expand; never assume non-nil. |
| `ratecard.go` | RateCard wraps productcatalog.RateCard with RateCardManagedFields; custom MarshalJSON/UnmarshalJSON dispatches on RateCardSerde.Type. | New rate card types require a case in UnmarshalJSON switch and the ratecard test. |
| `errors.go` | NotFoundError wrapping models.NewGenericNotFoundError; IsNotFound() for errors.As detection. | Never return raw entdb.IsNotFound — always wrap via NewNotFoundError. |
| `event.go` | All five lifecycle event structs; UserID sourced from session.GetSessionUserID(ctx). | AddonDeleteEvent.Validate() asserts Addon.DeletedAt != nil — soft-delete must be applied before constructing the event. |
| `repository.go` | Repository interface extends entutils.TxCreator — mandatory for TransactingRepo in the adapter layer. | TxCreator embedding is required; adapters without it cannot participate in ctx-bound transactions. |
| `validators.go` | IsAddonDeleted and HasAddonStatus validator funcs for use in service-layer guards. | Use via Addon.ValidateWith() instead of inline status checks. |

## Anti-Patterns

- Returning raw entdb.IsNotFound errors — always wrap in addon.NewNotFoundError.
- Using context.Background() in service or adapter code — always propagate ctx.
- Calling RateCards adapter operations without eager-loading related rate cards (WithRatecards).
- Publishing domain events outside a transaction closure — event failure should roll back the DB write.
- Creating multiple non-deleted draft versions for the same key without checking HasDraft().

## Decisions

- **RateCard is a separate managed type wrapping productcatalog.RateCard with AddonID** — Each rate card needs its own DB identity (NamespacedID) and foreign-key reference (AddonID) separate from the base productcatalog.RateCard so Ent can persist and eager-load them independently.
- **inputOptions.IgnoreNonCriticalIssues enables partial validation for draft addons** — Draft addons can be created with non-critical issues (e.g. missing features not yet registered); callers can opt into relaxed validation that only fails on ErrorSeverityCritical.
- **Domain events carry the full Addon pointer + UserID from session context** — Downstream Watermill consumers need the full entity for side-effects; UserID supports audit trails without requiring a separate lookup.

## Example: Creating and publishing a domain event after a successful adapter write inside a transaction

```
import (
    "github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*addon.Addon, error) {
    created, err := s.adapter.CreateAddon(ctx, params)
    if err != nil { return nil, err }
    event := addon.NewAddonCreateEvent(ctx, created)
    if err := s.publisher.Publish(ctx, event); err != nil {
        return nil, fmt.Errorf("publish addon created event: %w", err)
    }
    return created, nil
})
```

<!-- archie:ai-end -->
