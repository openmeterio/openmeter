# service

<!-- archie:ai-start -->

> Thin orchestration layer implementing app.Service — validates inputs, delegates persistence to app.Adapter, and publishes domain events (AppCreate, AppUpdate, AppDelete) via eventbus.Publisher after successful writes.

## Patterns

**Validate-then-delegate** — Every public method calls input.Validate() first; on error it wraps with models.NewGenericValidationError and returns early. Only after validation succeeds does it call the adapter. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**Publish domain events after adapter writes** — CreateApp, UpdateApp, UpdateAppStatus, and UninstallApp each publish an event (NewAppCreateEvent, NewAppUpdateEvent, NewAppDeleteEvent) via s.publisher.Publish after the adapter call succeeds. The event is built from the returned domain object. (`event := app.NewAppCreateEvent(ctx, appBase); if err := s.publisher.Publish(ctx, event); err != nil { return app.AppBase{}, err }`)
**transaction.Run for UpdateApp config update** — UpdateApp wraps the adapter update + AppConfigUpdate call + re-fetch in transaction.Run(ctx, s.adapter, ...) so the config update and the base update are atomic. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.App, error) { updatedApp, _ := s.adapter.UpdateApp(ctx, input); updatedApp.UpdateAppConfig(ctx, input.AppConfigUpdate); ... })`)
**Config struct with Validate()** — Service is constructed via New(Config) where Config holds Adapter and Publisher and has a Validate() method. Callers cannot construct Service with nil dependencies. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Compile-time interface assertions per file** — Each file that implements a sub-interface declares var _ app.AppService = (*Service)(nil) at the top. This ensures method set completeness is caught at compile time per logical grouping. (`var _ app.AppService = (*Service)(nil) // in app.go, customer.go, marketplace.go`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct definition, Config/Validate, New() constructor. Holds adapter and publisher only. | Service has no direct Ent dependency — all DB access goes through app.Adapter interface. |
| `app.go` | CreateApp, GetApp, UpdateApp, ListApps, UninstallApp, UpdateAppStatus — plus event publishing for each mutating operation. | UninstallApp calls GetApp before UninstallApp on adapter to capture event data (GetEventAppData). If GetApp fails the app is not uninstalled. UpdateApp re-fetches after config update to include latest state in the event. |
| `customer.go` | ListCustomerData, EnsureCustomer, DeleteCustomer — all thin pass-throughs to adapter with no event publishing. | No events are emitted for customer data mutations — this is intentional; customer data changes are considered app-internal. |
| `marketplace.go` | RegisterMarketplaceListing, GetMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, InstallMarketplaceListing, OAuth2 methods — all pass-throughs with input validation. | RegisterMarketplaceListing does NOT emit an event — it mutates the in-memory registry on the adapter and is called at startup, not per-request. |

## Anti-Patterns

- Bypassing input.Validate() before adapter calls — validation must happen at the service layer so the adapter only sees valid inputs.
- Adding Ent queries directly to service methods — all persistence must go through app.Adapter.
- Publishing events inside the adapter — event publishing is the service layer's responsibility only.
- Adding new dependencies to Config without a corresponding nil check in Config.Validate().

## Decisions

- **Service delegates all persistence to app.Adapter without business logic** — The app domain's business logic is in the concrete app implementations (appsandbox, appstripe) and their factories; the service is purely an orchestration and validation boundary.
- **Events published after adapter success, not inside transactions** — Event publishing to Kafka is not transactional with Postgres; publishing after a successful adapter call means the DB write is committed before the event is emitted, preventing phantom events on rollback.

## Example: UpdateApp: validate, transact, config update, re-fetch, publish event

```
func (s *Service) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.App, error) {
		updatedApp, err := s.adapter.UpdateApp(ctx, input)
		if err != nil { return nil, err }
		if input.AppConfigUpdate != nil {
			if err := updatedApp.UpdateAppConfig(ctx, input.AppConfigUpdate); err != nil { return nil, err }
			updatedApp, err = s.adapter.GetApp(ctx, input.AppID)
			if err != nil { return nil, err }
		}
		event, err := app.NewAppUpdateEvent(ctx, updatedApp)
		if err != nil { return nil, err }
		if err := s.publisher.Publish(ctx, event); err != nil { return nil, err }
		return updatedApp, nil
	})
// ...
```

<!-- archie:ai-end -->
