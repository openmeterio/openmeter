# client

<!-- archie:ai-start -->

> Pure Stripe API client layer with two client types: StripeClient (pre-install, OAuth/webhook setup) and StripeAppClient (post-install: invoices, customers, checkout, portal). No business logic; all Stripe-to-domain error translation lives here.

## Patterns

**Dual client hierarchy** — StripeClient (client.go) handles pre-install calls (GetAccount, SetupWebhook). StripeAppClient (appclient.go) handles installed-app calls. Never use StripeClient for post-install operations. (`stripeAppClient, _ := NewStripeAppClient(StripeAppClientConfig{AppService, AppID, APIKey, Logger})`)
**providerError translation on every Stripe call** — Route raw Stripe errors through providerError(). HTTP 400 -> models.NewGenericValidationError; 401 -> app.NewAppProviderAuthenticationError AND updates app status to Unauthorized; else app.NewAppProviderError. (`invoice, err := c.client.Invoices.New(params)
if err != nil { return nil, c.providerError(err) }`)
**OpenMeter metadata injection on created objects** — Every Stripe object OpenMeter creates injects StripeMetadataNamespace, StripeMetadataAppID, and (where applicable) StripeMetadataCustomerID/InvoiceID into Metadata. (`Metadata: map[string]string{StripeMetadataNamespace: input.AppID.Namespace, StripeMetadataAppID: input.AppID.ID}`)
**Currency helpers** — OpenMeter uses uppercase ISO codes (currencyx.Code), Stripe uses lowercase. Use Currency()/CurrencyPtr() to Stripe and FromStripeCurrency() back; never manual string conversion. (`out.Currency = lo.ToPtr(FromStripeCurrency(session.Currency))`)
**Idempotency keys on writes** — Stripe write ops use params.SetIdempotencyKey with a deterministic key from the OpenMeter entity ID to prevent duplicate creation on retries. (`params.SetIdempotencyKey(fmt.Sprintf("invoice-create-%s", input.InvoiceID))`)
**leveledLogger bridge for Stripe backend** — Construct stripe.Backend with the leveledLogger bridge (logger.go) so Stripe SDK output routes through slog. Errorf is deliberately downgraded to Warn. (`backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{LeveledLogger: leveledLogger{logger: config.Logger}})`)
**AddInvoiceLines list-after-create for line IDs** — AddInvoiceLines calls ListInvoiceLineItems after creating invoice items to resolve Stripe line IDs — CreateInvoiceItem does not return the line item ID. By design. (`lineItems, err := c.ListInvoiceLineItems(ctx, input.StripeInvoiceID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `appclient.go` | StripeAppClient interface + impl, webhook event-type constants, StripeMetadata* reserved keys, providerError. | providerError() calls context.Background() for the 401 app-status update — intentional exception to the no-Background rule, since the parent ctx may already be cancelled. |
| `client.go` | StripeClient interface, pre-install ops (GetAccount, SetupWebhook), Currency/FromStripeCurrency helpers, IsAPIKeyLiveMode. | Always use Currency()/FromStripeCurrency() for case conversion; never inline strings.ToLower/ToUpper. |
| `invoice.go` | Create/Update/Delete/Finalize/Get invoice with input/output types and Validate(). | AutoAdvance is always false on creation (advancement is via FinalizeInvoice); CollectionMethodSendInvoice requires DaysUntilDue, ChargeAutomatically forbids it. |
| `invoice_line.go` | AddInvoiceLines, UpdateInvoiceLines, RemoveInvoiceLines, ListInvoiceLineItems. | AddInvoiceLines always re-lists to resolve LineID — Stripe CreateInvoiceItem does not return it. |
| `errors.go` | Typed Stripe errors (StripeCustomerNotFoundError, StripePaymentMethodNotFoundError, StripeInvoiceCustomerTaxLocationInvalidError) with Is* helpers. | These are not models.GenericNotFoundError; callers use IsStripeCustomerNotFoundError etc., not models.IsNotFoundError. |
| `logger.go` | leveledLogger implementing stripe.LeveledLoggerInterface, routing SDK logs through slog. | Errorf intentionally logs at Warn so app-handled errors don't pollute logs. |

## Anti-Patterns

- Calling Stripe API methods without routing the error through providerError() — misses the 401 app-status update and domain mapping.
- Creating Stripe objects without injecting StripeMetadataNamespace/StripeMetadataAppID in Metadata.
- Using raw string currency codes instead of Currency()/FromStripeCurrency().
- Adding business logic here — this is a thin Stripe wrapper; orchestration belongs in appservice.
- Assuming AddInvoiceLines returns line IDs directly — always re-list to resolve them.

## Decisions

- **Two separate client interfaces (StripeClient vs StripeAppClient).** — Pre-install calls have no installed-app context; merging would require optional fields / nil checks throughout.
- **AddInvoiceLines does a separate ListInvoiceLineItems call.** — Stripe InvoiceItems.New returns the item but not its line item ID, which later update/delete operations require.

## Example: New stripeAppClient method calling the Stripe API

```
func (c *stripeAppClient) MyOperation(ctx context.Context, input MyInput) (MyOutput, error) {
	if err := input.Validate(); err != nil { return MyOutput{}, models.NewGenericValidationError(err) }
	result, err := c.client.SomeResource.New(&stripe.SomeParams{Metadata: map[string]string{StripeMetadataNamespace: input.AppID.Namespace, StripeMetadataAppID: input.AppID.ID}})
	if err != nil { return MyOutput{}, c.providerError(err) }
	return MyOutput{ID: result.ID}, nil
}
```

<!-- archie:ai-end -->
