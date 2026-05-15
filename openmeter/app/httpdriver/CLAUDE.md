# httpdriver

<!-- archie:ai-start -->

> v1 HTTP handler layer for the app domain — decodes requests, delegates to app.Service and billing.Service, encodes responses using the httptransport.HandlerWithArgs pattern. All domain-to-API mapping is centralised in mapper.go to keep handler files thin.

## Patterns

**HandlerWithArgs three-closure pattern** — Each endpoint is a function on *handler returning a typed HandlerWithArgs. First closure decodes (ctx, *http.Request, params) → Request. Second closure invokes service. Third arg is a response encoder (JSONResponseEncoderWithStatus or EmptyResponseEncoder). (`return httptransport.NewHandlerWithArgs(decodeFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listApps"))...)`)
**Request/Response/Handler type aliases per endpoint** — Each endpoint defines three type aliases: <Op>Request, <Op>Response, <Op>Handler. This lets the router reference the concrete handler type without importing httptransport. (`type (ListAppsRequest = app.ListAppInput; ListAppsResponse = api.AppPaginatedResponse; ListAppsHandler httptransport.HandlerWithArgs[ListAppsRequest, ListAppsResponse, ListAppsParams])`)
**Namespace resolved via NamespaceDecoder** — Namespace is always resolved from ctx via h.resolveNamespace(ctx) which calls h.namespaceDecoder.GetNamespace(ctx). Never read from query params or request body. (`namespace, err := h.resolveNamespace(ctx)`)
**Type-switch discrimination for polymorphic app types** — When the API body is a discriminated union (UpdateApp body, CustomerAppData), use body.Discriminator() then the appropriate As<Type>() method per known app type. Unknown types return GenericValidationError. (`switch updateType { case string(app.AppTypeStripe): payload, _ := body.AsStripeAppReplaceUpdate(); ... default: return UpdateAppRequest{}, models.NewGenericValidationError(...) }`)
**MapAppToAPI centralised in mapper.go** — All domain-to-API mapping lives in mapper.go via MapAppToAPI and MapEventAppToAPI. Handler files call these helpers and never inline mapping logic. MapAppToAPI type-asserts app.App to the concrete provider type before calling a per-provider mapper. (`case app.AppTypeStripe: stripeApp := item.(appstripe.App); app.FromStripeApp(mapStripeAppToAPI(stripeApp.Meta))`)
**Customer deletion guard in decode closure** — Customer existence and deletion checks (cus.IsDeleted()) are performed in the decode closure — not the operation closure — to fail fast before any service call. (`if cus != nil && cus.IsDeleted() { return ListCustomerDataRequest{}, models.NewGenericPreConditionFailedError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface definition (AppHandler embeds all endpoint groups), handler struct with all injected services, New() constructor, resolveNamespace helper. | handler holds stripeAppService stripeapp.Service — used only for GetSupplierContact during billing profile creation in marketplace.go. The handler is shared across requests; do not add per-request state fields. |
| `mapper.go` | All domain-to-API type conversions: MapAppToAPI, MapEventAppToAPI, ToAPIStripeCustomerAppData, fromAPIAppStripeCustomerData, and per-type private helpers. | Each new app type requires a new case in MapAppToAPI and in toAPICustomerAppData (customer.go). Missing cases return errors, not panics. |
| `app.go` | ListApps, GetApp, UpdateApp, UninstallApp handlers. | UninstallApp calls h.billingService.IsAppUsed before delegating to service.UninstallApp — this cross-domain check lives in the handler, not the service. |
| `customer.go` | ListCustomerData, UpsertCustomerData, DeleteCustomerData. Customer deletion guard runs in decode closures. resolveCustomerApp falls back to billingService.GetCustomerApp when no explicit appID is provided. | toCustomerData is a private method that discriminates app type then constructs domain CustomerData; it requires the concrete app package imports (appstripe, appsandbox, appcustominvoicing). |
| `marketplace.go` | ListMarketplaceListings, GetMarketplaceListing, MarketplaceAppAPIKeyInstall, MarketplaceAppInstall. Post-install billing profile logic in createBillingProfile and makeStripeDefaultBillingApp. | makeStripeDefaultBillingApp only promotes Stripe to default if the current default profile uses AppTypeSandbox — logic is idempotent for repeated installs. |

## Anti-Patterns

- Inline domain-to-API mapping inside handler closures — all mapping belongs in mapper.go.
- Reading namespace from query params or path directly — always use h.resolveNamespace(ctx).
- Adding cross-domain pre-checks (billing, customer existence) inside the operation closure — these belong in the decode closure to fail fast before service calls.
- Creating new handler struct fields for per-request state — the handler is shared across requests.

## Decisions

- **Handler holds both app.Service and billing.Service** — Marketplace install optionally creates a billing profile as a side-effect; keeping both services in the handler avoids forcing billing logic into the app service layer.
- **Type-switch in mapper.go rather than an interface method on app.App** — App types are discriminated unions at the API boundary; concrete Go types are known at compile time here, making explicit switches safer than runtime reflection or interface proliferation.

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
