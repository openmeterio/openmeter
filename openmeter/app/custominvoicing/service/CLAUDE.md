# service

<!-- archie:ai-start -->

> Business logic layer for the custominvoicing app — implements appcustominvoicing.Service (factory + customer data + sync) by orchestrating appcustominvoicing.Adapter, app.Service, and billing.Service within explicit transactions.

## Patterns

**transaction.Run wrapping multi-step writes** — Operations that touch multiple entities (e.g. CreateApp: create app base + upsert config) run inside transaction.Run or transaction.RunWithNoValue to ensure atomicity. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.AppBase, error) { appBase, err := s.appService.CreateApp(...); s.adapter.UpsertAppConfiguration(...); return appBase, nil })`)
**Config struct with Validate() + compile-time assertion** — Service constructor takes a Config struct; Validate() checks all required fields. var _ appcustominvoicing.Service = (*Service)(nil) enforces interface at compile time. (`var _ appcustominvoicing.Service = (*Service)(nil)`)
**ValidateInvoiceApp guard before billing mutations** — sync.go's sync methods call s.ValidateInvoiceApp(invoice) before and after billing mutations to assert the invoice belongs to the custom invoicing app. (`if err := s.ValidateInvoiceApp(invoice); err != nil { return billing.StandardInvoice{}, err }`)
**input.Validate() at service method entry** — Every exported service method calls input.Validate() before any business logic or transaction. (`if err := input.Validate(); err != nil { return billing.StandardInvoice{}, err }`)
**Critical ValidationIssue check after state-machine triggers** — HandlePaymentTrigger re-reads the invoice post-trigger and returns a billing.ValidationError if any critical ValidationIssues are present — causing transaction rollback. (`criticalIssues := lo.Filter(invoice.ValidationIssues, func(i billing.ValidationIssue, _ int) bool { return i.Severity == billing.ValidationIssueSeverityCritical }); if len(criticalIssues) > 0 { return billing.StandardInvoice{}, billing.ValidationError{Err: criticalIssues.AsError()} }`)
**Metadata timestamping via clock.Now()** — SyncDraftInvoice and SyncIssuingInvoice inject AdditionalMetadata with clock.Now() timestamps — uses pkg/clock not time.Now() for testability. (`AdditionalMetadata: map[string]string{appcustominvoicing.MetadataKeyDraftSyncedAt: clock.Now().Format(time.RFC3339)}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct definition, Config, Validate, New constructor. Sets up the three dependencies: adapter, appService, billingService. | Service struct is exported (uppercase) unlike most sibling packages — this is intentional to allow direct use in Wire provider sets. |
| `sync.go` | SyncService implementation: SyncDraftInvoice, SyncIssuingInvoice, HandlePaymentTrigger, ValidateInvoiceApp. | HandlePaymentTrigger does get-trigger-get — second GetStandardInvoiceById is needed to observe post-trigger state; do not collapse into one read. |
| `factory.go` | FactoryService implementation: CreateApp (two-step: appService.CreateApp + adapter.UpsertAppConfiguration), DeleteApp, UpsertAppConfiguration, GetAppConfiguration. | CreateApp wraps both steps in a single transaction.Run — if adapter.UpsertAppConfiguration fails, the app row is rolled back. |
| `customerdata.go` | CustomerDataService implementation: thin delegation to adapter wrapped in transaction.Run/RunWithNoValue. | No business logic here — if validation is needed, it belongs in the input types, not this file. |

## Anti-Patterns

- Calling time.Now() directly instead of clock.Now() — breaks deterministic tests
- Skipping ValidateInvoiceApp before billing mutations — allows cross-app invoice mutations
- Not wrapping multi-entity writes in transaction.Run — risks partial writes on appService + adapter calls
- Returning an error for a NotFound from adapter.GetAppConfiguration — the adapter returns zero value; callers expect that
- Adding billing.Service calls directly in customerdata.go — customer data operations must stay adapter-only

## Decisions

- **Service struct is exported (uppercase)** — Allows Wire to inject *Service directly when both FactoryService and SyncService sub-interfaces are needed at different wiring sites.
- **ValidateInvoiceApp is a standalone exported method** — Used both internally (before triggers) and passed as a callback (InvoiceValidator) to billing.Service sync methods, requiring a stable method signature.
- **Critical-issue check causes transaction rollback in HandlePaymentTrigger** — Ensures the state-machine trigger and any resulting side-effects are atomically reverted if the invoice ends up in an invalid state.

## Example: Add a new SyncService method that fires a billing trigger and validates the result

```
func (s *Service) VoidInvoice(ctx context.Context, input appcustominvoicing.VoidInvoiceInput) (billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return billing.StandardInvoice{}, err
	}
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.StandardInvoice, error) {
		invoice, err := s.billingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: input.InvoiceID})
		if err != nil { return billing.StandardInvoice{}, err }
		if err := s.ValidateInvoiceApp(invoice); err != nil { return billing.StandardInvoice{}, err }
		if err := s.billingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
			InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: input.InvoiceID, Trigger: billing.TriggerVoid},
			AppType: app.AppTypeCustomInvoicing,
			Capability: app.CapabilityTypeCollectPayments,
		}); err != nil { return billing.StandardInvoice{}, err }
		invoice, err = s.billingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: input.InvoiceID})
		if err != nil { return billing.StandardInvoice{}, err }
// ...
```

<!-- archie:ai-end -->
