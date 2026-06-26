# productcatalog

<!-- archie:ai-start -->

> Root of the product-catalog domain and the highest-fan-in non-pkg package. The root files define the shared catalog value types (Price, RateCard, Discounts, EntitlementTemplate, TaxConfig, EffectivePeriod, ProRatingConfig, SettlementMode, Alignment) plus the FeatureResolver interface; subscription, billing, and entitlement all build on these primitives. Sub-packages split by concern: feature/, plan/, addon/, planaddon/ (domains), adapter/ (Ent), driver/+http/ (HTTP), featureresolver/, subscription/ (plan->subscription bridge), testutils/.

## Patterns

**Discriminated union value types** — Price, RateCard, and EntitlementTemplate are sum types: a private `t TypeEnum` field plus one pointer per variant, an `AsX()/FromX()` accessor interface (pricer, entitlementTemplater), and type-switch MarshalJSON/UnmarshalJSON keyed on a `type` field. (`type Price struct { t PriceType; flat *FlatPrice; unit *UnitPrice; ... } with `var _ pricer = (*Price)(nil)``)
**ValidateWith + ValidatorFunc composition** — Validation is composed from `models.ValidatorFunc[T]` closures applied via `models.Validate(v, fns...)`, not ad-hoc inline checks. Named validators like PercentageDiscountWithValidValue() / ValidateEffectivePeriod() are reusable. (`func (d PercentageDiscount) Validate() error { return d.ValidateWith(PercentageDiscountWithValidValue()) }`)
**Collect-then-wrap validation** — Multi-field Validate() accumulates into `var errs []error`, prefixes each with models.ErrorWithFieldPrefix(NewFieldSelectorGroup(...)), and returns models.NewGenericValidationError / NewNillableGenericValidationError(errors.Join(errs...)). (`Discounts.Validate wraps percentage/usage errors under field selectors then under a 'discounts' group`)
**Models contract assertions** — Each value type asserts the interfaces it satisfies at package scope: models.Validator, models.Equaler[T], models.Clonable[T], models.CustomValidator[T], hasher.Hasher. (`var ( _ models.Validator = (*UsageDiscount)(nil); _ hasher.Hasher = (*UsageDiscount)(nil) )`)
**ValidateForPrice context validation** — Discounts carry a second validation pass that depends on the owning Price (e.g. usage discount illegal on FlatPriceType). New price-dependent rules go in ValidateForPrice(*Price), not Validate(). (`UsageDiscountWithPrice returns ErrUsageDiscountWithFlatPrice when price.Type()==FlatPriceType`)
**ISODuration cadence alignment** — Billing cadences are compared via datetime.ISODuration.Simplify(true) + DivisibleBy both ways, never by raw numeric comparison; ValidateBillingCadencesAlign is the canonical check. (`ok, err := pSimple.DivisibleBy(rcSimple)`)
**Hash-based equality for discounts** — Discount value types implement hasher.Hasher by concatenating canonical String() forms; Discounts.Equal compares via equal.HasherPtrEqual rather than reflect.DeepEqual. (`func (d PercentageDiscount) Hash() hasher.Hash { return hasher.NewHash([]byte(d.Percentage.String())) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `price.go` | Price sum type (flat/unit/tiered/dynamic/package), PaymentTermType, Commitments, pricer interface | Never call json.Marshal(p) inside Price.MarshalJSON; switch on p.t and serialize the embedded variant |
| `ratecard.go` | RateCard interface (FlatFee/UsageBased) with Merge/ChangeMeta/Clone and RateCardMeta | UsageBasedRateCard must carry a non-zero billing cadence; ChangeMeta is the only sanctioned mutation entry point |
| `entitlement.go` | EntitlementTemplate sum type over metered/static/boolean, marshals with a `type` discriminator | Default branch in MarshalJSON/UnmarshalJSON errors on unknown type — keep the switch exhaustive |
| `discount.go` | PercentageDiscount, UsageDiscount, Discounts container with Validate + ValidateForPrice | Usage discount is invalid with a flat price; keep Hash/Clone/Equal in sync when adding a field |
| `effectiveperiod.go` | EffectivePeriod (EffectiveFrom/To) embedded by plans/addons; AsPeriod()->timeutil.OpenPeriod | EffectiveTo without EffectiveFrom is rejected (ErrEffectivePeriodFromNotSet) |
| `alignment.go` | ValidateBillingCadencesAlign — plan vs ratecard cadence divisibility | P1M vs P4W are not numerically comparable; always go through Simplify+DivisibleBy |
| `pro_rating.go` | ProRatingConfig/ProRatingMode (only prorate_prices) | Validate short-circuits when !Enabled; add new modes to the switch and Values() |
| `featureresolver.go` | FeatureResolver / NamespacedFeatureResolver interfaces (impl in featureresolver/) | Interface only — do not put lookup logic here; it belongs in the featureresolver sub-package |

## Anti-Patterns

- Marshaling a union (Price/RateCard/EntitlementTemplate) via json.Marshal on the wrapper itself instead of switching on the discriminator and serializing the active variant (infinite recursion / wrong shape)
- Writing inline validation branches instead of composing models.ValidatorFunc closures via ValidateWith/models.Validate
- Returning on the first invalid field instead of collecting errs and returning NewGenericValidationError/NewNillableGenericValidationError(errors.Join(...)) with field prefixes
- Comparing billing cadences numerically (P1M==P4W) instead of ValidateBillingCadencesAlign / Simplify+DivisibleBy
- Adding DB or HTTP code to this root package — persistence lives in adapter/ and the per-domain adapter children, HTTP in driver/http/ and httpdriver children

## Decisions

- **Catalog value types are self-validating sum types with custom JSON, not Ent rows or plain structs** — Plan/addon/ratecard prices are polymorphic and reused across subscription, billing, and entitlement; a typed union with discriminated JSON keeps the wire shape and the domain invariants in one place
- **FeatureResolver is declared here but implemented in featureresolver/** — Avoids a dependency cycle: subscription/billing depend on the resolver interface while the implementation depends on feature.FeatureConnector

## Example: Compose a ValidatorFunc and accumulate field-prefixed errors

```
func (d Discounts) Validate() error {
	var errs []error
	if d.Percentage != nil {
		if err := d.Percentage.Validate(); err != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(
				models.NewFieldSelectorGroup(models.NewFieldSelector("percentage")), err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return models.NewGenericValidationError(err)
	}
	return nil
}
```

<!-- archie:ai-end -->
