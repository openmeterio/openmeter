# invoiceupdater

<!-- archie:ai-start -->

> Applies the subscription-sync reconciler's computed drift (a slice of Patch) to billing artifacts: provisions new gathering lines, updates/deletes lines on gathering and mutable standard invoices, and records warning validation issues (rather than mutating) on immutable invoices. It is the write side of the subscription->billing sync bridge; the reconciler decides what should change, this package executes it via billing.Service.

## Patterns

**Tagged-union Patch with op-checked accessors** — Patch is a struct holding op PatchOperation plus one field per variant; never construct Patch literals directly outside this package. Build via NewCreateLinePatch/NewDeleteLinePatch/NewUpdateLinePatch/NewDeleteSplitLineGroupPatch/NewUpdateSplitLineGroupPatch, and read via AsXxxPatch() which errors unless p.op matches. (`patch := invoiceupdater.NewDeleteLinePatch(line.GetLineID(), line.GetInvoiceID()); create, err := patch.AsCreateLinePatch() // err unless op == PatchOpLineCreate`)
**parsePatches bucketing before any write** — ApplyPatches first calls parsePatches to split patches into patchesParsed{newLines, updatedLinesByInvoiceID (keyed by invoice ID -> invoicePatches{updatedLines, deletedLines}), splitLineGroups}. New code that adds a PatchOperation MUST add a case in both parsePatches and Patch.Log (default branches error/log-unknown). (`case PatchOpLineDelete: lineUpdates := parsed.updatedLinesByInvoiceID[deletePatch.InvoiceID]; lineUpdates.deletedLines = append(...); parsed.updatedLinesByInvoiceID[deletePatch.InvoiceID] = lineUpdates`)
**Invoice-mutability branching** — Per invoice, dispatch by Type()/StatusDetails.Immutable: gathering -> updateGatheringInvoice, mutable standard -> updateMutableStandardInvoice, immutable -> updateImmutableInvoice (no mutation, only validation issues). Reuse this exact three-way branch; never mutate an immutable invoice's lines. (`if invoice.Type() == billing.InvoiceTypeGathering { ... }; standardInvoice, _ := invoice.AsStandardInvoice(); if !standardInvoice.StatusDetails.Immutable { updateMutableStandardInvoice(...) } else { updateImmutableInvoice(...) }`)
**Mutate inside billing.Service EditFn closures** — All line edits go through UpdateStandardInvoice/UpdateGatheringInvoice with an EditFn that mutates the passed *invoice in place (line.DeletedAt = lo.ToPtr(clock.Now()), invoice.Lines.ReplaceByID(...)). Set IncludeDeletedLines: true so deletes are visible. Re-snapshot quantity via billingService.SnapshotLineQuantity before replacing a line. (`u.billingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{Invoice: invoice.GetInvoiceID(), IncludeDeletedLines: true, EditFn: func(invoice *billing.StandardInvoice) error { ... }})`)
**Immutable invoice -> warning ValidationIssue, not error** — On immutable invoices, drift is reported via newValidationIssueOnLine (Severity Warning, Code ImmutableInvoiceHandlingNotSupportedErrorCode, Component subscriptionSyncComponentName) and merged idempotently with mergeValidationIssues (dedupes by Path+Component+Code+Message). Flat-fee per-unit amount and service-period changes, and usage-based quantity changes, are not applied — only flagged. (`validationIssues = append(validationIssues, newValidationIssueOnLine(existingLine, "flat fee line's per unit amount cannot be changed on immutable invoice (new per unit amount: %s)", targetPerUnitAmount.String()))`)
**Group new lines by currency for provisioning** — provisionUpcomingLines groups GatheringLines by l.Currency with lo.GroupBy and issues one billingService.CreatePendingInvoiceLines call per currency code, since pending-line creation is currency-scoped. (`linesByCurrency := lo.GroupBy(lines, func(l billing.GatheringLine) currencyx.Code { return l.Currency })`)
**Dry-run logging suppresses non-actionable patches** — LogPatches calls Patch.Log only when isDryRunLoggablePatch is true: current-period create-line patches and patches targeting immutable invoices are suppressed (isCurrentBillingPeriod / isMutableInvoice), keeping dry-run output focused on actionable drift on materialized resources. (`if !isDryRunLoggablePatch(patch, invoicesByID) { suppressedDryRunPatches++; continue }; patch.Log(u.logger)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `patch.go` | Defines PatchOperation constants, the Patch tagged-union and per-variant payload structs (PatchLineCreate/Delete/Update, PatchSplitLineGroup Delete/Update), constructors, AsXxxPatch accessors, Patch.Log, and GetDeletePatchesForLine. | GetDeletePatchesForLine skips lines annotated billing.AnnotationSubscriptionSyncIgnore and already-deleted lines; for a hierarchy it emits a group-delete plus per-line deletes but bails entirely if ANY line is sync-ignored. Adding a PatchOperation requires updating Patch.Log too. |
| `invoiceupdate.go` | Updater (New(billingService, logger)) with ApplyPatches as the single entry point; holds parsePatches, the three invoice-update paths, validation-issue helpers, and dry-run logging. | ApplyPatches order matters: provision new lines -> list affected invoices (IncludeDeleted: true) -> per-invoice line updates -> upsertSplitLineGroups last. updateMutableStandardInvoice deletes the invoice when NonDeletedLineCount()==0 (unless still gathering) and logs validation issues on StandardInvoiceStatusDeleteFailed. Service-period equality uses .Truncate(streaming.MinimumWindowSizeDuration). |
| `feehelper.go` | Flat-fee price helpers over billing line readers/writers: IsFlatFee, GetFlatFeePerUnitAmount, SetFlatFeePerUnitAmount. | All three nil-guard the line and price; AsFlat() can error if the line lacks usage-based metadata. SetFlatFeePerUnitAmount mutates the flatPrice then re-wraps via productcatalog.NewPriceFrom + line.SetPrice — don't assume in-place mutation of the price persists without SetPrice. |

## Anti-Patterns

- Constructing Patch{} literals or reading variant fields directly instead of using the NewXxxPatch constructors and AsXxxPatch accessors.
- Mutating lines or amounts on an immutable standard invoice instead of appending a warning ValidationIssue via newValidationIssueOnLine/mergeValidationIssues.
- Editing invoice lines outside an UpdateStandardInvoice/UpdateGatheringInvoice EditFn, or omitting IncludeDeletedLines: true so soft-deletes are not seen.
- Replacing a line on a mutable invoice without first re-running billingService.SnapshotLineQuantity to refresh usage quantity.
- Adding a new PatchOperation without extending parsePatches and Patch.Log (their default branches error / log 'unknown patch operation').

## Decisions

- **Patch is a hand-rolled tagged union with op-guarded accessors rather than an interface.** — Lets parsePatches exhaustively switch on Op() and lets accessors fail loudly when the variant is wrong, keeping create/delete/update of lines and split-line groups in one comparable, loggable type.
- **Immutable invoices are never mutated; drift becomes deduplicated warning ValidationIssues.** — Finalized invoices must stay stable, but subscription sync still needs to surface that target state diverged, so it records actionable warnings (ImmutableInvoiceHandlingNotSupportedErrorCode) idempotently.
- **Updater holds only billing.Service plus a logger and performs all writes through service methods.** — Keeps this reconciler write-side at the service layer (transaction/state-machine semantics owned by billing.Service) instead of touching adapters/Ent directly.

## Example: Update a line on a mutable standard invoice via EditFn with quantity re-snapshot

```
u.billingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
  Invoice:             invoice.GetInvoiceID(),
  IncludeDeletedLines: true,
  EditFn: func(invoice *billing.StandardInvoice) error {
    targetStandardLine, err := targetState.AsInvoiceLine().AsStandardLine()
    if err != nil { return err }
    updatedQtyLine, err := u.billingService.SnapshotLineQuantity(ctx, billing.SnapshotLineQuantityInput{Invoice: invoice, Line: &targetStandardLine})
    if err != nil { return err }
    targetStandardLine = *updatedQtyLine
    if ok := invoice.Lines.ReplaceByID(targetStandardLine.ID, &targetStandardLine); !ok {
      return fmt.Errorf("line[%s] not found", targetStandardLine.ID)
    }
    return nil
  },
})
```

<!-- archie:ai-end -->
