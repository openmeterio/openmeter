# meta

<!-- archie:ai-start -->

> Shared charge-meta package: defines the cross-type Charge/Intent/ChargeID/ChargeType/ChargeStatus value types, the registration Adapter interface, the period-patch system (extend/shrink/delete) built on qmuntal/stateless triggers, and timestamp normalization. It is the most depended-on charges package (39 in-edges) — every charge type embeds meta.Intent and registers via meta.Adapter.

## Patterns

**Service is an alias of Adapter** — service.go aliases `type Service = Adapter` with a comment: no transaction layer is needed because this package is a thin DB wrapper; add transaction.Run only if that changes (`type Service = Adapter`)
**Patch interface over stateless triggers** — Each patch (PatchExtend/PatchShrink/PatchDelete) implements Patch{ Op() PatchType; Trigger() stateless.Trigger; TriggerParams() any } with a package-level Trigger* = stateless.Trigger and a `var _ Patch = (*PatchX)(nil)` assertion (`TriggerExtend = stateless.Trigger("extend")`)
**Period patches: private fields, NormalizeTimestamp setters, Validate + ValidateWith** — PatchExtend/PatchShrink store private time fields set through Set*/normalized via NormalizeTimestamp; Validate() checks non-zero, ValidateWith(intent) enforces directional ordering against the existing periods (`PatchExtend.ValidateWith requires NewServicePeriodTo.After(intent.ServicePeriod.To)`)
**Enum types with Values()+Validate()+slices.Contains** — ChargeType, ChargeStatus, CreditRefundPolicy, InvoiceRefundPolicy each expose Values() and Validate() that error via models.NewGenericValidationError when not in Values() (`ChargeType.Validate uses slices.Contains(t.Values(), string(t))`)
**Expands via pkg/expand** — Expands = expand.Expand[Expand]; Expand constants (ExpandRealizations, ExpandRealtimeUsage, ExpandDetailedLines, ExpandDeletedRealizations) gate which edges adapters load (`input.Expands.Has(meta.ExpandRealizations)`)
**Timestamp normalization to streaming window** — NormalizeTimestamp truncates to UTC streaming.MinimumWindowSizeDuration; NormalizeClosedPeriod and Intent.Normalized() apply it to all period boundaries; patches normalize on Set* (`t.UTC().Truncate(streaming.MinimumWindowSizeDuration)`)
**ChargeAccessor interface** — Per-type charges satisfy meta.ChargeAccessor (GetChargeID/GetCustomerID/GetCurrency/ErrorAttributes) so generic engine code can operate over any charge type (`var _ meta.ChargeAccessor = (*Charge)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charge.go` | ChargeID/ChargeIDs, ChargeType, ChargeStatus, Expand, ChargeAccessor, and the generic Charge/Charges types | ChargeType drives the FK column in adapters; ChargeStatus semantics (created/active/final/deleted) — final means no further late events arrive |
| `intent.go` | Shared Intent with the three periods (Service/FullService/Billing), ManagedBy, tax config, and optional Subscription reference | ManagedBy is validated against billing.InvoiceLineManagedBy values; UniqueReferenceID must not be empty-string when set |
| `patch.go` | Patch interface, PatchType enum, and TriggerPatchResult carrying invoiceupdater.Patch side effects | TriggerParams() is `any` because stateless requires it; results carry InvoicePatches to feed invoiceupdater |
| `patchextend.go / patchshrink.go` | Period-extension/shrink patches with directional ValidateWith ordering rules | Extend requires periods grow (service strictly after); shrink requires service strictly before existing-to and after existing-from; both call NormalizeTimestamp on every setter |
| `patchdelete.go` | PatchDelete + PatchDeletePolicy (CreditRefundPolicy + InvoiceRefundPolicy) with RefundAsCreditsDeletePolicy default | Refund behavior is policy-driven (correct/ignore for credits; refund/grant_credits/ignore for invoices), not hardcoded |
| `adapter.go` | Adapter interface (RegisterCharges/DeleteRegisteredCharge + entutils.TxCreator) and RegisterChargesInput | DeleteRegisteredChargeInput is an alias of ChargeID; RegisterChargesInput requires a valid ChargeType and per-charge IDs |
| `timestamps.go` | NormalizeTimestamp/NormalizeOptionalTimestamp/NormalizeClosedPeriod and Intent.Normalized() | Truncation uses streaming.MinimumWindowSizeDuration — do not substitute a different granularity |
| `triggers.go` | Shared lifecycle triggers (TriggerNext, TriggerPartialInvoiceCreated, TriggerFinalInvoiceCreated, etc.) used by per-type state machines | Trigger = stateless.Trigger alias; add new charge lifecycle transitions here so all types share the vocabulary |

## Anti-Patterns

- Adding a transaction.Run/service layer here — Service is intentionally an alias of Adapter; this package is a thin DB wrapper
- Persisting non-normalized timestamps — every boundary must pass through NormalizeTimestamp (UTC + streaming-window truncation)
- Defining a new ChargeType/ChargeStatus/Patch without updating Values() and the corresponding Validate()
- Mutating a PatchExtend/PatchShrink field directly instead of the normalized Set* setters
- Calling only Validate() instead of ValidateWith(intent) — directional period ordering is only enforced by ValidateWith

## Decisions

- **meta.Service = meta.Adapter (no service layer)** — Charge-meta is pure persistence with no multi-step orchestration; aliasing avoids a redundant pass-through layer until business logic appears
- **Patches are stateless.Trigger-backed value objects, not imperative mutations** — Period changes feed a qmuntal/stateless state machine and emit invoiceupdater.Patch side effects, keeping lifecycle transitions declarative
- **All timestamps truncated to streaming.MinimumWindowSizeDuration** — Aligns charge periods with the metering window granularity so usage attribution and billing boundaries stay consistent

## Example: Defining a period patch as a stateless-trigger value object

```
var (
  _             Patch = (*PatchExtend)(nil)
  TriggerExtend       = stateless.Trigger("extend")
)

func (p PatchExtend) Op() PatchType              { return PatchTypeExtend }
func (p PatchExtend) Trigger() stateless.Trigger { return TriggerExtend }
func (p PatchExtend) TriggerParams() any         { return p }

func (p PatchExtend) ValidateWith(intent Intent) error {
  var errs []error
  if err := p.Validate(); err != nil { errs = append(errs, err) }
  if !p.GetNewServicePeriodTo().After(intent.ServicePeriod.To) {
    errs = append(errs, fmt.Errorf("new service period to must be greater than existing service period to"))
  }
// ...
```

<!-- archie:ai-end -->
