# invoiceupdater

<!-- archie:ai-start -->

> Translates charge-engine line patches into concrete billing.Service mutations on gathering and standard invoices. It is the only place charge state is projected back onto invoice lines; it owns no charge state itself.

## Patterns

**Discriminated Patch value type** — Patch carries a PatchOperation op plus one populated sub-struct; construct only via NewCreateLinePatch/NewDeleteLinePatch/NewUpdateLinePatch/NewDeleteGatheringLineByChargeIDPatch/NewUpdateGatheringLineByChargeIDPatch and read via As*Patch() which error on op mismatch. (`func (p Patch) AsCreateLinePatch() (PatchLineCreate, error) { if p.op != PatchOpLineCreate { return PatchLineCreate{}, fmt.Errorf(...) }; return p.createLinePatch, nil }`)
**Parse-then-route ApplyPatches pipeline** — ApplyPatches first parsePatches into patchesParsed, resolves by-charge-id deletes/updates against gathering invoices, provisions new lines, then routes per-invoice patches to gathering vs mutable-standard vs immutable handlers based on invoice.Type() and StatusDetails.Immutable. (`if invoice.Type()==billing.InvoiceTypeGathering { u.updateGatheringInvoice(...) } else if !standardInvoice.StatusDetails.Immutable { u.updateMutableStandardInvoice(...) } else { u.updateImmutableInvoice(...) }`)
**EditFn-based invoice mutation** — Standard/gathering invoice edits go through billing.UpdateStandardInvoiceInput/UpdateGatheringInvoiceInput with an EditFn closure that mutates invoice.Lines (GetByID, ReplaceByID, set DeletedAt), never direct adapter writes. (`u.billingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{Invoice: invoice.GetInvoiceID(), IncludeDeletedLines:true, EditFn: func(invoice *billing.StandardInvoice) error {...}})`)
**Charge ownership guard before mutating lines** — ensureLineHasChargeID is called before deleting/updating a line so the updater only touches charge-backed lines. (`if err := ensureLineHasChargeID(line, deletePatch.op); err != nil { return err }`)
**Dry-run patch log suppression** — LogPatches uses isDryRunLoggablePatch to suppress current-billing-period create-lines and immutable-invoice deletes/updates, focusing logs on actionable drift. (`return !isCurrentBillingPeriod(createPatch.Line)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `patch.go` | PatchOperation constants, the five Patch sub-structs, New* constructors, As* accessors, and Log(). | Patch fields are unexported — always go through constructors and accessors; adding an op requires updating Op() dispatch in parsePatches, isDryRunLoggablePatch, and Log. |
| `invoiceupdate.go` | Updater (wraps billing.Service + logger), ApplyPatches pipeline, gathering/standard/immutable routing, provisionUpcomingLines, by-charge-id resolution. | Empty mutable standard invoices (NonDeletedLineCount()==0, non-gathering) are auto-deleted; DeleteFailed status only logs validation issues. Target state is intentionally NOT passed through billing's generic snapshotter — charge engines own quantity snapshots. |
| `feehelper.go` | IsFlatFee / GetFlatFeePerUnitAmount / SetFlatFeePerUnitAmount over billing.GenericInvoiceLine price (productcatalog flat price). | SetFlatFeePerUnitAmount rebuilds the price via productcatalog.NewPriceFrom(flatPrice) and SetPrice; mutating flatPrice.Amount alone does not persist. |

## Anti-Patterns

- Constructing a Patch struct literal directly instead of via New* constructors.
- Writing invoice lines through ent/adapters instead of billing.Service EditFn closures.
- Mutating non-charge-backed lines — every delete/update must pass ensureLineHasChargeID.
- Pushing charge target-state through billing's generic snapshotter (charge engines own snapshots).

## Decisions

- **Invoice mutations are expressed as Patch values applied through billing.Service, not direct DB writes.** — Keeps charge engines decoupled from billing internals and lets billing enforce invoice state-machine rules (immutability, validation).
- **By-charge-id delete/update patches are resolved by listing gathering invoice lines and matching line.ChargeID.** — Charge engines don't always know the gathering line ID, so the updater bridges charge ID -> line ID at apply time.

<!-- archie:ai-end -->
