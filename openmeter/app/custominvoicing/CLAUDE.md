# custominvoicing

<!-- archie:ai-start -->

> Webhook-driven invoicing app that lets external systems receive invoice payloads and async-confirm sync completion. Implements billing.InvoicingApp and billing.InvoicingAppAsyncSyncer; advance gates (CanDraftSyncAdvance, CanIssuingSyncAdvance) block state-machine progression until the external system stamps metadata keys on the invoice.

## Patterns

**Composite Service interface** — Service = CustomerDataService + FactoryService + SyncService. New capability areas add a sub-interface and embed it in Service — never add methods directly to the Service interface. (`type Service interface { CustomerDataService; FactoryService; SyncService }`)
**Advance-gate via metadata keys** — CanDraftSyncAdvance/CanIssuingSyncAdvance return (true, nil) when MetadataKeyDraftSyncedAt/MetadataKeyFinalizedAt are present in invoice.Metadata, or when the hook is disabled via Configuration bool. New sync hooks must follow this exact two-check pattern. (`if _, ok := invoice.Metadata[MetadataKeyDraftSyncedAt]; ok { return true, nil }`)
**Config.Validate() on every input struct** — All input types implement Validate() returning models.NewNillableGenericValidationError(errors.Join(errs...)). Service methods call input.Validate() as the first statement. (`func (i SyncDraftInvoiceInput) Validate() error { return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Factory self-registers with app marketplace** — NewFactory calls config.AppService.RegisterMarketplaceListing with MarketplaceListing+Factory during construction; omitting this makes the app type invisible to the marketplace. (`err := config.AppService.RegisterMarketplaceListing(ctx, app.RegistryItem{Listing: MarketplaceListing, Factory: fact})`)
**App delegates to Service, never to Adapter directly** — App receiver methods (GetCustomerData, UpsertCustomerData, DeleteCustomerData, UpdateAppConfig) call a.customInvoicingService.* — App never imports or calls the adapter package. (`func (a App) GetCustomerData(ctx context.Context, input app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) { return a.customInvoicingService.GetCustomerData(...) }`)
**Compile-time interface assertions at package level** — var _ customerapp.App = (*App)(nil); var _ billing.InvoicingApp = (*App)(nil); var _ billing.InvoicingAppAsyncSyncer = (*App)(nil) — one assertion per satisfied interface in app.go. (`var _ billing.InvoicingApp = (*App)(nil)`)
**InvoicingApp methods are mostly no-ops** — ValidateStandardInvoice, UpsertStandardInvoice, and DeleteStandardInvoice are intentional no-ops; meaningful invoice logic flows through the SyncService HTTP endpoints. Only FinalizeStandardInvoice generates a non-draft invoice number after CanIssuingSyncAdvance passes. (`func (a App) UpsertStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) { return nil, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | App struct, Meta (AppBase + Configuration), compile-time assertions, and all InvoicingApp + InvoicingAppAsyncSyncer methods. FinalizeStandardInvoice generates invoice number only when CanIssuingSyncAdvance passes AND the number still has a draft prefix. | Both guards in FinalizeStandardInvoice must stay: CanIssuingSyncAdvance check AND draft-prefix check before calling GenerateInvoiceSequenceNumber. |
| `service.go` | Declares Service interface composed of three sub-interfaces. Source of truth for what business operations the package exposes. | Do not add methods directly here — embed a new sub-interface instead. |
| `sync.go` | Input types for SyncService methods: SyncDraftInvoiceInput, SyncIssuingInvoiceInput, HandlePaymentTriggerInput. All validate billing.InvoiceID and required pointer fields. | UpsertInvoiceResults and FinalizeInvoiceResult are pointer fields and required — validate they are non-nil. |
| `factory.go` | Factory struct + NewFactory (registers marketplace listing), InstallApp, UninstallApp, NewApp. MarketplaceListing var defines capabilities. | NewFactory must call RegisterMarketplaceListing or the app type is invisible. InstallApp must call NewApp after CreateApp to return a fully wired App. |
| `customerdata.go` | CustomerData type (implements app.CustomerData) and App receiver bridge methods to appcustominvoicing.Service. | Never add billing.Service calls here — customer data operations must stay service-only via customInvoicingService. |

## Anti-Patterns

- Calling billing.Service directly from App methods — all operations must go through appcustominvoicing.Service.
- Returning an error for NotFound from GetAppConfiguration — adapter returns zero-value Configuration; callers expect that.
- Skipping input.Validate() at the start of any SyncService method.
- Adding new sync hook logic without a corresponding CanXxxAdvance metadata-key gate and Configuration bool.
- Embedding InvoicingApp no-op methods (ValidateStandardInvoice, UpsertStandardInvoice, DeleteStandardInvoice) with real side-effects — they must remain no-ops.

## Decisions

- **InvoicingApp methods are mostly no-ops; meaningful logic flows through SyncService HTTP endpoints.** — External systems drive invoice lifecycle via webhook callbacks to httpdriver endpoints, not via state machine hooks. Only FinalizeStandardInvoice generates the non-draft invoice number as a final step.
- **Advance gates (CanDraftSyncAdvance, CanIssuingSyncAdvance) check invoice metadata rather than a DB flag.** — Metadata is already on the in-memory invoice passed to FinalizeStandardInvoice, avoiding an extra DB round-trip and keeping the gate logic stateless.

## Example: Add a new sync hook that blocks advance until external system stamps metadata

```
// In app.go — add metadata key constant and gate method:
const MetadataKeyIssuingSyncedAt = "openmeter.io/custominvoicing/issuing-synced-at"

func (a App) CanIssuingSyncAdvance(invoice billing.StandardInvoice) (bool, error) {
    if !a.Configuration.EnableIssuingSyncHook {
        return true, nil
    }
    if invoice.Metadata == nil {
        return false, nil
    }
    if _, ok := invoice.Metadata[MetadataKeyIssuingSyncedAt]; ok {
        return true, nil
    }
    return false, nil
}
```

<!-- archie:ai-end -->
