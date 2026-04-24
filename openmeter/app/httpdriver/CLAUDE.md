# httpdriver

<!-- archie:ai-start -->

> v1 HTTP handler layer for the app domain — decodes requests, delegates to app.Service and billing.Service, encodes responses using the httptransport.HandlerWithArgs pattern. All mapper functions live in mapper.go to keep handler files thin.

## Patterns

**httptransport.NewHandlerWithArgs three-closure pattern** — Each endpoint is a function on *handler returning a typed HandlerWithArgs. First closure: decode request from (ctx, *http.Request, params). Second closure: invoke service. Third arg: response encoder (JSONResponseEncoderWithStatus or EmptyResponseEncoder). (`return httptransport.NewHandlerWithArgs(decodeFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listApps"))...)`)
**Request/Response/Handler type aliases** — Each endpoint defines three type aliases: <Op>Request, <Op>Response, <Op>Handler — and a params type when needed. This allows the router to reference the concrete handler type without importing httptransport directly. (`type (ListAppsRequest = app.ListAppInput; ListAppsResponse = api.AppPaginatedResponse; ListAppsHandler httptransport.HandlerWithArgs[ListAppsRequest, ListAppsResponse, ListAppsParams])`)
**resolveNamespace via NamespaceDecoder** — Namespace is always resolved from ctx via h.namespaceDecoder.GetNamespace(ctx) in the decode closure. Never read from query params or body directly. (`namespace, err := h.resolveNamespace(ctx)`)
**Type-switch discrimination for polymorphic app types** — When the API body or customer data is a discriminated union (UpdateApp body, CustomerAppData), use body.Discriminator() then the appropriate As<Type>() method. Each known app type (Stripe, Sandbox, CustomInvoicing) has a case; unknown types return GenericValidationError. (`switch updateType { case string(app.AppTypeStripe): payload, _ := body.AsStripeAppReplaceUpdate(); ... default: return UpdateAppRequest{}, models.NewGenericValidationError(...) }`)
**MapAppToAPI in mapper.go** — All domain→API mapping is centralised in mapper.go via MapAppToAPI (and MapEventAppToAPI). Handler files call these helpers; they never inline mapping logic. MapAppToAPI type-asserts app.App to the concrete provider type before calling the provider-specific map function. (`case app.AppTypeStripe: stripeApp := item.(appstripe.App); app.FromStripeApp(mapStripeAppToAPI(stripeApp.Meta))`)
**Billing profile creation on marketplace install** — MarketplaceAppAPIKeyInstall and MarketplaceAppInstall optionally create a billing profile (CreateBillingProfile flag, default true). Only Stripe triggers real profile creation; Sandbox and CustomInvoicing return nil (TODOs). The handler calls h.billingService.CreateProfile directly. (`if request.CreateBillingProfile { defaultForCapabilityTypes, err := h.createBillingProfile(ctx, installedApp) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface definition (AppHandler embeds all endpoint groups), handler struct with all injected services, New() constructor, resolveNamespace helper. | handler holds stripeAppService stripeapp.Service — accessed only for GetSupplierContact during billing profile creation. Customer-deleted checks (IsDeleted) happen in the decode closure, not the operation closure. |
| `mapper.go` | All domain↔API type conversions for apps and customer data. MapAppToAPI, MapEventAppToAPI, ToAPIStripeCustomerAppData, fromAPIAppStripeCustomerData, and the private per-type helpers. | Each new app type requires a new case in both MapAppToAPI and toAPICustomerAppData in customer.go. Missing cases return errors, not panics. |
| `app.go` | ListApps, GetApp, UpdateApp, UninstallApp handlers. UpdateApp uses body.Discriminator() to branch per app type. | UninstallApp checks h.billingService.IsAppUsed before delegating to service.UninstallApp — this cross-domain check must remain in the handler, not the service. |
| `customer.go` | ListCustomerData, UpsertCustomerData, DeleteCustomerData. Customer deletion guard (cus.IsDeleted()) runs in decode closures. | toCustomerData is a private method doing type-discrimination + app resolution; resolveCustomerApp falls back to billingService.GetCustomerApp when no explicit appID is provided. |
| `marketplace.go` | ListMarketplaceListings, GetMarketplaceListing, MarketplaceAppAPIKeyInstall, MarketplaceAppInstall. Post-install billing profile logic in createBillingProfile / makeStripeDefaultBillingApp. | makeStripeDefaultBillingApp only promotes Stripe to default if the current default profile uses AppTypeSandbox. Logic is idempotent for repeated installs. |

## Anti-Patterns

- Inline domain→API mapping inside handler closures — all mapping belongs in mapper.go.
- Reading namespace from query params or path directly — always use h.resolveNamespace(ctx).
- Adding cross-domain pre-checks (billing, customer existence) inside the operation closure — these belong in the decode closure to fail-fast before service calls.
- Creating new handler struct fields for per-request state — the handler is shared across requests; all state must be injected at construction or derived from ctx.

## Decisions

- **Handler holds both app.Service and billing.Service** — Marketplace install triggers billing profile creation as a side-effect; keeping both services in the handler avoids a separate endpoint or forcing billing logic into the app service layer.
- **Type-switch in mapper.go rather than interface method** — App types are discriminated unions at the API boundary; the concrete Go types (appstripe.App, appsandbox.App) are known at compile time here, making explicit switches safer than runtime reflection.

## Example: Standard HandlerWithArgs endpoint wiring

```
func (h *handler) ListApps() ListAppsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAppsParams) (ListAppsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return ListAppsRequest{}, err }
			return ListAppsRequest{Namespace: ns, Page: pagination.Page{PageSize: lo.FromPtrOr(params.PageSize, app.DefaultPageSize), PageNumber: lo.FromPtrOr(params.Page, app.DefaultPageNumber)}}, nil
		},
		func(ctx context.Context, req ListAppsRequest) (ListAppsResponse, error) {
			result, err := h.service.ListApps(ctx, req)
			if err != nil { return ListAppsResponse{}, err }
			// map result.Items via MapAppToAPI ...
			return ListAppsResponse{Items: items, TotalCount: result.TotalCount}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListAppsResponse](http.StatusOK),
		httptransport.AppendOptions(h.options, httptransport.WithOperationName("listApps"))...,
// ...
```

<!-- archie:ai-end -->
