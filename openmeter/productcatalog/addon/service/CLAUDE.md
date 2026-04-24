# service

<!-- archie:ai-start -->

> Business logic layer implementing addon.Service: versioned add-on lifecycle (create/update/delete/publish/archive), feature and tax code resolution for rate cards, and event publishing via Watermill.

## Patterns

**transaction.Run wrapping for mutating operations** — Create, Delete, Publish, Archive, and UpdateAddon (when rate cards change) wrap their fn in transaction.Run(ctx, s.adapter, fn) to ensure atomicity across adapter calls + event publish. (`return transaction.Run(ctx, s.adapter, fn)`)
**resolveFeatures before persisting rate cards** — CreateAddon and UpdateAddon call s.resolveFeatures(ctx, ns, &params.RateCards) to bi-directionally populate missing FeatureKey/FeatureID from the feature connector before saving. (`if err = s.resolveFeatures(ctx, params.Namespace, &params.RateCards); err != nil { ... }`)
**resolveTaxCodes before persisting rate cards** — After resolveFeatures, s.resolveTaxCodes populates TaxConfig.TaxCodeID by calling taxCode.GetOrCreateByAppMapping for each rate card with a Stripe code. (`if err = s.resolveTaxCodes(ctx, params.Namespace, &params.RateCards); err != nil { ... }`)
**addonVersions collection for version management** — getAddonVersions fetches all versions (including deleted) and returns addonVersions slice. HasDraft() guards against multiple concurrent draft versions. Latest() drives auto-increment of params.Version. (`params.Version = lo.FromPtr(versions.Latest()).Version + 1`)
**Event publishing after every mutation** — Every mutating operation publishes a typed domain event (NewAddonCreateEvent, NewAddonDeleteEvent, etc.) via s.publisher.Publish(ctx, event) inside the transaction fn. (`event := addon.NewAddonCreateEvent(ctx, aa); if err = s.publisher.Publish(ctx, event); err != nil { ... }`)
**Config struct with validation in New constructor** — service.New(Config) validates all required dependencies (Adapter, Feature, TaxCode, Logger, Publisher) before constructing the service. Returns addon.Service interface. (`var _ addon.Service = (*service)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, New constructor, service struct definition. All dependencies injected at construction time. | All five dependencies are required — never make them optional without explicit noop fallbacks. |
| `addon.go` | All addon.Service method implementations. Contains resolveFeatures, resolveTaxCodes, addonVersions helper type, and getAddonVersions. | resolveFeatures converts GenericNotFoundError to GenericValidationError (feature not found = invalid input). Only draft add-ons can be updated; active/archived add-ons reject changes via ValidateAddonWithStatus. |
| `service_test.go` | Integration tests for full add-on lifecycle: create→publish→v2 create→publish→archive. Uses pctestutils.NewTestEnv. | Tests verify single-draft enforcement (HasDraft), version auto-increment, and status transition correctness. |
| `taxcode_test.go` | Dedicated tests for tax code dual-write (TaxConfig.TaxCodeID population) covering create/update/remove scenarios. | assertAddonRCDBCols directly queries the DB to verify dedicated tax_code_id and tax_behavior columns — use this pattern to verify low-level persistence in future tax-related changes. |

## Anti-Patterns

- Calling s.adapter methods directly without transaction.Run for operations that combine multiple writes or event publishing.
- Skipping resolveFeatures/resolveTaxCodes when accepting rate cards from callers — FeatureKey/FeatureID and TaxCodeID must be resolved before persisting.
- Allowing multiple non-deleted draft versions for the same key — HasDraft() guard must be checked before creating.
- Publishing domain events outside the transaction fn — event publish failure should roll back the DB write.
- Using context.Background() instead of the passed ctx in service methods.

## Decisions

- **Version auto-increment is computed from existing versions rather than relying on a DB sequence.** — Enables multi-version branching (draft on top of active) with explicit version semantics visible in the domain model, and avoids coupling version numbering to DB identity.
- **Tax code resolution (GetOrCreateByAppMapping) is performed at the service layer, not the adapter layer.** — Tax code entity creation is a side effect that requires cross-domain coordination (taxcode.Service); the adapter layer must stay focused on Ent persistence of what it receives.

## Example: Add a new addon mutating operation following the established pattern

```
func (s service) CloneAddon(ctx context.Context, params addon.CloneAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid clone add-on params: %w", err)
		}
		src, err := s.adapter.GetAddon(ctx, addon.GetAddonInput{NamespacedID: params.NamespacedID})
		if err != nil { return nil, err }
		// ... build CreateAddonInput from src ...
		cloned, err := s.adapter.CreateAddon(ctx, createInput)
		if err != nil { return nil, err }
		event := addon.NewAddonCreateEvent(ctx, cloned)
		if err = s.publisher.Publish(ctx, event); err != nil { return nil, err }
		return cloned, nil
	}
	return transaction.Run(ctx, s.adapter, fn)
// ...
```

<!-- archie:ai-end -->
