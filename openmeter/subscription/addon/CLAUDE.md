# addon

<!-- archie:ai-start -->

> Domain package for subscription addon lifecycle: core types (SubscriptionAddon, SubscriptionAddonInstance, SubscriptionAddonQuantity), the Service interface, domain events, and the Apply/Restore RateCard mutation logic. Primary constraint: quantities are an append-only timeline; instances are derived by pairing adjacent open periods.

## Patterns

**Append-only quantity timeline** — SubscriptionAddonQuantity rows are never updated or deleted. GetInstances() derives SubscriptionAddonInstance values by pairing adjacent open periods from the Timeline after filtering by DeletedAt. (`sa.Quantities = timeutil.NewTimeline([]timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity]{q1.AsTimed(), q2.AsTimed()})`)
**Apply/Restore RateCard mutation** — SubscriptionAddonRateCard.Apply adds price/entitlement/discount deltas to a target RateCard pointer; Restore subtracts them. Both require a non-nil pointer target and non-nil annotations map. (`rc.Apply(targetRateCardPtr, annotations) // target must be *productcatalog.FlatFeeRateCard or *UsageBasedRateCard`)
**Input type Validate() pattern** — All input structs implement Validate() returning errors.Join result. CreateSubscriptionAddonInput.Validate() rejects quantity==0; CreateSubscriptionAddonQuantityInput.Validate() allows 0. (`if err := inp.InitialQuantity.Validate(); err != nil { return fmt.Errorf("initialQuantity: %w", err) }`)
**Event construction via constructor functions** — Domain events implement marshaler.Event. Use NewCreatedEvent/NewChangeQuantityEvent constructors to capture session.UserID from ctx; never construct CreatedEvent{} directly. (`event := subscriptionaddon.NewCreatedEvent(ctx, customer, subscriptionAddon)`)
**GetInstances DeletedAt truncation** — If SubscriptionAddon.DeletedAt is set, GetInstances() filters quantities via timeline.Before(*deletedAt) before pairing periods — ensuring soft-deleted addons only surface active instances. (`if a.DeletedAt != nil { quantities = quantities.Before(*a.DeletedAt) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `addon.go` | Core domain types (SubscriptionAddon, CreateSubscriptionAddonInput) and GetInstances()/GetInstanceAt() derivation logic. | GetInstances() skips the zeroth sentinel period (periods = periods[1:]); any Timeline semantics change breaks instance derivation. |
| `extend.go` | Apply and Restore methods on SubscriptionAddonRateCard — additive/subtractive price, entitlement, and discount mutations. | Restore checks instanceType to nil out SingleInstance price; Apply/Restore both validate target is a non-nil pointer with matching key/cadence/entitlement type. |
| `quantity.go` | SubscriptionAddonQuantity type and CreateSubscriptionAddonQuantityInput with Validate(). | Quantity 0 is valid for CreateSubscriptionAddonQuantityInput (signals removal) but CreateSubscriptionAddonInput rejects quantity==0 for initial creation. |
| `events.go` | CreatedEvent and ChangeQuantityEvent implementing marshaler.Event. | EventName constants must stay stable — they are Watermill routing keys. EventSubsystem = 'subscriptionaddon' routes to SystemEventsTopic. |
| `repository.go` | SubscriptionAddonRepository (Create/Get/List) and SubscriptionAddonQuantityRepository (Create-only) interfaces. | SubscriptionAddonQuantityRepository has no Update or Delete method by design — quantities are immutable. |
| `service.go` | Service interface: Create, Get, List, ChangeQuantity and ListSubscriptionAddonsInput with Validate(). | ChangeQuantity appends a new quantity row; it does not update the previous one. |
| `instance.go` | SubscriptionAddonInstance: the virtual derived view merging addon+quantity for a time period. | Instance is purely derived; never persist it directly. |

## Anti-Patterns

- Calling SubscriptionAddonRateCard.Apply or Restore with a nil target or nil annotations — both return explicit errors.
- Mutating SubscriptionAddonQuantity rows after creation — they are immutable append-only records.
- Setting Quantity=0 on CreateSubscriptionAddonInput.InitialQuantity — Validate() returns an error; 0-quantity is only valid for subsequent ChangeQuantity calls.
- Deriving instance periods manually instead of using GetInstances() — the DeletedAt truncation logic is non-trivial.
- Reading or writing annotation map keys by string literal instead of via subscription.AnnotationParser.

## Decisions

- **Quantities stored as append-only timeline** — Preserves full audit history of quantity changes and enables point-in-time instance derivation without mutable state.
- **Apply/Restore separated from diff sub-package** — extend.go owns single-RateCard mutation semantics; diff/ orchestrates multi-item spec-level application, preventing the pure in-memory diff layer from gaining business-logic dependencies.
- **Events published inside transaction (in service sub-package)** — Ensures DB write and event publish are consistent; rollback prevents orphaned events.

## Example: Apply a boolean-entitlement addon rate card to a target rate card

```
import (
    subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
    "github.com/openmeterio/openmeter/pkg/models"
)

rc := subscriptionaddon.SubscriptionAddonRateCard{AddonRateCard: addonRC}
annotations := models.Annotations{}
if err := rc.Apply(targetRateCardPtr, annotations); err != nil {
    return fmt.Errorf("apply addon rate card: %w", err)
}
```

<!-- archie:ai-end -->
