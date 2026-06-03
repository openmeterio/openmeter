# apps

<!-- archie:ai-start -->

> TypeSpec definitions for the v3 billing apps subsystem (Stripe, Sandbox, ExternalInvoicing): installed-app models, the discriminated App union, customer-data linkage, Stripe checkout/portal session types, and the list/get operations interface. Compiled to the BillingApp* section of api/v3/openapi.yaml.

## Patterns

**AppBase generic spread** — Every concrete app model spreads AppBase<AppType.X> rather than duplicating the id/type/definition/status fields. (`model AppStripe { ...AppBase<AppType.Stripe>; account_id: string; }`)
**Discriminated union for polymorphic App** — The top-level App union uses @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }); every member must spread AppBase so the type discriminator is always present. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union App { stripe: AppStripe, sandbox: AppSandbox, external_invoicing: AppExternalInvoicing }`)
**Visibility-gated fields for secrets** — Secret/write-only fields carry @secret with @visibility(Lifecycle.Create, Lifecycle.Update) only; read-only computed fields carry @visibility(Lifecycle.Read) only. (`@visibility(Lifecycle.Create, Lifecycle.Update) @secret secret_api_key?: string;`)
**operations.tsp owns all HTTP decorators** — Only operations.tsp imports @typespec/http, @typespec/rest, @typespec/openapi3 and declares 'using TypeSpec.Http'. Model files (app.tsp, stripe.tsp, ...) must not import HTTP decorators. (`// operations.tsp only: import "@typespec/http"; using TypeSpec.Http; interface AppsOperations { @get list(...): ... }`)
**index.tsp as barrel re-export** — index.tsp imports all sibling .tsp files (app, catalog, customer, external_invoicing, sandbox, stripe, operations) and nothing else; it is the sole entry point consumed by parent packages. (`import "./app.tsp"; import "./catalog.tsp"; import "./operations.tsp";`)
**@friendlyName on every model/enum** — Every exported model and enum carries @friendlyName with a BillingApp* prefix to control the generated SDK type name. (`@friendlyName("BillingAppStripe") model AppStripe { ... }`)
**Shared.Resource spread for identity entities** — Entities with identity (AppBase) spread Shared.Resource to inherit id, created_at, updated_at. (`model AppBase<T extends AppType> { ...Shared.Resource; type: T; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.tsp` | Defines the AppType enum (sandbox/stripe/external_invoicing), AppStatus enum (ready/unauthorized), AppBase<T> generic, the App discriminated union, and AppReference. | Adding a new app type requires: a new AppType member, a model spreading AppBase, a new union member in App, a new file imported in index.tsp, and an optional field in customer.tsp. |
| `stripe.tsp` | Full Stripe app model plus all Checkout Session and Customer Portal session request/result types (~250+ lines). Only the secret_api_key field uses @secret. | @secret must be on create/update-only fields; never add it to read-only fields. Use @maxLength on free-text fields (e.g. checkout custom_text messages cap at 1200). |
| `external_invoicing.tsp` | ExternalInvoicing app model with enable_draft_sync_hook and enable_issuing_sync_hook boolean flags controlling bi-directional sync pausing. | Sync hooks are plain booleans, not enums; adding a new hook state would require a discriminated-union change. |
| `operations.tsp` | Declares the AppsOperations interface with list and get operations only. All HTTP decoration lives here. | Stripe-specific and customer-scoped operations are not in this file. |
| `customer.tsp` | AppCustomerData aggregate grouping per-app customer linkage (Stripe customer ID, external invoicing labels). | Adding a new app requires an optional field here with matching @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update). |
| `catalog.tsp` | AppCatalogItem read-only model describing an available app (type, name, description) referenced by AppBase.definition. | All fields are @visibility(Lifecycle.Read); this is a marketplace listing, not a create/update body. |

## Anti-Patterns

- Adding HTTP decorators (@get, @post) in model files — HTTP decoration belongs only in operations.tsp
- Defining a new app model without spreading AppBase<AppType.X> — breaks discriminated-union typing
- Omitting @friendlyName on a new model — generates an uncontrolled SDK type name
- Using @visibility(Lifecycle.Read) on secret_api_key-style fields — secrets must be write-only
- Importing from outside the aip/src tree without going through the Shared.* or Common.* namespaces

## Decisions

- **AppBase is a generic model rather than an interface.** — TypeSpec lacks interface inheritance for models; generic spread lets all app types share id/type/status/definition without copy-paste and keeps the discriminated union working.
- **Stripe checkout/portal session types live in stripe.tsp, not operations.tsp.** — Model definitions are kept separate from operation declarations so model files can be imported by other namespaces without pulling in HTTP routing concerns.

## Example: Add a new billing app type (e.g. Adyen)

```
// 1. app.tsp — add to AppType enum:  Adyen: "adyen",
// 2. adyen.tsp:
import "./app.tsp";
namespace Apps;
@friendlyName("BillingAppAdyen")
model AppAdyen { ...AppBase<AppType.Adyen>; merchant_account: string; }
// 3. app.tsp — add to App union:  adyen: AppAdyen,
// 4. index.tsp — add:  import "./adyen.tsp";
// 5. customer.tsp — add optional field with @visibility(Read, Create, Update)
```

<!-- archie:ai-end -->
