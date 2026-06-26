# addon

<!-- archie:ai-start -->

> Root domain package for product-catalog add-ons: defines the Addon/RateCard/Plan value types, the Repository and Service interfaces, lifecycle events, validators, and not-found errors. Holds no DB or HTTP code itself — those live in adapter/, httpdriver/, and service/ children.

## Patterns

**Validate() collects errors then wraps** — Every domain type's Validate() appends into var errs []error and returns models.NewNillableGenericValidationError(errors.Join(errs...)) rather than returning on first failure. (`func (a Addon) Validate() error { var errs []error; ...; return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Embedded productcatalog meta + AsProductCatalog converter** — Domain Addon embeds productcatalog.AddonMeta and exposes AsProductCatalogAddon() to drop managed fields when delegating to shared productcatalog validation/logic. (`func (a Addon) AsProductCatalogAddon() productcatalog.Addon { return productcatalog.Addon{AddonMeta: a.AddonMeta, RateCards: a.RateCards.AsProductCatalogRateCards()} }`)
**Managed RateCard wraps productcatalog.RateCard with custom JSON** — RateCard embeds productcatalog.RateCard + RateCardManagedFields and implements MarshalJSON/UnmarshalJSON via productcatalog.RateCardSerde, switching on Type (FlatFeeRateCardType/UsageBasedRateCardType) to pick the concrete struct. (`switch s.Type { case productcatalog.FlatFeeRateCardType: serde.RateCard = &productcatalog.FlatFeeRateCard{} ... }`)
**Repository embeds entutils.TxCreator** — Repository interface embeds entutils.TxCreator so the service can drive transactions; Service mirrors Repository methods plus lifecycle ops (PublishAddon/ArchiveAddon/NextAddon). (`type Repository interface { entutils.TxCreator; ListAddons(...); CreateAddon(...); ... }`)
**Input structs own their Validate + IgnoreNonCriticalIssues** — Create/Update inputs embed inputOptions and filter validation issues via issues.WithSeverityOrHigher(models.ErrorSeverityCritical) when IgnoreNonCriticalIssues is set. (`if i.IgnoreNonCriticalIssues { issues = issues.WithSeverityOrHigher(models.ErrorSeverityCritical) }`)
**Typed NotFoundError over GenericNotFoundError** — NewNotFoundError builds a *NotFoundError wrapping models.NewGenericNotFoundError; IsNotFound(err) uses errors.As. Lifecycle validators (IsAddonDeleted, HasAddonStatus) are models.ValidatorFunc[Addon]. (`func IsNotFound(err error) bool { var e *NotFoundError; return errors.As(err, &e) }`)
**Events carry UserID from session + ComposeResourcePath** — Each lifecycle event (create/update/delete/publish/archive) pulls session.GetSessionUserID(ctx) and builds Source/Subject via metadata.ComposeResourcePath(ns, metadata.EntityAddon, id). (`resourcePath := metadata.ComposeResourcePath(e.Addon.Namespace, metadata.EntityAddon, e.Addon.ID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `addon.go` | Addon domain type, Validate, AsProductCatalogAddon | RateCards must validate per-element; Plans is only populated when expanded |
| `ratecard.go` | Managed RateCard, RateCards slice, custom JSON marshalling | UnmarshalJSON must dispatch on RateCardSerde.Type or it errors with 'invalid RateCard type' |
| `repository.go` | Repository interface (persistence contract) | Must embed entutils.TxCreator; do not add business ops here (those go on Service) |
| `service.go` | Service interface + all Input structs and their Validate | Publish/Archive inputs validate EffectiveFrom/To against clock.Now() with a 30s timeJitter |
| `event.go` | AddonCreate/Update/Delete/Publish/Archive events | Delete event requires DeletedAt set; all events emit metadata.EventType{Version:"v1"} |
| `validators.go` | IsAddonDeleted / HasAddonStatus ValidatorFunc[Addon] | Use via Addon.ValidateWith(...) before mutations rather than ad-hoc status checks |
| `errors.go` | Typed NotFoundError + IsNotFound | Adapter/service must convert raw ent errors into this type |

## Anti-Patterns

- Returning on first validation failure instead of collecting errs and wrapping with NewNillableGenericValidationError
- Adding DB queries or HTTP decode logic in this root package instead of the adapter/ or httpdriver/ child
- Mutating a RateCard's concrete type without updating MarshalJSON/UnmarshalJSON dispatch on RateCardSerde.Type
- Putting lifecycle/business operations (Publish/Archive/Next) on the Repository instead of Service
- Hand-comparing addons in tests instead of using assert.go helpers (AssertAddonEqual / AssertRateCardEqual)

## Decisions

- **Domain types wrap productcatalog.* meta plus managed fields** — Keeps shared catalog logic (pricing, entitlement templates, validation) in productcatalog while this package adds DB-managed identity and persistence-aware JSON.
- **Publish/Archive validate effective dates with a 30s timeJitter** — Tolerates clock skew between client request and server clock.Now() so near-now schedules are not rejected as 'in the past'.

## Example: Domain Validate aggregating sub-validations

```
func (a Addon) Validate() error {
	var errs []error
	if err := a.NamespacedID.Validate(); err != nil { errs = append(errs, err) }
	if err := a.ManagedModel.Validate(); err != nil { errs = append(errs, err) }
	if err := a.AddonMeta.Validate(); err != nil { errs = append(errs, err) }
	for _, rc := range a.RateCards { if err := rc.Validate(); err != nil { errs = append(errs, err) } }
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

<!-- archie:ai-end -->
