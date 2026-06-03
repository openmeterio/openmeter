# app

<!-- archie:ai-start -->

> TypeSpec definitions for the installed Apps API — marketplace listings, OAuth2/API-key install flows, Stripe/Sandbox/CustomInvoicing app types, and per-customer app data. Schema-only; every model feeds api/openapi.yaml and the generated Go/JS/Python SDKs.

## Patterns

**Discriminated union for polymorphic app types** — App, AppReplaceUpdate, and CustomerAppData are @discriminated(#{ envelope: 'none', discriminatorPropertyName: 'type' }) unions. A new app type requires a new variant in all three. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union App { stripe: StripeApp, sandbox: SandboxApp, custom_invoicing: CustomInvoicingApp }`)
**AppBase spread for concrete app models** — Concrete app models (StripeApp, SandboxApp, CustomInvoicingApp) spread ...AppBase and add a literal type: AppType.<Value> field; never duplicate AppBase fields. (`model SandboxApp { ...AppBase; type: AppType.Sandbox; }`)
**CustomerAppBase generic for per-app customer data** — Per-app customer data models spread ...CustomerAppBase<AppType.X> to inherit id?/type, then add provider-specific fields. (`model StripeCustomerAppData { ...CustomerAppBase<AppType.Stripe>; ...StripeCustomerAppDataBase; @visibility(Lifecycle.Read) app?: StripeApp; }`)
**Interface-per-resource with @route, @tag, @operationId, @summary** — Each HTTP resource is an interface decorated with @route and @tag; operations carry @operationId and @summary. Never write bare operations outside an interface. (`@route("/api/v1/apps") @tag("Apps") interface AppsEndpoints { @get @operationId("listApps") list(...ListAppsRequest): PaginatedResponse<App> | CommonErrors; }`)
**main.tsp as the import manifest** — main.tsp only imports sibling .tsp files and contains no type definitions; it is the entry point imported by cloud/main.tsp. New files must be imported here or are excluded from compilation. (`import "./app.tsp"; import "./stripe.tsp"; import "./custominvoicing.tsp";`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.tsp` | AppsEndpoints interface, AppBase, App union, AppStatus enum, AppType enum, AppReference — scaffolding for all app types. | App and AppReplaceUpdate unions must stay in sync — adding an app type to App without AppReplaceUpdate breaks PUT. |
| `marketplace.tsp` | MarketplaceEndpoints (listing discovery, OAuth2 install, API-key install, plain install); MarketplaceListing, AppInstallMethod, install request/response models. | installMethods on MarketplaceListing is informational only — actual install routes live in MarketplaceEndpoints. |
| `stripe.tsp` | AppStripeEndpoints (webhook, deprecated updateStripeAPIKey, createCheckoutSession) plus Stripe checkout session option models. | updateStripeAPIKey is deprecated; new stripe config changes go through PUT /api/v1/apps/{id}. |
| `custominvoicing.tsp` | CustomInvoicingApp and AppCustomInvoicingEndpoints for draft/issuing sync callbacks and payment status updates. | Uses async hooks (enableDraftSyncHook, enableIssuingSyncHook); consumers must POST draftSynchronized/issuingSynchronized to progress invoice state. |
| `oauth.tsp` | OAuth2 namespace with AuthorizationCodeGrantParams combining success and error query params. | Uses OAuth2 namespace, not OpenMeter — references must be OAuth2.ClientAppStartResponse etc. |
| `customer.tsp` | CustomerAppData union and per-provider customer app data models (Stripe/Sandbox/CustomInvoicing). | This customer.tsp is under app/ and is separate from the top-level customer/ folder's customer.tsp. |

## Anti-Patterns

- Adding app-type-specific fields directly to AppBase — each app type has its own concrete model
- Defining endpoints outside a named interface with @route and @tag
- Adding a new AppType variant only to App without updating AppReplaceUpdate and CustomerAppData
- Using @body on list/GET operations — paginated params use spread models like ...QueryPagination
- Importing types from sibling app/*.tsp files without going through main.tsp at build time

## Decisions

- **Discriminated unions with envelope:none for App/AppReplaceUpdate/CustomerAppData** — Produces a single OpenAPI oneOf with a flat discriminator property, matching client SDK codegen and the Go handler type-switch pattern.
- **Separate AppBase model spread into each concrete app** — Avoids deep inheritance chains that TypeSpec/oapi-codegen struggle with; each concrete model is self-contained for SDK generation.

## Example: Adding a new app type (e.g. PayPal)

```
// 1. app.tsp: enum AppType { ..., PayPal: "paypal" }
// 2. paypal.tsp: model PayPalApp { ...AppBase; type: AppType.PayPal; }
// 3. app.tsp: union App { ..., paypal: PayPalApp }; union AppReplaceUpdate { ..., paypal: TypeSpec.Rest.Resource.ResourceReplaceModel<PayPalApp> }
// 4. customer.tsp: union CustomerAppData { ..., paypal: PayPalCustomerAppData }
// 5. main.tsp: import "./paypal.tsp";
```

<!-- archie:ai-end -->
