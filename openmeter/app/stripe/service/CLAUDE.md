# service

<!-- archie:ai-start -->

> Business logic service implementing appstripe.Service and app.AppFactory; orchestrates Stripe app install/uninstall lifecycle, webhook event handling, invoice state transitions, and checkout session creation. Delegates persistence to appstripe.Adapter and Stripe API calls to the client factories injected through the adapter.

## Patterns

**transaction.Run wrapping every service method** — Service methods that call the adapter must wrap in transaction.Run(ctx, s.adapter, func(ctx) ...) or transaction.RunWithNoValue. This ensures a single transaction spans the full operation. (`func (s *Service) GetWebhookSecret(ctx context.Context, input appstripe.GetWebhookSecretInput) (appstripe.GetWebhookSecretOutput, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripe.GetWebhookSecretOutput, error) {
		return s.adapter.GetWebhookSecret(ctx, input)
	})
}`)
**Marketplace self-registration in New()** — New() must call config.AppService.RegisterMarketplaceListing with the factory (Service itself) and the stripe marketplace listing. This self-registration is what makes the Stripe app type installable. Without it the Stripe app type is invisible to the marketplace registry. (`config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: appstripe.StripeMarketplaceListing, Factory: service})`)
**Config.Validate() in New() before construction** — All constructor dependencies are validated via Config.Validate() before constructing the Service. Return error from New() if validation fails — never panic. (`func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil { return nil, err }
	...
}`)
**publisher.Publish after adapter mutation in same transaction** — Side-effect events (checkout session created, payment setup succeeded) are published via s.publisher.Publish() inside the transaction.Run closure, after the adapter write succeeds. (`event := appstripe.NewAppCheckoutSessionEvent(ctx, input.Namespace, output.SessionID, output.AppID.ID, output.CustomerID.ID)
if err := s.publisher.Publish(ctx, event); err != nil { return appstripe.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to publish event: %w", err) }`)
**HandleInvoiceStateTransition idempotency guards** — Before triggering a billing state transition, check if the invoice is already in TargetStatuses (skip) or matches IgnoreInvoiceInStatus (skip). getInvoiceByStripeID returns nil (not error) when the Stripe invoice is unknown — callers must nil-check before proceeding. (`if slices.Contains(input.TargetStatuses, invoice.Status) { return nil } // already in target state
if invoice.Status.Matches(input.IgnoreInvoiceInStatus...) { return nil } // terminal state`)
**newApp combines AppBase with AppData** — After any adapter call that returns AppBase + AppData, call s.newApp(appBase, stripeApp) to produce the concrete appstripe.App struct with all injected service references. Never construct appstripe.App directly. (`app, err := s.newApp(stripeApp.AppBase, stripeApp.AppData)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config, New() constructor with marketplace self-registration. | DisableWebhookRegistration flag is only used in factory.go — do not reference it in other files. |
| `factory.go` | NewApp (app.AppFactory), InstallAppWithAPIKey, UninstallApp, newApp helper. | InstallAppWithAPIKey generates the app ID before creating secrets/webhook (three external systems: secret store, Stripe, DB). A TODO acknowledges no rollback exists; partial failures can leave orphaned secrets or webhooks. |
| `billing.go` | HandleInvoiceStateTransition, HandleInvoiceSentEvent, GetSupplierContact, getInvoiceByStripeID. | getInvoiceByStripeID returns nil (not error) when the Stripe invoice is unknown to OpenMeter — callers must check for nil before proceeding, not treat nil as an error. |
| `app.go` | Remaining appstripe.Service methods: GetWebhookSecret, UpdateAPIKey, CreateCheckoutSession, GetStripeAppData, GetStripeCustomerData, UpsertStripeCustomerData, DeleteStripeCustomerData, HandleSetupIntentSucceeded, CreatePortalSession. | generateMaskedSecretAPIKey is a local helper — keep masking format (first 8 chars + *** + last 3 chars) in sync with the UI display format. |
| `webhook.go` | WebhookURLGenerator implementations: baseURLWebhookURLGenerator and patternWebhookURLGenerator. | patternWebhookURLGenerator requires a %s placeholder in the pattern string; validated at construction time. |

## Anti-Patterns

- Calling s.adapter methods directly without transaction.Run — bypasses the transaction boundary and risks partial writes.
- Implementing HandleInvoiceStateTransition without idempotency guards (target-status check, ignore-status check) — causes duplicate billing state transitions.
- Not calling RegisterMarketplaceListing in New() — makes the Stripe app type invisible to the marketplace registry.
- Treating nil return from getInvoiceByStripeID as an error — nil means the invoice is not managed by this app, which is a valid expected case.
- Constructing appstripe.App directly instead of using s.newApp() — misses injected service references.

## Decisions

- **Service self-registers into the app marketplace in New() rather than at wire time.** — Keeps the registration co-located with the implementation; RegisterMarketplaceListing is idempotent so duplicate registrations are safe.
- **InstallAppWithAPIKey generates the app ID before creating secrets/webhook.** — Secrets must be tagged with AppID for retrieval; the ID must exist before any secret or webhook is created, so it is generated early with ulid.Make().
- **getInvoiceByStripeID returns nil rather than error for unknown Stripe invoices.** — Non-OpenMeter-initiated Stripe invoices are a normal case (e.g. manual Stripe invoices); treating them as errors would cause unnecessary noise in webhook processing.

## Example: New service method that writes to the adapter and publishes an event

```
func (s *Service) MyAction(ctx context.Context, input appstripe.MyInput) (appstripe.MyOutput, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripe.MyOutput, error) {
		out, err := s.adapter.MyAdapterOp(ctx, input)
		if err != nil {
			return appstripe.MyOutput{}, fmt.Errorf("my action: %w", err)
		}
		event := appstripe.NewMyActionEvent(ctx, input.Namespace, out.ID)
		if err := s.publisher.Publish(ctx, event); err != nil {
			return appstripe.MyOutput{}, fmt.Errorf("failed to publish event: %w", err)
		}
		return out, nil
	})
}
```

<!-- archie:ai-end -->
