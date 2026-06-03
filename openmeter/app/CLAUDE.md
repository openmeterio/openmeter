# app

<!-- archie:ai-start -->

> Marketplace registry and runtime lifecycle for installed billing apps (Stripe, Sandbox, CustomInvoicing). Owns the app.App/app.Service contracts and the AppFactory/RegistryItem self-registration mechanism, so the billing core can drive invoicing through provider-agnostic interfaces without hardcoding provider logic.

## Patterns

**Per-provider sub-package implements app.App + billing.InvoicingApp** — Each provider folder (stripe/, sandbox/, custominvoicing/) is a self-contained app implementation embedding app.AppBase for shared identity methods (GetID/GetType/GetStatus/ValidateCapabilities) and implementing billing.InvoicingApp. Provider logic lives in the sub-package's service/, never in the root or adapter/. (`type App struct { app.AppBase; ... } // stripe.App, sandbox.App, custominvoicing.App`)
**Factory self-registration at constructor, not at wire time** — A provider's New()/NewFactory() calls app.Service.RegisterMarketplaceListing(RegistryItem{Listing, Factory}). The adapter holds the in-memory registry map[AppType]RegistryItem; every DB read (mapAppFromDB) reconstructs concrete apps via the registered Factory. (`svc.appService.RegisterMarketplaceListing(app.RegistryItem{Listing: stripeListing, Factory: svc})`)
**Root package = contracts + base; layers split service/adapter/httpdriver** — Root files (service.go, registry.go, appbase.go, errors.go, event.go, input.go) define the API surface and shared types. service/ orchestrates (Validate-then-delegate, fires ServiceHooks[AppBase] inside transaction.Run, publishes events); adapter/ is pure Ent persistence; httpdriver/ is the v1 HTTP layer with mapping centralized in mapper.go. (`service/service.go: s.hooks.PostCreate(ctx, &appBase) inside transaction.Run, then eventbus.Publish`)
**Type-neutral event serialization via GetEventAppData** — App types never enter event payloads directly. They implement GetEventAppData() returning a map[string]any; NewEventApp(app) builds the EventApp payload and consumers ParseInto(&target). AppCreateEvent is v1 (AppBase only), AppUpdateEvent/AppDeleteEvent are v2 (full EventApp). (`event.go: NewEventApp(app) calls app.GetEventAppData() to build EventApp payload`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/app/service.go` | Composite app.Service: MarketplaceService + AppLifecycleService + CustomerDataService + models.ServiceHooks[AppBase]. | RegisterHooks (lifecycle) and RegisterRequestValidator (pre-mutation) are distinct; cross-domain wiring needs both, and both must be registered from app/common, never a domain constructor. |
| `openmeter/app/registry.go` | AppFactory/RegistryItem extension point for new app types. | RegistryItem.Factory must be non-nil (RegistryItem.Validate() enforces it); a new provider needs both a Factory and a registered MarketplaceListing. |
| `openmeter/app/appbase.go` | AppBase value type with canonical GetAppBase/GetID/GetType/GetStatus/ValidateCapabilities. | AppType constants must stay in sync with the Ent schema app_type enum; do not add provider-specific fields here. |
| `openmeter/app/errors.go` | Typed domain errors (AppNotFoundError, AppProviderError, AppProviderAuthenticationError, ...) wrapping models.Generic* sentinels with IsXxxError helpers. | Never return raw fmt.Errorf from service/adapter — the HTTP encoder falls through to 500 for unwrapped errors. |

## Anti-Patterns

- Calling adapter/Ent methods or billing.Service directly from App method implementations — provider logic must flow through the provider's own service layer.
- Importing provider sub-packages (appstripe/appsandbox/appcustominvoicing) from adapter/ — concrete construction is delegated to the registered Factory.
- Publishing domain events inside the adapter — event publishing is exclusively the service layer's responsibility after a successful write.
- Adding a new AppType without updating AppType.Validate() and the Ent schema app_type enum — causes runtime panics on unknown type.
- Returning errors not wrapped in models.Generic* types — breaks RFC 7807 status mapping.

## Decisions

- **Marketplace listings live in-memory on the adapter, not in the database.** — Listings are static metadata supplied by factories at startup; persisting them would add DB round-trips to every install with no benefit.
- **OAuth2 install methods are stubbed (GenericNotImplementedError) on the adapter; only API-key and no-credential installs are storage-backed.** — OAuth2 flow state is transient and provider-specific; storing it would couple the adapter to provider protocols.

<!-- archie:ai-end -->
