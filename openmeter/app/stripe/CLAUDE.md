# stripe

<!-- archie:ai-start -->

> Stripe billing app implementing billing.InvoicingApp (upsert/finalize/delete invoices in Stripe), customer payment data management, OAuth2 install, webhook ingestion, and checkout/portal sessions. Organized as: package-level types + App methods (appinvoice.go, appcustomer.go, clientapp.go) backed by appstripe.Service (sub-package service/) with persistence via appstripe.Adapter (sub-package adapter/) and Stripe API via appstripe client (sub-package client/).

## Patterns

**Four-file App method split** — App methods are split by concern into appinvoice.go (InvoicingApp), appcustomer.go (customerapp.App + CustomerData), clientapp.go (getStripeClient helper), config.go (UpdateAppConfig). New App methods go in the matching file. (`var _ billing.InvoicingApp = (*App)(nil) // in appinvoice.go`)
**getStripeClient encapsulates secret retrieval** — All App methods needing a Stripe API client call a.getStripeClient(ctx, operationName, logKV...) which fetches AppData + secret + builds StripeAppClient. Never fetch the API key secret inline. (`_, stripeClient, err := a.getStripeClient(ctx, "createInvoice", "invoice_id", invoice.ID)`)
**UpsertStandardInvoice branches on ExternalIDs.Invoicing** — If invoice.ExternalIDs.Invoicing is empty → createInvoice; otherwise → updateInvoice. The result always sets ExternalID + InvoiceNumber from Stripe's response. (`if invoice.ExternalIDs.Invoicing == "" { return a.createInvoice(ctx, invoice) }
return a.updateInvoice(ctx, invoice)`)
**StripeCalculator for all currency arithmetic** — Every amount sent to Stripe goes through NewStripeCalculator(invoice.Currency).RoundToAmount(). Never pass raw float64 or alpacadecimal directly to Stripe params. (`calculator, err := NewStripeCalculator(invoice.Currency)
amount := calculator.RoundToAmount(line.Amount)`)
**Compile-time interface assertions per file** — Each file that implements an interface declares var _ <Interface> = (*App)(nil) at the top. Add a new assertion whenever App gains a new interface. (`var _ billing.InvoicingApp = (*App)(nil)
var _ customerapp.App = (*App)(nil)`)
**Input.Validate() namespace cross-check** — Input types that carry both AppID and CustomerID always validate AppID.Namespace == CustomerID.Namespace. Add the same check to any new input type combining two namespaced IDs. (`if i.AppID.Namespace != i.CustomerID.Namespace { return errors.New("app and customer must be in the same namespace") }`)
**HandleInvoiceStateTransition idempotency guards** — Service.HandleInvoiceStateTransition checks TargetStatuses (already-in-target = skip) and IgnoreInvoiceInStatus (out-of-order = skip). New Stripe webhook handlers must supply both. (`HandleInvoiceStateTransitionInput{TargetStatuses: []billing.StandardInvoiceStatus{...}, IgnoreInvoiceInStatus: [...]}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `appinvoice.go` | Full InvoicingApp implementation: ValidateStandardInvoice (calls ValidateCustomerByID), UpsertStandardInvoice (create/update branch), FinalizeStandardInvoice (finalize + set invoice number + PaymentIntent), DeleteStandardInvoice, createInvoice, updateInvoice helpers. | updateInvoice builds stripeLinesToRemove map from existing Stripe line items and reconciles add/update/remove — any new line attribute must update both the add and update paths. |
| `appcustomer.go` | ValidateCustomer / ValidateCustomerByID (checks Stripe customer existence, payment method, tax status per capability), GetCustomerData, UpsertCustomerData, DeleteCustomerData. | ValidateCustomerByID switches on CollectionMethod (ChargeAutomatically requires default payment method + billing address; SendInvoice requires email). New collection methods need a new case. |
| `calculator.go` | StripeCalculator wraps currencyx.Calculator to convert alpacadecimal amounts to Stripe integer subunits (RoundToAmount) and format display strings (FormatAmount, FormatQuantity). | RoundToAmount multiplies by 10^subunits and truncates to int64 — only use for amounts going to Stripe, not for intermediate billing calculations. |
| `types.go` | All input/output structs with Validate() methods: CreateAppStripeInput, CustomerData, AppData, HandleInvoiceStateTransitionInput, CreateCheckoutSessionInput, etc. Also defines APIKeySecretKey / WebhookSecretKey constants. | CreateCheckoutSessionInput has mutually exclusive fields (CreateCustomerInput vs CustomerID) validated inline — new options must add similar mutual-exclusion checks. |
| `service.go` | Service interface composed of AppFactoryService, StripeAppService, CustomerService, BillingService sub-interfaces. | New Stripe-specific operations go in the matching sub-interface — never add methods directly to Service. |
| `marketplace.go` | StripeMarketplaceListing var and the three capability vars (StripeCollectPaymentCapability, StripeCalculateTaxCapability, StripeInvoiceCustomerCapability). | Capability keys are stored in DB; renaming them is a breaking migration. |
| `event.go` | AppCheckoutSessionEvent implements watermill marshaler.Event (EventName, EventMetadata). EventName uses versioned metadata.EventType. | New domain events must follow the same EventName() / EventMetadata() pattern and must be versioned (v1, v2, ...). |

## Anti-Patterns

- Calling stripe API methods without routing errors through the client's providerError() — misses 401 app-status update and domain error mapping.
- Creating Stripe objects without injecting StripeMetadataNamespace / StripeMetadataAppID in Metadata — breaks reconciliation.
- Using raw float64 or alpacadecimal for Stripe amounts instead of StripeCalculator.RoundToAmount.
- Adding HandleInvoiceStateTransition webhook handlers without TargetStatuses and IgnoreInvoiceInStatus guards — causes duplicate state transitions.
- Implementing new App business logic in the adapter sub-package — adapter is pure persistence; all orchestration belongs in service/.

## Decisions

- **Two separate Stripe client interfaces (StripeClient for pre-install OAuth/webhook setup vs StripeAppClient for post-install operations).** — Pre-install calls only need OAuth scopes; post-install calls need API key + app context. Splitting prevents accidentally using an unauthenticated client for billing operations.
- **Service self-registers into the app marketplace in New() (service/factory.go) rather than at wire time.** — Keeps the registration co-located with the factory construction; Wire only needs to call New() and does not need to know about the marketplace registry interface.
- **Stripe is the source of truth for invoice numbers — FinalizeStandardInvoice reads stripeInvoice.Number and stores it via result.SetInvoiceNumber.** — Stripe's invoice numbering is deterministic and customer-facing; generating our own number separately would cause reconciliation mismatch.

## Example: Add a new Stripe webhook handler that transitions an invoice to a new status with idempotency guards

```
// In service/webhook.go — new handler method on Service:
func (s *service) HandleInvoiceVoidedEvent(ctx context.Context, input HandleInvoiceStateTransitionInput) error {
    return s.HandleInvoiceStateTransition(ctx, HandleInvoiceStateTransitionInput{
        AppID:   input.AppID,
        Invoice: input.Invoice,
        Trigger: billing.TriggerVoid,
        TargetStatuses: []billing.StandardInvoiceStatus{
            billing.StandardInvoiceStatusVoided,
        },
        IgnoreInvoiceInStatus: []billing.StandardInvoiceStatusMatcher{
            billing.StandardInvoiceStatusVoided,
            billing.StandardInvoiceStatusDeleted,
        },
    })
}
```

<!-- archie:ai-end -->
