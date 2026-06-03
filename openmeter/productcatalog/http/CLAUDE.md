# http

<!-- archie:ai-start -->

> Shared HTTP utilities for the productcatalog domain: ValidationErrorEncoder (wraps ValidationIssues into problem+json), bidirectional API↔domain mapping for RateCards/Prices/EntitlementTemplates/Discounts, and ResourceKind constants. Consumed by plan, addon, and subscription HTTP handlers to avoid duplication.

## Patterns

**ValidationErrorEncoder with ResourceKind** — ValidationErrorEncoder(kind ResourceKind) converts []models.ValidationIssue into a validationErrors extension on the problem+json 400 response; attach per-resource-kind in handler options. (`httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan))`)
**Discriminator-based RateCard mapping** — AsRateCard dispatches on r.Discriminator(); FromRateCard switches on r.Type(). New RateCard types require cases in both directions. (`rType, _ := r.Discriminator(); switch rType { case string(productcatalog.FlatFeeRateCardType): return AsFlatFeeRateCard(r) }`)
**Price type dispatch via price.Type()** — FromRateCardUsageBasedPrice dispatches on price.Type() (Flat, Unit, Tiered, Dynamic, Package); new price models need a case in both FromRateCardUsageBasedPrice and AsRateCard. (`switch price.Type() { case productcatalog.FlatPriceType: ...; case productcatalog.TieredPriceType: ... }`)
**UsageBasedRateCard requires non-nil BillingCadence** — In FromRateCard for UsageBasedRateCardType, billingCadence != nil is mandatory (return error if nil); FlatFeeRateCard BillingCadence is optional. (`if billingCadence == nil { return resp, errors.New("invalid UsageBasedRateCard: billing cadence must be set") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mapping.go` | FromRateCard, AsRateCard, AsFlatFeeRateCard, AsUsageBasedRateCard, FromRateCardUsageBasedPrice, FromTaxConfig, FromEntitlementTemplate, FromDiscounts and inverses. | UsageBasedRateCard requires non-nil BillingCadence; all functions must stay pure conversions with no side-effects. |
| `errors.go` | ValidationErrorEncoder and private validationError type with AsErrorExtension serialization. | Returns false (not handled) if models.AsValidationIssues yields no issues — falls through to the generic encoder. |
| `resource.go` | ResourceKind string type with ResourceKindPlan and ResourceKindAddon constants. | Add new ResourceKind constants here when introducing new top-level productcatalog resources. |

## Anti-Patterns

- Adding business logic or side-effects to mapping functions — they must be pure bidirectional conversions.
- Skipping the error return from FromRateCard/AsRateCard — both can fail on unsupported types.
- Importing this package from outside productcatalog without knowing it imports api (v1) types directly.

## Decisions

- **Separate http package for shared productcatalog HTTP utilities** — RateCard and price mappings are reused across plan, addon, and subscription handlers; centralising avoids duplication and keeps conversion logic in one reviewable place.

<!-- archie:ai-end -->
