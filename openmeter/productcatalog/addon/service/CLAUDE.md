# service

<!-- archie:ai-start -->

> Business logic layer implementing addon.Service — versioned add-on lifecycle (create/update/delete/publish/archive), feature and tax-code resolution for rate cards before persistence, and Watermill domain-event publishing inside each transaction.

## Patterns

**transaction.Run for all mutations** — Create, Delete, Publish, Archive, UpdateAddon wrap their fn in transaction.Run(ctx, s.adapter, fn) for atomicity across adapter calls and event publishing. (`return transaction.Run(ctx, s.adapter, fn)`)
**resolveFeatures before persisting rate cards** — Create/Update call s.resolveFeatures to bi-directionally populate FeatureKey/FeatureID; GenericNotFoundError is converted to GenericValidationError. (`if err = s.resolveFeatures(ctx, params.Namespace, &params.RateCards); err != nil { if models.IsGenericNotFoundError(err) { err = models.NewGenericValidationError(err) } }`)
**resolveTaxCodes after resolveFeatures** — s.resolveTaxCodes populates TaxConfig.TaxCodeID via productcatalog.ResolveTaxConfig (taxCode.GetOrCreateByAppMapping) for each rate card with a Stripe code. (`if err = s.resolveTaxCodes(ctx, params.Namespace, &params.RateCards); err != nil { return nil, fmt.Errorf("failed to resolve tax codes: %w", err) }`)
**addonVersions for version management** — getAddonVersions fetches all versions incl. deleted; HasDraft() guards concurrent drafts; Latest() drives auto-increment of params.Version. (`params.Version = lo.FromPtr(versions.Latest()).Version + 1`)
**Event publish inside the transaction fn** — Each mutating op publishes a typed event (NewAddonCreateEvent, etc.) via s.publisher.Publish inside the tx fn; publish failure rolls back the DB write. (`event := addon.NewAddonCreateEvent(ctx, aa); if err = s.publisher.Publish(ctx, event); err != nil { return nil, err }`)
**Config validation in New** — service.New validates all five deps (Adapter, Feature, TaxCode, Logger, Publisher); var _ addon.Service = (*service)(nil) asserts compliance. (`if config.Adapter == nil { return nil, errors.New("add-on adapter is required") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, New constructor with mandatory dependency validation, service struct. | All five dependencies are required — never make them optional without explicit noop fallbacks. |
| `addon.go` | All addon.Service methods plus resolveFeatures, resolveTaxCodes, addonVersions, getAddonVersions. | Only draft add-ons can be updated (ValidateAddonWithStatus). Delete requires Plans loaded (Expand.PlanAddons=true) before checking active assignments. |
| `service_test.go` | Integration tests covering create->publish->v2->archive via pctestutils.NewTestEnv. | Verify single-draft enforcement (HasDraft) and version auto-increment. |
| `taxcode_test.go` | Tests for tax-code dual-write (TaxConfig.TaxCodeID) via assertAddonRCDBCols querying the DB directly. | assertAddonRCDBCols checks tax_code_id and tax_behavior columns at DB level — reuse for future tax-persistence tests. |

## Anti-Patterns

- Calling s.adapter methods directly without transaction.Run for multi-write or event-publishing ops.
- Skipping resolveFeatures/resolveTaxCodes when accepting rate cards.
- Allowing multiple non-deleted draft versions for the same key — check HasDraft() first.
- Publishing domain events outside the transaction fn.
- Using context.Background() instead of the passed ctx.

## Decisions

- **Version auto-increment computed from addonVersions.Latest() rather than a DB sequence.** — Enables draft-on-active branching with explicit version semantics in the domain model, decoupled from DB identity generation.
- **Tax code resolution (GetOrCreateByAppMapping) happens at the service layer, not the adapter.** — Tax-code creation is a cross-domain side effect requiring taxcode.Service; the adapter stays focused on persisting what it receives.

## Example: Add a mutating addon service operation

```
func (s service) CloneAddon(ctx context.Context, params addon.CloneAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid clone params: %w", err) }
		if err = s.resolveFeatures(ctx, params.Namespace, &newRateCards); err != nil { return nil, err }
		if err = s.resolveTaxCodes(ctx, params.Namespace, &newRateCards); err != nil { return nil, err }
		cloned, err := s.adapter.CreateAddon(ctx, createInput)
		if err != nil { return nil, err }
		if err = s.publisher.Publish(ctx, addon.NewAddonCreateEvent(ctx, cloned)); err != nil { return nil, err }
		return cloned, nil
	}
	return transaction.Run(ctx, s.adapter, fn)
}
```

<!-- archie:ai-end -->
