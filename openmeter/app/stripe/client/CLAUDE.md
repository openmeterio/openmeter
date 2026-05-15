# client

<!-- archie:ai-start -->

> Pure Stripe API client layer with two distinct client types: StripeClient (pre-install, for OAuth/webhook setup) and StripeAppClient (post-install, for invoice, customer, checkout, portal operations). No business logic; all error translation to OpenMeter domain errors happens here.

## Patterns

**Dual client hierarchy** — StripeClient (client.go) handles unauthenticated/pre-install calls (GetAccount, SetupWebhook). StripeAppClient (appclient.go) handles installed-app calls (invoices, customers, portal). Never use StripeClient for post-install operations. (`stripeClient, _ := NewStripeClient(StripeClientConfig{...})
stripeAppClient, _ := NewStripeAppClient(StripeAppClientConfig{AppService: ..., AppID: ..., APIKey: ..., Logger: ...})`)
**providerError translation on every Stripe API call** — Every method that calls the Stripe API must pass raw errors through providerError(). HTTP 400 → models.NewGenericValidationError; HTTP 401 → app.NewAppProviderAuthenticationError AND updates app status to Unauthorized via context.Background(); others → app.NewAppProviderError. (`result, err := c.client.Invoices.New(params)
if err != nil { return nil, c.providerError(err) }`)
**OpenMeter metadata injection on all created objects** — Every Stripe object created by OpenMeter (customer, invoice, checkout session, setup intent) must have StripeMetadataNamespace, StripeMetadataAppID, and StripeMetadataCustomerID injected into its Metadata map. (`Metadata: map[string]string{
	StripeMetadataNamespace: input.AppID.Namespace,
	StripeMetadataAppID:     input.AppID.ID,
	StripeMetadataCustomerID: input.CustomerID.ID,
}`)
**Currency helper functions** — OpenMeter uses uppercase ISO codes (currencyx.Code), Stripe uses lowercase. Always use Currency() to convert to Stripe format and FromStripeCurrency() to convert back. Never do manual string conversion. (`params.Currency = lo.ToPtr(string(Currency(input.Currency)))
out.Currency = lo.ToPtr(FromStripeCurrency(session.Currency))`)
**Idempotency keys on write operations** — Stripe write operations (CreateInvoice) must use params.SetIdempotencyKey with a deterministic key derived from the OpenMeter entity ID to prevent duplicate creation on retries. (`params.SetIdempotencyKey(fmt.Sprintf("invoice-create-%s", input.InvoiceID))`)
**leveledLogger bridge for Stripe backend** — All stripe.Backend instances must be constructed with the leveledLogger bridge (logger.go) so Stripe SDK log output routes through slog. Never use the default Stripe logger. (`backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{LeveledLogger: leveledLogger{logger: config.Logger}})`)
**AddInvoiceLines list-after-create for line IDs** — AddInvoiceLines calls ListInvoiceLineItems after creating invoice items to resolve Stripe line IDs — Stripe's CreateInvoiceItem does not return the line item ID. This is by design, not a bug. (`// After creating all items:
lineItems, err := c.ListInvoiceLineItems(ctx, input.StripeInvoiceID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `appclient.go` | StripeAppClient interface, concrete stripeAppClient, webhook event type constants, and StripeMetadata* reserved keys. | providerError() calls context.Background() for the app status update on 401 — intentional exception to the no-context.Background() rule; the parent ctx may already be cancelled. |
| `client.go` | StripeClient interface, pre-install operations (GetAccount, SetupWebhook), Currency/FromStripeCurrency helpers, and IsAPIKeyLiveMode. | Currency conversion uses lowercase for Stripe; always use Currency()/FromStripeCurrency() helpers, never manual string conversion. |
| `invoice.go` | CreateInvoice, UpdateInvoice, DeleteInvoice, FinalizeInvoice, GetInvoice with input/output types. | AutoAdvance is always set to false on creation; advancement is driven by FinalizeInvoice. CollectionMethod determines whether DaysUntilDue is required. |
| `invoice_line.go` | AddInvoiceLines, UpdateInvoiceLines, RemoveInvoiceLines. | AddInvoiceLines always calls ListInvoiceLineItems after creation to resolve LineID — required because Stripe CreateInvoiceItem does not return the line item ID. |
| `errors.go` | Typed Stripe-specific errors (StripeCustomerNotFoundError, StripePaymentMethodNotFoundError, StripeInvoiceCustomerTaxLocationInvalidError) with Is* helpers. | These are NOT the same as generic models.GenericNotFoundError; callers must use IsStripeCustomerNotFoundError etc., not models.IsNotFoundError. |

## Anti-Patterns

- Calling Stripe API methods without routing the error through providerError() — misses the 401 app-status update and domain error mapping.
- Creating Stripe objects (customers, invoices, checkout sessions) without injecting StripeMetadataNamespace / StripeMetadataAppID in Metadata.
- Using raw string currency codes instead of Currency() / FromStripeCurrency() helpers.
- Adding business logic here — this package is a thin Stripe API wrapper; all orchestration belongs in appservice.
- Assuming AddInvoiceLines returns line IDs directly — always call ListInvoiceLineItems after creation to resolve them.

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
