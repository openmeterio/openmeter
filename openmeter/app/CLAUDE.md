# app

<!-- archie:ai-start -->

> Marketplace registry and runtime lifecycle for installed billing apps (Stripe, Sandbox, CustomInvoicing). Owns the App interface, AppFactory/RegistryItem pattern for pluggable app types, and orchestrates install/uninstall, OAuth2 flows, and customer data delegation to each app type.

## Patterns

**AppFactory self-registration at constructor** — Each app type's factory registers its MarketplaceListing with app.Service.RegisterMarketplaceListing inside its own New()/NewFactory() constructor, not at wire time. (`stripe/service/factory.go: svc.appService.RegisterMarketplaceListing(RegistryItem{Listing: stripeListing, Factory: svc})`)
**ServiceHookRegistry[AppBase] for cross-domain lifecycle reactions** — models.ServiceHookRegistry[AppBase] is embedded on app.Service; external packages register hooks via RegisterHooks(). Hooks fire at five points: PostCreate, PreUpdate, PostUpdate, PreDelete, PostDelete — all inside the mutation transaction so a failing hook rolls back the DB write. (`service/service.go: s.hooks.PostCreate(ctx, &appBase) inside transaction.Run`)
**App embeds AppBase** — Concrete App structs (stripe.App, sandbox.App, custominvoicing.App) embed app.AppBase which provides GetAppBase(), GetID(), GetType(), GetStatus(), ValidateCapabilities() — never duplicate these method implementations. (`type App struct { app.AppBase; ... }`)
**Input.Validate() at every entry point** — Every service method calls input.Validate() before delegating to adapter or external provider. Every input struct has a Validate() error method that enforces namespace cross-checks, required fields, and namespace equality. (`EnsureCustomerInput.Validate() checks AppID.Namespace == CustomerID.Namespace`)
**Compile-time interface assertion per file** — Each file that implements an interface has a package-level var _ <Interface> = (*ConcreteType)(nil) or var _ <Interface> = ConcreteType{} assertion to catch drift at compile time. (`var _ app.Service = (*service)(nil)`)
**GetEventAppData for event serialization** — App types implement GetEventAppData() (EventAppData, error) to produce a map[string]any for domain events. Use NewEventAppData(v) for serialization; consumers call ParseInto(&target). (`app/event.go: NewEventApp(app) calls app.GetEventAppData() to build the EventApp payload`)
**models.Generic* error wrapping for domain errors** — All domain errors wrap a models.GenericNotFoundError, GenericValidationError, GenericConflictError, or GenericPreConditionFailedError so the HTTP encoder chain maps them to the correct status code. (`errors.go: AppNotFoundError wraps models.NewGenericNotFoundError(...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/app/adapter.go` | Adapter interface definition — the persistence boundary for the app domain. Composes AppAdapter + entutils.TxCreator. | All methods that write state must also appear in entutils.TxCreator chain; never call db directly outside TransactingRepo in the adapter implementation. |
| `openmeter/app/app.go` | App runtime interface + all input/output types for installed-app operations (GetAppInput, UpdateAppInput, CreateAppInput, ListAppInput, UpdateAppStatusInput). | AppConfigUpdate is an open interface — new app types inject a typed config update struct; must call AppConfigUpdate.Validate() inside UpdateAppInput.Validate(). |
| `openmeter/app/appbase.go` | AppBase value type with all shared fields (Type, Status, Listing, Metadata) and canonical implementations of GetAppBase, GetID, GetType, GetStatus, ValidateCapabilities. | Do not add app-type-specific fields here. AppType constants must be kept in sync with Ent schema app_type enum. |
| `openmeter/app/registry.go` | AppFactory, AppFactoryInstallWithAPIKey, AppFactoryInstall interfaces and RegistryItem. This is the extension point for new app types. | RegistryItem.Factory must be non-nil; Validate() enforces this. A new app type needs both a Factory implementation and a registered MarketplaceListing. |
| `openmeter/app/service.go` | Service interface — composes MarketplaceService, AppLifecycleService, CustomerDataService, and models.ServiceHooks[AppBase]. | Marketplace methods operate on the in-memory registry. Installed-app mutating methods (CreateApp, UpdateApp, UninstallApp) open a plain transaction.Run in service/ and fire hook pairs around the adapter call. |
| `openmeter/app/errors.go` | Typed domain errors: AppNotFoundError, AppDefaultNotFoundError, AppProviderAuthenticationError, AppProviderError, AppProviderPreConditionError, AppCustomerPreConditionError. | Each error type has a corresponding IsXxxError(err) helper. Always use these typed errors; never return raw fmt.Errorf from service or adapter methods. |
| `openmeter/app/event.go` | Domain event types (AppCreateEvent, AppUpdateEvent, AppDeleteEvent) and EventApp/EventAppData serialization helpers. | AppUpdateEvent is v2 (carries full EventApp); AppCreateEvent is v1 (carries only AppBase). Do not bump version numbers in-place — add a new versioned struct. |

## Anti-Patterns

- Calling adapter/Ent methods directly inside App method implementations — app-type-specific logic must go through the service layer, not bypass it.
- Returning errors without wrapping in a models.Generic* type — HTTP error encoder will fall through to 500 for unrecognized error types.
- Constructing a RegistryItem without setting Factory — RegistryItem.Validate() panics and the app cannot be installed.
- Adding new AppType constants without updating both AppType.Validate() and the Ent schema app_type enum — causes runtime panics on unknown type.
- Publishing events inside the adapter layer — event publishing is exclusively the service layer's responsibility after successful adapter writes.
- Registering hooks from inside a domain package constructor — always register from app/common provider functions to avoid circular imports between billing, subscription, and app.

## Decisions

- **Registry (marketplace listings) lives in-memory on the adapter, not in the database.** — Listings are static metadata provided by factories at startup; they don't need persistence and would add unnecessary DB round-trips for every install call.
- **App.GetEventAppData() decouples app-type-specific serialization from the event infrastructure.** — EventApp is a type-neutral map[string]any so the Watermill bus doesn't need to know about concrete app types; consumers use ParseInto to decode.
- **OAuth2 install methods on the adapter return 'not implemented' — only API-key and no-credentials installs are storage-backed.** — OAuth2 flow state is transient and provider-specific; adapter-level storage would couple the adapter to provider protocols.
- **No per-app-type advisory lock — multiple apps of the same AppType per namespace are supported by design.** — The Ent schema index on (namespace, type) is non-unique; users legitimately run multiple Stripe region apps or multiple sandbox apps in one namespace. PostgreSQL row-level locking handles concurrent mutations of a single app row; cross-row serialization is not needed.

## Example: Implementing a new app type factory that self-registers

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
