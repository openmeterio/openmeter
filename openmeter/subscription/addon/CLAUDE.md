# addon

<!-- archie:ai-start -->

> Subscription-addon sub-domain (package subscriptionaddon): models a 1:1 SubscriptionAddon (subscription + productcatalog Addon) plus its append-only quantity timeline, and the rate-card Apply/Restore math that folds addon prices/entitlements/discounts onto a target RateCard. Root holds domain types + events; diff/, repo/, service/, http/ provide the engine, persistence, service, and transport layers.

## Patterns

**Quantity as a timeutil.Timeline** — Quantities is a timeutil.Timeline[SubscriptionAddonQuantity]; GetInstances() derives SubscriptionAddonInstance segments from open periods, never from stored end times. (`periods := quantities.GetOpenPeriods(); periods = periods[1:]; cad, _ := models.NewCadencedModelFromPeriod(period)`)
**Reversible Apply/Restore rate-card math** — SubscriptionAddonRateCard.Apply adds price/entitlement/discount onto a target RateCard; Restore subtracts the exact same amounts. They must stay inverse and validate compatibility first. (`productcatalog.NewRateCardWithOverlay(a.AddonRateCard.RateCard, target).ValidateWith(ValidateRateCardsShareSameKey, ...)`)
**ChangeMeta-only mutation of rate cards** — All rate-card edits go through target.ChangeMeta(func(m RateCardMeta) (RateCardMeta, error){...}); target must be a non-nil pointer (checked via reflect) and annotations must be non-nil. (`return target.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) { ... })`)
**Boolean-entitlement counting via AnnotationParser** — Stacked boolean entitlements are tracked by an annotation count (subscription.AnnotationParser Get/SetBooleanEntitlementCount); metered entitlements sum/subtract IssueAfterReset treating nil as 0. (`count := subscription.AnnotationParser.GetBooleanEntitlementCount(annotations); annotations, _ = subscription.AnnotationParser.SetBooleanEntitlementCount(annotations, count+1)`)
**Validate() collecting errors.Join** — Input Validate() methods append to var errs []error and return errors.Join(errs...) (or models.NewNillableGenericValidationError), never first-error-return. (`if i.AddonID == "" { errs = append(errs, errors.New("addonID is required")) }; return errors.Join(errs...)`)
**Customer-scoped marshaler.Event pair** — CreatedEvent and ChangeQuantityEvent are aliases of a shared event struct; Source is the subscriptionAddon resource path, Subject is the customer path, UserID from session.GetSessionUserID(ctx). (`Source: metadata.ComposeResourcePath(ns, metadata.EntitySubscriptionAddon, id), Subject: metadata.ComposeResourcePath(ns, metadata.EntityCustomer, customer.ID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `addon.go` | SubscriptionAddon aggregate + GetInstances/GetInstanceAt deriving instances from the quantity timeline; CreateSubscriptionAddonInput.Validate. | Deleted addons truncate quantities via .Before(*DeletedAt); the several 'this should never happen' guards return empty rather than panicking. |
| `extend.go` | SubscriptionAddonRateCard.Apply / Restore — the core price/entitlement/discount fold and unfold math. | Apply and Restore must remain exact inverses; Restore guards against negative flat amounts/IssueAfterReset; FeatureID validation intentionally disabled (FIXME OM-1337). |
| `quantity.go` | SubscriptionAddonQuantity (ActiveFrom+Quantity) and AsTimed() adapter into timeutil.Timed. | Quantity may be 0 in a segment but CreateSubscriptionAddonInput rejects an initial quantity of 0. |
| `instance.go` | SubscriptionAddonInstance — virtual effective addon for a CadencedModel period, embeds CadencedModel. | Instances are computed, never persisted; merging quantity with Addon happens in GetInstances. |
| `events.go` | CreatedEvent / ChangeQuantityEvent definitions and constructors. | Event names are versioned (subscriptionaddon.created v1); changing them breaks the consumer contract. |
| `repository.go` | SubscriptionAddonRepository + SubscriptionAddonQuantityRepository interfaces and their input types. | Quantity repo only has Create — quantity is append-only, there is no Update/Delete. |
| `service.go` | Service interface (Create/Get/List/ChangeQuantity), OrderBy enum, list/get input validation. | ListSubscriptionAddonsInput requires SubscriptionID; GetSubscriptionAddonInput supports lookup by ID or by subscription+addon. |
| `ratecard.go` | Thin wrapper of addon.RateCard as SubscriptionAddonRateCard. | All rate-card behavior lives in extend.go, not here. |

## Anti-Patterns

- Mutating a RateCard outside target.ChangeMeta or passing a non-pointer / nil annotations (Apply/Restore reject via reflect checks).
- Making Apply and Restore non-inverse — restoring must subtract exactly what apply added and error on negative results.
- Updating or deleting a quantity row instead of appending a new (ActiveFrom, Quantity) segment.
- Deriving instance end times from stored fields instead of timeline open periods (GetInstances).
- Indexing the boolean-entitlement count with raw annotation keys instead of subscription.AnnotationParser.

## Decisions

- **Quantity is an append-only timeline; effective instances are computed from open periods at read time.** — Preserves full history for billing/proration and avoids destructive in-place edits.
- **Addon effects on rate cards are expressed as a reversible Apply/Restore pair.** — Lets the workflow layer re-sync the full before/after addon set via diffs and cleanly undo an addon.
- **Stacked boolean entitlements counted via annotations rather than duplicate templates.** — A rate card can hold only one entitlement template, so count tracks how many addons contributed it.

## Example: Folding an addon's flat price onto a target rate card

```
return target.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
    aMeta := a.AddonRateCard.AsMeta()
    tMeta := m.Clone()
    if aMeta.Price != nil && tMeta.Price.Type() == productcatalog.FlatPriceType {
        tFlat, _ := tMeta.Price.AsFlat()
        aFlat, _ := aMeta.Price.AsFlat()
        m.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: tFlat.Amount.Add(aFlat.Amount), PaymentTerm: tFlat.PaymentTerm})
    }
    return m, nil
})
```

<!-- archie:ai-end -->
