# sandbox

<!-- archie:ai-start -->

> Reference/test app implementation of the app framework: a no-credentials Sandbox app that satisfies customerapp.App and billing.InvoicingApp (+PostAdvanceHook) to let OpenMeter run invoicing end-to-end without external integrations. Also provides AutoProvision and a mockable factory for tests.

## Patterns

**Compile-time interface assertions** — App and CustomerData are pinned to their contracts with var _ blocks: customerapp.App, billing.InvoicingApp, billing.InvoicingAppPostAdvanceHook, app.CustomerData, app.EventAppParser. Adding/removing methods must keep these satisfied. (`var _ billing.InvoicingApp = (*App)(nil)`)
**Factory registers itself in the marketplace** — NewFactory(config) validates config then calls config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: MarketplaceListing, Factory: fact}); the Factory implements NewApp/InstallAppWithAPIKey/UninstallApp. (`config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: MarketplaceListing, Factory: fact})`)
**Invoicing hooks delegate to billing.Service helpers** — FinalizeStandardInvoice generates a number via billingService.GenerateInvoiceSequenceNumber(...) with the package-level InvoiceSequenceNumber definition; results are built with billing.NewFinalize/UpsertStandardInvoiceResult fluent setters. (`billing.NewFinalizeStandardInvoiceResult().SetInvoiceNumber(invoiceNumber).SetSentToCustomerAt(clock.Now())`)
**Simulated payment via metadata-driven triggers** — PostAdvanceStandardInvoiceHook only acts on PaymentProcessingPending, reads TargetPaymentStatusMetadataKey from invoice metadata, and returns out.InvokeTrigger(billing.InvoiceTriggerInput{...}) with TriggerPaid/Failed/Uncollectible/ActionRequired. (`out.InvokeTrigger(billing.InvoiceTriggerInput{Invoice: invoice.GetInvoiceID(), Trigger: billing.TriggerPaid})`)
**Mockable factory wrapping the real one** — MockableFactory embeds *Factory and overrideFactory; EnableMock(t) swaps NewApp to return a recording mockAppInstance; MockApp uses mo.Option fields + AssertExpectations for call verification. (`fact.EnableMock(t) returns *MockApp; NewApp delegates to overrideFactory when set`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | Meta (embeds app.AppBase), App, CustomerData, Factory, Config; all invoicing/customer-data methods; PostAdvanceStandardInvoiceHook payment simulation | Most customer-data/invoice methods are intentional no-ops returning empty results; FinalizeStandardInvoice needs a non-nil billingService |
| `marketplace.go` | MarketplaceListing + Collect/CalculateTax/InvoiceCustomer capabilities; InstallMethodNoCredentials | Capabilities listed here must match what ValidateCapabilities enforces in ValidateCustomer |
| `helpers.go` | AutoProvision + AutoProvisionInput: installs a default Sandbox app on first run if none exists | AutoProvision returns the first existing sandbox app when one is present; do not assume it always creates |
| `errors.go` | ErrSimulatedPaymentFailure as a billing.NewValidationError | Used as the validation error attached to TriggerFailed in the post-advance hook |
| `mock.go` | MockApp, mockAppInstance, MockableFactory, NewMockableFactory, MockWithAppType | MockApp methods use mo.Option.MustGet() and panic if a response was not set via On*; Reset(t) asserts expectations and clears state |
| `config.go` | Empty Configuration{} with no-op Validate (sandbox has no config) | Update flows still pass appsandbox.Configuration{} as AppConfigUpdate |

## Anti-Patterns

- Breaking a var _ interface assertion by changing a method signature without updating the contract
- Adding real external calls to the sandbox app (it must run with InstallMethodNoCredentials and no credentials)
- Forgetting to register the listing in NewFactory so the app type is unknown to the registry
- Returning payment triggers from PostAdvanceStandardInvoiceHook when status is not PaymentProcessingPending

## Decisions

- **Sandbox is auto-provisioned by default at namespace setup** — Lets users exercise billing/invoicing immediately without configuring Stripe or another external provider
- **Payment outcome is controllable via invoice metadata** — Enables unit and customer tests to deterministically simulate paid/failed/uncollectible/action-required flows through the real advance machinery

## Example: Self-registering factory implementing the app plugin contract

```
func NewFactory(config Config) (*Factory, error) {
	if err := config.Validate(); err != nil { return nil, err }
	fact := &Factory{appService: config.AppService, billingService: config.BillingService}
	if err := config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: MarketplaceListing, Factory: fact}); err != nil {
		return nil, fmt.Errorf("failed to register marketplace listing: %w", err)
	}
	return fact, nil
}
```

<!-- archie:ai-end -->
