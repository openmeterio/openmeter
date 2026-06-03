# httpdriver

<!-- archie:ai-start -->

> v1 HTTP handler layer for the app domain — decodes requests, delegates to app.Service and billing.Service, encodes responses via httptransport.HandlerWithArgs. All domain-to-API mapping is centralised in mapper.go to keep handler files thin.

## Patterns

**HandlerWithArgs three-closure pattern** — Each endpoint returns a typed HandlerWithArgs: first closure decodes to Request, second invokes the service, third is a response encoder. (`return httptransport.NewHandlerWithArgs(decodeFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listApps"))...)`)
**Request/Response/Handler type aliases per endpoint** — Each endpoint defines <Op>Request, <Op>Response, <Op>Handler aliases so the router references the concrete handler type without importing httptransport. (`type (ListAppsRequest = app.ListAppInput; ListAppsResponse = api.AppPaginatedResponse; ListAppsHandler httptransport.HandlerWithArgs[ListAppsRequest, ListAppsResponse, ListAppsParams])`)
**Namespace resolved via NamespaceDecoder** — Namespace is always resolved from ctx via h.resolveNamespace(ctx); never read from query params or body. (`namespace, err := h.resolveNamespace(ctx)`)
**Type-switch discrimination for polymorphic app types** — For discriminated-union API bodies (UpdateApp, CustomerAppData), call body.Discriminator() then the appropriate As<Type>(). Unknown types return GenericValidationError. (`switch updateType { case string(app.AppTypeStripe): payload, _ := body.AsStripeAppReplaceUpdate(); ... default: return UpdateAppRequest{}, models.NewGenericValidationError(...) }`)
**MapAppToAPI centralised in mapper.go** — All domain-to-API mapping lives in mapper.go (MapAppToAPI, MapEventAppToAPI). MapAppToAPI type-asserts to the concrete provider type before a per-provider mapper. (`case app.AppTypeStripe: stripeApp := item.(appstripe.App); app.FromStripeApp(mapStripeAppToAPI(stripeApp.Meta))`)
**Customer deletion guard in decode closure** — Customer existence and deletion checks (cus.IsDeleted()) are done in the decode closure to fail fast before any service call. (`if cus != nil && cus.IsDeleted() { return ListCustomerDataRequest{}, models.NewGenericPreConditionFailedError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (AppHandler embeds endpoint groups), handler struct with injected services, New(), resolveNamespace helper. | handler holds stripeAppService stripeapp.Service used only for GetSupplierContact during billing profile creation. Shared across requests — do not add per-request state fields. |
| `mapper.go` | All domain-to-API conversions: MapAppToAPI, MapEventAppToAPI, ToAPIStripeCustomerAppData, fromAPIAppStripeCustomerData, per-type helpers. | Each new app type needs a case in MapAppToAPI and in toAPICustomerAppData. Missing cases return errors, not panics. |
| `app.go` | ListApps, GetApp, UpdateApp, UninstallApp handlers. | UninstallApp calls h.billingService.IsAppUsed before service.UninstallApp — this cross-domain check lives in the handler, not the service. |
| `customer.go` | ListCustomerData, UpsertCustomerData, DeleteCustomerData. resolveCustomerApp falls back to billingService.GetCustomerApp when no appID is given. | toCustomerData discriminates app type then constructs domain CustomerData; requires concrete app package imports (appstripe, appsandbox, appcustominvoicing). |
| `marketplace.go` | ListMarketplaceListings, GetMarketplaceListing, MarketplaceAppAPIKeyInstall, MarketplaceAppInstall; post-install billing profile in createBillingProfile/makeStripeDefaultBillingApp. | makeStripeDefaultBillingApp only promotes Stripe to default if the current default profile uses AppTypeSandbox — idempotent for repeated installs. |

## Anti-Patterns

- Inline domain-to-API mapping inside handler closures — all mapping belongs in mapper.go
- Reading namespace from query params or path directly — always use h.resolveNamespace(ctx)
- Adding cross-domain pre-checks (billing, customer existence) inside the operation closure — these belong in the decode closure
- Creating new handler struct fields for per-request state — the handler is shared across requests

## Decisions

- **Handler holds both app.Service and billing.Service** — Marketplace install optionally creates a billing profile as a side-effect; keeping both services in the handler avoids forcing billing logic into the app service layer.
- **Type-switch in mapper.go rather than an interface method on app.App** — App types are discriminated unions at the API boundary; concrete Go types are known at compile time here, making explicit switches safer than reflection or interface proliferation.

## Example: Standard HandlerWithArgs endpoint wiring with namespace resolution and MapAppToAPI

```
func (h *handler) ListApps() ListAppsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAppsParams) (ListAppsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return ListAppsRequest{}, err }
			return ListAppsRequest{Namespace: ns, Page: pagination.Page{PageSize: lo.FromPtrOr(params.PageSize, app.DefaultPageSize)}}, nil
		},
		func(ctx context.Context, req ListAppsRequest) (ListAppsResponse, error) {
			result, err := h.service.ListApps(ctx, req)
			if err != nil { return ListAppsResponse{}, err }
			items, _ := lo.MapE(result.Items, MapAppToAPI)
			return ListAppsResponse{Items: items, TotalCount: result.TotalCount}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListAppsResponse](http.StatusOK),
		httptransport.AppendOptions(h.options, httptransport.WithOperationName("listApps"))...,
// ...
```

<!-- archie:ai-end -->
