# app

<!-- archie:ai-start -->

> Third-party app/marketplace framework. The root package declares the domain contracts (app.App, app.Service, app.Adapter, app.AppFactory/RegistryItem, AppBase, MarketplaceListing, Capability) and the app event/error vocabulary; concrete apps (stripe/, custominvoicing/, sandbox/) plug into billing via these interfaces and self-register in the marketplace registry.

## Patterns

**App identity = AppBase data + factory-built behavior** — AppBase (embeds models.ManagedResource + Type/Status/Listing/Metadata) is the persisted row; an AppFactory.NewApp(ctx, AppBase) turns a row into a live App implementing the full interface (UpdateAppConfig, customer-data methods, ValidateCapabilities). (`type App interface { GetAppBase() AppBase; ValidateCapabilities(...CapabilityType) error; UpsertCustomerData(...) error }`)
**Every input/value type is self-validating** — All *Input and value types implement Validate() returning models.NewGenericValidationError or errors.Join(errs...); namespace-coherence checks (e.g. CustomerID.Namespace == AppID.Namespace) live in Validate, not the service. (`func (a EnsureCustomerInput) Validate() error { ...; if a.AppID.Namespace != a.CustomerID.Namespace { return fmt.Errorf(...) } }`)
**Typed errors with IsX helpers** — Each error (AppNotFoundError, AppCustomerPreConditionError, AppProviderError) is built via NewX(...), wraps a models.Generic*Error, and exposes Error/Unwrap plus an IsX(err) helper backed by errors.As; a `var _ models.GenericError` assertion guards each. (`var _ models.GenericError = (*AppCustomerPreConditionError)(nil)`)
**AppType is a closed enum gating the discriminated union** — AppType (stripe|sandbox|custom_invoicing) validates against a fixed switch; adding a type requires extending AppType.Validate plus every httpdriver Discriminator switch. (`AppTypeStripe AppType = "stripe"`)
**Events versioned via metadata.EventType** — Lifecycle events (AppCreateEvent v1, AppUpdate/Delete v2) embed AppBase/EventApp, set EventName via metadata.GetEventName, build EventMetadata with metadata.ComposeResourcePath(namespace, EntityApp, id). (`metadata.GetEventName(metadata.EventType{Subsystem: AppEventSubsystem, Name: AppUpdateEventName, Version: "v2"})`)
**App config carried as serializable EventAppData** — EventAppData is a map[string]any from a JSON round-trip (NewEventAppData) consumed via ParseInto(ptr); apps surface config through GetEventAppData rather than concrete structs. (`func NewEventApp(app App) (EventApp, error) { appData, err := app.GetEventAppData(); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `app.go` | App interface + app-customer-data input types (Get/Upsert/Delete + Validate) | App is the live-behavior interface, not the row; no persistence here. |
| `appbase.go` | AppType/AppStatus/CapabilityType enums, AppBase + getters, AppID | AppType.Validate is a closed switch; new types must be added here AND in every consuming discriminator switch. |
| `adapter.go / service.go` | Adapter and Service interfaces (marketplace + installed-app + customer-data); adapter embeds entutils.TxCreator | Keep Service and Adapter method sets in lockstep; service is a thin validate-then-delegate wrapper. |
| `registry.go` | AppFactory contracts (NewApp, InstallApp, InstallAppWithAPIKey), RegistryItem (Listing+Factory) | RegistryItem.Validate requires a non-nil Factory; the marketplace is in-memory, not a DB table. |
| `errors.go` | Typed app errors wrapping models.Generic*Error with IsX helpers | Return these (esp. AppCustomerPreConditionError) instead of bare errors so HTTP mapping stays correct. |
| `event.go / events.go` | App lifecycle events and app_customer payment-setup event, versioned | Delete EventMetadata dereferences DeletedAt; ensure it is set. |
| `marketplace.go` | MarketplaceListing, Capability, InstallMethod enums, install input types | MarketplaceListing.Validate requires non-empty Type/Name/Description and validates each capability/install-method. |

## Anti-Patterns

- Putting Ent/SQL or HTTP logic in the root app package (contracts + events only; persistence in adapter/, transport in httpdriver/).
- Adding an AppType without extending AppType.Validate and every discriminator switch (UpdateApp, MapAppToAPI, toCustomerData).
- Returning bare errors instead of the typed Generic*Error wrappers, breaking IsX detection and status mapping.
- Constructing an App by hand instead of routing AppBase through an AppFactory.NewApp.
- Skipping the namespace-coherence checks in *Input.Validate (AppID.Namespace must match CustomerID.Namespace).

## Decisions

- **App is split into a persisted AppBase row plus a factory-built live App.** — Lets generic CRUD/marketplace logic stay app-agnostic while per-type behavior (Stripe/sandbox) is injected via factories registered in the in-memory marketplace.
- **App config is serialized through EventAppData (JSON map) rather than concrete union types.** — Apps are an interface, not a closed union; the map keeps events app-neutral until a proper union refactor (noted in TODOs).

## Example: Declaring a versioned app event with resource-path metadata

```
func (e AppUpdateEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{Subsystem: AppEventSubsystem, Name: AppUpdateEventName, Version: "v2"})
}

func (e AppUpdateEvent) EventMetadata() metadata.EventMetadata {
	appBase := e.AppBase.GetAppBase()
	resourcePath := metadata.ComposeResourcePath(appBase.Namespace, metadata.EntityApp, appBase.ID)
	return metadata.EventMetadata{ID: ulid.Make().String(), Source: resourcePath, Subject: resourcePath, Time: appBase.UpdatedAt}
}
```

<!-- archie:ai-end -->
