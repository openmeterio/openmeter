# addon

<!-- archie:ai-start -->

> Domain package for subscription-addon lifecycle: defines the core types (SubscriptionAddon, SubscriptionAddonInstance, SubscriptionAddonQuantity, SubscriptionAddonRateCard), the Service/Repository interfaces, and domain events; its sub-packages split into diff (pure Apply/Restore spec transform), repo (Ent persistence), http (v1 handlers), and service (validation + event publishing). Primary constraint: quantities are an append-only timeline and instances are derived, never persisted.

## Patterns

**Append-only quantity timeline** — SubscriptionAddonQuantity rows are never updated or deleted; GetInstances() derives SubscriptionAddonInstance values by pairing adjacent open periods from the Timeline after filtering by DeletedAt. The repo child has no quantity Update/Delete by design. (`if a.DeletedAt != nil { quantities = quantities.Before(*a.DeletedAt) }; periods := quantities.GetOpenPeriods()[1:]`)
**Apply/Restore split from the diff layer** — extend.go owns single-RateCard additive (Apply) / subtractive (Restore) mutation; the diff/ child orchestrates multi-item spec-level invertible application. Keep DB/service logic out of the pure diff transform. (`rc.Apply(targetRateCardPtr, annotations) // target must be a non-nil *FlatFeeRateCard or *UsageBasedRateCard`)
**Input Validate() with quantity asymmetry** — All input structs implement Validate() via errors.Join. CreateSubscriptionAddonInput rejects InitialQuantity==0; CreateSubscriptionAddonQuantityInput allows 0 (signals removal). (`if err := inp.InitialQuantity.Validate(); err != nil { return fmt.Errorf("initialQuantity: %w", err) }`)
**Events via constructors, published inside the transaction** — Use NewCreatedEvent/NewChangeQuantityEvent to capture session.UserID from ctx (never construct event structs directly); the service child publishes them inside transaction.Run so DB write and event stay consistent. (`event := subscriptionaddon.NewCreatedEvent(ctx, customer, subscriptionAddon)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `addon.go` | Core types plus GetInstances()/GetInstanceAt() derivation logic. | GetInstances() skips the zeroth sentinel period (periods = periods[1:]); any Timeline change breaks instance derivation. |
| `extend.go` | Apply/Restore additive and subtractive RateCard mutations. | Both validate target is a non-nil pointer with matching key/cadence/entitlement type; Restore nils out SingleInstance price by instanceType. |
| `quantity.go` | SubscriptionAddonQuantity and its CreateSubscriptionAddonQuantityInput.Validate(). | Quantity 0 is valid here (removal) but rejected by CreateSubscriptionAddonInput for initial creation. |
| `events.go` | CreatedEvent / ChangeQuantityEvent implementing marshaler.Event. | EventName constants are Watermill routing keys; EventSubsystem='subscriptionaddon' routes to SystemEventsTopic — keep stable. |
| `repository.go` | SubscriptionAddonRepository (Create/Get/List) and quantity repository (Create-only) interfaces. | Quantity repository has no Update/Delete by design — quantities are immutable. |
| `instance.go` | SubscriptionAddonInstance: derived addon+quantity view for a period. | Purely derived — never persist directly. |

## Anti-Patterns

- Calling Apply or Restore with a nil target or nil annotations — both return explicit errors.
- Mutating SubscriptionAddonQuantity rows after creation — they are immutable append-only records.
- Setting InitialQuantity=0 on CreateSubscriptionAddonInput — only ChangeQuantity may use 0.
- Deriving instance periods manually instead of GetInstances() — DeletedAt truncation is non-trivial.
- Reading/writing annotation keys by string literal instead of via subscription.AnnotationParser.

## Decisions

- **Quantities stored as an append-only timeline.** — Preserves full audit history of quantity changes and enables point-in-time instance derivation without mutable state.
- **Apply/Restore (extend.go) separated from the diff sub-package.** — extend.go owns single-RateCard mutation; diff/ orchestrates multi-item spec application, keeping the pure in-memory diff layer free of business logic.
- **Events published inside the transaction in the service sub-package.** — Ensures DB write and event publish are consistent; rollback prevents orphaned events.

## Example: Apply an addon rate card to a target rate card

```
import (
    subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
    "github.com/openmeterio/openmeter/pkg/models"
)

rc := subscriptionaddon.SubscriptionAddonRateCard{AddonRateCard: addonRC}
if err := rc.Apply(targetRateCardPtr, models.Annotations{}); err != nil {
    return fmt.Errorf("apply addon rate card: %w", err)
}
```

<!-- archie:ai-end -->
