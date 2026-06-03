# customer

<!-- archie:ai-start -->

> TypeSpec definitions for the Customer API — CRUD, listing with filters, per-customer app data (Stripe, Sandbox, CustomInvoicing), and a legacy Stripe convenience endpoint. The Customer model feeds api/openapi.yaml and SDKs and is referenced across subscriptions, billing, and entitlements.

## Patterns

**ResourceCreateModel / ResourceReplaceModel for mutation bodies** — Create bodies use TypeSpec.Rest.Resource.ResourceCreateModel<Customer> and updates use ResourceReplaceModel<Customer>. Never inline field lists for create/update. (`@post create(@body customer: TypeSpec.Rest.Resource.ResourceCreateModel<Customer>): { @statusCode _: 201; @body body: Customer; } | CommonErrors;`)
**@visibility annotations control lifecycle exposure** — Read-only fields (currentSubscriptionId, subscriptions, annotations) carry @visibility(Lifecycle.Read) to exclude them from create/update; spread ...Resource for id/timestamps. (`@visibility(Lifecycle.Read) annotations?: Annotations;`)
**Filter params as spread models in interface operations** — List filter params (name, key, primaryEmail, subject, planKey) are declared as ListCustomersParams and spread into the list operation. (`@get list(...ListCustomersParams): PaginatedResponse<Customer> | CommonErrors;`)
**ULIDOrExternalKey path param for customer lookup** — Per-customer endpoints use @path customerIdOrKey: ULIDOrExternalKey to support both ULID and string key without duplicate routes. (`@get @route("/{customerIdOrKey}") get(@path customerIdOrKey: ULIDOrExternalKey, ...GetCustomerParams): Customer | NotFoundError | CommonErrors;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer.tsp` | Customer model, CustomersEndpoints interface, and all list/filter param models. Customer spreads ...Resource and exposes subscriptions as a Lifecycle.Read expand. | subscriptions is only populated when expand=subscriptions is passed; not in default responses. |
| `app.tsp` | CustomerAppsEndpoints (list/upsert/delete CustomerAppData) and ListCustomerAppDataParams. | This app.tsp defines customer-scoped app data, not the app-domain app.tsp. |
| `stripe.tsp` | CustomerStripeEndpoints: get/upsert Stripe app data and createPortalSession — Stripe convenience endpoints separate from the generic app data API. | createPortalSession returns 201, not 200 — matches the Go handler encoding. |
| `main.tsp` | Imports app.tsp, customer.tsp, stripe.tsp; no definitions. | Add new customer sub-resource .tsp files here. |

## Anti-Patterns

- Adding billing-specific fields to the Customer model — billing overrides live in the billing/ sub-folder
- Hardcoding filter params inline in an operation instead of a spread model
- Using @body on GET list operations — list filters must be @query params
- Duplicating Stripe-specific customer fields in the generic CustomerAppData models

## Decisions

- **CustomerUsageAttribution as a nested model rather than a flat subjectKeys array on Customer** — Preserves room for future attribution dimensions (e.g. account-level) without a breaking change.
- **Stripe customer portal session in customer/stripe.tsp rather than app/stripe.tsp** — The portal session is customer-scoped (requires customerIdOrKey path param), not app-scoped.

<!-- archie:ai-end -->
