# app

<!-- archie:ai-start -->

> Marketplace registry and runtime lifecycle for installed billing apps (Stripe, Sandbox, CustomInvoicing). Owns the App interface, AppFactory/RegistryItem self-registration pattern, install/uninstall flows, OAuth2, and per-customer data delegation — without hardcoding provider logic in the billing core.

## Patterns

**AppFactory self-registration at constructor** — Each app type's factory calls app.Service.RegisterMarketplaceListing inside its own New()/NewFactory(), not at wire time. The registry lives in-memory on the adapter. (`stripe/service/factory.go: svc.appService.RegisterMarketplaceListing(RegistryItem{Listing: stripeListing, Factory: svc})`)
**App embeds AppBase for shared identity methods** — Concrete App structs (stripe.App, sandbox.App, custominvoicing.App) embed app.AppBase which provides GetAppBase(), GetID(), GetType(), GetStatus(), ValidateCapabilities(). Never duplicate these implementations. (`type App struct { app.AppBase; ... }`)
**Input.Validate() at every service entry point** — Every service method calls input.Validate() before delegating to adapter or provider. Input structs enforce namespace cross-checks, required fields, and namespace equality via models.NewNillableGenericValidationError. (`EnsureCustomerInput.Validate() checks AppID.Namespace == CustomerID.Namespace`)
**models.Generic* error wrapping for all domain errors** — Domain errors (AppNotFoundError, AppProviderError, etc.) embed models.NewGenericNotFoundError / GenericPreConditionFailedError so HTTP encoders map them to correct RFC 7807 status codes without special-casing. (`errors.go: AppNotFoundError embeds models.NewGenericNotFoundError(...)`)
**Compile-time interface assertion per file** — Every file implementing an interface has a package-level var _ <Interface> = (*ConcreteType)(nil) assertion to catch drift at compile time. (`var _ app.Service = (*service)(nil)`)
**GetEventAppData() for type-neutral event serialization** — App types implement GetEventAppData() (EventAppData, error) producing a map[string]any. Use NewEventApp(app) for serialization; consumers call ParseInto(&target). Never pass typed app structs directly to event payloads. (`app/event.go: NewEventApp(app) calls app.GetEventAppData() to build EventApp payload`)
**ServiceHooks[AppBase] for cross-domain lifecycle reactions** — models.ServiceHookRegistry[AppBase] is embedded on app.Service; hooks fire at five points (PostCreate, PreUpdate, PostUpdate, PreDelete, PostDelete) inside the mutation transaction so a failing hook rolls back the DB write. (`service/service.go: s.hooks.PostCreate(ctx, &appBase) inside transaction.Run`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/app/service.go` | Full public API: composes MarketplaceService, AppLifecycleService, CustomerDataService, and models.ServiceHooks[AppBase]. | RegisterHooks and RegisterRequestValidator are separate — one for lifecycle hooks, one for pre-mutation validation. Both are needed when wiring cross-domain constraints. |
| `openmeter/app/registry.go` | AppFactory, AppFactoryInstallWithAPIKey, AppFactoryInstall interfaces and RegistryItem — the extension point for new app types. | RegistryItem.Factory must be non-nil; RegistryItem.Validate() enforces this. A new app type needs both a Factory implementation and a registered MarketplaceListing. |
| `openmeter/app/appbase.go` | AppBase value type with shared fields (Type, Status, Listing, Metadata) and canonical GetAppBase/GetID/GetType/GetStatus/ValidateCapabilities implementations. | AppType constants must be kept in sync with Ent schema app_type enum. Do not add app-type-specific fields here. |
| `openmeter/app/errors.go` | Typed domain errors: AppNotFoundError, AppDefaultNotFoundError, AppProviderAuthenticationError, AppProviderError, AppProviderPreConditionError, AppCustomerPreConditionError. | Each error type has an IsXxxError(err) helper. Always use these typed errors; never return raw fmt.Errorf from service or adapter methods. |
| `openmeter/app/event.go` | Domain event types (AppCreateEvent v1, AppUpdateEvent v2, AppDeleteEvent v2) and EventApp/EventAppData serialization helpers. | AppUpdateEvent is v2 (carries full EventApp); AppCreateEvent is v1 (carries only AppBase). Do not bump version numbers in-place — add a new versioned struct. |

## Anti-Patterns

- Calling adapter/Ent methods directly inside App method implementations — app-type logic must go through the service layer.
- Returning errors without wrapping in a models.Generic* type — HTTP error encoder falls through to 500 for unrecognized types.
- Constructing RegistryItem without setting Factory — RegistryItem.Validate() panics and the app cannot be installed.
- Adding new AppType constants without updating AppType.Validate() and the Ent schema app_type enum — causes runtime panics on unknown type.
- Publishing events inside the adapter layer — event publishing is exclusively the service layer's responsibility after successful adapter writes.

## Decisions

- **Registry (marketplace listings) lives in-memory on the adapter, not the database.** — Listings are static metadata provided by factories at startup; they don't need persistence and would add unnecessary DB round-trips for every install call.
- **OAuth2 install methods on the adapter return 'not implemented' — only API-key and no-credentials installs are storage-backed.** — OAuth2 flow state is transient and provider-specific; adapter-level storage would couple the adapter to provider protocols.
- **No per-app-type advisory lock — multiple apps of the same AppType per namespace are supported by design.** — The Ent schema index on (namespace, type) is non-unique; users legitimately run multiple Stripe region apps or sandbox apps in one namespace.

## Example: Implementing a new app type factory that self-registers at construction time

```
package myprovider

import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/app"
)

func NewFactory(appSvc app.Service) *Factory {
	f := &Factory{appSvc: appSvc}
	_ = appSvc.RegisterMarketplaceListing(app.RegistryItem{
		Listing: app.MarketplaceListing{
			Type:         app.AppType("myprovider"),
			Name:         "My Provider",
			Description:  "My billing provider",
			Capabilities: []app.Capability{{Type: app.CapabilityTypeInvoiceCustomers, Key: "invoice", Name: "Invoice", Description: "Invoice customers"}},
// ...
```

<!-- archie:ai-end -->
