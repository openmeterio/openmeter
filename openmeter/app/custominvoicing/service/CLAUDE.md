# service

<!-- archie:ai-start -->

> Business logic layer for the custominvoicing app — implements appcustominvoicing.Service (factory, customer data, and sync sub-interfaces) by orchestrating appcustominvoicing.Adapter, app.Service, and billing.Service within explicit transactions.

## Patterns

**transaction.Run wrapping multi-step writes** — Operations that touch multiple entities run inside transaction.Run or transaction.RunWithNoValue to ensure atomicity. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.AppBase, error) { appBase, _ := s.appService.CreateApp(...); s.adapter.UpsertAppConfiguration(...); return appBase, nil })`)
**Config struct with Validate() + compile-time assertion** — Service constructor takes a Config struct; Validate() checks all required fields. var _ appcustominvoicing.Service = (*Service)(nil) enforces interface at compile time. (`var _ appcustominvoicing.Service = (*Service)(nil)`)
**ValidateInvoiceApp guard before billing mutations** — sync.go methods call s.ValidateInvoiceApp(invoice) after fetching the invoice to assert it belongs to the custom invoicing app before firing triggers. (`if err := s.ValidateInvoiceApp(invoice); err != nil { return billing.StandardInvoice{}, err }`)
**input.Validate() at method entry** — Every exported service method calls input.Validate() before any business logic or transaction. (`if err := input.Validate(); err != nil { return billing.StandardInvoice{}, err }`)
**Double read around billing trigger in HandlePaymentTrigger** — HandlePaymentTrigger does get → trigger → get: the second GetStandardInvoiceById is needed to observe post-trigger state and detect critical ValidationIssues. (`invoice, _ = s.billingService.GetStandardInvoiceById(ctx, ...); criticalIssues := lo.Filter(invoice.ValidationIssues, func(i billing.ValidationIssue, _ int) bool { return i.Severity == billing.ValidationIssueSeverityCritical })`)
**Critical ValidationIssue check causes transaction rollback** — HandlePaymentTrigger returns billing.ValidationError if any critical issues are present after the trigger — this causes the wrapping transaction.Run to roll back. (`if len(criticalIssues) > 0 { return billing.StandardInvoice{}, billing.ValidationError{Err: criticalIssues.AsError()} }`)
**clock.Now() for testable timestamps** — SyncDraftInvoice and SyncIssuingInvoice inject metadata timestamps using clock.Now(), not time.Now(), for deterministic testing. (`AdditionalMetadata: map[string]string{appcustominvoicing.MetadataKeyDraftSyncedAt: clock.Now().Format(time.RFC3339)}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct definition, Config, Validate, New constructor. Sets up the three dependencies: adapter, appService, billingService. | Service struct is exported (uppercase) unlike most sibling packages — intentional to allow direct *Service injection in Wire provider sets. |
| `sync.go` | SyncService implementation: SyncDraftInvoice, SyncIssuingInvoice, HandlePaymentTrigger, ValidateInvoiceApp. | HandlePaymentTrigger does get-trigger-get — do not collapse the two reads; the second is required to observe post-trigger state. |
| `factory.go` | FactoryService implementation: CreateApp (two-step: appService.CreateApp + adapter.UpsertAppConfiguration), DeleteApp, UpsertAppConfiguration, GetAppConfiguration. | CreateApp wraps both steps in a single transaction.Run — if adapter.UpsertAppConfiguration fails, the app row is rolled back. |
| `customerdata.go` | CustomerDataService implementation: thin delegation to adapter wrapped in transaction.Run/RunWithNoValue. | No business logic here — validation belongs in input types, not this file. |

## Anti-Patterns

- Calling time.Now() directly instead of clock.Now() — breaks deterministic tests
- Skipping ValidateInvoiceApp before billing mutations — allows cross-app invoice mutations
- Not wrapping multi-entity writes in transaction.Run — risks partial writes on appService + adapter calls
- Returning an error for NotFound from adapter.GetAppConfiguration — the adapter returns zero value; callers expect that
- Adding billing.Service calls directly in customerdata.go — customer data operations must stay adapter-only

## Decisions

- **Service struct is exported (uppercase)** — Allows Wire to inject *Service directly when both FactoryService and SyncService sub-interfaces are needed at different wiring sites.
- **ValidateInvoiceApp is a standalone exported method** — Used both internally (before triggers) and passed as InvoiceValidator callback to billing.Service sync methods, requiring a stable method signature.
- **Critical-issue check causes transaction rollback in HandlePaymentTrigger** — Ensures the state-machine trigger and any resulting side-effects are atomically reverted if the invoice ends up in an invalid state.

## Example: Add a new SyncService method that fires a billing trigger and validates the post-trigger invoice state

```
func (s *Service) VoidInvoice(ctx context.Context, input appcustominvoicing.VoidInvoiceInput) (billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return billing.StandardInvoice{}, err
	}
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.StandardInvoice, error) {
		invoice, err := s.billingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: input.InvoiceID})
		if err != nil { return billing.StandardInvoice{}, err }
		if err := s.ValidateInvoiceApp(invoice); err != nil { return billing.StandardInvoice{}, err }
		err = s.billingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
			InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: input.InvoiceID, Trigger: billing.TriggerVoid},
			AppType:    app.AppTypeCustomInvoicing,
			Capability: app.CapabilityTypeCollectPayments,
		})
		if err != nil { return billing.StandardInvoice{}, err }
		invoice, err = s.billingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: input.InvoiceID})
// ...
```

<!-- archie:ai-end -->
