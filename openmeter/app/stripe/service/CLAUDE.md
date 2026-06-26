# service

<!-- archie:ai-start -->

> Service layer for the Stripe app (package appservice). Wraps the adapter in transactions, emits eventbus events, implements the app.AppFactory (install/uninstall/NewApp) and the marketplace registration, and bridges Stripe webhooks into billing state transitions.

## Patterns

**transaction.Run / RunWithNoValue wrapping the adapter** — Most Service methods are thin transactional wrappers: transaction.Run(ctx, s.adapter, func(ctx)...) delegating to s.adapter. Post-commit side effects (event publish) happen inside the same closure. (`func (s *Service) GetStripeAppData(ctx, input) (appstripe.AppData, error) { return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripe.AppData, error) { return s.adapter.GetStripeAppData(ctx, input) }) }`)
**Event emission after adapter writes** — Successful operations publish domain events via s.publisher.Publish — appstripe.NewAppCheckoutSessionEvent on checkout, app.CustomerPaymentSetupSucceededEvent on setup-intent success (metadata stripped of reserved keys via lo.OmitByKeys). (`event := appstripe.NewAppCheckoutSessionEvent(ctx, input.Namespace, output.SessionID, output.AppID.ID, output.CustomerID.ID); s.publisher.Publish(ctx, event)`)
**AppFactory implementation + marketplace self-registration** — Service implements app.AppFactory (var _ app.AppFactory = (*Service)(nil)) with NewApp/InstallAppWithAPIKey/UninstallApp; New() calls AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: appstripe.StripeMarketplaceListing, Factory: service}). (`config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: appstripe.StripeMarketplaceListing, Factory: service})`)
**Config + New with full dependency validation** — Config.Validate() requires every dependency (Adapter, AppService, SecretService, BillingService, Logger, Publisher, WebhookURLGenerator) be non-nil; logger is injected, never slog.Default(). (`if c.WebhookURLGenerator == nil { return errors.New("webhook url generator cannot be null") }`)
**Webhook->billing bridge via HandleInvoiceStateTransition** — billing.go resolves the local invoice by Stripe external ID, applies status/ignore guards, optionally re-fetches the Stripe invoice (ShouldTriggerOnEvent/GetValidationErrors), then calls billingService.TriggerInvoice with the mapped trigger and capability CollectPayments. (`s.billingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{InvoiceTriggerInput: billing.InvoiceTriggerInput{Invoice: invoice.GetInvoiceID(), Trigger: input.Trigger, ValidationErrors: validationErrors}, AppType: app.AppTypeStripe, Capability: app.CapabilityTypeCollectPayments})`)
**External-ID invoice lookup with cardinality guards** — getInvoiceByStripeID lists StandardInvoices filtered by InvoicingExternalIDType; returns nil (non-managed) when empty, errors on >1, and verifies AppReferences.Invoicing.ID matches the app. (`if len(invoices.Items) == 0 { return nil, nil } if len(invoices.Items) > 1 { return nil, fmt.Errorf(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config/Validate, New + marketplace registration | New deps must be added to Config, Validate, and the struct; registration failure returns the service AND an error (caller decides). Publisher and WebhookURLGenerator are mandatory. |
| `app.go` | core app operations (checkout, customer data, setup-intent, portal, API key, masked key) | generateMaskedSecretAPIKey slices [:8] and [len-3:] — will panic on short keys; callers must pass real Stripe keys. Event publish is inside the tx closure. |
| `billing.go` | webhook->billing bridge: HandleInvoiceStateTransition, HandleInvoiceSentEvent, getInvoiceByStripeID, stripeErrorToValidationError | Returns nil (no-op) for non-managed invoices; relies on guard ordering (TargetStatuses already-set check, IgnoreInvoiceInStatus, ShouldTriggerOnEvent) before TriggerInvoice. |
| `factory.go` | app.AppFactory: NewApp, InstallAppWithAPIKey, UninstallApp, newApp | Install coordinates secret store + Stripe + DB and is explicitly NOT transactional (commented TODO). disableWebhookRegistration injects a fake secret for dev. Uninstall tolerates SecretNotFoundError and provider-auth errors. |
| `webhook.go` | app.WebhookURLGenerator implementations (baseURL and pattern variants) | patternWebhookURLGenerator requires the pattern to contain %s; baseURL variant joins /api/v1/apps/{id}/stripe/webhook — keep the path in sync with the router and Stripe webhook registration. |
| `const.go` | log attribute name constants for invoice/stripe IDs | Used in s.logger.With(...) for structured logging in billing.go. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run/RunWithNoValue when atomicity or event emission is required.
- Triggering a billing state transition without first resolving the managed invoice via getInvoiceByStripeID (non-managed invoices must be skipped).
- Using slog.Default() instead of the injected s.logger.
- Adding a Service dependency without updating Config, Config.Validate, and the struct together.
- Assuming InstallAppWithAPIKey is transactional — secret/webhook/db steps are independent remote calls.

## Decisions

- **Service self-registers in the marketplace inside New()** — Wires the Stripe AppFactory into the app registry at construction so app.Service can install Stripe apps; failure is surfaced to the caller.
- **Webhook events are translated to billing triggers with explicit guard sets per event** — Stripe delivers events at-least-once and out-of-order; status/ignore/ShouldTrigger guards make state transitions idempotent and ordering-safe.
- **InstallAppWithAPIKey is intentionally non-transactional** — Coordinating three remote services (secret store, Stripe API, DB) in one transaction is not feasible; the code documents this trade-off.

## Example: Transactional adapter wrap plus event emission

```
func (s *Service) CreateCheckoutSession(ctx context.Context, input appstripe.CreateCheckoutSessionInput) (appstripe.CreateCheckoutSessionOutput, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripe.CreateCheckoutSessionOutput, error) {
		output, err := s.adapter.CreateCheckoutSession(ctx, input)
		if err != nil { return appstripe.CreateCheckoutSessionOutput{}, err }
		event := appstripe.NewAppCheckoutSessionEvent(ctx, input.Namespace, output.SessionID, output.AppID.ID, output.CustomerID.ID)
		if err := s.publisher.Publish(ctx, event); err != nil {
			return appstripe.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to publish event: %w", err)
		}
		return output, nil
	})
}
```

<!-- archie:ai-end -->
