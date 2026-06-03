# invoiceupdater

<!-- archie:ai-start -->

> Translates charge-lifecycle patch operations into billing.Service mutations: creates new gathering lines, updates/deletes existing lines on gathering and standard invoices, and resolves charge-ID-based patches by scanning gathering invoices for matching lines.

## Patterns

**Patch tagged-union with typed accessor methods** — All patch operations are Patch{op, ...} with named constructors (NewCreateLinePatch, NewDeleteLinePatch, NewUpdateLinePatch...) and typed As* accessors that validate op before returning the payload. Never construct Patch{} directly. (`patch := invoiceupdater.NewDeleteLinePatch(lineID, invoiceID)
deletePatch, err := patch.AsDeleteLinePatch()`)
**ApplyPatches fans out by invoice type** — Updater.ApplyPatches first resolves charge-ID-based patches to concrete line IDs, then groups patches by invoice ID and dispatches to updateGatheringInvoice or updateMutableStandardInvoice/updateImmutableInvoice based on invoice.Type() and StatusDetails.Immutable. (`if invoice.Type() == billing.InvoiceTypeGathering { return u.updateGatheringInvoice(ctx, ...) }
if !invoice.StatusDetails.Immutable { return u.updateMutableStandardInvoice(ctx, ...) }`)
**Resolve charge-ID-based patches via gathering invoice scan** — PatchOpDeleteGatheringLineByChargeID and PatchOpUpdateGatheringLineByChargeID are resolved by listing all gathering invoices for the customer and matching line.ChargeID, converting them to concrete line-level entries before dispatch. (`invoices, _ := u.billingService.ListGatheringInvoices(ctx, ...)
for _, line := range invoice.Lines.OrEmpty() { if _, ok := chargeIDs[*line.ChargeID]; ok { /* add to deletedLines */ } }`)
**Only charge-backed lines mutated** — All update/delete patches must target lines with a non-nil ChargeID; ensureLineHasChargeID enforces this before applying any patch. Non-charge lines are left untouched. (`if err := ensureLineHasChargeID(line, updatePatch.op); err != nil { return err }`)
**Empty mutable standard invoice is deleted** — After updateMutableStandardInvoice, if NonDeletedLineCount() == 0 and not in Gathering status, DeleteInvoice is called automatically; deletion failures are logged as warnings, not errors. (`if updatedInvoice.Lines.NonDeletedLineCount() == 0 { invoice, err := u.billingService.DeleteInvoice(ctx, updatedInvoice.GetInvoiceID()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `patch.go` | Defines PatchOperation constants, all patch payload structs, the Patch tagged-union, named constructors, As* accessors, and Log() for structured logging. | New patch ops must add a constructor (NewXxxPatch), an As* accessor, a Log() case, and handling in parsePatches. Missing any one causes a panic or silent no-op. |
| `invoiceupdate.go` | Updater struct, ApplyPatches orchestration, patch resolution and dispatch to billing.Service methods; also LogPatches for dry-run output. | LogPatches suppresses current-period pending-line creates and gathering invoice patches to reduce dry-run noise — do not remove this filtering without understanding the intent. |
| `feehelper.go` | Pure helpers for reading/writing flat-fee per-unit amount from a billing.GenericInvoiceLine price field. | SetFlatFeePerUnitAmount mutates line.Price in place; callers must ensure the line struct is not shared across goroutines. |

## Anti-Patterns

- Constructing Patch{} struct literals with op set manually — use named constructors so payload fields are always populated
- Adding line mutations directly to the billing adapter from outside invoiceupdater — all charge-line mutations flow through Updater.ApplyPatches
- Calling ListGatheringInvoices per individual patch instead of batching — resolution is already batched per customer in resolveGatheringLineDeletesByChargeID
- Skipping ensureLineHasChargeID on update/delete patches — non-charge lines must not be mutated
- Returning errors for immutable invoice patches without implementing updateImmutableInvoice — immutable paths must be explicitly handled

## Decisions

- **Patch operations use a tagged-union struct with private op discriminator rather than an interface** — Exhaustive dispatch in the parsePatches switch is compile-checked on adding new constants; interface dispatch would silently swallow unknown types.
- **Charge-ID-based patches resolved at ApplyPatches time, not at patch creation** — The charge service caller does not have gathering line IDs at patch generation; lazy resolution inside ApplyPatches keeps the caller API simple.

## Example: Adding a new patch operation for a new mutation type

```
// 1. const PatchOpLineExtend PatchOperation = "line_extend"
// 2. type PatchLineExtend struct { LineID billing.LineID; NewEndTime time.Time }
// 3. add extendLinePatch field to the Patch union
func NewExtendLinePatch(lineID billing.LineID, endTime time.Time) Patch {
    return Patch{op: PatchOpLineExtend, extendLinePatch: PatchLineExtend{LineID: lineID, NewEndTime: endTime}}
}
// 4. add AsExtendLinePatch accessor; 5. add Log() case; 6. handle in parsePatches
```

<!-- archie:ai-end -->
