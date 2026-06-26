# http

<!-- archie:ai-start -->

> Shared API<->domain mapping (package http) for productcatalog value types — rate cards, prices, entitlement templates, tax config, discounts. Reused by plan, addon, planaddon, subscription, and billing HTTP drivers.

## Patterns

**FromX / AsX mapping convention** — FromRateCard/FromTaxConfig/FromEntitlementTemplate map domain->api; AsRateCard/AsFlatFeeRateCard/AsUsageBasedRateCard/AsTaxConfig map api->domain. Follow this naming for new conversions. (`func FromRateCard(r productcatalog.RateCard) (api.RateCard, error)`)
**Discriminator-driven union mapping** — Mapping branches on r.Type()/r.Discriminator() (FlatFeeRateCardType, UsageBasedRateCardType) and price.Type() (Flat/Unit/Tiered/Dynamic/Package), using oapi-codegen FromX setters on the union. (`switch price.Type() { case productcatalog.TieredPriceType: ... resp.FromTieredPriceWithCommitments(...) }`)
**Decimal and ISO-duration string conversion** — Amounts cross the boundary as strings (alpacadecimal .String() / decimal.NewFromString); billing cadences as datetime.ISODurationString parsed via Parse/ParsePtrOrNil. (`rc.BillingCadence, err = datetime.ISODurationString(usage.BillingCadence).Parse()`)
**ValidationIssue error encoder per ResourceKind** — ValidationErrorEncoder(kind) extracts models.AsValidationIssues and emits a validationError with a validationErrors extension; resource.go defines ResourceKind constants (plan, add-on). (`http.ValidationErrorEncoder(http.ResourceKindPlan)`)
**Wrap every cast failure with context** — Conversions return fmt.Errorf("failed to cast X: %w", err); never swallow union-cast errors. (`return resp, fmt.Errorf("failed to cast FlatPrice: %w", err)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mapping.go` | All rate-card / price / entitlement-template / tax / discount conversions in both directions. | UsageBasedRateCard requires a non-nil billing cadence (errors otherwise); DynamicPrice omits Multiplier when equal to DynamicPriceDefaultMultiplier; entitlement template mapping switches on entitlement.EntitlementType*. |
| `errors.go` | ValidationErrorEncoder + validationError carrying ResourceKind and ValidationIssues. | Only fires when models.AsValidationIssues yields issues; returns false (passes through) otherwise. |
| `resource.go` | ResourceKind enum (plan, add-on) used to label validation errors. | Add a kind here before referencing it in a new driver's ValidationErrorEncoder. |

## Anti-Patterns

- Passing decimals or durations as floats/raw numbers instead of strings across the API boundary
- Skipping the discriminator switch and casting a union to the wrong concrete type
- Swallowing oapi-codegen FromX/AsX cast errors instead of wrapping with context
- Emitting a 0/empty billing cadence for a UsageBasedRateCard

## Decisions

- **Shared mapping lives in one http package consumed by many drivers** — Plan/addon/planaddon/subscription/billing all serialize the same rate-card and price unions, so conversion is centralized to stay consistent.
- **Validation surfaces as a typed validationError with a validationErrors extension** — Lets the HTTP layer return structured per-field issues (models.ValidationIssues) with a stable resource label.

## Example: Wiring a resource validation error encoder

```
import pchttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"

httptransport.WithErrorEncoder(
  pchttp.ValidationErrorEncoder(pchttp.ResourceKindPlan),
)
```

<!-- archie:ai-end -->
