# service

<!-- archie:ai-start -->

> Service layer of the custom-invoicing app: creates/deletes the app and its sync-hook config (FactoryService), manages per-customer external data, and bridges external sync webhooks into billing (SyncService) by delegating to billing.Service while stamping metadata.

## Patterns

**Single Service implementing multiple domain interfaces** — One *Service satisfies appcustominvoicing.Service, FactoryService, and SyncService via compile-time asserts; behavior split across customerdata.go / factory.go / sync.go. (`var _ appcustominvoicing.SyncService = (*Service)(nil)`)
**Config-validated constructor with injected dependencies** — New(Config) requires Adapter, Logger, AppService, BillingService all non-nil (Config.Validate); no slog.Default fallback. Service holds adapter + appService + billingService. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**transaction.Run wraps multi-step writes** — Methods that combine app creation + config upsert (CreateApp) or read-trigger-reread (HandlePaymentTrigger) wrap the body in transaction.Run / RunWithNoValue against s.adapter so partial failures roll back. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.AppBase, error) { appBase, err := s.appService.CreateApp(...); ...; s.adapter.UpsertAppConfiguration(...) })`)
**Sync delegates to billing with metadata + validator injection** — SyncDraftInvoice/SyncIssuingInvoice call billing.Service.Sync*StandardInvoice, attaching AdditionalMetadata (MetadataKeyDraftSyncedAt/FinalizedAt via clock.Now()) and InvoiceValidator: s.ValidateInvoiceApp. (`return s.billingService.SyncDraftInvoice(ctx, billing.SyncDraftStandardInvoiceInput{InvoiceID: input.InvoiceID, UpsertInvoiceResults: input.UpsertInvoiceResults, AdditionalMetadata: map[string]string{...}, InvoiceValidator: s.ValidateInvoiceApp})`)
**Invoice ownership validation** — ValidateInvoiceApp asserts Workflow.Apps.Invoicing exists and GetType()==app.AppTypeCustomInvoicing before any sync/payment mutation, returning models.NewGenericValidationError otherwise. (`if invoice.Workflow.Apps.Invoicing.GetType() != app.AppTypeCustomInvoicing { return models.NewGenericValidationError(...) }`)
**Critical-issue rollback after trigger** — HandlePaymentTrigger re-reads the invoice after TriggerInvoice and, if any ValidationIssueSeverityCritical issues exist, returns billing.ValidationError to force a transaction rollback. (`criticalIssues := lo.Filter(invoice.ValidationIssues, func(issue billing.ValidationIssue, _ int) bool { return issue.Severity == billing.ValidationIssueSeverityCritical }); if len(criticalIssues) > 0 { return ..., billing.ValidationError{Err: criticalIssues.AsError()} }`)
**Validate inputs first** — Every public method starts with input.Validate() (CreateApp wraps as 'invalid input: %w') and returns before doing work. (`if err := input.Validate(); err != nil { return app.AppBase{}, fmt.Errorf("invalid input: %w", err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config/Validate, New; compile-time assert for appcustominvoicing.Service | All four deps mandatory; logger injected, never slog.Default |
| `factory.go` | CreateApp/DeleteApp/Upsert/GetAppConfiguration (FactoryService) | CreateApp must run app create + config upsert in one transaction.Run; uses app.AppTypeCustomInvoicing |
| `sync.go` | SyncDraftInvoice/SyncIssuingInvoice/HandlePaymentTrigger + ValidateInvoiceApp (SyncService) | Always pass InvoiceValidator: s.ValidateInvoiceApp; HandlePaymentTrigger re-reads invoice and rolls back on critical issues; TriggerInvoice uses CapabilityTypeCollectPayments |
| `customerdata.go` | Get/Upsert/Delete CustomerData delegating to adapter inside transaction.Run | Thin pass-throughs wrapped in transaction.Run/RunWithNoValue |

## Anti-Patterns

- Mutating an invoice via billing without first calling ValidateInvoiceApp
- Performing CreateApp's two writes (app + config) outside a single transaction.Run
- Swallowing critical ValidationIssues instead of returning billing.ValidationError to roll back
- Using slog.Default() or time.Now() directly instead of injected logger / pkg/clock.Now()
- Bypassing billing.Service and writing invoice state directly from this layer

## Decisions

- **Sync methods are thin wrappers over billing.Service with injected validator + metadata** — Billing owns invoice state machine; the app only stamps custom-invoicing metadata and enforces app-ownership
- **HandlePaymentTrigger re-reads and inspects ValidationIssues to decide rollback** — TriggerInvoice may produce critical issues without erroring; explicit re-read lets the service abort the tx
- **clock.Now() for sync timestamps** — Test-controllable time for deterministic metadata assertions

## Example: Drive a payment trigger through billing and roll back on critical validation issues

```
func (s *Service) HandlePaymentTrigger(ctx context.Context, input appcustominvoicing.HandlePaymentTriggerInput) (billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil { return billing.StandardInvoice{}, err }
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.StandardInvoice, error) {
		invoice, err := s.billingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: input.InvoiceID})
		if err != nil { return billing.StandardInvoice{}, err }
		if err := s.ValidateInvoiceApp(invoice); err != nil { return billing.StandardInvoice{}, err }
		err = s.billingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
			InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: input.InvoiceID, Trigger: input.Trigger},
			AppType: app.AppTypeCustomInvoicing, Capability: app.CapabilityTypeCollectPayments,
		})
		if err != nil { return billing.StandardInvoice{}, err }
		invoice, err = s.billingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{Invoice: input.InvoiceID})
		if err != nil { return billing.StandardInvoice{}, err }
		if len(invoice.ValidationIssues) > 0 {
			criticalIssues := lo.Filter(invoice.ValidationIssues, func(issue billing.ValidationIssue, _ int) bool { return issue.Severity == billing.ValidationIssueSeverityCritical })
// ...
```

<!-- archie:ai-end -->
