# custominvoicing

<!-- archie:ai-start -->

> Webhook-driven invoicing app that lets external systems receive invoice payloads and async-confirm sync completion. Implements billing.InvoicingApp and billing.InvoicingAppAsyncSyncer; advance gates (CanDraftSyncAdvance, CanIssuingSyncAdvance) block state-machine progression until the external system stamps metadata on the invoice.

## Patterns

**Composite Service interface** — Service = CustomerDataService + FactoryService + SyncService. New capability areas add a sub-interface and embed it — never add methods directly to Service. (`type Service interface { CustomerDataService; FactoryService; SyncService }`)
**Advance-gate via metadata keys** — CanDraftSyncAdvance / CanIssuingSyncAdvance return (true, nil) when MetadataKeyDraftSyncedAt / MetadataKeyFinalizedAt are present in invoice.Metadata, or when the hook is disabled. New sync hooks must follow this exact pattern. (`if _, ok := invoice.Metadata[MetadataKeyDraftSyncedAt]; ok { return true, nil }`)
**Config.Validate() on every input struct** — All input types in sync.go, customerdata.go, factory.go implement Validate() returning models.NewNillableGenericValidationError(errors.Join(errs...)). Service methods call input.Validate() as the first statement. (`func (i SyncDraftInvoiceInput) Validate() error { return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Factory self-registers with app marketplace** — NewFactory calls config.AppService.RegisterMarketplaceListing with MarketplaceListing+Factory; omitting this makes the app type invisible. (`err := config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: MarketplaceListing, Factory: fact})`)
**App delegates to Service, never to Adapter directly** — App methods (GetCustomerData, UpsertCustomerData, DeleteCustomerData, UpdateAppConfig) call a.customInvoicingService.* — the App struct never imports or calls the adapter package. (`func (a App) GetCustomerData(ctx context.Context, input app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) { return a.customInvoicingService.GetCustomerData(...) }`)
**Compile-time interface assertions at package level** — var _ customerapp.App = (*App)(nil); var _ billing.InvoicingApp = (*App)(nil); var _ billing.InvoicingAppAsyncSyncer = (*App)(nil) — one assertion per satisfied interface in app.go. (`var _ billing.InvoicingApp = (*App)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | Defines App struct, Meta (AppBase + Configuration), compile-time assertions, and all InvoicingApp + InvoicingAppAsyncSyncer methods. UpsertStandardInvoice is a no-op; FinalizeStandardInvoice generates a non-draft invoice number only after CanIssuingSyncAdvance passes. | FinalizeStandardInvoice calls billingService.GenerateInvoiceSequenceNumber only when CanIssuingSyncAdvance returns true AND the number still has a draft prefix — both guards must stay. |
| `service.go` | Declares Service interface composed of three sub-interfaces. Source of truth for what business operations the package exposes. | Do not add methods directly here — embed a new sub-interface instead. |
| `sync.go` | Input types for SyncService methods: SyncDraftInvoiceInput, SyncIssuingInvoiceInput, HandlePaymentTriggerInput. All validate billing.InvoiceID and required pointer fields. | UpsertInvoiceResults and FinalizeInvoiceResult are pointers and required — validate they are non-nil. |
| `factory.go` | Factory struct + NewFactory (registers marketplace listing), InstallApp, UninstallApp, NewApp (fetches configuration from service). MarketplaceListing var defines capabilities. | NewFactory must call RegisterMarketplaceListing or the app type is invisible to the marketplace. InstallApp must call NewApp after CreateApp to return a fully wired App. |
| `customerdata.go` | CustomerData type (implements app.CustomerData), per-field input types with Validate(). App receiver methods bridge app.CustomerData interface to appcustominvoicing.Service. | CustomerData.Validate() is a no-op today; never add billing.Service calls here — customer data operations must stay service-only. |

## Anti-Patterns

- Calling billing.Service directly from App methods in customerdata.go — customer data operations must stay adapter-only via Service.
- Returning an error for NotFound from GetAppConfiguration — adapter returns zero-value Configuration; callers expect that.
- Skipping input.Validate() at the start of any SyncService method.
- Adding new sync hook logic without a corresponding CanXxxAdvance metadata-key gate and Configuration bool.
- Embedding app.InvoicingApp methods that are no-ops (ValidateStandardInvoice, UpsertStandardInvoice, DeleteStandardInvoice) with real side-effects — they must remain no-ops.

## Decisions

- **InvoicingApp methods are mostly no-ops; meaningful logic flows through SyncService HTTP endpoints.** — External systems drive invoice lifecycle via webhook callbacks to the httpdriver endpoints, not via the state machine hooks. Only FinalizeStandardInvoice generates the non-draft invoice number as a final step.
- **Advance gates (CanDraftSyncAdvance, CanIssuingSyncAdvance) check invoice metadata rather than a DB flag.** — Metadata is already on the in-memory invoice passed to FinalizeStandardInvoice, avoiding an extra DB round-trip and keeping the gate logic stateless.

## Example: Implement a new sync hook: block issuing advance until external system stamps metadata

```
// In app.go — add a new metadata key constant and gate method:
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
