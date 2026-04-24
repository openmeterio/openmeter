# productcatalog

<!-- archie:ai-start -->

> Root domain package for the product catalog: defines shared value types (RateCard, Price, Discount, EntitlementTemplate, EffectivePeriod, PlanAddon, Phase, ProRatingConfig) and validators consumed by all sub-packages (addon, plan, planaddon, feature, subscription). Acts as the shared type contract between billing, entitlement, and subscription layers.

## Patterns

**ValidatorFunc composition** — All domain types implement models.Validator and models.CustomValidator[T]; validation is composed from named ValidatorFunc[T] functions (e.g. ValidateAddonMeta(), ValidateAddonRateCards()) and called via models.Validate(). Never inline validation logic directly in Validate(). (`func (a Addon) Validate() error { return a.ValidateWith(ValidateAddonMeta(), ValidateAddonRateCards()) }`)
**ValidationIssue sentinel errors** — All domain errors are package-level models.ValidationIssue vars with ErrorCode constants (ErrCodeXxx / ErrXxx pairs). HTTP status codes are attached via commonhttp.WithHTTPStatusCodeAttribute. Never return plain fmt.Errorf at the package boundary. (`var ErrRateCardFeatureNotFound = models.NewValidationIssue(ErrCodeRateCardFeatureNotFound, "feature not found", models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**Discriminated union types with custom JSON** — Polymorphic types (RateCard, Price, EntitlementTemplate) use unexported type-switch structs with explicit MarshalJSON/UnmarshalJSON. Always read the type discriminator field first, then unmarshal the concrete variant. (`EntitlementTemplate.MarshalJSON embeds typed sub-structs with a 'type' field; UnmarshalJSON reads serde.Type then delegates to MeteredEntitlementTemplate / StaticEntitlementTemplate / BooleanEntitlementTemplate.`)
**EffectivePeriod status derivation** — Status (Draft/Active/Archived/Invalid) is always computed from EffectivePeriod.EffectiveFrom/EffectiveTo relative to clock.Now() or a supplied time — never stored as a separate field. Use StatusAt(t) for deterministic testing. (`AddonMeta.Status() calls StatusAt(clock.Now()); zero EffectivePeriod = Draft, EffectiveFrom set + EffectiveTo zero or future = Active.`)
**RateCards collection helpers** — RateCards ([]RateCard) exposes collection-level methods: SingleBillingCadence(), AsProductCatalogRateCards(), ValidateRateCards(). Use these helpers rather than ranging over the slice directly to preserve validation semantics. (`ValidateAddonHasSingleBillingCadence checks a.RateCards.SingleBillingCadence() before publishing.`)
**models.ErrorWithFieldPrefix for structured error paths** — Field-path context is always attached via models.ErrorWithFieldPrefix(models.NewFieldSelectorGroup(models.NewFieldSelector("field")), err) so that HTTP encoders can generate JSONPath selectors for client-side field highlighting. (`models.ErrorWithFieldPrefix(models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").WithExpression(models.WildCard)), ErrRateCardMultipleBillingCadence)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/productcatalog/errors.go` | Single source of truth for all productcatalog-level ValidationIssue sentinel errors. Every new validation error must be added here as an ErrCode const + var pair. | Forgetting commonhttp.WithHTTPStatusCodeAttribute on errors that need a non-500 HTTP mapping; using plain errors.New instead of models.NewValidationIssue. |
| `openmeter/productcatalog/ratecard.go` | Defines the RateCard interface (Type, AsMeta, Key, Merge, Clone, Compatible, GetBillingCadence, IsBillable) and RateCardMeta. All concrete rate card types (FlatFeeRateCard, UsageBasedRateCard) implement this interface. | Adding fields to RateCardMeta without updating Clone(); forgetting to add a case in the type-switch JSON serializer. |
| `openmeter/productcatalog/price.go` | Defines the Price discriminated union (FlatPrice, UnitPrice, TieredPrice, DynamicPrice, PackagePrice) with custom JSON and the PaymentTermType enum. | Adding a new PriceType without updating the pricer interface, MarshalJSON/UnmarshalJSON switch, and all downstream adapters that map Price entities. |
| `openmeter/productcatalog/entitlement.go` | Defines EntitlementTemplate discriminated union (Metered, Static, Boolean) with custom JSON. Used as a field of RateCardMeta. | Adding a new EntitlementType without updating all four switch statements in MarshalJSON, UnmarshalJSON, Validate, and Equal. |
| `openmeter/productcatalog/addon.go` | Defines AddonMeta, Addon, AddonInstanceType and their validators including Publishable() which enforces stricter rules than Validate(). | Calling Validate() where Publishable() is required; adding validators without registering them in ValidateWith chains. |
| `openmeter/productcatalog/effectiveperiod.go` | Shared EffectivePeriod type embedded by Plan and Addon. Governs Draft/Active/Archived status transitions. | Setting EffectivePeriod fields directly to alter status — only Publish/Archive operations should mutate these fields. |
| `openmeter/productcatalog/alignment.go` | Provides ValidateBillingCadencesAlign for checking that a rate card's billing cadence is divisible by (or equal to) the plan cadence. | Calling this with un-simplified ISODurations — always pass through ISODuration.Simplify(true) or rely on the validator helpers. |

## Anti-Patterns

- Returning plain fmt.Errorf or errors.New at the package boundary — always use models.NewValidationIssue sentinels or wrap with models.NewGenericValidationError.
- Directly mutating EffectivePeriod.EffectiveFrom/EffectiveTo outside of Publish/Archive operations — status is derived, not stored.
- Adding a new discriminated union variant (RateCardType, PriceType, EntitlementType) without updating MarshalJSON, UnmarshalJSON, Equal, Validate, and all downstream adapter type-switches.
- Importing app/common or any DI package from this package — it must remain a pure domain types package with no wiring dependencies.
- Calling RateCard.Compatible() or RateCard.Merge() without first verifying Type() equality — mismatched types cause panics or silent data loss.

## Decisions

- **ValidatorFunc[T] composition instead of inline Validate()** — Allows callers to run stricter validation (Publishable) by composing additional ValidatorFunc values on top of the base set, without duplicating logic or adding conditional branches inside Validate().
- **EffectivePeriod-based status derivation (no stored status field)** — Avoids status synchronization bugs: status is always consistent with the stored timestamps and requires no additional write when times are updated.
- **Discriminated union types with unexported fields and custom JSON** — Prevents accidental construction of zero-valued union types; forces callers through typed constructors (NewEntitlementTemplateFrom, NewPriceFrom) that set the discriminator correctly.

## Example: Adding a new ValidatorFunc and wiring it into Publishable validation

```
// productcatalog/addon.go
func ValidateAddonHasCompatiblePrices() models.ValidatorFunc[Addon] {
    return func(a Addon) error {
        switch a.InstanceType {
        case AddonInstanceTypeMultiple:
            for _, rc := range a.RateCards {
                if price := rc.AsMeta().Price; price != nil && price.Type() != FlatPriceType {
                    return models.ErrorWithFieldPrefix(
                        models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").
                            WithExpression(models.NewFieldAttrValue("key", rc.Key()))),
                        ErrAddonInvalidPriceForMultiInstance,
                    )
                }
            }
        }
// ...
```

<!-- archie:ai-end -->
