# productcatalog

<!-- archie:ai-start -->

> Root domain package for the product catalog and the shared type contract between billing, entitlement, and subscription. It owns the polymorphic value types (RateCard, Price, Discount, EntitlementTemplate, EffectivePeriod, Phase, PlanAddon, ProRatingConfig) and their validators; sub-packages split each entity (feature, plan, addon, planaddon) into the standard domain / adapter / driver / service three-layer shape, plus bridge packages (featureresolver, subscription) that break import cycles.

## Patterns

**Root holds shared value types; sub-packages hold entity aggregates** — The package root (price.go, ratecard.go, entitlement.go, effectiveperiod.go, errors.go) defines the polymorphic value types and ValidationIssue sentinels reused by every sub-package. Entity lifecycle (feature, plan, addon, planaddon) lives one level down, each with adapter/driver/service children. (`productcatalog.RateCard / productcatalog.Price consumed by addon.RateCard and plan.RateCard which wrap them with managed identity (NamespacedID, ManagedModel).`)
**Three-layer per entity: domain root + adapter + httpdriver + service** — Each entity sub-package defines Service+Repository interfaces and aggregate types at its root; an adapter/ child for Ent persistence (TransactingRepo + Tx/WithTx/Self triad), an httpdriver/ child for v1 HTTP, and a service/ child for validation + feature/taxcode resolution + event publishing. (`openmeter/productcatalog/plan/{plan.go,service.go} + plan/adapter + plan/httpdriver + plan/service`)
**Validation and event publishing in the service/connector layer only** — Adapters do pure persistence; all input validation, feature/taxcode resolution, and Watermill event publishing happen in the service (or featureConnector) layer, with events published inside the transaction.Run closure so a publish failure rolls back the DB write. (`feature.featureConnector wraps feature.FeatureRepo; addon/plan service resolve features+taxcodes then publish events inside transaction.Run.`)
**EffectivePeriod-derived status, gated to Publish/Archive** — Draft/Active/Archived status is computed from EffectivePeriod relative to clock.Now(), never stored. EffectivePeriod fields are zeroed in UpdatePlan and may only change via Publish/Archive. (`AddonMeta.Status() = StatusAt(clock.Now()); plan service zeroes EffectivePeriod in UpdatePlan.`)
**Bridge packages break import direction** — featureresolver adapts feature.FeatureConnector into productcatalog.FeatureResolver so plan/addon code never imports feature directly; the subscription/ bridge (package plansubscription) routes all subscription writes through subscriptionworkflow.Service, never subscription.Service. (`ResolveFeaturesForRateCards aggregates field-prefixed errors; PlanSubscriptionService delegates persistence to WorkflowService.`)
**Discriminated unions with custom JSON + ValidationIssue sentinels** — Polymorphic types (RateCard, Price, EntitlementTemplate) read a type discriminator then unmarshal the concrete variant; all package-boundary errors are package-level models.ValidationIssue vars with ErrCode pairs and commonhttp.WithHTTPStatusCodeAttribute, never plain fmt.Errorf. (`EntitlementTemplate.UnmarshalJSON reads serde.Type then delegates; ErrRateCardFeatureNotFound = models.NewValidationIssue(...).`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | Single source of truth for productcatalog-level ValidationIssue sentinels (ErrCode const + var pairs). | Forgetting commonhttp.WithHTTPStatusCodeAttribute for non-500 mappings; using errors.New. |
| `ratecard.go` | RateCard interface (Type, AsMeta, Key, Merge, Clone, Compatible, GetBillingCadence, IsBillable) + RateCardMeta. | Adding RateCardMeta fields without updating Clone(); missing a JSON type-switch case. |
| `price.go` | Price discriminated union (FlatPrice, UnitPrice, TieredPrice, DynamicPrice, PackagePrice) + PaymentTermType. | Adding a PriceType without updating MarshalJSON/UnmarshalJSON and all downstream adapter mappings. |
| `entitlement.go` | EntitlementTemplate union (Metered, Static, Boolean) used as a RateCardMeta field. | Adding an EntitlementType without updating all four switches (MarshalJSON, UnmarshalJSON, Validate, Equal). |
| `effectiveperiod.go` | Shared EffectivePeriod embedded by Plan and Addon, governs status derivation. | Mutating EffectiveFrom/EffectiveTo outside Publish/Archive — status is derived, not stored. |
| `featureresolver.go / featureresolver/` | Breaks the productcatalog→feature import cycle by resolving rate-card feature refs. | Calling feature.FeatureConnector directly from plan/addon code instead of through the resolver. |

## Anti-Patterns

- Returning plain fmt.Errorf/errors.New at a package boundary instead of models.NewValidationIssue sentinels or models.NewGeneric*Error.
- Adding a discriminated-union variant (RateCardType, PriceType, EntitlementType) without updating MarshalJSON, UnmarshalJSON, Equal, Validate, and all downstream adapter type-switches.
- Putting validation or event publishing inside an entity adapter — it belongs in the service/connector layer.
- Mutating EffectivePeriod directly to change status, or setting it via UpdatePlan instead of Publish/Archive.
- Importing app/common (or any DI package) from this domain or its testutils — breaks the leaf-node import direction and causes cycles.

## Decisions

- **ValidatorFunc[T] composition instead of inline Validate() logic.** — Lets Publishable() layer stricter validators on the base set without conditional branches inside Validate().
- **EffectivePeriod-derived status with no stored status field.** — Status stays consistent with stored timestamps and needs no extra write, avoiding sync bugs.
- **Bridge packages (featureresolver, subscription) instead of direct cross-package imports.** — Breaks import cycles between productcatalog, feature, and subscription while keeping persistence routed through the workflow layer.

## Example: Adding a ValidatorFunc and composing it into Publishable validation

```
// productcatalog/addon.go
func ValidateAddonHasCompatiblePrices() models.ValidatorFunc[Addon] {
    return func(a Addon) error {
        for _, rc := range a.RateCards {
            if p := rc.AsMeta().Price; p != nil && p.Type() != FlatPriceType {
                return models.ErrorWithFieldPrefix(
                    models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").
                        WithExpression(models.NewFieldAttrValue("key", rc.Key()))),
                    ErrAddonInvalidPriceForMultiInstance,
                )
            }
        }
        return nil
    }
}
```

<!-- archie:ai-end -->
