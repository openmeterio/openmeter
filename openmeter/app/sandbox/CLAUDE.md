# sandbox

<!-- archie:ai-start -->

> Sandbox billing app — a no-op InvoicingApp used for development and testing that simulates payment outcomes via invoice metadata, auto-provisions itself at namespace startup, and registers its MarketplaceListing at factory construction time.

## Patterns

**App embeds Meta embeds AppBase** — App struct embeds Meta (which embeds app.AppBase) plus injected services. Meta holds only AppBase and implements EventAppParser (FromEventAppData). This two-level embed keeps base data separate from runtime behaviour. (`type App struct { Meta; billingService billing.Service }`)
**Factory self-registers on construction** — NewFactory calls config.AppService.RegisterMarketplaceListing inside the constructor. If registration fails (duplicate type), New returns an error. This ensures the listing is always registered before any app instance is created. (`err := config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: MarketplaceListing, Factory: fact})`)
**Payment simulation via metadata key** — PostAdvanceStandardInvoiceHook reads invoice.Metadata[TargetPaymentStatusMetadataKey] to decide which billing trigger to fire (TriggerPaid, TriggerFailed, TriggerPaymentUncollectible, TriggerActionRequired). Default is paid. (`override, ok := invoice.Metadata[TargetPaymentStatusMetadataKey]; if ok { targetStatus = override }`)
**AutoProvision idempotent helper** — AutoProvision in helpers.go lists existing sandbox apps and creates one only if none exist. Called at server startup to ensure a default sandbox app exists for the default namespace. (`if sandboxAppList.TotalCount == 0 { appBase, _ = input.AppService.CreateApp(ctx, ...) }`)
**MockableFactory for tests** — MockableFactory wraps Factory with an overrideFactory field. EnableMock(t) injects a MockApp as the factory; DisableMock() restores the real factory. Use NewMockableFactory in tests to get both real and mock behaviour. (`mock := fact.EnableMock(t); mock.OnUpsertStandardInvoice(cb); defer fact.DisableMock()`)
**No-op customer data methods** — GetCustomerData, UpsertCustomerData, DeleteCustomerData all return nil / empty CustomerData{}. CustomerData{} implements app.CustomerData with a no-op Validate(). This is intentional — sandbox requires no external customer setup. (`func (a App) GetCustomerData(...) (app.CustomerData, error) { return CustomerData{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | App struct, Meta struct, CustomerData struct, Factory struct, Config/Validate, NewFactory, NewApp, InstallAppWithAPIKey, UninstallApp, all billing.InvoicingApp methods, PostAdvanceStandardInvoiceHook. | FinalizeStandardInvoice calls billingService.GenerateInvoiceSequenceNumber with InvoiceSequenceNumber (OM-SANDBOX prefix). UninstallApp is a no-op. InstallAppWithAPIKey creates the app DB row then calls NewApp. |
| `mock.go` | MockApp (per-call response stubs with mo.Option), mockAppInstance (wraps MockApp into app.App interface), MockableFactory (toggles between real and mock), NewMockApp, NewMockableFactory. | MockApp.Reset calls AssertExpectations — any staged expectation not consumed causes a test failure. Expectations are set via On<Method>() helpers. The mock stubs use MustGet() which panics if no expectation was staged. |
| `marketplace.go` | Declares the singleton MarketplaceListing var and the three Capability vars (CollectPayment, CalculateTax, InvoiceCustomer) that are registered at startup. | MarketplaceListing.InstallMethods = [InstallMethodNoCredentials] — this is the only method supported; the adapter type-asserts to AppFactoryInstallWithAPIKey which this factory also satisfies. |
| `helpers.go` | AutoProvision — idempotent sandbox app provisioning. Called by server startup code after namespace creation. | Returns the first sandbox app found if multiple exist — ordering is non-deterministic. Meant for single-sandbox-per-namespace use. |
| `errors.go` | ErrSimulatedPaymentFailure — a billing.ValidationError used in PostAdvanceStandardInvoiceHook for TargetPaymentStatusFailed simulations. | This is exported so tests can assert on it directly. |

## Anti-Patterns

- Storing actual payment or customer data in the sandbox — all customer data methods are intentionally no-ops.
- Registering the sandbox listing outside NewFactory — the constructor owns the registration lifecycle.
- Using MockApp directly without Reset(t) between test cases — unstaged mo.Option MustGet() panics.
- Calling billing triggers other than the four defined constants in PostAdvanceStandardInvoiceHook — other triggers are not sandboxed.

## Decisions

- **Payment outcome controlled by invoice metadata at hook time** — Allows per-invoice test configuration without changing app configuration, making it easy to test different payment paths in the same test suite without reinstalling the sandbox app.
- **MockableFactory embeds real Factory** — Tests that don't need mocking can use the real factory path; EnableMock only overrides NewApp, so registration and uninstall logic remain tested through the real factory.

## Example: NewFactory: register listing and validate config

```
func NewFactory(config Config) (*Factory, error) {
	if err := config.Validate(); err != nil { return nil, err }
	fact := &Factory{appService: config.AppService, billingService: config.BillingService}
	err := config.AppService.RegisterMarketplaceListing(app.RegistryItem{
		Listing: MarketplaceListing,
		Factory: fact,
	})
	if err != nil { return nil, fmt.Errorf("failed to register marketplace listing: %w", err) }
	return fact, nil
}
```

<!-- archie:ai-end -->
