# httpdriver

<!-- archie:ai-start -->

> HTTP transport layer for the Stripe app: API-key replacement, checkout sessions, customer stripe-data, portal sessions, and the inbound Stripe webhook. Decodes requests, resolves namespace/customer/app, delegates to appstripe.Service / billing.Service, and encodes API responses.

## Patterns

**httptransport.NewHandler(WithArgs) triad** — Each endpoint is a method on *handler returning a typed Handler built from (decode func, business func, response encoder, options). Request/response types are package-level aliases to api.* and appstripe.* types. (`httptransport.NewHandlerWithArgs(decodeFn, businessFn, commonhttp.EmptyResponseEncoder[Resp](http.StatusNoContent), httptransport.AppendOptions(h.options, httptransport.WithOperationName("replaceStripeAPIKey"))...)`)
**Namespace resolution via resolveNamespace** — Non-webhook handlers call h.resolveNamespace(ctx) (backed by namespacedriver.NamespaceDecoder); webhook handler deliberately does NOT — it derives namespace from the signed webhook secret's SecretID. (`namespace, err := h.resolveNamespace(ctx)`)
**Customer polymorphic resolution + deleted guard** — Handlers accept customer by id, key, or create-input; resolve to a customer.CustomerID and reject deleted customers with models.NewGenericPreConditionFailedError. (`if cus != nil && cus.IsDeleted() { return ..., models.NewGenericPreConditionFailedError(...) }`)
**App resolution falls back to billing profile** — When no AppId is in the request, the handler resolves the Stripe app via billingService.ResolveStripeAppIDFromBillingProfile / GetCustomerApp(AppTypeStripe), then type-asserts to appstripe.App. (`stripeApp, ok := genericApp.(appstripe.App); if !ok { return ..., fmt.Errorf("customer app is not a stripe app") }`)
**Webhook event dispatch switch routing to service methods** — AppStripeWebhook validates the signature, then switches on stripeclient.WebhookEventType* and maps each invoice event to service.HandleInvoiceStateTransition with explicit Trigger / TargetStatuses / IgnoreInvoiceInStatus / ShouldTriggerOnEvent. (`case stripeclient.WebhookEventTypeInvoicePaid: ... HandleInvoiceStateTransition(ctx, appstripe.HandleInvoiceStateTransitionInput{Trigger: billing.TriggerPaid, TargetStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusPaid}, ...})`)
**FromAPI / ToAPI mapping helpers** — Conversions live in mapping.go (toAPIStripePortalSession, fromAPIAppStripeCustomerDataBase); the Handler interface enumerates every endpoint and is asserted with var _ Handler = (*handler)(nil). (`func fromAPIAppStripeCustomerDataBase(d api.StripeCustomerAppDataBase) appstripe.CustomerData { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/AppStripeHandler interfaces, handler struct, New constructor, resolveNamespace | New endpoints must be added to the AppStripeHandler interface (compile-checked by var _ Handler). Deps: appstripe.Service, billing.Service, customer.Service, namespacedriver.NamespaceDecoder. |
| `webhook.go` | signature verification (webhook.ConstructEventWithTolerance) + event-type switch | No namespace resolver here — trust comes from the secret. Setup-intent handler validates om_app_id / om_namespace metadata and silently ignores events from other apps. Each invoice case re-fetches the Stripe invoice via ShouldTriggerOnEvent to rule out late events. |
| `checkout_session.go` | create checkout session; parses customer as create/id/key | Exactly one of createCustomerInput/customerId/customerKey must resolve; AppID falls back to ResolveStripeAppIDFromBillingProfile when body.AppId is nil. |
| `customer.go` | GetCustomerStripeAppData, UpsertCustomerStripeAppData, CreateStripeCustomerPortalSession + getAPIStripeCustomerAppData helper | Resolves the concrete stripe app through billingService.GetCustomerApp then type-asserts to appstripe.App; rejects deleted customers. |
| `mapping.go` | API<->domain conversions for portal session and customer data | Follow toAPI.../fromAPI... naming; keep nil-checks for optional fields like Configuration. |
| `const.go` | context-key attribute names for structured logging (stripe_event_id, stripe_event_type, app_id) | Stored on ctx via context.WithValue in the webhook handler for downstream log enrichment. |

## Anti-Patterns

- Calling resolveNamespace inside the webhook handler (namespace must come from the verified secret).
- Skipping the cus.IsDeleted() pre-condition guard when resolving a customer.
- Acting on a webhook setup-intent event without validating om_app_id/om_namespace metadata against the request app.
- Putting business logic in the decode func instead of the dedicated business func of the handler triad.
- Returning the generic app from GetCustomerApp without type-asserting to appstripe.App.

## Decisions

- **Webhook signature verification replaces namespace authentication** — Stripe cannot send a namespace; validating the payload with the app's stored webhook secret proves authenticity and yields the namespace from the secret's SecretID.
- **Invoice webhook handlers re-fetch the live Stripe invoice before transitioning state** — ShouldTriggerOnEvent guards against stale/out-of-order webhook deliveries by checking the current upstream invoice status.

## Example: Webhook event mapped to a billing state transition

```
case stripeclient.WebhookEventTypeInvoiceVoided:
	invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
	if err != nil { return AppStripeWebhookResponse{}, err }
	err = h.service.HandleInvoiceStateTransition(ctx, appstripe.HandleInvoiceStateTransitionInput{
		AppID: request.AppID, Invoice: invoice,
		Trigger: billing.TriggerVoid,
		TargetStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusVoided},
		IgnoreInvoiceInStatus: []billing.StandardInvoiceStatusMatcher{billing.StandardInvoiceStatusCategoryPaid},
		ShouldTriggerOnEvent: func(si *stripe.Invoice) (bool, error) { return si.Status == stripe.InvoiceStatusVoid, nil },
	})
```

<!-- archie:ai-end -->
