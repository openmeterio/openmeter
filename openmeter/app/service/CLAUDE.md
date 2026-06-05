# service

<!-- archie:ai-start -->

> Service layer for the app/marketplace framework: thin orchestration over app.Adapter that validates inputs, runs transactions, and publishes domain events. Implements app.Service (app CRUD, app-customer data, marketplace listing/install).

## Patterns

**Validate-then-delegate** — Every public method calls input.Validate() first and wraps the error in models.NewGenericValidationError before delegating to s.adapter. Read methods are pure pass-throughs after validation. (`if err := input.Validate(); err != nil { return ..., models.NewGenericValidationError(err) }; return s.adapter.GetApp(ctx, input)`)
**Publish events after mutations** — Create/Update/Uninstall/UpdateAppStatus emit events via s.publisher.Publish: app.NewAppCreateEvent, NewAppUpdateEvent, NewAppDeleteEvent. UninstallApp captures existingApp.GetEventAppData() before delete. (`event := app.NewAppCreateEvent(ctx, appBase); s.publisher.Publish(ctx, event)`)
**Transaction wraps multi-step updates** — UpdateApp runs inside transaction.Run(ctx, s.adapter, ...) so the adapter update, per-app UpdateAppConfig, re-fetch, and event publish are atomic. (`transaction.Run(ctx, s.adapter, func(ctx) { adapter.UpdateApp(...); updatedApp.UpdateAppConfig(...); publish(...) })`)
**App-config update routed to the typed app** — When input.AppConfigUpdate != nil, the service calls updatedApp.UpdateAppConfig(ctx, ...) on the typed app then re-fetches via adapter.GetApp to return fresh state. (`if input.AppConfigUpdate != nil { updatedApp.UpdateAppConfig(ctx, input.AppConfigUpdate); updatedApp, _ = s.adapter.GetApp(ctx, input.AppID) }`)
**Constructor with required deps** — Service{adapter, publisher}; Config.Validate rejects nil adapter or publisher; New(config) returns *Service. No slog.Default fallbacks. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &Service{...}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct + Config + Validate + New; pins var _ app.Service = (*Service)(nil) | Both adapter and publisher are mandatory; do not introduce a default publisher |
| `app.go` | CreateApp/GetApp/UpdateApp/ListApps/UninstallApp/UpdateAppStatus with validation + event publishing | UpdateApp must re-fetch after UpdateAppConfig so the published event/return reflect config changes; UninstallApp reads EventAppData from the existing app before deletion |
| `customer.go` | ListCustomerData/EnsureCustomer/DeleteCustomer as direct adapter pass-throughs | These do not re-validate (validation happens in adapter/http layer); no events emitted here |
| `marketplace.go` | RegisterMarketplaceListing + Get/List/Install[WithAPIKey] + Oauth2 install URL/authorize, all validate-then-delegate | Oauth2 methods delegate to adapter which returns 'not implemented' |

## Anti-Patterns

- Skipping input.Validate()/NewGenericValidationError before calling the adapter on mutating methods
- Performing multi-step mutations (update + config + event) outside transaction.Run
- Publishing events before the adapter mutation succeeds, or omitting events on create/update/delete
- Putting Ent/SQL access in the service instead of delegating to app.Adapter

## Decisions

- **Service is a validation + eventing wrapper, persistence lives in the adapter** — Standard OpenMeter service/adapter split keeps DB concerns isolated and makes events a service-layer responsibility
- **UpdateApp re-fetches after applying typed AppConfigUpdate** — App config is stored in app-type-specific entities, so the generic app row must be reloaded to emit an accurate update event

## Example: Transactional update applying typed config then publishing an event

```
func (s *Service) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.App, error) {
		updatedApp, err := s.adapter.UpdateApp(ctx, input)
		if err != nil { return nil, err }
		if input.AppConfigUpdate != nil {
			if err := updatedApp.UpdateAppConfig(ctx, input.AppConfigUpdate); err != nil { return nil, err }
			if updatedApp, err = s.adapter.GetApp(ctx, input.AppID); err != nil { return nil, err }
		}
		event, err := app.NewAppUpdateEvent(ctx, updatedApp)
		if err != nil { return nil, err }
		return updatedApp, s.publisher.Publish(ctx, event)
	})
}
```

<!-- archie:ai-end -->
