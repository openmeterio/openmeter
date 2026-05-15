# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for all Stripe-specific API endpoints: webhook ingestion, API key rotation, checkout session creation, customer Stripe data CRUD, and portal session creation. Uses the httptransport.Handler / HandlerWithArgs pattern; delegates all business logic to appstripe.Service and billing.Service.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs** — Every handler function returns httptransport.Handler[Req, Resp] or httptransport.HandlerWithArgs[Req, Resp, Params]. The three arguments are: decoder (context + http.Request → Req), operation (context + Req → Resp), and encoder. Never implement ServeHTTP directly. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("myAction"))...)`)
**resolveNamespace from context** — All decoders call h.resolveNamespace(ctx) to extract the namespace. Never read the namespace from the URL path or request body directly. (`ns, err := h.resolveNamespace(ctx)
if err != nil { return Req{}, err }`)
**Deleted customer guard** — After resolving a customer from the service, always check cus.IsDeleted() and return models.NewGenericPreConditionFailedError if true. Never operate on a deleted customer. (`if cus != nil && cus.IsDeleted() { return Req{}, models.NewGenericPreConditionFailedError(fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID)) }`)
**Webhook namespace from signed secret, not request** — The webhook handler (AppStripeWebhook) does NOT call resolveNamespace. It recovers the namespace from secret.SecretID.Namespace after validating the Stripe-Signature header. Never inject a namespace from the URL for webhooks. (`secret, err := h.service.GetWebhookSecret(ctx, ...)
event, err := webhook.ConstructEventWithTolerance(..., secret.Value, ...)
appID := app.AppID{Namespace: secret.SecretID.Namespace, ID: params.AppID}`)
**Type aliases for request/response types** — Use type aliases (`type X = appstripe.Y`) rather than wrapper struct types for request/response to avoid unnecessary indirection. Keep mapping functions in mapping.go pure (no service calls). (`type UpdateStripeAPIKeyRequest = appstripe.UpdateAPIKeyInput
type UpdateStripeAPIKeyResponse = struct{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface definition, concrete handler struct (holds service, billingService, customerService), New() constructor, resolveNamespace helper. | The handler holds both billingService and customerService in addition to the Stripe service — new handlers needing billing profile lookups can access them directly. |
| `webhook.go` | AppStripeWebhook handler — validates Stripe-Signature, routes 12+ event types to service calls. | setup_intent.succeeded events must validate StripeMetadataAppID and StripeMetadataNamespace in payment intent metadata before acting. Events missing the metadata are silently ignored (non-OpenMeter-initiated payments). |
| `checkout_session.go` | CreateAppStripeCheckoutSession — handles three customer resolution paths: create-by-inline-data, resolve-by-ID, resolve-by-key. | App ID resolution falls back to billing profile lookup (h.billingService.ResolveStripeAppIDFromBillingProfile) when body.AppId is nil — do not remove this fallback. |
| `mapping.go` | Pure conversion functions between domain and API types (toAPIStripePortalSession, fromAPIAppStripeCustomerDataBase). | Keep mapping functions pure — no service calls. All enrichment happens in the operation closure. |

## Anti-Patterns

- Calling service methods inside the decoder function — decoders must only parse and validate the request shape.
- Reading namespace from URL path or body instead of h.resolveNamespace(ctx).
- Adding business logic to handlers — all domain decisions belong in appstripe.Service.
- Deriving namespace for webhooks from URL parameters — namespace must come from the signed webhook secret.
- Operating on a deleted customer without the IsDeleted() guard.

## Decisions

- **Webhook handler recovers namespace from the signed secret rather than a URL segment.** — Stripe webhook calls include the app ID in the URL but not the namespace; the only secure source of namespace is the stored webhook secret whose ID carries the namespace.
- **Handler holds billingService in addition to appstripe.Service.** — Checkout session creation requires resolving the Stripe app ID from the billing profile when no explicit app ID is provided in the request body.

## Example: New Stripe HTTP handler that resolves a customer and delegates to the service

```
func (h *handler) MyStripeAction() httptransport.HandlerWithArgs[MyReq, MyResp, MyParams] {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params MyParams) (MyReq, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return MyReq{}, err }
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &customer.CustomerID{Namespace: ns, ID: params.CustomerID}})
			if err != nil { return MyReq{}, err }
			if cus != nil && cus.IsDeleted() { return MyReq{}, models.NewGenericPreConditionFailedError(fmt.Errorf("customer is deleted")) }
			return MyReq{CustomerID: cus.GetID()}, nil
		},
		func(ctx context.Context, req MyReq) (MyResp, error) {
			return h.service.MyAction(ctx, req)
		},
		commonhttp.JSONResponseEncoderWithStatus[MyResp](http.StatusOK),
		httptransport.AppendOptions(h.options, httptransport.WithOperationName("myStripeAction"))...,
// ...
```

<!-- archie:ai-end -->
