# apps

<!-- archie:ai-start -->

> TypeSpec definitions for the billing apps subsystem (Stripe, Sandbox, ExternalInvoicing). Defines installed-app models, customer data, checkout sessions, and app operations — all compiled to api/v3/openapi.yaml and Go server stubs.

## Patterns

**AppBase generic spread** — Every concrete app model spreads AppBase<AppType.X> instead of duplicating id/type/status/definition fields. (`model AppStripe { ...AppBase<AppType.Stripe>; account_id: string; }`)
**Discriminated union for polymorphic App** — The top-level App union uses @discriminated with envelope:none and discriminatorPropertyName:type; every member must be a model that spreads AppBase so the discriminator field is always present. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union App { stripe: AppStripe, sandbox: AppSandbox, external_invoicing: AppExternalInvoicing }`)
**Visibility-gated fields** — Secret/write-only fields (e.g. secret_api_key) carry @visibility(Lifecycle.Create, Lifecycle.Update) only; read-only computed fields carry @visibility(Lifecycle.Read) only. (`@secret secret_api_key?: string; // Create+Update only`)
**operations.tsp imports all HTTP decorators** — Only operations.tsp imports @typespec/http, @typespec/rest, @typespec/openapi3 and declares 'using TypeSpec.Http'. Model files must not import HTTP decorators. (`import "@typespec/http"; using TypeSpec.Http; namespace Apps; interface AppsOperations { @get list(...): ... }`)
**index.tsp as barrel re-export** — index.tsp imports all sibling .tsp files and nothing else; it is the entry point consumed by parent packages. (`import "./app.tsp"; import "./catalog.tsp"; import "./operations.tsp";`)
**@friendlyName on every model/enum** — Every exported model and enum carries @friendlyName with a BillingApp* prefix to control the generated SDK type name. (`@friendlyName("BillingAppStripe") model AppStripe { ... }`)
**Shared.Resource spread for domain entities** — Entities with identity (AppBase, not value types) spread Shared.Resource to inherit id, created_at, updated_at. (`model AppBase<T> { ...Shared.Resource; type: T; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.tsp` | Defines AppType enum, AppStatus enum, AppBase generic, App discriminated union, and AppReference. Core types consumed by all other files. | Adding a new app type here requires: new AppType member, new model that spreads AppBase, new union member in App, and a new file (e.g. newapp.tsp) imported in index.tsp. |
| `stripe.tsp` | Full Stripe app model plus all Checkout Session and Customer Portal session types (~250 lines). Only secret fields use @secret. | @secret must be on create/update-only fields; do not add it to read-only fields. Use @maxLength on free-text fields. |
| `external_invoicing.tsp` | ExternalInvoicing app model with enable_draft_sync_hook and enable_issuing_sync_hook boolean flags controlling bi-directional sync pausing. | Sync hooks are plain booleans, not enums; adding a new hook state would require a discriminated union change. |
| `operations.tsp` | Declares AppsOperations interface with list and get operations only. All HTTP decoration lives here. | Stripe-specific operations (createCheckoutSession, createCustomerPortalSession) are NOT in this file — they live elsewhere in the aip spec. Do not add customer-scoped ops here. |
| `customer.tsp` | AppCustomerData aggregate model grouping per-app customer linkage data (Stripe customer ID, external invoicing labels). | Adding a new app requires adding an optional field here with matching @visibility(Read, Create, Update). |

## Anti-Patterns

- Adding HTTP decorators (@get, @post) in model files (app.tsp, stripe.tsp, etc.) — HTTP decoration belongs only in operations.tsp
- Defining a new app model without spreading AppBase<AppType.X> — breaks discriminated union typing
- Omitting @friendlyName on a new model — generates an uncontrolled SDK type name
- Using @visibility(Lifecycle.Read) on secret_api_key-style fields — secrets must be write-only
- Importing from outside the aip/src tree without using Shared.* or Common.* namespaces

## Decisions

- **AppBase is a generic model rather than an interface** — TypeSpec does not have interface inheritance for models; generic spread lets all app types share id/type/status/definition without copy-paste and keeps the discriminated union working.
- **Stripe checkout/portal session types live in stripe.tsp not operations.tsp** — Model definitions are kept separate from operation declarations so model files can be imported by other namespaces without pulling in HTTP routing concerns.

## Example: Add a new billing app type (e.g. Adyen)

```
// 1. In app.tsp — add to AppType enum:
//   Adyen: "adyen",
// 2. Create adyen.tsp:
import "./app.tsp";
namespace Apps;
@friendlyName("BillingAppAdyen")
model AppAdyen {
  ...AppBase<AppType.Adyen>;
  merchant_account: string;
}
// 3. In app.tsp — add to App union:
//   adyen: AppAdyen,
// 4. In index.tsp — add:
//   import "./adyen.tsp";
// 5. In customer.tsp — add optional field:
// ...
```

<!-- archie:ai-end -->
