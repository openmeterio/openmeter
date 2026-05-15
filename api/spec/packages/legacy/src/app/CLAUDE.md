# app

<!-- archie:ai-start -->

> TypeSpec definitions for the installed Apps API — marketplace listings, OAuth2/API-key install flows, Stripe, Sandbox, and CustomInvoicing app types, and per-customer app data. Every model here feeds api/openapi.yaml and the generated Go/JS/Python SDKs.

## Patterns

**Discriminated union for polymorphic app types** — App, AppReplaceUpdate, and CustomerAppData are all `@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })` unions. Adding a new app type requires a new union variant in all three. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union App { stripe: StripeApp, sandbox: SandboxApp, custom_invoicing: CustomInvoicingApp }`)
**AppBase spread for concrete app models** — All concrete app models (StripeApp, SandboxApp, CustomInvoicingApp) spread `...AppBase` and add a literal `type: AppType.<Value>` field. Never duplicate AppBase fields. (`model SandboxApp { ...AppBase; type: AppType.Sandbox; }`)
**CustomerAppBase generic for per-app customer data** — Per-app customer data models spread `...CustomerAppBase<AppType.X>` to inherit common `id?` and `type` fields, then add provider-specific fields. (`model StripeCustomerAppData { ...CustomerAppBase<AppType.Stripe>; ...StripeCustomerAppDataBase; @visibility(Lifecycle.Read) app?: StripeApp; }`)
**Interface-per-resource with @route, @tag, @operationId** — Each HTTP resource is an `interface` decorated with `@route`, `@tag`, and operations use `@operationId` + `@summary`. Never write bare operations outside an interface. (`@route("/api/v1/apps") @tag("Apps") interface AppsEndpoints { @get @operationId("listApps") list(...ListAppsRequest): PaginatedResponse<App> | CommonErrors; }`)
**main.tsp as the import manifest** — main.tsp only imports other .tsp files; it contains no type definitions. It is the entry point imported by cloud/main.tsp. (`// main.tsp
import "./app.tsp";
import "./stripe.tsp";
import "./custominvoicing.tsp";`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.tsp` | Defines AppsEndpoints interface, AppBase, App union, AppStatus enum, AppType enum, and AppReference. Core scaffolding for all app types. | App and AppReplaceUpdate unions must stay in sync — adding an app type to App without adding it to AppReplaceUpdate breaks PUT. |
| `marketplace.tsp` | MarketplaceEndpoints interface covering listing discovery, OAuth2 install, API-key install, and plain install. Defines MarketplaceListing, AppInstallMethod, and install request/response models. | installMethods on MarketplaceListing is informational only — actual install routes live in MarketplaceEndpoints. |
| `stripe.tsp` | AppStripeEndpoints (webhook, deprecated updateStripeAPIKey, createCheckoutSession) plus all Stripe-specific checkout session option models. | updateStripeAPIKey is deprecated; new stripe config changes go through PUT /api/v1/apps/{id}. |
| `custominvoicing.tsp` | CustomInvoicingApp model and AppCustomInvoicingEndpoints for draft/issuing sync callbacks and payment status updates. | CustomInvoicing uses async hooks (enableDraftSyncHook, enableIssuingSyncHook); consumers must POST to draftSynchronized/issuingSynchronized to progress invoice state. |
| `oauth.tsp` | OAuth2 namespace with AuthorizationCodeGrantParams combining success and error query params into a single model. | Uses `OAuth2` namespace, not `OpenMeter` — imports from this file must reference `OAuth2.ClientAppStartResponse` etc. |
| `customer.tsp` | CustomerAppData union and per-provider customer app data models (StripeCustomerAppData, SandboxCustomerAppData, CustomInvoicingCustomerAppData). | customer.tsp here is under the `app/` sub-folder and is separate from the `customer/` folder's customer.tsp. |

## Anti-Patterns

- Adding app-type-specific fields directly to AppBase — each app type has its own concrete model.
- Defining new endpoints outside a named interface with @route and @tag.
- Adding a new AppType variant only to App union without updating AppReplaceUpdate and CustomerAppData.
- Using `@body` on list/GET operations — paginated params use spread models like `...QueryPagination`.
- Importing types from sibling app/*.tsp files without going through main.tsp at build time.

## Decisions

- **Discriminated unions with envelope:none for App/AppReplaceUpdate/CustomerAppData** — Allows a single OpenAPI oneOf with a flat discriminator property, matching client SDK code-gen expectations and the Go handler type-switch pattern.
- **Separate AppBase model spread into each concrete app** — Avoids deep inheritance chains that TypeSpec/oapi-codegen struggle with; each concrete model is self-contained for SDK generation.

## Example: Adding a new app type (e.g. PayPal)

```
// 1. Add to AppType enum in app.tsp
enum AppType { ..., PayPal: "paypal" }

// 2. Create paypal.tsp
model PayPalApp { ...AppBase; type: AppType.PayPal; }

// 3. Add to App and AppReplaceUpdate unions in app.tsp
union App { ..., paypal: PayPalApp }
union AppReplaceUpdate { ..., paypal: TypeSpec.Rest.Resource.ResourceReplaceModel<PayPalApp> }

// 4. Add to CustomerAppData in customer.tsp
union CustomerAppData { ..., paypal: PayPalCustomerAppData }

// 5. Import in main.tsp
import "./paypal.tsp";
```

<!-- archie:ai-end -->
