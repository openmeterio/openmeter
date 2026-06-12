# billing

<!-- archie:ai-start -->

> Root of the billing domain: the contract surface that declares the billing.Service and billing.Adapter interfaces, all invoice/line/profile/customer-override/discount/tax domain models, the invoice state machine vocabulary (StandardInvoiceStatus/Trigger/Operation), the LineEngine plug-in contract, and the billing error/ValidationIssue taxonomy. This package is interfaces + value types + pure model logic only; orchestration lives in service/, persistence in adapter/, and concrete pricing/line/charge/worker logic in the sibling sub-packages.

## Patterns

**Service and Adapter are interface compositions** — billing.Service and billing.Adapter are each a single interface that embeds many narrow sub-interfaces (ProfileService, InvoiceService, LineEngineService, ... / ProfileAdapter, GatheringInvoiceAdapter, SequenceAdapter, ...). New capability = a new sub-interface added to the composite, implemented in service/ or adapter/, never inline here. (`type Service interface { ProfileService; CustomerOverrideService; InvoiceLineService; LineEngineService; ...; ConfigService } and type Adapter interface { ProfileAdapter; ...; entutils.TxCreator }`)
**Validate() collects then NewNillableGenericValidationError** — Validate() on model/input structs accumulates into var errs []error and returns models.NewNillableGenericValidationError(errors.Join(errs...)) rather than returning on the first failure (see SyncDraftStandardInvoiceInput.Validate, EventStandardInvoice.Validate, the LineEngine input Validates). (`func (i SyncDraftStandardInvoiceInput) Validate() error { var errs []error; if err := i.InvoiceID.Validate(); err != nil { errs = append(errs, err) }; ...; return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Error sentinels are coded ValidationIssue values, not error strings** — All billing-facing sentinels in errors.go are built with NewValidationError(code, message), which returns a billing.ValidationIssue (code + severity). Add new failures as a coded sentinel so HTTP status, severity, and frontend code-matching survive; never surface a bare fmt.Errorf to callers. (`ErrInvoiceCannotAdvance = NewValidationError("invoice_cannot_advance", "invoice cannot advance")`)
**Snapshot models, never live references** — Invoice-scoped models snapshot upstream entities at creation time (InvoiceCustomer copies customer.Customer fields; usage-based quantity is captured into the line and never updated). New invoice fields depending on Customer/Profile/CustomerOverride must be copied in, not referenced. (`func NewInvoiceCustomer(cust customer.Customer) InvoiceCustomer { ... } // copies Key/Name/BillingAddress so later customer edits cannot rewrite issued invoices`)
**Events carry circular-ref-stripped, app-stripped, versioned payloads** — Invoice events are constructed via NewEventStandardInvoice which calls invoice.RemoveCircularReferences() and nils Workflow.Apps (not JSON-marshallable), promoting apps into a separate InvoiceApps struct. EventName() is versioned via metadata.GetEventName (e.g. invoice.created v2, invoice.updated v3, invoice.advance v1). (`payload, err := invoice.RemoveCircularReferences(); payload.Workflow.Apps = nil; apps.Invoicing, _ = app.NewEventApp(invoice.Workflow.Apps.Invoicing)`)
**Status strings are dotted; ShortStatus drives category matching** — StandardInvoiceStatus values are "category.detail" (e.g. draft.collecting); ShortStatus() splits on the first dot so a StandardInvoiceStatusCategory matches any sub-state. New statuses must be added to validStatuses, and to finalStatuses/failedStatuses if terminal/failing. (`func (s StandardInvoiceStatus) ShortStatus() string { parts := strings.SplitN(string(s), ".", 2); return parts[0] }`)
**LineEngine hooks reuse input line IDs and preview is side-effect-free** — billing.LineEngine is the per-line-type plug-in (invoicing + the three charge types). BuildStandardInvoiceLines/preview must reuse the exact input gathering-line IDs; the preview variant must not persist, allocate credits, mutate IDs, or emit events. Engines register via Service.RegisterLineEngine keyed by GetLineEngineType(). (`LineEngineTypeInvoice / LineEngineTypeChargeFlatFee / ...ChargeUsageBased / ...ChargeCreditPurchase; BuildStandardLinesForGatheringPreview "must be side-effect free"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Declares the composite billing.Service interface and every sub-interface (Profile, CustomerOverride, InvoiceLine, LineEngine, SplitLineGroup, Invoice, GatheringInvoice, StandardInvoice, Sequence, Lockable, InvoiceApp, Config). | Add new methods to the correct narrow sub-interface; service/ must then implement them. ConfigService.WithAdvancementStrategy/WithLockedNamespaces return a (cloned) Service, they do not mutate in place. |
| `adapter.go` | Declares billing.Adapter as a composition of persistence sub-interfaces plus entutils.TxCreator; the pure Ent-backed contract. | Adapter is persistence only — no business logic. Gathering vs standard invoice operations live in distinct sub-interfaces; pick the right one. |
| `errors.go` | Central billing error taxonomy: coded ValidationError sentinels (Err*), NotFoundError, AppError, UpdateAfterDeleteError, ErrSnapshotInvalidDatabaseState, plus EncodeValidationIssues for HTTP encoding. | NewValidationError returns a billing.ValidationIssue (defined in validationissue.go), not a plain error. ErrSnapshotInvalidDatabaseState is special-cased to move the invoice to draft.invalid; do not collapse it into a generic error. |
| `stdinvoice.go` | StandardInvoice model plus the full StandardInvoiceStatus enum, StandardInvoiceStatusCategory, validStatuses/finalStatuses/failedStatuses, and matcher helpers. | A new status must be appended to validStatuses or Validate() rejects it; classify into finalStatuses/failedStatuses as appropriate; statuses are dotted and matched by ShortStatus(). |
| `stdinvoicestate.go` | State-machine vocabulary: InvoiceTrigger aliases over stateless.Trigger (TriggerNext/Approve/Retry/Failed/ForceCollect/...) and the StandardInvoiceOperation enum. | Triggers/operations are declared here but transition wiring lives in service/ (qmuntal/stateless). Adding a trigger here without wiring it does nothing. |
| `lineengine.go` | billing.LineEngine + billing.LineCalculator interfaces, LineEngineType discriminators, and per-hook input structs (BuildStandardInvoiceLinesInput, CalculateLinesInput, StandardLineEventInput, SplitGatheringLineInput). | Adding a hook means updating every engine impl plus testutils.NoopLineEngine. Keep BuildStandardLinesForGatheringPreview side-effect-free. NewLineEngineValidationError wraps failures into a ValidationIssue with a component name. |
| `app.go` | InvoicingApp / InvoicingAppPostAdvanceHook / InvoicingAppAsyncSyncer interfaces apps implement, plus Upsert/Finalize result builders and SyncInput (draft/issuing) types that MergeIntoInvoice. | The invoice passed to app callbacks is read-only and may be a stale in-memory snapshot — never call back into billing.Service from these hooks. Result structs use fluent Set*/Get* (bool-presence) accessors. |
| `events.go` | Versioned invoice domain events (StandardInvoiceCreated v2, Updated v3, AdvanceStandardInvoice v1) and the InvoiceApps payload; eventsgathering.go covers GatheringInvoiceCreatedEvent. | Bump Version in EventName() on a breaking payload change. Always build via NewEventStandardInvoice so circular refs and non-marshallable Apps are stripped. |

## Anti-Patterns

- Putting orchestration, DB access, or state-machine transition logic in this root package — it is interfaces + value types + pure model math; behavior belongs in service/ and adapter/.
- Surfacing a new failure as a bare fmt.Errorf instead of a coded NewValidationError sentinel in errors.go, losing HTTP status, severity, and frontend code-matching.
- Returning on the first invalid field in a Validate() method instead of collecting into errs and returning models.NewNillableGenericValidationError(errors.Join(errs...)).
- Storing a live pointer to Customer/Profile/CustomerOverride on an invoice/line model instead of snapshotting the needed fields (e.g. via NewInvoiceCustomer) — issued invoices must not change retroactively.
- Adding a StandardInvoiceStatus or LineEngine hook without updating the central lists (validStatuses/finalStatuses) or every LineEngine implementation incl. testutils.NoopLineEngine, leaving the state machine / engine registry inconsistent.

## Decisions

- **billing.Service and billing.Adapter are large interface compositions of narrow sub-interfaces rather than one flat interface.** — Lets handlers and workers depend on just the slice they need and keeps the service/ struct's surface organised by concern (profile, invoice, line, sequence, lock, config).
- **Invoice-facing errors are modeled as coded ValidationIssue sentinels returned alongside the (still-returned) invoice, not as fatal errors.** — Invoices can have issues at many independent layers (missing profile, provider failure, etc.); clients need the invoice plus machine-readable codes (EncodeValidationIssues) rather than string-matched failures.
- **Invoice models snapshot upstream entities and the package uses goderive (generate.go) for derived equality/clone helpers (derived.gen.go).** — Snapshotting guarantees an issued invoice is immutable to later customer/profile edits; goderive avoids hand-writing the deep-equality and clone logic the snapshot/diff flows rely on.

## Example: Adding an invoice failure: declare it as a coded ValidationIssue sentinel and validate inputs by collecting errors

```
// errors.go — sentinel is a billing.ValidationIssue (carries code + severity)
var ErrInvoiceCannotAdvance = NewValidationError("invoice_cannot_advance", "invoice cannot advance")

// any input Validate(): collect, don't early-return
func (i SyncDraftStandardInvoiceInput) Validate() error {
	var errs []error
	if err := i.InvoiceID.Validate(); err != nil {
		errs = append(errs, err)
	}
	if i.AdditionalMetadata == nil {
		errs = append(errs, fmt.Errorf("additional metadata is required"))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

<!-- archie:ai-end -->
