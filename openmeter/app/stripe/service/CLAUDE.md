# service

<!-- archie:ai-start -->

> Business logic service implementing appstripe.Service and app.AppFactory; orchestrates Stripe app install/uninstall lifecycle, webhook event handling, invoice state transitions, and checkout session creation. Delegates persistence to appstripe.Adapter and Stripe API calls to the client factories injected through the adapter.

## Patterns

**transaction.Run wrapping every service method** — Service methods that call the adapter must wrap in transaction.Run(ctx, s.adapter, func(ctx) ...) or transaction.RunWithNoValue. This ensures a single transaction spans the operation. (`func (s *Service) GetWebhookSecret(...) (..., error) { return transaction.Run(ctx, s.adapter, func(ctx context.Context) (..., error) { return s.adapter.GetWebhookSecret(ctx, input) }) }`)
**Marketplace registration in New()** — New() must call config.AppService.RegisterMarketplaceListing with the factory (Service itself) and the stripe marketplace listing. This self-registration is what makes the Stripe app type installable. (`config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: appstripe.StripeMarketplaceListing, Factory: service})`)
**Config.Validate() then New()** — All constructor dependencies are validated via Config.Validate() before constructing the Service. Return error from New() if validation fails — never panic. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Publisher.Publish after adapter mutation** — Side-effect events (checkout session created, payment setup succeeded) are published via s.publisher.Publish() inside the transaction.Run closure, after the adapter write succeeds. (`event := appstripe.NewAppCheckoutSessionEvent(ctx, ...); if err := s.publisher.Publish(ctx, event); err != nil { return ..., fmt.Errorf("failed to publish event: %w", err) }`)
**HandleInvoiceStateTransition idempotency guards** — Before triggering a billing state transition, check if the invoice is already in TargetStatuses (skip) or matches IgnoreInvoiceInStatus (skip). Then optionally re-fetch from Stripe via GetStripeInvoice to validate the event is not stale. (`if slices.Contains(input.TargetStatuses, invoice.Status) { return nil } // already there
if invoice.Status.Matches(input.IgnoreInvoiceInStatus...) { return nil } // terminal`)
**newApp combines AppBase with AppData** — After any adapter call that returns AppBase + AppData, call s.newApp(appBase, stripeApp) to produce the concrete appstripe.App struct with all injected service references. (`app, err := s.newApp(stripeApp.AppBase, stripeApp.AppData)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config, New() constructor with marketplace self-registration. | DisableWebhookRegistration flag (for dev/test) is respected in factory.go; do not reference it in other files. |
| `factory.go` | NewApp (app.AppFactory), InstallAppWithAPIKey, UninstallApp, newApp helper. | InstallAppWithAPIKey creates secrets and the Stripe webhook before the DB record — a partial failure can leave orphaned secrets or webhooks; the TODO comment acknowledges this and no rollback exists yet. |
| `billing.go` | HandleInvoiceStateTransition, HandleInvoiceSentEvent, GetSupplierContact, getInvoiceByStripeID. | getInvoiceByStripeID returns nil (not error) when the Stripe invoice is unknown to OpenMeter — callers must check for nil before proceeding. |
| `webhook.go` | WebhookURLGenerator implementations: baseURLWebhookURLGenerator and patternWebhookURLGenerator. | patternWebhookURLGenerator requires a %s placeholder in the pattern; validated at construction time. |
| `app.go` | Remaining appstripe.Service methods: GetWebhookSecret, UpdateAPIKey, CreateCheckoutSession, GetStripeAppData, GetStripeCustomerData, UpsertStripeCustomerData, DeleteStripeCustomerData, HandleSetupIntentSucceeded, CreatePortalSession. | generateMaskedSecretAPIKey is a local helper that must be kept in sync with the masking format displayed in the UI. |

## Anti-Patterns

- Calling s.adapter methods directly without transaction.Run — bypasses transaction boundary.
- Implementing InvoiceStateTransition without the idempotency guards (target-status check, ignore-status check) — causes duplicate billing state transitions.
- Not calling RegisterMarketplaceListing in New() — makes Stripe app type invisible to the marketplace registry.

## Decisions

- **Service self-registers into the app marketplace in New() rather than at wire time.** — Keeps the registration co-located with the implementation; the app.Service.RegisterMarketplaceListing is idempotent so duplicate registrations are safe.
- **InstallAppWithAPIKey generates the app ID before creating secrets/webhook to enable pre-creation secret labeling.** — Secrets must be tagged with AppID for retrieval; the ID must exist before any secret or webhook is created, so it is generated early with ulid.Make().

## Example: Adding a new service method that writes to the adapter and publishes an event

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
