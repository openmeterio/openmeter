# invoiceupdater

<!-- archie:ai-start -->

> Applies a typed patch set (create/update/delete lines and split-line-groups) to billing invoices during subscription sync reconciliation. It is the write-side of the reconciler: it translates abstract diff patches into concrete billing.Service calls, branching on invoice type (gathering vs standard) and mutability (mutable vs immutable).

## Patterns

**Sealed Patch discriminated union** — Patch is a value type with a private op field; callers construct via New*Patch() constructors and type-assert via As*Patch() methods that error on wrong op. Never add public fields to Patch or bypass As*Patch(). (`patch := NewUpdateLinePatch(line); create, err := patch.AsCreateLinePatch() // errors if op != PatchOpLineCreate`)
**Invoice-type branch in ApplyPatches** — After fetching the invoice, branch on invoice.Type() == billing.InvoiceTypeGathering first, then StandardInvoice.StatusDetails.Immutable. Each branch calls a dedicated private method (updateGatheringInvoice, updateMutableStandardInvoice, updateImmutableInvoice). (`if invoice.Type() == billing.InvoiceTypeGathering { updateGatheringInvoice(...) } else if !standardInvoice.StatusDetails.Immutable { updateMutableStandardInvoice(...) } else { updateImmutableInvoice(...) }`)
**EditFn closure for atomic invoice updates** — Mutations to gathering and mutable-standard invoices go through billing.Service's UpdateGatheringInvoice / UpdateStandardInvoice with an EditFn closure. Never call low-level adapter methods directly. (`u.billingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{EditFn: func(invoice *billing.StandardInvoice) error { ... }})`)
**Immutable invoice → ValidationIssue accumulation** — When the invoice is immutable, differences are not applied; instead they are collected as billing.ValidationIssue values with Severity=Warning and Code=ImmutableInvoiceHandlingNotSupportedErrorCode, then upserted via billing.Service.UpsertValidationIssues. (`validationIssues = append(validationIssues, newValidationIssueOnLine(existingLine, "flat fee line's per unit amount cannot be changed..."))`)
**Currency-grouped line provisioning** — New gathering lines are grouped by currencyx.Code via lo.GroupBy before calling CreatePendingInvoiceLines once per currency group. (`linesByCurrency := lo.GroupBy(lines, func(l billing.GatheringLine) currencyx.Code { return l.Currency })`)
**Dry-run log suppression for current-period creates** — isDryRunLoggablePatch suppresses PatchOpLineCreate patches whose line is in the current billing period (isCurrentBillingPeriod using clock.Now()); other patch types are always logged. (`case PatchOpLineCreate: return !isCurrentBillingPeriod(createPatch.Line)`)
**FlatFee helpers isolate price-type logic** — feehelper.go exposes IsFlatFee, GetFlatFeePerUnitAmount, SetFlatFeePerUnitAmount to keep price-type branching out of invoiceupdate.go. Add new price-type helpers here, not inline in update logic. (`if IsFlatFee(targetState) { existingAmt, _ := GetFlatFeePerUnitAmount(existingLine); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `invoiceupdate.go` | Core Updater struct and ApplyPatches orchestration; owns all billing.Service call sites | Invoice fetch happens once per invoiceID inside the loop — do not add extra fetches; empty-invoice deletion after mutable-standard update is intentional and must be preserved |
| `patch.go` | Patch value type, PatchOperation constants, New*/As* constructors, GetDeletePatchesForLine helper | Adding a new PatchOperation requires: new constant, new payload struct, new As* method, new New* constructor, new Log case, and handling in parsePatches switch — all five must be updated together |
| `feehelper.go` | Flat-fee price-type helpers operating on billing.GenericInvoiceLineReader/billing.GenericInvoiceLine | SetFlatFeePerUnitAmount mutates the line in place via line.SetPrice(); callers must hold a mutable reference |

## Anti-Patterns

- Calling billing.Adapter methods directly instead of going through billing.Service (EditFn pattern)
- Adding a new PatchOperation without updating all five locations: constant, struct, As*, New*, parsePatches switch
- Fetching the same invoice multiple times within one ApplyPatches call
- Producing ValidationIssues for mutable invoices — ValidationIssues are only for immutable invoice drift
- Using context.Background() instead of propagating the ctx parameter through all billing.Service calls

## Decisions

- **Patch is a sealed value type (private op, As* accessors) rather than an interface** — Prevents callers from constructing invalid patch combinations and keeps the switch exhaustive and compiler-checked within this package
- **Immutable invoices receive ValidationIssues instead of hard errors** — Invoice immutability is a billing state (e.g. issued/paid); reporting drift as warnings lets operators see subscription/invoice divergence without blocking the reconciler loop
- **New lines are grouped by currency before CreatePendingInvoiceLines** — billing.CreatePendingInvoiceLinesInput requires a single currency per call; grouping here prevents the caller from needing to pre-split lines

## Example: Applying a mixed patch set to invoices of different types

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

patches := []invoiceupdater.Patch{
	invoiceupdater.NewCreateLinePatch(newGatheringLine),
	invoiceupdater.NewUpdateLinePatch(updatedLine),
	invoiceupdater.NewDeleteLinePatch(lineID, invoiceID),
}
updater := invoiceupdater.New(billingService, logger)
if err := updater.ApplyPatches(ctx, customer.CustomerID{Namespace: ns, ID: cid}, patches); err != nil {
	return fmt.Errorf("applying patches: %w", err)
// ...
```

<!-- archie:ai-end -->
