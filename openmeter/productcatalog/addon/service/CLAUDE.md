# service

<!-- archie:ai-start -->

> Business-logic service implementing addon.Service: version lifecycle (create/publish/archive), validation, feature & tax-code resolution, transactional orchestration over addon.Repository, and event publishing.

## Patterns

**Config-validated constructor returning addon.Service** — New(Config) requires Adapter, FeatureResolver, TaxCode, Logger, Publisher (each nil-checked, returns errors.New). var _ addon.Service = (*service)(nil). Service holds adapter/taxCode/logger/publisher/featureResolver. (`func New(config Config) (addon.Service, error) { if config.Adapter == nil { return nil, errors.New("add-on adapter is required") } ... }`)
**transaction.Run wrapping for mutations** — Write methods (Create/Update/Delete/Publish/Archive) run their closure via transaction.Run(ctx, s.adapter, fn); read methods (List/Get) just call fn(ctx) directly without a transaction. (`return transaction.Run(ctx, s.adapter, fn)`)
**Validate params then re-validate domain status before mutating** — Each method validates params, then loads the addon and calls add.AsProductCatalogAddon().ValidateWith(ValidateAddonWithStatus(...)) to enforce status gates (e.g. update only Draft, delete only Draft/Archived). (`add.AsProductCatalogAddon().ValidateWith(productcatalog.ValidateAddonWithStatus(productcatalog.AddonStatusDraft))`)
**Feature + tax-code resolution before persisting ratecards** — Create/Update resolve features via featureresolver.ResolveFeaturesForRateCards(ctx, s.featureResolver, ns, rateCards) and tax codes via s.resolveTaxCodes -> productcatalog.ResolveTaxConfig, then merge resolved RateCardMeta back via rc.Merge. (`featureresolver.ResolveFeaturesForRateCards(ctx, s.featureResolver, params.Namespace, &params.RateCards)`)
**Single-draft + auto-increment version invariant** — CreateAddon lists all versions via getAddonVersions, rejects if addonVersions.HasDraft() (NewGenericValidationError), and sets params.Version = Latest().Version + 1. addonVersions implements sort.Interface (Len/Less/Swap/Sort/Latest/HasDraft). (`if versions.HasDraft() { return nil, models.NewGenericValidationError(...) }; params.Version = lo.FromPtr(versions.Latest()).Version + 1`)
**Publish archives previous active version then sets EffectivePeriod** — PublishAddon validates Publishable() + ValidateAddonWithFeatures, and for Version>1 looks up the active version by key and ArchiveAddon's it at EffectiveFrom before updating the new version's EffectivePeriod. (`s.ArchiveAddon(ctx, addon.ArchiveAddonInput{NamespacedID: ..., EffectiveTo: lo.FromPtr(params.EffectiveFrom)})`)
**Emit eventbus event after every mutation** — After each successful write the service publishes via s.publisher.Publish(ctx, addon.NewAddonCreateEvent/Update/Delete/Publish event); a publish failure fails the whole transaction. (`event := addon.NewAddonCreateEvent(ctx, aa); err = s.publisher.Publish(ctx, event)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, New constructor with nil-checks, and the private service struct + addon.Service assertion. | All five collaborators are mandatory; FeatureResolver is productcatalog.FeatureResolver, TaxCode is taxcode.Service, Publisher is eventbus.Publisher. |
| `addon.go` | All service methods (List/Create/Get/Update/Delete/Publish/Archive) plus helpers resolveTaxCodes, getAddonVersions, and addonVersions sort type. | UpdateAddon zeroes params.EffectivePeriod (state changes only via Publish/Archive). DeleteAddon requires Plans expanded (Expand.PlanAddons) and refuses delete if any active plan assignment exists. |
| `service_test.go` | Integration test TestAddonService over pctestutils.NewTestEnv covering create/publish/archive lifecycle with features, meters, tax codes. | Requires Postgres; builds features per meter via NewTestFeatureFromMeter and a real taxcode entity before referencing TaxConfig.TaxCodeID. |
| `taxcode_test.go` | Tests stripe-code -> TaxCode entity resolution and that changing a stripe code creates a new TaxCode while preserving the old entity. | Relies on resolveTaxCodes creating TaxCode entities lazily; old entities are never deleted on code change. |

## Anti-Patterns

- Mutating addon state through UpdateAddon's EffectivePeriod instead of Publish/Archive (it is zeroed).
- Skipping the AsProductCatalogAddon().ValidateWith status gate before a mutation.
- Calling s.adapter mutations outside transaction.Run, losing atomicity with event publishing.
- Creating a second draft version (violates single-draft invariant enforced by HasDraft).
- Persisting ratecards before resolving features/tax codes, leaving FeatureID/TaxCodeID unset.

## Decisions

- **Event publish happens inside the same transaction as the write.** — Guarantees no event is emitted for a rolled-back mutation; a publish error aborts the change.
- **Versions are immutable; publish archives the prior active version rather than editing it.** — Preserves historical add-on definitions and keeps exactly one active version per key.
- **Tax-code resolution is lazy and additive (new stripe code -> new TaxCode entity).** — Avoids mutating tax codes shared by historical versions; old entities stay valid for archived add-ons.

## Example: Mutating service method with validation, resolution, transaction, and event publish

```
func (s service) CreateAddon(ctx context.Context, params addon.CreateAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid create add-on params: %w", err) }
		versions, err := s.getAddonVersions(ctx, params.Namespace, params.Key)
		if err != nil { return nil, err }
		if versions.HasDraft() { return nil, models.NewGenericValidationError(fmt.Errorf("only a single draft version is allowed for add-on")) }
		params.Version = lo.FromPtr(versions.Latest()).Version + 1
		if len(params.RateCards) > 0 {
			if err = featureresolver.ResolveFeaturesForRateCards(ctx, s.featureResolver, params.Namespace, &params.RateCards); err != nil { return nil, err }
			if err = s.resolveTaxCodes(ctx, params.Namespace, &params.RateCards); err != nil { return nil, err }
		}
		aa, err := s.adapter.CreateAddon(ctx, params)
		if err != nil { return nil, err }
		if err = s.publisher.Publish(ctx, addon.NewAddonCreateEvent(ctx, aa)); err != nil { return nil, err }
		return aa, nil
// ...
```

<!-- archie:ai-end -->
