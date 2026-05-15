# sandbox

<!-- archie:ai-start -->

> Sandbox billing app — a no-op InvoicingApp used for development and testing that simulates payment outcomes via invoice metadata, auto-provisions itself at namespace startup, and self-registers its MarketplaceListing at factory construction time.

## Patterns

**App embeds Meta embeds AppBase** — App struct embeds Meta (which embeds app.AppBase) plus injected services. Meta holds only AppBase and implements EventAppParser via FromEventAppData. This two-level embed separates base data from runtime behaviour. (`type App struct { Meta; billingService billing.Service }; type Meta struct { app.AppBase }`)
**Factory self-registers on construction** — NewFactory calls config.AppService.RegisterMarketplaceListing inside the constructor. If registration fails (duplicate type) the factory returns an error, ensuring the listing is always registered before any app instance is created. (`err := config.AppService.RegisterMarketplaceListing(ctx, app.RegistryItem{Listing: MarketplaceListing, Factory: fact})`)
**Payment simulation via invoice metadata key** — PostAdvanceStandardInvoiceHook reads invoice.Metadata[TargetPaymentStatusMetadataKey] to decide which billing trigger to fire (TriggerPaid, TriggerFailed, TriggerPaymentUncollectible, TriggerActionRequired). Default is TriggerPaid. (`override, ok := invoice.Metadata[TargetPaymentStatusMetadataKey]; if ok && override != "" { targetStatus = override }`)
**AutoProvision idempotent helper** — AutoProvision in helpers.go lists existing sandbox apps and creates one only if none exist. Called at server startup to ensure a default sandbox app exists for the default namespace. (`if sandboxAppList.TotalCount == 0 { appBase, _ = input.AppService.CreateApp(ctx, ...) }`)
**MockableFactory for tests** — MockableFactory wraps Factory with an overrideFactory field. EnableMock(t) injects a MockApp as the factory; DisableMock() restores the real factory. Expectations are set via On<Method>() helpers and asserted in Reset(t). (`mock := fact.EnableMock(t); mock.OnUpsertStandardInvoice(cb); defer fact.DisableMock()`)
**No-op customer data methods** — GetCustomerData, UpsertCustomerData, DeleteCustomerData all return nil/empty CustomerData{}. This is intentional — sandbox requires no external customer setup. (`func (a App) GetCustomerData(...) (app.CustomerData, error) { return CustomerData{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | App, Meta, CustomerData, Factory structs; Config/Validate; NewFactory; all billing.InvoicingApp methods; PostAdvanceStandardInvoiceHook. | FinalizeStandardInvoice calls billingService.GenerateInvoiceSequenceNumber with InvoiceSequenceNumber (OM-SANDBOX prefix). UninstallApp is a no-op. InstallAppWithAPIKey creates the DB row then calls NewApp. |
| `mock.go` | MockApp (per-call response stubs via mo.Option), mockAppInstance (wraps MockApp as app.App), MockableFactory (toggles between real and mock), NewMockApp, NewMockableFactory. | MockApp.Reset calls AssertExpectations — unstaged expectations cause test failure. mo.Option MustGet() panics if no expectation was staged. Always call Reset between test cases. |
| `marketplace.go` | Singleton MarketplaceListing var and the three Capability vars (CollectPayment, CalculateTax, InvoiceCustomer) registered at startup. | InstallMethods = [InstallMethodNoCredentials]. Factory also implements AppFactoryInstallWithAPIKey. |
| `helpers.go` | AutoProvision — idempotent sandbox app provisioning called by server startup code after namespace creation. | Returns the first sandbox app found if multiple exist — ordering is non-deterministic. Intended for single-sandbox-per-namespace use. |
| `errors.go` | ErrSimulatedPaymentFailure — a billing.ValidationError used in PostAdvanceStandardInvoiceHook for TargetPaymentStatusFailed simulations. | Exported so tests can assert on it directly with errors.Is. |

## Anti-Patterns

- Storing actual payment or customer data in the sandbox — all customer data methods are intentionally no-ops.
- Registering the sandbox listing outside NewFactory — the constructor owns the registration lifecycle.
- Using MockApp without Reset(t) between test cases — unstaged mo.Option MustGet() panics.
- Calling billing triggers other than the four defined constants in PostAdvanceStandardInvoiceHook.

## Decisions

- **Payment outcome controlled by invoice metadata at hook time** — Allows per-invoice test configuration without changing app configuration, making it easy to test different payment paths in the same suite without reinstalling the sandbox app.
- **MockableFactory embeds real Factory** — Tests that don't need mocking use the real factory path; EnableMock only overrides NewApp, so registration and uninstall logic remain tested through the real factory.

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
