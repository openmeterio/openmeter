# invoiceupdater

<!-- archie:ai-start -->

> Applies a typed patch set (create/update/delete lines and split-line-groups) to billing invoices during subscription sync reconciliation. It is the write-side of the reconciler: translates abstract diff patches into concrete billing.Service calls, branching on invoice type (gathering vs standard) and mutability.

## Patterns

**Sealed Patch discriminated union** — Patch is a value type with a private op field. Callers construct via New*Patch() constructors and type-assert via As*Patch() methods that error on wrong op. Never add public fields to Patch or bypass As*Patch(). (`patch := NewUpdateLinePatch(line); update, err := patch.AsUpdateLinePatch() // errors if op != PatchOpLineUpdate`)
**Invoice-type branch in ApplyPatches** — After fetching the invoice, branch on invoice.Type() == billing.InvoiceTypeGathering first, then check StandardInvoice.StatusDetails.Immutable. Each branch calls a dedicated private method: updateGatheringInvoice, updateMutableStandardInvoice, or updateImmutableInvoice. (`if invoice.Type() == billing.InvoiceTypeGathering { updateGatheringInvoice(...) } else if !standardInvoice.StatusDetails.Immutable { updateMutableStandardInvoice(...) } else { updateImmutableInvoice(...) }`)
**EditFn closure for atomic invoice updates** — All mutations to gathering and mutable-standard invoices go through billing.Service's UpdateGatheringInvoice / UpdateStandardInvoice with an EditFn closure. Never call low-level adapter methods directly. (`u.billingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{Invoice: id, EditFn: func(invoice *billing.StandardInvoice) error { /* mutate */ return nil }})`)
**Immutable invoice ValidationIssue accumulation** — When the invoice is immutable, diff patches are not applied; instead they are collected as billing.ValidationIssue values with Severity=Warning and Code=ImmutableInvoiceHandlingNotSupportedErrorCode, then upserted via billing.Service.UpsertValidationIssues. (`validationIssues = append(validationIssues, newValidationIssueOnLine(existingLine, "flat fee line's per unit amount cannot be changed..."))`)
**Currency-grouped line provisioning** — New gathering lines are grouped by currencyx.Code via lo.GroupBy before calling CreatePendingInvoiceLines once per currency group. billing.CreatePendingInvoiceLinesInput requires a single currency per call. (`linesByCurrency := lo.GroupBy(lines, func(l billing.GatheringLine) currencyx.Code { return l.Currency }); for currency, lines := range linesByCurrency { billingService.CreatePendingInvoiceLines(ctx, ...) }`)
**FlatFee helpers isolate price-type logic** — feehelper.go exposes IsFlatFee, GetFlatFeePerUnitAmount, SetFlatFeePerUnitAmount to keep price-type branching out of invoiceupdate.go. Add new price-type helpers here, not inline in update logic. (`if IsFlatFee(targetState) { existingAmt, _ := GetFlatFeePerUnitAmount(existingLine) }`)
**New PatchOperation requires five coordinated changes** — Adding a new PatchOperation requires: new constant, new payload struct, new As* accessor method, new New* constructor, new case in parsePatches switch, and new Log case. All five must be updated atomically. (`const PatchOpLineCreate PatchOperation = "line_create" // + PatchLineCreate struct + AsCreateLinePatch() + NewCreateLinePatch() + parsePatches case + Log case`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `invoiceupdate.go` | Core Updater struct and ApplyPatches orchestration; owns all billing.Service call sites. Fetches invoices once per invoiceID in listInvoicesByID then distributes to type-specific handlers. | Invoice fetch happens once per invoiceID inside the loop — do not add extra fetches. Empty-invoice deletion after mutable-standard update (when NonDeletedLineCount()==0) is intentional and must be preserved. |
| `patch.go` | Patch value type, PatchOperation constants, New*/As* constructors/accessors, GetDeletePatchesForLine helper for building delete patches from a LineOrHierarchy. | GetDeletePatchesForLine checks billing.AnnotationSubscriptionSyncIgnore and skips lines/groups with it set. Adding a new PatchOperation requires all five coordinated changes listed in patterns. |
| `feehelper.go` | Flat-fee price-type helpers operating on billing.GenericInvoiceLineReader / billing.GenericInvoiceLine. | SetFlatFeePerUnitAmount mutates the line in place via line.SetPrice(); callers must hold a mutable reference. Returns error if price type is not flat. |

## Anti-Patterns

- Calling billing.Adapter methods directly instead of going through billing.Service EditFn pattern
- Adding a new PatchOperation without updating all five locations: constant, struct, As*, New*, parsePatches switch (and Log case)
- Fetching the same invoice multiple times within one ApplyPatches call
- Producing ValidationIssues for mutable invoices — ValidationIssues are only for immutable invoice drift
- Using context.Background() instead of propagating the ctx parameter through all billing.Service calls

## Decisions

- **Patch is a sealed value type (private op, As* accessors) rather than an interface** — Prevents callers from constructing invalid patch combinations and keeps the parsePatches switch exhaustive and compiler-checked within this package.
- **Immutable invoices receive ValidationIssues instead of hard errors** — Invoice immutability is a billing state (e.g. issued/paid); reporting drift as warnings lets operators see subscription/invoice divergence without blocking the reconciler loop.
- **New lines are grouped by currency before CreatePendingInvoiceLines** — billing.CreatePendingInvoiceLinesInput requires a single currency per call; grouping here prevents the caller from needing to pre-split lines.

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
if err := updater.ApplyPatches(ctx, customer.CustomerID{Namespace: ns, ID: cid}, patches); err != nil {
	return fmt.Errorf("applying patches: %w", err)
}
```

<!-- archie:ai-end -->
