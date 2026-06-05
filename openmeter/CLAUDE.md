# openmeter

<!-- archie:ai-start -->

> Structural root of OpenMeter's core business logic. It owns no direct source files; each child is a self-contained domain or infrastructure package (billing, subscription, entitlement, customer, ledger, notification, streaming, sink, watermill, etc.) following the service/adapter layering, and downstream wiring (app/common, cmd/*) plus the v3 handlers compose them.

## Patterns

**Domain package = interface root + layered sub-packages** — A domain's root declares Service/Adapter (or Repository) interfaces plus value types; persistence lives in a child adapter/, orchestration in service/, transport in httpdriver/httphandler/ or the v3 handlers. See billing, customer, subject, notification, taxcode. (`openmeter/subject/{subject.go,service.go,adapter.go} -> service/, adapter/, httphandler/, testutils/`)
**Write path goes through Service, never Adapter** — Adapters are Ent/transport-only; the Service owns transaction.Run, hooks, and Validate(). Calling the Adapter directly skips all three. (`subject.Service.Create wraps adapter writes in transaction.Run + hooks; direct adapter writes are an anti-pattern`)
**Collect-then-wrap validation** — Validate()/Input.Validate() accumulate into var errs []error and return models.NewNillableGenericValidationError(errors.Join(errs...)) rather than returning on first invalid field. Pervasive across billing, productcatalog, notification, taxcode, app. (`errs = append(errs, fmt.Errorf("field: %w", err)); return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**Coded ValidationIssue error catalogs** — Domain errors are typed, coded sentinels (ValidationIssue with attrs / IsX helpers) in an errors.go, not raw fmt.Errorf — they carry HTTP status, severity and frontend code matching. See billing, ledger, taxcode, app, customer. (`billing.NewValidationError(...) sentinels in errors.go; ledger ValidationIssue.WithAttrs`)
**Discriminated-union value types with central registries** — Sum types keyed by a Type discriminator (Price/RateCard, Channel/Rule config, EventType, AppType, EntitlementType) must switch on the discriminator and be added to the central Values()/list. Marshaling the wrapper directly causes wrong shape / recursion. (`productcatalog Price/RateCard custom JSON; notification EventType registered in eventTypes slice; app AppType.Validate`)
**Namespace-tenant seam** — Components that provision per-tenant state register a namespace.Handler (CreateNamespace/DeleteNamespace) on the central Manager; the active tenant is resolved from context via the namespacedriver decoder, never from body/query. (`taxcode/namespacehandler.go seeds org defaults inside one CreateNamespace transaction`)
**Feature-gated wiring (credits.enabled)** — ledger and related credit/customer-account paths are gated by credits.enabled; when off, app/common wires noop implementations. Code touching ledger accounts must respect this gate at every layer. (`openmeter/ledger/noop used when credits disabled`)

## Anti-Patterns

- Putting Ent/SQL or HTTP code in a domain root package — roots are interfaces + value types + pure model logic; persistence belongs in adapter/, transport in the httpdriver/handlers
- Writing through an Adapter directly instead of the Service, bypassing transaction.Run, hooks, and Validate()
- Returning bare fmt.Errorf instead of the domain's coded ValidationIssue sentinels, losing HTTP status / severity / frontend-code matching
- Adding a discriminated-union variant (Price, ChannelType, EventType, AppType) without updating its central registry/Values() and every Validate() switch
- Letting Service and Adapter/Repository method sets drift apart, or constructing services via struct literals instead of their New constructors

## Decisions

- **Each domain is an interface-rooted package with separate adapter/service/transport children rather than one flat package.** — Enforces dependency direction (transport -> service -> adapter), keeps roots import-light, and lets app/common compose narrow contracts via Wire.
- **Domain errors and validation are modeled as coded ValidationIssue sentinels with errors.Join accumulation.** — Gives consistent HTTP status mapping, severity, and frontend code-matching, and surfaces all invalid fields at once.
- **Tenancy and feature gating are cross-cutting seams (namespace.Handler, credits.enabled noop wiring) rather than per-package conditionals.** — Lets every domain provision/disable consistently without duplicating multi-tenant or feature-flag logic.

<!-- archie:ai-end -->
