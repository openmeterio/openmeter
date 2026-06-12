# httpdriver

<!-- archie:ai-start -->

> HTTP transport layer for the app/marketplace API: app CRUD, app-customer data upsert/list/delete, and marketplace listing/install endpoints. Each endpoint is a typed httptransport handler that decodes requests, calls app.Service, and maps domain apps to api.* discriminated unions.

## Patterns

**httptransport.NewHandlerWithArgs triple** — Every endpoint is built from (decode func, business func, encoder) plus AppendOptions(h.options, WithOperationName(...)). Request/Response types are aliased to api.* or app.* via type ( ... = ... ) blocks. (`httptransport.NewHandlerWithArgs(decode, handle, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listApps"))...)`)
**Namespace from request context** — Decode funcs call h.resolveNamespace(ctx) which reads namespaceDecoder.GetNamespace(ctx); failure is a 500 HTTPError. Never trust a namespace from the body. (`namespace, err := h.resolveNamespace(ctx)`)
**Discriminated-union dispatch on app type** — Update/customer-data handlers call body.Discriminator() then switch on app.AppTypeStripe/Sandbox/CustomInvoicing, parsing via body.AsStripeAppReplaceUpdate() etc.; unknown types return NewGenericValidationError. (`switch updateType { case string(app.AppTypeStripe): body.AsStripeAppReplaceUpdate() ... }`)
**Typed map functions to/from api.App** — mapper.go centralizes MapAppToAPI / MapEventAppToAPI / ToAPIStripeCustomerAppData / fromAPIAppStripeCustomerData and per-type mapStripeAppToAPI/mapSandboxAppToAPI/mapCustomInvoicingAppToAPI, building api.App via FromStripeApp/FromSandboxApp/FromCustomInvoicingApp. (`app := api.App{}; app.FromStripeApp(mapStripeAppToAPI(stripeApp.Meta))`)
**Cross-service orchestration in handler body** — Handlers reach into billingService and stripeAppService for side effects: UninstallApp checks billingService.IsAppUsed first; install handlers optionally createBillingProfile/makeStripeDefaultBillingApp. (`if err := h.billingService.IsAppUsed(ctx, request); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/AppHandler interfaces, handler struct (service, stripeAppService, billingService, customerService, namespaceDecoder, options), New() constructor, resolveNamespace | New() positional args must match wiring order; resolveNamespace returns 500 not 400 when namespace missing |
| `app.go` | ListApps/GetApp/UpdateApp/UninstallApp handlers; UpdateApp dispatches on Discriminator to build AppConfigUpdate per type | UpdateApp embeds the per-type Configuration (appstripe.Configuration etc.) in AppConfigUpdate; missing a new app type here silently rejects updates |
| `customer.go` | ListCustomerData/UpsertCustomerData/DeleteCustomerData + toCustomerData/resolveCustomerApp/toAPICustomerAppData | All three first GetCustomer and reject deleted customers with NewGenericPreConditionFailedError; resolveCustomerApp falls back to billingService.GetCustomerApp when no appID given |
| `marketplace.go` | ListMarketplaceListings/GetMarketplaceListing/MarketplaceAppInstall[APIKey] + createBillingProfile/makeStripeDefaultBillingApp/mapMarketplaceListing | createBillingProfile only implemented for Stripe (sandbox/custom-invoicing are TODO nil); makeStripeDefaultBillingApp only sets default when current default is the sandbox app |
| `mapper.go` | Domain<->api.App conversions for all app types and stripe customer data | MapAppToAPI/MapEventAppToAPI use unchecked type assertions item.(appstripe.App); a new app type must be added in every switch or it returns 'unsupported app type' |

## Anti-Patterns

- Reading namespace from request body/params instead of resolveNamespace(ctx)
- Adding a new app type without extending every Discriminator switch (UpdateApp, toCustomerData, toAPICustomerAppData, MapAppToAPI, createBillingProfile)
- Putting business/persistence logic in handlers instead of delegating to app.Service
- Returning raw errors without wrapping; handlers consistently wrap with fmt.Errorf context

## Decisions

- **App types are surfaced as oapi-codegen discriminated unions (api.App, api.CustomerAppData)** — TypeSpec models each app variant separately; handlers must Discriminator()/As*()/From*() to bridge the union to typed domain apps
- **Install endpoints optionally provision a billing profile** — Installing Stripe should make it the default billing app when only the sandbox is configured, removing a manual setup step

## Example: Discriminated-union request decode for UpdateApp

```
updateType, err := body.Discriminator()
if err != nil { return UpdateAppRequest{}, models.NewGenericValidationError(err) }
switch updateType {
case string(app.AppTypeStripe):
	payload, err := body.AsStripeAppReplaceUpdate()
	if err != nil { return UpdateAppRequest{}, err }
	return UpdateAppRequest{AppID: app.AppID{ID: appId, Namespace: namespace}, Name: payload.Name, AppConfigUpdate: appstripe.Configuration{SecretAPIKey: payload.SecretAPIKey}}, nil
default:
	return UpdateAppRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid app type: %s", updateType))
}
```

<!-- archie:ai-end -->
