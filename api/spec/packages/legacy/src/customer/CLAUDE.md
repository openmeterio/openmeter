# customer

<!-- archie:ai-start -->

> TypeSpec definitions for the Customer API — CRUD, listing with filters, per-customer app data (Stripe, Sandbox, CustomInvoicing), and a legacy Stripe convenience endpoint. Models feed api/openapi.yaml and SDKs; the Customer model is referenced throughout subscriptions, billing, and entitlements.

## Patterns

**ResourceCreateModel / ResourceReplaceModel for mutation bodies** — Create bodies use `TypeSpec.Rest.Resource.ResourceCreateModel<Customer>` and update bodies use `TypeSpec.Rest.Resource.ResourceReplaceModel<Customer>`. Never inline field lists for create/update. (`@post create(@body customer: TypeSpec.Rest.Resource.ResourceCreateModel<Customer>): { @statusCode _: 201; @body body: Customer; } | CommonErrors;`)
**@visibility annotations control lifecycle exposure** — Fields like `currentSubscriptionId`, `subscriptions`, and `annotations` carry `@visibility(Lifecycle.Read)` so they are excluded from create/update. Spread `...Resource` on entity models to inherit id/timestamps automatically. (`@visibility(Lifecycle.Read) annotations?: Annotations;`)
**Filter params as spread models in interface operations** — List filter params (name, key, primaryEmail, subject, planKey) are declared as a `model ListCustomersParams` and spread into the list operation, keeping operation signatures short. (`@get list(...ListCustomersParams): PaginatedResponse<Customer> | CommonErrors;`)
**ULIDOrExternalKey path param for customer lookup** — All per-customer endpoints use `@path customerIdOrKey: ULIDOrExternalKey` to support both ULID and string key lookup without duplicate routes. (`@get @route("/{customerIdOrKey}") get(@path customerIdOrKey: ULIDOrExternalKey, ...GetCustomerParams): Customer | NotFoundError | CommonErrors;`)

## Key Files

| File           | Role                                                                                                                                                                                | Watch For                                                                                                           |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `customer.tsp` | Defines Customer model, CustomersEndpoints interface, and all list/filter param models. Customer spreads `...Resource` and exposes `subscriptions` as a Lifecycle.Read-only expand. | `subscriptions` field is only populated when `expand=subscriptions` is passed; it is not part of default responses. |
| `app.tsp`      | CustomerAppsEndpoints interface for list/upsert/delete of CustomerAppData, and ListCustomerAppDataParams model.                                                                     | This app.tsp is under customer/ and defines customer-scoped app data endpoints, not the app-domain app.tsp.         |
| `stripe.tsp`   | CustomerStripeEndpoints with get/upsert Stripe app data and createPortalSession. Stripe-specific convenience endpoints separate from the generic app data API.                      | createPortalSession returns 201, not 200 — matches the Go handler response encoding.                                |
| `main.tsp`     | Imports app.tsp, customer.tsp, stripe.tsp. No definitions.                                                                                                                          | Add new customer sub-resource .tsp files here.                                                                      |

## Anti-Patterns

- Adding billing-specific fields to the Customer model — billing overrides live in the billing/ sub-folder.
- Hardcoding filter params inline in an operation instead of a spread model.
- Using `@body` on GET list operations — list filters must be `@query` params.
- Duplicating Stripe-specific customer fields in the generic CustomerAppData models.

## Decisions

- **CustomerUsageAttribution as a nested model rather than flat subjectKeys array on Customer** — Preserves room for future attribution dimensions (e.g. account-level attribution) without a breaking change.
- **Stripe customer portal session in customer/stripe.tsp rather than app/stripe.tsp** — The portal session is customer-scoped (requires customerIdOrKey path param), not app-scoped.

<!-- archie:ai-end -->
