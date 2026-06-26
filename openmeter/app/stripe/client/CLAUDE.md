# client

<!-- archie:ai-start -->

> Thin, typed wrapper over the stripe-go/v80 SDK. Translates OpenMeter inputs into Stripe params and Stripe responses/errors into OpenMeter domain types. Defines StripeAppClient (installed-app operations) and StripeClient (pre-install), plus their factory function types.

## Patterns

**Interface + factory-type + struct triad** — Each client is an interface (StripeAppClient), a factory type alias (StripeAppClientFactory = func(config) (StripeAppClient, error)), and a concrete struct (stripeAppClient) constructed by NewStripeAppClient(config) after config.Validate(). (`type StripeAppClientFactory = func(config StripeAppClientConfig) (StripeAppClient, error)`)
**Central providerError translation** — All Stripe SDK errors funnel through c.providerError(err), which maps HTTP 400 -> models.NewGenericValidationError, 401 -> updates app status to Unauthorized + app.NewAppProviderAuthenticationError, else app.NewAppProviderError. (`if stripeErr, ok := err.(*stripe.Error); ok { switch stripeErr.HTTPStatusCode { case http.StatusBadRequest: return models.NewGenericValidationError(...) } }`)
**Sentinel typed errors with New/Is pair** — errors.go declares typed errors (StripeCustomerNotFoundError, StripePaymentMethodNotFoundError, StripeInvoiceCustomerTaxLocationInvalidError) each with a New... constructor and an Is... helper using errors.As, asserting models.GenericError. (`func IsStripeCustomerNotFoundError(err error) bool { var e *StripeCustomerNotFoundError; return errors.As(err, &e) }`)
**Reserved OpenMeter metadata keys on Stripe objects** — om_namespace / om_app_id / om_customer_id / om_invoice_id are injected into Stripe metadata and listed in SetupIntentReservedMetadataKeys; user-supplied metadata containing these keys must be rejected. (`metadata[StripeMetadataNamespace] = input.AppID.Namespace; metadata[StripeMetadataAppID] = input.AppID.ID`)
**Webhook event type constants** — Stripe webhook event type strings are centralized as WebhookEventType* consts (setup_intent.* and invoice.*) and consumed by the httpdriver webhook switch. (`const WebhookEventTypeInvoicePaid = "invoice.paid"`)
**Explicit toStripeX / FromStripeX mappers** — Conversions between Stripe SDK structs and OpenMeter domain types are dedicated functions (toStripePaymentMethod, FromStripeCurrency) rather than inline. (`func toStripePaymentMethod(stripePaymentMethod *stripe.PaymentMethod) StripePaymentMethod { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `appclient.go` | StripeAppClient interface, factory, NewStripeAppClient, providerError, metadata + webhook-event consts | providerError mutates app status on 401 using context.Background(); all SDK calls must route through it. Adding an interface method requires updating the concrete struct and test fakes. |
| `checkout.go` | CreateCheckoutSession param building, StripeCheckoutSession type + Validate, CreateCheckoutSessionInput + Validate | Mode is always Setup (Validate enforces it); embedded vs hosted UI mode have different required URL fields; metadata is validated against reserved keys. |
| `errors.go` | typed sentinel errors with New/Is helpers | Constructors return pointers but the var _ models.GenericError assertion is on the value type — keep both consistent when adding a new error. |
| `invoice.go / invoice_line.go` | Stripe invoice + line-item CRUD mapping | These translate between billing line items and Stripe InvoiceItem IDs; preserve the StripeInvoiceItemWithLineID linkage. |
| `client.go` | pre-install StripeClient (GetAccount, SetupWebhook) used before an app row exists | Used by service.InstallAppWithAPIKey before persistence; do not assume an installed app context here. |
| `logger.go` | leveledLogger adapter bridging slog into stripe.LeveledLogger | Injected into stripe.BackendConfig in NewStripeAppClient. |

## Anti-Patterns

- Returning a raw *stripe.Error to callers instead of routing through providerError.
- Letting OpenMeter-reserved metadata keys (om_*) be set by user input without rejecting them.
- Hard-coding webhook event type strings instead of the WebhookEventType* constants.
- Using a Stripe checkout mode other than Setup (StripeCheckoutSession.Validate enforces Setup).
- Building param/response conversions inline instead of a toStripeX/FromStripeX helper.

## Decisions

- **Two distinct clients (StripeClient pre-install, StripeAppClient post-install)** — Before installation there is no app row/status to mutate on auth errors; afterwards providerError can update app status.
- **Stripe HTTP errors mapped to OpenMeter typed errors at the client boundary** — Upstream layers (adapter/service/httpdriver) stay decoupled from stripe-go and can branch on app.*/models.* error categories.

## Example: Centralized Stripe error translation

```
func (c *stripeAppClient) providerError(err error) error {
	if stripeErr, ok := err.(*stripe.Error); ok {
		switch stripeErr.HTTPStatusCode {
		case http.StatusBadRequest:
			return models.NewGenericValidationError(fmt.Errorf("stripe error: %s", stripeErr.Msg))
		case http.StatusUnauthorized:
			_ = c.appService.UpdateAppStatus(context.Background(), app.UpdateAppStatusInput{ID: c.appID, Status: app.AppStatusUnauthorized})
			return app.NewAppProviderAuthenticationError(&c.appID, c.appID.Namespace, errors.New(stripeErr.Msg))
		default:
			return app.NewAppProviderError(&c.appID, c.appID.Namespace, errors.New(stripeErr.Msg))
		}
	}
	return err
}
```

<!-- archie:ai-end -->
