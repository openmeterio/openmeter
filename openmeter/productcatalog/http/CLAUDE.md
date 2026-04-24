# http

<!-- archie:ai-start -->

> Shared HTTP utilities for the productcatalog domain: validation error encoding (ValidationErrorEncoder with ValidationIssues extension), bidirectional API↔domain mapping for RateCards/EntitlementTemplates/Prices/Discounts, and ResourceKind constants.

## Patterns

**ValidationErrorEncoder wraps models.ValidationIssues** — ValidationErrorEncoder(kind ResourceKind) converts []models.ValidationIssue into a validationErrors extension field on the problem+json response. Attach per-resource-kind; call with ResourceKindPlan or ResourceKindAddon. (`httptransport.WithErrorEncoder(http.ValidationErrorEncoder(http.ResourceKindPlan))`)
**Discriminator-based RateCard mapping** — AsRateCard uses r.Discriminator() to dispatch to AsFlatFeeRateCard or AsUsageBasedRateCard. FromRateCard uses r.Type() switch. Any new RateCard type requires cases in both directions. (`rType, _ := r.Discriminator(); switch rType { case string(productcatalog.FlatFeeRateCardType): ... }`)
**Price type dispatch via price.Type()** — FromRateCardUsageBasedPrice dispatches on price.Type() (Flat, Unit, Tiered, Dynamic, Package). New price models require a new case in both FromRateCardUsageBasedPrice and AsRateCard. (`switch price.Type() { case productcatalog.FlatPriceType: ... case productcatalog.TieredPriceType: ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mapping.go` | FromRateCard, AsRateCard, AsFlatFeeRateCard, AsUsageBasedRateCard, FromRateCardUsageBasedPrice, FromTaxConfig, FromEntitlementTemplate, FromDiscounts and inverses. | UsageBasedRateCard requires non-nil BillingCadence; returns error if nil. FlatFeeRateCard BillingCadence is optional. |
| `errors.go` | ValidationErrorEncoder and validationError private type with AsErrorExtension serialization. | Returns false (not handled) if models.AsValidationIssues returns no issues — falls through to generic encoder. |
| `resource.go` | ResourceKind string type with ResourceKindPlan and ResourceKindAddon constants. | Add new ResourceKind constants here when adding new top-level resources to the productcatalog. |

## Anti-Patterns

- Adding business logic to mapping functions — they must be pure conversions with no side-effects.
- Skipping error return from FromRateCard or AsRateCard — both can return errors on unsupported types.
- Using this package from outside productcatalog without considering that it imports api (v1) types.

## Decisions

- **Separate http package for shared productcatalog HTTP utilities rather than embedding in each sub-package** — RateCard and price mappings are reused across plan, addon, and subscription HTTP handlers; centralising avoids duplication.

<!-- archie:ai-end -->
