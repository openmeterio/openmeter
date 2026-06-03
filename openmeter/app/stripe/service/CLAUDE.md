# service

<!-- archie:ai-start -->

> Business-logic service implementing appstripe.Service and app.AppFactory — orchestrates Stripe install/uninstall lifecycle, webhook event handling, invoice state transitions, and checkout sessions. Delegates persistence to appstripe.Adapter and Stripe calls to the client factories.

## Patterns

**transaction.Run wrapping every service method** — Methods calling the adapter wrap in transaction.Run / RunWithNoValue so one transaction spans the full operation. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripe.GetWebhookSecretOutput, error) { return s.adapter.GetWebhookSecret(ctx, input) })`)
**Marketplace self-registration in New()** — New() calls config.AppService.RegisterMarketplaceListing with the Stripe listing and the Service as factory; without it the Stripe app type is invisible to the registry. (`config.AppService.RegisterMarketplaceListing(app.RegistryItem{Listing: appstripe.StripeMarketplaceListing, Factory: service})`)
**Config.Validate() in New()** — Validate all dependencies via Config.Validate() before constructing the Service; return error, never panic. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } }`)
**publisher.Publish inside the transaction** — Side-effect events are published via s.publisher.Publish() inside the transaction.Run closure after the adapter write succeeds. (`event := appstripe.NewAppCheckoutSessionEvent(ctx, input.Namespace, output.SessionID, output.AppID.ID, output.CustomerID.ID)
if err := s.publisher.Publish(ctx, event); err != nil { return ..., err }`)
**HandleInvoiceStateTransition idempotency guards** — Skip if invoice is already in TargetStatuses or matches IgnoreInvoiceInStatus; getInvoiceByStripeID returns nil (not error) for unknown invoices — nil-check before proceeding. (`if slices.Contains(input.TargetStatuses, invoice.Status) { return nil }`)
**newApp combines AppBase with AppData** — After an adapter call returning AppBase+AppData, build the concrete app via s.newApp(appBase, stripeApp); never construct appstripe.App directly. (`app, err := s.newApp(stripeApp.AppBase, stripeApp.AppData)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config, New() with marketplace self-registration. | DisableWebhookRegistration is used only in factory.go — do not reference it elsewhere. |
| `factory.go` | NewApp (app.AppFactory), InstallAppWithAPIKey, UninstallApp, newApp helper. | InstallAppWithAPIKey generates the app ID before creating secrets/webhook across three external systems; a TODO notes no rollback — partial failures can orphan secrets/webhooks. |
| `billing.go` | HandleInvoiceStateTransition, HandleInvoiceSentEvent, GetSupplierContact, getInvoiceByStripeID. | getInvoiceByStripeID returns nil (not error) for invoices unknown to OpenMeter — callers must nil-check, not treat as error. |
| `app.go` | GetWebhookSecret, UpdateAPIKey, CreateCheckoutSession, GetStripeAppData/CustomerData, Upsert/DeleteStripeCustomerData, HandleSetupIntentSucceeded, CreatePortalSession. | generateMaskedSecretAPIKey masking (first 8 + *** + last 3) must stay in sync with UI display format. |
| `webhook.go` | WebhookURLGenerator impls: baseURLWebhookURLGenerator and patternWebhookURLGenerator. | patternWebhookURLGenerator requires a %s placeholder, validated at construction. |

## Anti-Patterns

- Calling s.adapter methods directly without transaction.Run — risks partial writes.
- Implementing HandleInvoiceStateTransition without the target-status and ignore-status idempotency guards.
- Not calling RegisterMarketplaceListing in New() — Stripe app type becomes invisible to the registry.
- Treating nil from getInvoiceByStripeID as an error — nil means the invoice isn't managed by this app.
- Constructing appstripe.App directly instead of s.newApp() — misses injected service references.

## Decisions

- **Service self-registers into the marketplace in New() rather than at wire time.** — Keeps registration co-located with the implementation; RegisterMarketplaceListing is idempotent.
- **InstallAppWithAPIKey generates the app ID before creating secrets/webhook.** — Secrets are tagged with AppID, so the ID (ulid.Make()) must exist before any secret or webhook is created.
- **getInvoiceByStripeID returns nil for unknown Stripe invoices.** — Non-OpenMeter Stripe invoices are normal; treating them as errors would add webhook-processing noise.

## Example: Service method that writes to the adapter and publishes an event

```
func (s *Service) MyAction(ctx context.Context, input appstripe.MyInput) (appstripe.MyOutput, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appstripe.MyOutput, error) {
		out, err := s.adapter.MyAdapterOp(ctx, input)
		if err != nil { return appstripe.MyOutput{}, fmt.Errorf("my action: %w", err) }
		event := appstripe.NewMyActionEvent(ctx, input.Namespace, out.ID)
		if err := s.publisher.Publish(ctx, event); err != nil { return appstripe.MyOutput{}, fmt.Errorf("failed to publish event: %w", err) }
		return out, nil
	})
}
```

<!-- archie:ai-end -->
