# meta

<!-- archie:ai-start -->

> Shared domain primitives for all charge types: ChargeID, ChargeType, ChargeStatus, ManagedResource, Intent (embedded in every charge intent), Expands, Patch interface (PatchDelete/Extend/Shrink backed by stateless.Trigger constants), and timestamp normalization utilities. Service is a type alias for Adapter — no separate service layer exists.

## Patterns

**Patch interface with stateless.Trigger** — Every patch type implements Patch: Validate() + Op() PatchType + Trigger() stateless.Trigger + TriggerParams() any. Trigger constants live in triggers.go. New patch types must follow this structure; never define trigger strings inline. (`type PatchExtend struct { ... }
func (p PatchExtend) Trigger() stateless.Trigger { return TriggerExtend }
func (p PatchExtend) TriggerParams() any { return p }`)
**NormalizeTimestamp on all time.Time fields** — All timestamps stored in Intent and patches must pass through NormalizeTimestamp (UTC + Truncate(streaming.MinimumWindowSizeDuration)) before persistence. Call Intent.Normalized() or patch setter methods which call NormalizeTimestamp internally. (`i.ServicePeriod = NormalizeClosedPeriod(i.ServicePeriod)`)
**ValidateWith for period bounds checking** — PatchExtend and PatchShrink provide ValidateWith(intent Intent) to check new periods against existing intent periods. Always call ValidateWith in addition to Validate() when the existing intent is available. (`if err := patch.ValidateWith(charge.Intent); err != nil { return err }`)
**ChargeID = models.NamespacedID alias** — ChargeID is a type alias of models.NamespacedID; validate with ChargeID.Validate() which delegates to NamespacedID.Validate(). Never access Namespace or ID fields without first calling Validate(). (`func (i ChargeID) Validate() error { return models.NamespacedID(i).Validate() }`)
**Service = Adapter type alias** — meta.Service is a type alias for meta.Adapter — no separate service implementation exists. If complex orchestration is ever needed, remove this alias and add a proper service layer with transaction.Run. (`type Service = Adapter`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charge.go` | ChargeID, ChargeType, ChargeStatus, Expand, Expands, ChargeAccessor interface, Charge struct, Charges slice. ChargeAccessor requires GetChargeID(), GetCustomerID(), GetCurrency(), ErrorAttributes(). | ExpandNone is nil (not empty slice); every new charge type must implement ChargeAccessor. |
| `intent.go` | Shared Intent struct embedded in all charge intents. Validates ManagedBy, CustomerID, Currency, three periods, TaxConfig, Subscription, UniqueReferenceID. | UniqueReferenceID must be nil or non-empty string — empty string fails Validate(). |
| `patchdelete.go` | PatchDelete + PatchDeletePolicy (CreditRefundPolicy + InvoiceRefundPolicy). RefundAsCreditsDeletePolicy is the recommended default. | Always set both refund policies; leaving either at zero value fails Validate(). |
| `timestamps.go` | NormalizeTimestamp, NormalizeOptionalTimestamp, NormalizeClosedPeriod, Intent.Normalized(). Truncates to streaming.MinimumWindowSizeDuration. | Truncation precision is streaming.MinimumWindowSizeDuration, NOT time.Microsecond — different from the lineage package. |
| `triggers.go` | All stateless.Trigger constants shared across charge state machines: TriggerNext, TriggerPartialInvoiceCreated, TriggerFinalInvoiceCreated, etc. | New triggers must be added here; never define trigger strings inline in state machine configuration files. |
| `patchextend.go / patchshrink.go` | PatchExtend and PatchShrink with private fields, typed setters that auto-normalize, and ValidateWith(intent Intent) for bounds checking. | All setters call NormalizeTimestamp internally; use NewPatchExtend/NewPatchShrink constructors which validate before construction. |

## Anti-Patterns

- Defining charge-type-specific copies of Intent period fields instead of embedding meta.Intent.
- Calling NormalizeTimestamp with different truncation precision than streaming.MinimumWindowSizeDuration.
- Adding complex business logic to the meta package — it is a shared primitives package; logic belongs in type-specific service packages.
- Using string literals for trigger constants instead of the vars in triggers.go.
- Creating a PatchDelete with only one of the two refund policies set — the other defaults to empty string and fails Validate().

## Decisions

- **Service = Adapter type alias with no separate service layer** — meta is a persistence-thin registry; until orchestration is needed here, the alias avoids premature abstraction.
- **Patch interface uses stateless.Trigger rather than a string enum** — Compile-time type safety with the stateless library; callers pass trigger + params to Machine.FireAndActivate without a string-switch.
- **All timestamps truncated to streaming.MinimumWindowSizeDuration** — ClickHouse window aggregation requires consistent truncation granularity across all charge-related timestamps.

## Example: Create and apply a PatchExtend with ValidateWith

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"

patch, err := meta.NewPatchExtend(meta.NewPatchExtendInput{
    NewServicePeriodTo:     newTo,
    NewFullServicePeriodTo: newFullTo,
    NewBillingPeriodTo:     newBillingTo,
    NewInvoiceAt:           newInvoiceAt,
})
if err != nil { return err }
if err := patch.ValidateWith(charge.Intent); err != nil { return err }
// Fire through state machine:
err = machine.FireAndActivate(ctx, patch.Trigger(), patch.TriggerParams())
```

<!-- archie:ai-end -->
