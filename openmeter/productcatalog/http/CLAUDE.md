# http

<!-- archie:ai-start -->

> Shared HTTP utilities for the productcatalog domain: ValidationErrorEncoder (wraps ValidationIssues into problem+json extension), bidirectional API↔domain mapping for RateCards/Prices/EntitlementTemplates/Discounts, and ResourceKind constants. Consumed by plan, addon, and subscription HTTP handlers to avoid duplication.

## Patterns

**ValidationErrorEncoder with ResourceKind** — ValidationErrorEncoder(kind ResourceKind) converts []models.ValidationIssue into a validationErrors extension on the problem+json 400 response. Attach per-resource-kind in handler options. (`httptransport.WithErrorEncoder(http.ValidationErrorEncoder(http.ResourceKindPlan))`)
**Discriminator-based RateCard mapping** — AsRateCard uses r.Discriminator() to dispatch to AsFlatFeeRateCard or AsUsageBasedRateCard. FromRateCard uses r.Type() switch. New RateCard types require cases in both directions. (`rType, _ := r.Discriminator(); switch rType { case string(productcatalog.FlatFeeRateCardType): return AsFlatFeeRateCard(r) }`)
**Price type dispatch via price.Type()** — FromRateCardUsageBasedPrice dispatches on price.Type() (Flat, Unit, Tiered, Dynamic, Package). New price models require a new case in both FromRateCardUsageBasedPrice and AsRateCard. (`switch price.Type() { case productcatalog.FlatPriceType: ... case productcatalog.TieredPriceType: ... }`)
**UsageBasedRateCard requires non-nil BillingCadence** — In FromRateCard for UsageBasedRateCardType, check billingCadence != nil and return an error if nil. FlatFeeRateCard BillingCadence is optional. (`if billingCadence == nil { return resp, errors.New("invalid UsageBasedRateCard: billing cadence must be set") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mapping.go` | FromRateCard, AsRateCard, AsFlatFeeRateCard, AsUsageBasedRateCard, FromRateCardUsageBasedPrice, FromTaxConfig, FromEntitlementTemplate, FromDiscounts and their inverses. | UsageBasedRateCard requires non-nil BillingCadence; returns error if nil. All functions must remain pure conversions with no side-effects. |
| `errors.go` | ValidationErrorEncoder and private validationError type with AsErrorExtension serialization. | Returns false (not handled) if models.AsValidationIssues returns no issues — falls through to the generic encoder. |
| `resource.go` | ResourceKind string type with ResourceKindPlan and ResourceKindAddon constants. | Add new ResourceKind constants here when adding new top-level productcatalog resources. |

## Anti-Patterns

- Adding business logic or side-effects to mapping functions — they must be pure bidirectional conversions.
- Skipping error return from FromRateCard or AsRateCard — both can return errors on unsupported types.
- Importing this package from outside productcatalog without knowing it imports api (v1) types directly.

## Decisions

- **Separate http package for shared productcatalog HTTP utilities rather than embedding in each sub-package** — RateCard and price mappings are reused across plan, addon, and subscription HTTP handlers; centralising avoids duplication and keeps conversion logic in one reviewable place.

<!-- archie:ai-end -->
