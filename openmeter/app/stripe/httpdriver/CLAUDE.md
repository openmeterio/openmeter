# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for all Stripe-specific endpoints: webhook ingestion, API key rotation, checkout session creation, customer Stripe-data CRUD, portal sessions. Uses the httptransport.Handler/HandlerWithArgs pattern; delegates all logic to appstripe.Service and billing.Service.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs** — Each handler returns httptransport.Handler/HandlerWithArgs via (decoder, operation, encoder). Never implement ServeHTTP directly. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("myAction"))...)`)
**resolveNamespace from context** — Decoders call h.resolveNamespace(ctx) for the namespace; never read it from URL path or body. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, err }`)
**Deleted customer guard** — After resolving a customer, check cus.IsDeleted() and return models.NewGenericPreConditionFailedError if true. (`if cus != nil && cus.IsDeleted() { return Req{}, models.NewGenericPreConditionFailedError(fmt.Errorf("customer is deleted")) }`)
**Webhook namespace from signed secret** — AppStripeWebhook does NOT call resolveNamespace; it recovers namespace from secret.SecretID.Namespace after validating the Stripe-Signature header. (`event, _ := webhook.ConstructEventWithTolerance(..., secret.Value, ...)
appID := app.AppID{Namespace: secret.SecretID.Namespace, ID: params.AppID}`)
**Type aliases for request/response** — Use type aliases (type X = appstripe.Y) instead of wrapper structs; keep mapping.go conversion functions pure (no service calls). (`type UpdateStripeAPIKeyRequest = appstripe.UpdateAPIKeyInput`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface, concrete struct (holds service, billingService, customerService), New(), resolveNamespace helper. | Handler also holds billingService and customerService — checkout flows use them for billing-profile and customer lookups. |
| `webhook.go` | AppStripeWebhook — validates Stripe-Signature and routes 12+ event types to service calls. | setup_intent.succeeded must validate StripeMetadataAppID/Namespace in metadata; events missing it are silently ignored (non-OpenMeter payments). |
| `checkout_session.go` | CreateAppStripeCheckoutSession — handles create-by-inline-data, resolve-by-ID, resolve-by-key customer paths. | App ID falls back to h.billingService.ResolveStripeAppIDFromBillingProfile when body.AppId is nil — do not remove. |
| `mapping.go` | Pure domain<->API conversion (toAPIStripePortalSession, fromAPIAppStripeCustomerDataBase). | Keep mapping functions pure; all enrichment happens in the operation closure. |
| `apikey.go` | UpdateStripeAPIKey handler decoding body then delegating to h.service.UpdateAPIKey. | Returns 204 No Content via EmptyResponseEncoder; response type is struct{}. |

## Anti-Patterns

- Calling service methods inside the decoder — decoders only parse/validate request shape.
- Reading namespace from URL path or body instead of h.resolveNamespace(ctx).
- Adding business logic to handlers — domain decisions belong in appstripe.Service.
- Deriving webhook namespace from URL parameters instead of the signed secret.
- Operating on a deleted customer without the IsDeleted() guard.

## Decisions

- **Webhook handler recovers namespace from the signed secret, not a URL segment.** — Stripe webhooks carry the app ID in the URL but not the namespace; the only secure source is the stored webhook secret whose ID carries the namespace.
- **Handler holds billingService in addition to appstripe.Service.** — Checkout session creation must resolve the Stripe app ID from the billing profile when no explicit app ID is in the body.

## Example: New Stripe HTTP handler resolving a customer and delegating

```
func (h *handler) MyStripeAction() httptransport.HandlerWithArgs[MyReq, MyResp, MyParams] {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params MyParams) (MyReq, error) {
			ns, err := h.resolveNamespace(ctx); if err != nil { return MyReq{}, err }
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &customer.CustomerID{Namespace: ns, ID: params.CustomerID}})
			if err != nil { return MyReq{}, err }
			if cus != nil && cus.IsDeleted() { return MyReq{}, models.NewGenericPreConditionFailedError(fmt.Errorf("customer is deleted")) }
			return MyReq{CustomerID: cus.GetID()}, nil
		},
		func(ctx context.Context, req MyReq) (MyResp, error) { return h.service.MyAction(ctx, req) },
		commonhttp.JSONResponseEncoderWithStatus[MyResp](http.StatusOK),
		httptransport.AppendOptions(h.options, httptransport.WithOperationName("myStripeAction"))...,
	)
}
```

<!-- archie:ai-end -->
