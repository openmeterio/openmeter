# client

<!-- archie:ai-start -->

> Pure Stripe API client layer with two distinct client types: StripeClient (pre-install, for OAuth/webhook setup) and StripeAppClient (post-install, for all app operations including invoices, customers, checkout). No business logic; all error translation to OpenMeter domain errors happens here.

## Patterns

**Dual client hierarchy** — StripeClient (client.go) is for unauthenticated/pre-install calls (GetAccount, SetupWebhook). StripeAppClient (appclient.go) is for installed-app calls (invoices, customers, portal). Never use StripeClient for post-install operations. (`stripeClient, _ := NewStripeClient(StripeClientConfig{...}); stripeAppClient, _ := NewStripeAppClient(StripeAppClientConfig{...})`)
**providerError translation** — Every method that calls the Stripe API must pass raw errors through providerError(). 400 maps to models.NewGenericValidationError, 401 maps to app.NewAppProviderAuthenticationError (and also updates app status to Unauthorized for stripeAppClient), others map to app.NewAppProviderError. (`if err != nil { return ..., c.providerError(err) }`)
**OpenMeter metadata injection on Stripe objects** — Every Stripe object created by OpenMeter (customer, invoice, checkout session, webhook, setup intent) must have StripeMetadataNamespace, StripeMetadataAppID, and StripeMetadataCustomerID injected into its Metadata map. (`Metadata: map[string]string{StripeMetadataNamespace: input.AppID.Namespace, StripeMetadataAppID: input.AppID.ID, ...}`)
**Input.Validate() on every public method** — All Input structs implement Validate() error; call it at the start of every method before touching the Stripe API. (`if err := input.Validate(); err != nil { return ..., models.NewGenericValidationError(err) }`)
**Idempotency keys on write operations** — Stripe write operations (CreateInvoice) use params.SetIdempotencyKey with a deterministic key derived from the OpenMeter entity ID to prevent duplicate creation on retries. (`params.SetIdempotencyKey(fmt.Sprintf("invoice-create-%s", input.InvoiceID))`)
**leveledLogger bridge** — All stripe.Backend instances must be constructed with the leveledLogger bridge (logger.go) so Stripe SDK log output routes through slog. Never use the default Stripe logger. (`stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{LeveledLogger: leveledLogger{logger: config.Logger}})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `appclient.go` | StripeAppClient interface + concrete stripeAppClient implementation; also defines all webhook event type constants and reserved metadata keys. | providerError in stripeAppClient calls UpdateAppStatus to AppStatusUnauthorized on 401 — it uses context.Background() for the app status update, which is intentional but violates the general no-context.Background() rule. |
| `client.go` | StripeClient interface + concrete stripeClient; also Currency/FromStripeCurrency helpers and IsAPIKeyLiveMode. | Currency conversion: OpenMeter uses uppercase ISO codes (currencyx.Code), Stripe uses lowercase. Always use Currency() / FromStripeCurrency() helpers, never manual string conversion. |
| `invoice.go` | CreateInvoice, UpdateInvoice, DeleteInvoice, FinalizeInvoice, GetInvoice with all their input/output types. | AutoAdvance is always set to false on creation; Stripe advancement is driven by FinalizeInvoice. CollectionMethod determines whether DaysUntilDue is required. |
| `invoice_line.go` | AddInvoiceLines, UpdateInvoiceLines, RemoveInvoiceLines. Adding lines requires a post-create list call to get line IDs because Stripe CreateInvoiceItem does not return the line ID. | AddInvoiceLines always calls ListInvoiceLineItems after creation to resolve LineID — this is by design, not a bug. |
| `errors.go` | Typed Stripe-specific errors (StripeCustomerNotFoundError, StripePaymentMethodNotFoundError, StripeInvoiceCustomerTaxLocationInvalidError) with Is* helpers for errors.As checks. | These errors implement models.GenericError but are NOT the same as generic not-found errors; callers must use IsStripeCustomerNotFoundError etc. rather than models.IsNotFoundError. |

## Anti-Patterns

- Calling stripe API methods without routing the error through providerError() — misses 401 app-status update and domain error mapping.
- Creating Stripe objects (customers, invoices) without injecting StripeMetadataNamespace / StripeMetadataAppID in Metadata.
- Using raw string currency codes instead of Currency() / FromStripeCurrency() helpers.
- Adding business logic here — this package is a thin API wrapper; all orchestration belongs in appservice.

## Decisions

- **Two separate client interfaces (StripeClient vs StripeAppClient) rather than one unified client.** — Pre-install calls (webhook setup, account retrieval) do not have an installed app context; merging them would require optional fields or nil checks throughout.
- **AddInvoiceLines does a separate ListInvoiceLineItems call to resolve Stripe line IDs.** — Stripe's InvoiceItems.New returns the invoice item but not its line item ID; the line ID is required for subsequent update/delete operations.

## Example: Adding a new stripeAppClient method that calls the Stripe API

```
func (c *stripeAppClient) MyOperation(ctx context.Context, input MyInput) (MyOutput, error) {
	if err := input.Validate(); err != nil {
		return MyOutput{}, models.NewGenericValidationError(err)
	}
	result, err := c.client.SomeResource.New(&stripe.SomeParams{
		Metadata: map[string]string{
			StripeMetadataNamespace: input.AppID.Namespace,
			StripeMetadataAppID:     input.AppID.ID,
		},
	})
	if err != nil {
		return MyOutput{}, c.providerError(err)
	}
	return MyOutput{ID: result.ID}, nil
}
```

<!-- archie:ai-end -->
