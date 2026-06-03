# custominvoicing

<!-- archie:ai-start -->

> Webhook-driven marketplace billing app: the App type (this folder) implements billing.InvoicingApp + billing.InvoicingAppAsyncSyncer, deferring meaningful invoice logic to external systems via the httpdriver/ HTTP callbacks and appcustominvoicing.Service (service/), with persistence in adapter/. Primary constraint: invoice state-machine progression is gated on external metadata stamps rather than DB flags.

## Patterns

**Composite Service interface** — Service = CustomerDataService + FactoryService + SyncService. New capability areas add a sub-interface and embed it — never add methods directly to Service. (`type Service interface { CustomerDataService; FactoryService; SyncService }`)
**Advance-gate via metadata keys** — CanDraftSyncAdvance/CanIssuingSyncAdvance return (true, nil) when MetadataKeyDraftSyncedAt/MetadataKeyFinalizedAt are present in invoice.Metadata, or when the hook is disabled via Configuration bool. New sync hooks follow this exact two-check pattern. (`if !a.Configuration.EnableDraftSyncHook { return true, nil }
if _, ok := invoice.Metadata[MetadataKeyDraftSyncedAt]; ok { return true, nil }`)
**Validate() on every input struct** — Input types implement Validate() returning models.NewNillableGenericValidationError(errors.Join(errs...)); Service/Sync methods call input.Validate() as the first statement. (`func (i SyncDraftInvoiceInput) Validate() error { return models.NewNillableGenericValidationError(errors.Join(errs...)) }`)
**Factory self-registers with the app marketplace** — NewFactory calls config.AppService.RegisterMarketplaceListing(ctx, app.RegistryItem{Listing: MarketplaceListing, Factory: fact}) during construction; omitting this makes the app type invisible to the marketplace. (`err := config.AppService.RegisterMarketplaceListing(ctx, app.RegistryItem{Listing: MarketplaceListing, Factory: fact})`)
**App delegates to Service, never to Adapter directly** — App receiver methods (GetCustomerData, UpsertCustomerData, DeleteCustomerData, UpdateAppConfig) call a.customInvoicingService.*; App never imports the adapter package. (`func (a App) GetCustomerData(ctx, input) (app.CustomerData, error) { return a.customInvoicingService.GetCustomerData(ctx, ...) }`)
**Compile-time interface assertions at package level** — var _ customerapp.App = (*App)(nil); var _ billing.InvoicingApp = (*App)(nil); var _ billing.InvoicingAppAsyncSyncer = (*App)(nil) — one assertion per satisfied interface in app.go. (`var _ billing.InvoicingApp = (*App)(nil)`)
**InvoicingApp methods are mostly no-ops** — ValidateStandardInvoice, UpsertStandardInvoice, DeleteStandardInvoice are intentional no-ops; only FinalizeStandardInvoice generates a non-draft number after CanIssuingSyncAdvance passes AND the draft prefix still matches. (`func (a App) UpsertStandardInvoice(ctx, invoice) (*billing.UpsertStandardInvoiceResult, error) { return nil, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | App struct, Meta (AppBase + Configuration), compile-time assertions, and all InvoicingApp + InvoicingAppAsyncSyncer methods. | FinalizeStandardInvoice must keep both guards: CanIssuingSyncAdvance check AND DraftInvoiceSequenceNumber.PrefixMatches before GenerateInvoiceSequenceNumber. |
| `service.go` | Declares Service = CustomerDataService + FactoryService + SyncService — source of truth for exposed business operations. | Do not add methods directly here — embed a new sub-interface instead. |
| `sync.go` | SyncService input types (SyncDraftInvoiceInput, SyncIssuingInvoiceInput, HandlePaymentTriggerInput) with Validate(). | UpsertInvoiceResults and FinalizeInvoiceResult are required pointer fields — validate non-nil. |
| `factory.go` | Factory + NewFactory (registers marketplace listing), InstallApp, UninstallApp, NewApp; MarketplaceListing defines capabilities. | NewFactory must call RegisterMarketplaceListing; InstallApp must call NewApp after CreateApp to return a fully wired App. |
| `customerdata.go` | CustomerData type (implements app.CustomerData) and App bridge methods to appcustominvoicing.Service. | Never add billing.Service calls here — customer data operations must stay service-only. |
| `adapter.go` | Adapter = CustomerDataAdapter + AppConfigAdapter + TxCreator. Implemented by adapter/ with soft-delete and upserts. | GetAppConfiguration/GetCustomerData return zero-value on NotFound — callers expect that, not an error. |

## Anti-Patterns

- Calling billing.Service directly from App methods — all operations must go through appcustominvoicing.Service.
- Returning an error for NotFound from GetAppConfiguration — the adapter returns zero-value Configuration; callers expect that.
- Skipping input.Validate() at the start of any SyncService method.
- Adding new sync-hook logic without a corresponding CanXxxAdvance metadata-key gate and Configuration bool.
- Giving the no-op InvoicingApp methods (ValidateStandardInvoice, UpsertStandardInvoice, DeleteStandardInvoice) real side-effects — they must remain no-ops.

## Decisions

- **InvoicingApp methods are mostly no-ops; meaningful logic flows through SyncService HTTP endpoints.** — External systems drive invoice lifecycle via webhook callbacks to httpdriver endpoints, not via state-machine hooks; only FinalizeStandardInvoice generates the non-draft number as a final step.
- **Advance gates check invoice metadata rather than a DB flag.** — Metadata is already on the in-memory invoice passed to FinalizeStandardInvoice, avoiding an extra DB round-trip and keeping the gate stateless.

## Example: Add a new sync hook that blocks advance until an external system stamps metadata

```
const MetadataKeyIssuingSyncedAt = "openmeter.io/custominvoicing/issuing-synced-at"

func (a App) CanIssuingSyncAdvance(invoice billing.StandardInvoice) (bool, error) {
  if !a.Configuration.EnableIssuingSyncHook { return true, nil }
  if invoice.Metadata == nil { return false, nil }
  if _, ok := invoice.Metadata[MetadataKeyIssuingSyncedAt]; ok { return true, nil }
  return false, nil
}
```

<!-- archie:ai-end -->
