# sandbox

<!-- archie:ai-start -->

> Sandbox billing app — a no-op InvoicingApp for development and testing that simulates payment outcomes via invoice metadata, auto-provisions itself at namespace startup, and self-registers its MarketplaceListing at factory construction time.

## Patterns

**App embeds Meta embeds AppBase** — App embeds Meta (which embeds app.AppBase) plus injected services. Meta implements EventAppParser via FromEventAppData. Two-level embed separates base data from runtime behaviour. (`type App struct { Meta; billingService billing.Service }; type Meta struct { app.AppBase }`)
**Factory self-registers on construction** — NewFactory calls config.AppService.RegisterMarketplaceListing inside the constructor; a registration failure returns an error so the listing is always registered before any app instance. (`err := config.AppService.RegisterMarketplaceListing(ctx, app.RegistryItem{Listing: MarketplaceListing, Factory: fact})`)
**Payment simulation via invoice metadata key** — PostAdvanceStandardInvoiceHook reads invoice.Metadata[TargetPaymentStatusMetadataKey] to choose TriggerPaid/Failed/PaymentUncollectible/ActionRequired. Default is TriggerPaid. (`override, ok := invoice.Metadata[TargetPaymentStatusMetadataKey]; if ok && override != "" { targetStatus = override }`)
**AutoProvision idempotent helper** — AutoProvision lists existing sandbox apps and creates one only if none exist; called at server startup for the default namespace. (`if sandboxAppList.TotalCount == 0 { appBase, _ = input.AppService.CreateApp(ctx, ...) }`)
**MockableFactory for tests** — MockableFactory wraps Factory with an overrideFactory. EnableMock(t) injects a MockApp; DisableMock() restores the real factory. Expectations set via On<Method>() and asserted in Reset(t). (`mock := fact.EnableMock(t); mock.OnUpsertStandardInvoice(cb); defer fact.DisableMock()`)
**No-op customer data methods** — GetCustomerData, UpsertCustomerData, DeleteCustomerData all return nil/empty CustomerData{} — sandbox needs no external customer setup. (`func (a App) GetCustomerData(...) (app.CustomerData, error) { return CustomerData{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | App, Meta, CustomerData, Factory; Config/Validate; NewFactory; all billing.InvoicingApp methods; PostAdvanceStandardInvoiceHook. | FinalizeStandardInvoice calls GenerateInvoiceSequenceNumber with OM-SANDBOX prefix. UninstallApp is a no-op. InstallAppWithAPIKey creates the DB row then NewApp. |
| `mock.go` | MockApp (per-call stubs via mo.Option), mockAppInstance, MockableFactory, NewMockApp, NewMockableFactory. | MockApp.Reset calls AssertExpectations — unstaged expectations fail. mo.Option MustGet() panics if no expectation staged. Always Reset between cases. |
| `marketplace.go` | Singleton MarketplaceListing and Capability vars (CollectPayment, CalculateTax, InvoiceCustomer). | InstallMethods = [InstallMethodNoCredentials]. Factory also implements AppFactoryInstallWithAPIKey. |
| `helpers.go` | AutoProvision — idempotent sandbox provisioning called at server startup after namespace creation. | Returns the first sandbox app if multiple exist (non-deterministic ordering). Intended for single-sandbox-per-namespace use. |
| `errors.go` | ErrSimulatedPaymentFailure — a billing.ValidationError used for TargetPaymentStatusFailed simulations. | Exported so tests assert on it with errors.Is. |

## Anti-Patterns

- Storing actual payment or customer data in the sandbox — all customer data methods are intentionally no-ops
- Registering the sandbox listing outside NewFactory — the constructor owns the registration lifecycle
- Using MockApp without Reset(t) between test cases — unstaged mo.Option MustGet() panics
- Firing billing triggers other than the four defined constants in PostAdvanceStandardInvoiceHook

## Decisions

- **Payment outcome controlled by invoice metadata at hook time** — Allows per-invoice test configuration without changing app config, making it easy to test different payment paths in one suite without reinstalling the sandbox.
- **MockableFactory embeds real Factory** — Tests not needing mocking use the real factory path; EnableMock only overrides NewApp, so registration and uninstall logic stay tested through the real factory.

## Example: NewFactory: validate config, construct factory, self-register listing

```
func NewFactory(ctx context.Context, config Config) (*Factory, error) {
	if err := config.Validate(); err != nil { return nil, fmt.Errorf("failed to validate config: %w", err) }
	fact := &Factory{appService: config.AppService, billingService: config.BillingService}
	err := config.AppService.RegisterMarketplaceListing(ctx, app.RegistryItem{Listing: MarketplaceListing, Factory: fact})
	if err != nil { return nil, fmt.Errorf("failed to register marketplace listing: %w", err) }
	return fact, nil
}
```

<!-- archie:ai-end -->
