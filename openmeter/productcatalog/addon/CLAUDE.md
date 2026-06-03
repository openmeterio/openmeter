# addon

<!-- archie:ai-start -->

> Domain package for add-on lifecycle management: defines the Addon aggregate (with RateCards), the Service + Repository interfaces, typed errors, domain events, and status/deletion validators; its children split into adapter (Ent persistence), httpdriver (v1 HTTP), and service (validation, feature/taxcode resolution, event publishing). Primary constraint: domain types always carry both productcatalog.* base types and managed identity fields (NamespacedID, ManagedModel).

## Patterns

**Addon aggregate embedding** — Addon embeds models.NamespacedID, models.ManagedModel, productcatalog.AddonMeta, and RateCards; RateCard wraps productcatalog.RateCard with RateCardManagedFields (AddonID). Never use struct literals missing these fields. (`type Addon struct { models.NamespacedID; models.ManagedModel; productcatalog.AddonMeta; RateCards RateCards }`)
**RateCards.AsProductCatalogRateCards() for cross-type conversion** — Never cast addon.RateCards to productcatalog.RateCards directly — call AsProductCatalogRateCards(), which extracts the inner productcatalog.RateCard from each wrapper. (`func (c RateCards) AsProductCatalogRateCards() productcatalog.RateCards { for _, rc := range c { rcs = append(rcs, rc.RateCard) }; return rcs }`)
**RateCard type-discriminator JSON** — RateCard.UnmarshalJSON reads productcatalog.RateCardSerde.Type then switches FlatFeeRateCardType vs UsageBasedRateCardType to pick the concrete struct; new rate card types need a case here and in the ratecard test. (`switch s.Type { case productcatalog.FlatFeeRateCardType: serde.RateCard = &productcatalog.FlatFeeRateCard{} }`)
**Typed NotFoundError + draft validation options** — Adapters return addon.NewNotFoundError(...) wrapping models.NewGenericNotFoundError (never raw entdb.IsNotFound); input structs embed inputOptions{IgnoreNonCriticalIssues} so drafts can pass with non-critical issues via WithSeverityOrHigher(ErrorSeverityCritical). (`issues := models.AsValidationIssues(errors.Join(errs...)); if i.IgnoreNonCriticalIssues { issues = issues.WithSeverityOrHigher(models.ErrorSeverityCritical) }`)
**Events via constructors, published inside the transaction** — Lifecycle events (Create/Update/Delete/Publish/Archive) carry the full Addon pointer + UserID from session.GetSessionUserID(ctx); the service child publishes them inside transaction.Run so DB write and event stay consistent. (`event := addon.NewAddonCreateEvent(ctx, created); s.publisher.Publish(ctx, event)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (List/Create/Delete/Get/Update/Publish/Archive/Next) and input types with Validate(). | PublishAddonInput.Validate() uses a 30s jitter via clock.Now() for EffectiveFrom — do not substitute time.Now(). |
| `addon.go` | Addon aggregate with Validate(), ValidateWith(), AsProductCatalogAddon(). | Plans field is *[]Plan — only populated on expand; never assume non-nil. |
| `ratecard.go` | RateCard wrapping productcatalog.RateCard with RateCardManagedFields; custom Marshal/UnmarshalJSON dispatching on RateCardSerde.Type. | New rate card types require a case in the UnmarshalJSON switch and the ratecard test. |
| `errors.go` | NotFoundError wrapping models.NewGenericNotFoundError; IsNotFound() for errors.As detection. | Never return raw entdb.IsNotFound — always wrap via NewNotFoundError. |
| `event.go` | Five lifecycle event structs; UserID sourced from session.GetSessionUserID(ctx). | AddonDeleteEvent.Validate() asserts Addon.DeletedAt != nil — soft-delete must be applied before constructing the event. |
| `repository.go` | Repository interface embedding entutils.TxCreator — mandatory for TransactingRepo in the adapter. | Without TxCreator embedding the adapter cannot participate in ctx-bound transactions. |
| `validators.go` | IsAddonDeleted and HasAddonStatus models.ValidatorFunc[Addon] for service-layer guards. | Use via Addon.ValidateWith() instead of inline status checks. |

## Anti-Patterns

- Returning raw entdb.IsNotFound errors — always wrap in addon.NewNotFoundError.
- Using context.Background() in service or adapter code — always propagate ctx.
- Querying add-ons without eager-loading rate cards (WithRatecards).
- Publishing domain events outside a transaction closure — event failure should roll back the DB write.
- Creating multiple non-deleted draft versions for the same key without checking HasDraft().

## Decisions

- **RateCard is a separate managed type wrapping productcatalog.RateCard with AddonID.** — Each rate card needs its own DB identity (NamespacedID) and foreign key (AddonID) so Ent can persist and eager-load them independently.
- **inputOptions.IgnoreNonCriticalIssues enables partial validation for draft addons.** — Drafts can be created with non-critical issues (e.g. features not yet registered); callers opt into relaxed validation that only fails on ErrorSeverityCritical.
- **Tax code and feature resolution happen at the service layer, not the adapter.** — resolveFeatures/resolveTaxCodes (GetOrCreateByAppMapping) are cross-domain orchestration that the persistence layer must stay free of.

## Example: Publishing a domain event after a successful adapter write inside a transaction

```
import (
    "github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*addon.Addon, error) {
    created, err := s.adapter.CreateAddon(ctx, params)
    if err != nil { return nil, err }
    if err := s.publisher.Publish(ctx, addon.NewAddonCreateEvent(ctx, created)); err != nil {
        return nil, fmt.Errorf("publish addon created event: %w", err)
    }
    return created, nil
})
```

<!-- archie:ai-end -->
