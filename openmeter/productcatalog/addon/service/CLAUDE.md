# service

<!-- archie:ai-start -->

> Business logic layer implementing addon.Service: versioned add-on lifecycle (create/update/delete/publish/archive), bidirectional feature and tax code resolution for rate cards before persistence, and domain event publishing via Watermill inside each transaction.

## Patterns

**transaction.Run for all mutating operations** — Create, Delete, Publish, Archive, and UpdateAddon wrap their fn in transaction.Run(ctx, s.adapter, fn) to ensure atomicity across adapter calls and event publishing. (`return transaction.Run(ctx, s.adapter, fn)`)
**resolveFeatures before persisting rate cards** — CreateAddon and UpdateAddon call s.resolveFeatures(ctx, ns, &params.RateCards) to bi-directionally populate missing FeatureKey/FeatureID from the feature connector. GenericNotFoundError is converted to GenericValidationError. (`if err = s.resolveFeatures(ctx, params.Namespace, &params.RateCards); err != nil { if models.IsGenericNotFoundError(err) { err = models.NewGenericValidationError(err) } ... }`)
**resolveTaxCodes after resolveFeatures** — After resolveFeatures, s.resolveTaxCodes populates TaxConfig.TaxCodeID by calling productcatalog.ResolveTaxConfig (which calls taxCode.GetOrCreateByAppMapping) for each rate card with a Stripe code. (`if err = s.resolveTaxCodes(ctx, params.Namespace, &params.RateCards); err != nil { return nil, fmt.Errorf("failed to resolve tax codes: %w", err) }`)
**addonVersions collection for version management** — getAddonVersions fetches all versions including deleted. HasDraft() guards against multiple concurrent drafts. Latest() drives auto-increment of params.Version. (`params.Version = lo.FromPtr(versions.Latest()).Version + 1`)
**Event publishing inside the transaction fn** — Every mutating operation publishes a typed domain event (NewAddonCreateEvent, NewAddonDeleteEvent, etc.) via s.publisher.Publish(ctx, event) inside the transaction fn. Event publish failure rolls back the DB write. (`event := addon.NewAddonCreateEvent(ctx, aa); if err = s.publisher.Publish(ctx, event); err != nil { return nil, fmt.Errorf("failed to publish event: %w", err) }`)
**Config validation in New constructor** — service.New(Config) validates all five required dependencies (Adapter, Feature, TaxCode, Logger, Publisher) before constructing. var _ addon.Service = (*service)(nil) provides compile-time assertion. (`if config.Adapter == nil { return nil, errors.New("add-on adapter is required") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, New constructor with mandatory dependency validation, service struct definition. | All five dependencies are required — never make them optional without explicit noop fallbacks. |
| `addon.go` | All addon.Service method implementations plus resolveFeatures, resolveTaxCodes, addonVersions helper type, and getAddonVersions. | Only draft add-ons can be updated; active/archived reject changes via ValidateAddonWithStatus. Delete checks Plans are loaded (Expand.PlanAddons=true) before checking for active assignments. |
| `service_test.go` | Integration tests covering full lifecycle: create->publish->v2 create->publish->archive. Uses pctestutils.NewTestEnv. | Tests verify single-draft enforcement (HasDraft), version auto-increment (Version field), and status transition correctness. |
| `taxcode_test.go` | Dedicated tests for tax code dual-write (TaxConfig.TaxCodeID population) covering create/update/remove scenarios. Uses assertAddonRCDBCols to query the DB directly. | assertAddonRCDBCols verifies dedicated tax_code_id and tax_behavior columns at DB level — use this pattern for future tax-related persistence verification. |

## Anti-Patterns

- Calling s.adapter methods directly without transaction.Run for operations combining multiple writes or event publishing.
- Skipping resolveFeatures/resolveTaxCodes when accepting rate cards — FeatureKey/FeatureID and TaxCodeID must be resolved before persisting.
- Allowing multiple non-deleted draft versions for the same key — HasDraft() guard must be checked before creating.
- Publishing domain events outside the transaction fn — event publish failure should roll back the DB write.
- Using context.Background() instead of the passed ctx in service methods.

## Decisions

- **Version auto-increment is computed from existing versions via addonVersions.Latest() rather than a DB sequence.** — Enables multi-version branching (draft on top of active) with explicit version semantics visible in the domain model, decoupled from DB identity generation.
- **Tax code resolution (GetOrCreateByAppMapping) is performed at the service layer, not the adapter layer.** — Tax code entity creation is a cross-domain side effect requiring taxcode.Service; the adapter must stay focused on persisting what it receives without cross-domain calls.

## Example: Add a new mutating addon service operation following established pattern

```
func (s service) CloneAddon(ctx context.Context, params addon.CloneAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid clone add-on params: %w", err)
		}
		src, err := s.adapter.GetAddon(ctx, addon.GetAddonInput{NamespacedID: params.NamespacedID})
		if err != nil { return nil, err }
		// resolve features and tax codes for new rate cards
		if err = s.resolveFeatures(ctx, params.Namespace, &newRateCards); err != nil { ... }
		if err = s.resolveTaxCodes(ctx, params.Namespace, &newRateCards); err != nil { ... }
		cloned, err := s.adapter.CreateAddon(ctx, createInput)
		if err != nil { return nil, err }
		event := addon.NewAddonCreateEvent(ctx, cloned)
		if err = s.publisher.Publish(ctx, event); err != nil { return nil, err }
		return cloned, nil
// ...
```

<!-- archie:ai-end -->
