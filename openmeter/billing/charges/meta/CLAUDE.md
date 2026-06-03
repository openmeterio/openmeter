# meta

<!-- archie:ai-start -->

> Shared domain primitives for every charge type (flatfee, usagebased, creditpurchase): ChargeID, ChargeType, ChargeStatus, ManagedResource, the embedded Intent, Expands, the Patch interface (PatchDelete/Extend/Shrink backed by stateless.Trigger constants), and timestamp normalization. meta.Service is a type alias for meta.Adapter (a persistence-thin charge cross-reference registry, implemented in adapter/).

## Patterns

**Patch interface with stateless.Trigger** — Every patch implements Validate() + Op() PatchType + Trigger() stateless.Trigger + TriggerParams() any. Trigger constants live in triggers.go; never define trigger strings inline. (`func (p PatchExtend) Trigger() stateless.Trigger { return TriggerExtend }
func (p PatchExtend) TriggerParams() any { return p }`)
**NormalizeTimestamp on all time fields** — Timestamps in Intent and patches pass through NormalizeTimestamp (UTC + Truncate(streaming.MinimumWindowSizeDuration)) before persistence, via Intent.Normalized() or patch setters that normalize internally. (`i.ServicePeriod = NormalizeClosedPeriod(i.ServicePeriod)`)
**ValidateWith for period bounds checking** — PatchExtend/PatchShrink expose ValidateWith(intent Intent) to check new periods against the existing intent; always call it in addition to Validate() when the existing intent is available. (`if err := patch.ValidateWith(charge.Intent); err != nil { return err }`)
**ChargeID = models.NamespacedID alias** — ChargeID is a type alias of models.NamespacedID; validate with ChargeID.Validate() before reading Namespace/ID. (`func (i ChargeID) Validate() error { return models.NamespacedID(i).Validate() }`)
**Service = Adapter type alias** — meta.Service is a type alias for meta.Adapter — no separate service layer exists. If orchestration is ever needed, remove the alias and add a real service with transaction.Run. (`type Service = Adapter`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charge.go` | ChargeID, ChargeType, ChargeStatus, Expand/Expands, ChargeAccessor interface, Charge/Charges. ChargeAccessor requires GetChargeID/GetCustomerID/GetCurrency/ErrorAttributes. | ExpandNone is nil (not empty slice); every new charge type must implement ChargeAccessor. |
| `intent.go` | Shared Intent embedded in all charge intents; validates ManagedBy, CustomerID, Currency, three periods, TaxConfig, Subscription, UniqueReferenceID. | UniqueReferenceID must be nil or non-empty — empty string fails Validate(). |
| `timestamps.go` | NormalizeTimestamp/NormalizeOptionalTimestamp/NormalizeClosedPeriod/Intent.Normalized(). Truncates to streaming.MinimumWindowSizeDuration. | Truncation precision is streaming.MinimumWindowSizeDuration, NOT time.Microsecond (differs from lineage). |
| `triggers.go` | All stateless.Trigger constants shared across charge state machines. | Add new triggers here; never inline trigger strings in state-machine config files. |
| `patchdelete.go` | PatchDelete + PatchDeletePolicy (Credit/Invoice refund policies). RefundAsCreditsDeletePolicy is the recommended default. | Both refund policies must be set; a zero-value policy fails Validate(). |
| `patchextend.go / patchshrink.go` | Private-field patches with auto-normalizing setters and ValidateWith bounds checks; NewPatchExtend/NewPatchShrink constructors. | Use the constructors which validate before construction; setters normalize internally. |
| `adapter.go` | meta.Adapter: RegisterCharges/DeleteRegisteredCharge + TxCreator. Implemented by adapter/ via TransactingRepoWithNoValue with soft-delete. | Adapter is persistence-only; no business logic belongs here. |

## Anti-Patterns

- Defining charge-type-specific copies of Intent period fields instead of embedding meta.Intent.
- Calling NormalizeTimestamp with different precision than streaming.MinimumWindowSizeDuration.
- Adding complex business logic to meta — it is a shared primitives package; logic belongs in type-specific service packages.
- Using string literals for trigger constants instead of the vars in triggers.go.
- Creating a PatchDelete with only one of the two refund policies set — the other defaults to empty and fails Validate().

## Decisions

- **Service = Adapter type alias with no separate service layer.** — meta is a persistence-thin registry; the alias avoids premature abstraction until orchestration is needed.
- **Patch interface uses stateless.Trigger rather than a string enum.** — Compile-time type safety with the stateless library; callers pass trigger + params to Machine.FireAndActivate without a string switch.
- **All timestamps truncated to streaming.MinimumWindowSizeDuration.** — ClickHouse window aggregation needs consistent truncation granularity across all charge timestamps.

## Example: Create and apply a PatchExtend with ValidateWith

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"

patch, err := meta.NewPatchExtend(meta.NewPatchExtendInput{
  NewServicePeriodTo: newTo, NewFullServicePeriodTo: newFullTo,
  NewBillingPeriodTo: newBillingTo, NewInvoiceAt: newInvoiceAt,
})
if err != nil { return err }
if err := patch.ValidateWith(charge.Intent); err != nil { return err }
err = machine.FireAndActivate(ctx, patch.Trigger(), patch.TriggerParams())
```

<!-- archie:ai-end -->
