# addon

<!-- archie:ai-start -->

> Manages subscription addon lifecycle: domain types (SubscriptionAddon, SubscriptionAddonInstance, SubscriptionAddonQuantity), the Service interface, events (CreatedEvent, ChangeQuantityEvent), and the RateCard extend/restore logic. Primary constraint: quantities are an append-only timeline; instances are derived by pairing adjacent quantity entries into open periods.

## Patterns

**Append-only quantity timeline** — SubscriptionAddonQuantity rows are never updated or deleted. Quantities are stored as a timeutil.Timeline and GetInstances() derives SubscriptionAddonInstance values by pairing adjacent open periods. (`sa.Quantities = timeutil.NewTimeline([]timeutil.Timed[SubscriptionAddonQuantity]{q1.AsTimed(), q2.AsTimed()})`)
**Apply/Restore RateCard mutation** — SubscriptionAddonRateCard.Apply adds price/entitlement/discount deltas to a target RateCard pointer; Restore subtracts them. Both require a non-nil pointer target and non-nil annotations map. Restore uses instanceType to decide whether to nil out a SingleInstance price. (`rc.Apply(target, annotations) // target must be *productcatalog.FlatFeeRateCard or *UsageBasedRateCard`)
**CreatedEvent / ChangeQuantityEvent via marshaler.Event** — Domain events implement marshaler.Event (EventName() + EventMetadata() + Validate()). Event names follow the pattern io.openmeter.subscriptionaddon.v1.subscriptionaddon.<action>. Use NewCreatedEvent / NewChangeQuantityEvent constructors to capture session.UserID from ctx. (`event := subscriptionaddon.NewCreatedEvent(ctx, customer, subscriptionAddon)`)
**Input type Validate() method** — All input structs (CreateSubscriptionAddonInput, CreateSubscriptionAddonQuantityInput) implement Validate() returning a joined errors.Join error. Call Validate() before passing to service. (`if err := inp.InitialQuantity.Validate(); err != nil { return fmt.Errorf("initialQuantity: %w", err) }`)
**GetInstances() derives instances from DeletedAt truncation** — If SubscriptionAddon.DeletedAt is set, GetInstances() first filters quantities via timeline.Before(deletedAt) before pairing periods, ensuring soft-deleted addons surface only active instances. (`if a.DeletedAt != nil { quantities = quantities.Before(*a.DeletedAt) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `addon.go` | Core domain types (SubscriptionAddon, CreateSubscriptionAddonInput) and GetInstances() / GetInstanceAt() derivation logic. | GetInstances() relies on GetOpenPeriods() having exactly len(quantities) periods after the zeroth sentinel — any change to Timeline semantics breaks instance derivation. |
| `instance.go` | SubscriptionAddonInstance: the virtual merged view of addon + quantity for a time period. | Instance is purely derived; never persist it directly. |
| `quantity.go` | SubscriptionAddonQuantity type and CreateSubscriptionAddonQuantityInput with Validate(). | Quantity 0 is valid for CreateSubscriptionAddonQuantityInput (used to signal removal) but CreateSubscriptionAddonInput rejects quantity==0 for initial creation. |
| `extend.go` | Apply and Restore methods on SubscriptionAddonRateCard — additive/subtractive price, entitlement, and discount mutations. | Apply/Restore must receive a pointer RateCard and non-nil annotations. Restoring a flat price to negative or entitlement count to negative is an error. |
| `events.go` | CreatedEvent and ChangeQuantityEvent implementing marshaler.Event. | EventName must stay stable — it is a Watermill routing key. |
| `repository.go` | SubscriptionAddonRepository and SubscriptionAddonQuantityRepository interfaces used by service and repo sub-packages. | SubscriptionAddonQuantityRepository.Create only — there is no Update or Delete method by design. |
| `service.go` | Service interface: Create, Get, List, ChangeQuantity. | ChangeQuantity appends a new quantity row; it does not update the previous one. |

## Anti-Patterns

- Calling SubscriptionAddonRateCard.Apply or Restore with a nil target or nil annotations — both are rejected with explicit errors.
- Mutating SubscriptionAddonQuantity rows after creation — they are immutable append-only records.
- Setting Quantity=0 on CreateSubscriptionAddonInput.InitialQuantity — Validate() returns an error; 0-quantity is only valid for subsequent ChangeQuantity calls.
- Reading annotation map keys by string literal instead of via subscription.AnnotationParser.
- Deriving instance periods manually instead of using GetInstances() — the derivation logic with DeletedAt truncation is non-trivial.

## Decisions

- **Quantities stored as append-only timeline** — Preserves full audit history of quantity changes and enables point-in-time instance derivation without mutable state.
- **Apply/Restore separated from diff sub-package** — extend.go owns the single-RateCard mutation semantics; diff/ orchestrates multi-item spec-level application. Keeping them separate prevents the pure in-memory diff layer from gaining business-logic dependencies.
- **Events published inside transaction (in service sub-package)** — Ensures DB write and event publish are consistent; rollback prevents orphaned events.

## Example: Add a boolean-entitlement addon rate card to a target rate card

```
import (
    subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
    "github.com/openmeterio/openmeter/openmeter/productcatalog"
    "github.com/openmeterio/openmeter/pkg/models"
)

rc := subscriptionaddon.SubscriptionAddonRateCard{AddonRateCard: addonRC}
annotations := models.Annotations{}
if err := rc.Apply(targetRateCardPtr, annotations); err != nil {
    return fmt.Errorf("apply addon rate card: %w", err)
}
```

<!-- archie:ai-end -->
