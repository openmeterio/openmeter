# stripe

<!-- archie:ai-start -->

> Stripe marketplace billing app: the App type (this folder, split across appinvoice.go/appcustomer.go/clientapp.go/config.go) implements billing.InvoicingApp and customerapp.App, backed by appstripe.Service (service/), persistence (adapter/), and the Stripe API client factories (client/), with all Stripe HTTP endpoints in httpdriver/. Primary constraint: Stripe is the source of truth for invoice numbers and all Stripe errors flow through providerError translation.

## Patterns

**Four-file App method split** — App methods split by concern: appinvoice.go (InvoicingApp), appcustomer.go (customerapp.App + CustomerData), clientapp.go (getStripeClient helper), config.go (UpdateAppConfig). New App methods go in the matching file. (`var _ billing.InvoicingApp = (*App)(nil) // in appinvoice.go`)
**getStripeClient encapsulates secret retrieval** — App methods needing a Stripe client call a.getStripeClient(ctx, operationName, logKV...) which fetches AppData + secret + builds StripeAppClient. Never fetch the API key secret inline. (`_, stripeClient, err := a.getStripeClient(ctx, "createInvoice", "invoice_id", invoice.ID)`)
**UpsertStandardInvoice branches on ExternalIDs.Invoicing** — Empty ExternalIDs.Invoicing -> createInvoice; otherwise -> updateInvoice. The result always sets ExternalID + InvoiceNumber from Stripe's response (Stripe is source of truth for numbers). (`if invoice.ExternalIDs.Invoicing == "" { return a.createInvoice(ctx, invoice) }
return a.updateInvoice(ctx, invoice)`)
**StripeCalculator for all currency arithmetic** — Every amount sent to Stripe goes through NewStripeCalculator(invoice.Currency).RoundToAmount(). Never pass raw float64 or alpacadecimal to Stripe params. (`calculator, _ := NewStripeCalculator(invoice.Currency)
amount := calculator.RoundToAmount(line.Amount)`)
**HandleInvoiceStateTransition idempotency guards** — Service.HandleInvoiceStateTransition requires both TargetStatuses (already-in-target = skip) and IgnoreInvoiceInStatus (out-of-order = skip). New Stripe webhook handlers must supply both guards. (`HandleInvoiceStateTransitionInput{TargetStatuses: []billing.StandardInvoiceStatus{...}, IgnoreInvoiceInStatus: [...]}`)
**Input namespace cross-check** — Input types carrying both AppID and CustomerID validate AppID.Namespace == CustomerID.Namespace. Apply the same check to any new input combining two namespaced IDs. (`if i.AppID.Namespace != i.CustomerID.Namespace { return errors.New("app and customer must be in the same namespace") }`)
**Compile-time interface assertions per file** — Each file implementing an interface declares var _ <Interface> = (*App)(nil) at the top; add an assertion whenever App gains a new interface. (`var _ billing.InvoicingApp = (*App)(nil)
var _ customerapp.App = (*App)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `appinvoice.go` | Full InvoicingApp impl: Validate/Upsert/Finalize/Delete StandardInvoice plus createInvoice/updateInvoice helpers. | updateInvoice builds stripeLinesToRemove from existing line items — new line attributes must update both add and update paths; AddInvoiceLines resolves line IDs via a separate ListInvoiceLineItems call. |
| `appcustomer.go` | ValidateCustomer/ValidateCustomerByID (per-capability Stripe checks), GetCustomerData/UpsertCustomerData/DeleteCustomerData. | ValidateCustomerByID switches on CollectionMethod (ChargeAutomatically vs SendInvoice); new collection methods need a new case. |
| `calculator.go` | StripeCalculator: alpacadecimal -> Stripe integer subunits (RoundToAmount) + display formatting. | RoundToAmount multiplies by 10^subunits and truncates to int64 — only for amounts going to Stripe, not intermediate billing math. |
| `types.go` | Input/output structs with Validate(), API/Webhook secret key constants, HandleInvoiceStateTransitionInput (Target/Ignore status + hooks). | CreateCheckoutSessionInput has mutually exclusive fields (CreateCustomerInput vs CustomerID) validated inline — new options need similar mutual-exclusion checks. |
| `service.go` | Service interface = AppFactoryService + StripeAppService + CustomerService + BillingService sub-interfaces. | New Stripe operations go in the matching sub-interface — never add methods directly to Service. |
| `marketplace.go` | StripeMarketplaceListing var and the three capability vars. | Capability keys are stored in DB; renaming them is a breaking migration. |
| `event.go` | AppCheckoutSessionEvent implements watermill marshaler.Event with versioned metadata.EventType (subsystem 'app.stripe'). | New domain events must follow the same EventName()/EventMetadata() pattern and be versioned (v1, v2, ...). |

## Anti-Patterns

- Calling Stripe API methods without routing errors through the client's providerError() — misses the 401 app-status update and domain error mapping.
- Creating Stripe objects without injecting StripeMetadataNamespace/StripeMetadataAppID in Metadata — breaks reconciliation.
- Using raw float64 or alpacadecimal for Stripe amounts instead of StripeCalculator.RoundToAmount.
- Adding HandleInvoiceStateTransition webhook handlers without TargetStatuses and IgnoreInvoiceInStatus guards — causes duplicate state transitions.
- Implementing App business logic in the adapter sub-package — adapter is pure persistence; orchestration belongs in service/.

## Decisions

- **Two separate Stripe client interfaces: StripeClient (pre-install OAuth/webhook setup) vs StripeAppClient (post-install billing).** — Pre-install calls need only OAuth scopes; post-install needs API key + app context. Splitting prevents using an unauthenticated client for billing operations.
- **Service self-registers into the marketplace in New() rather than at wire time.** — Keeps registration co-located with factory construction; Wire only calls New() and needs no knowledge of the marketplace registry.
- **Stripe is the source of truth for invoice numbers — FinalizeStandardInvoice reads stripeInvoice.Number and stores it.** — Stripe's numbering is deterministic and customer-facing; generating a separate number would cause reconciliation mismatch.

## Example: Add a Stripe webhook handler that transitions an invoice with idempotency guards

```
func (s *service) HandleInvoiceVoidedEvent(ctx context.Context, input HandleInvoiceStateTransitionInput) error {
  return s.HandleInvoiceStateTransition(ctx, HandleInvoiceStateTransitionInput{
    AppID: input.AppID, Invoice: input.Invoice, Trigger: billing.TriggerVoid,
    TargetStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusVoided},
    IgnoreInvoiceInStatus: []billing.StandardInvoiceStatusMatcher{billing.StandardInvoiceStatusVoided, billing.StandardInvoiceStatusDeleted},
  })
}
```

<!-- archie:ai-end -->
