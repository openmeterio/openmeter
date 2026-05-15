# service

<!-- archie:ai-start -->

> Thin orchestration layer implementing app.Service — validates inputs, fires ServiceHooks[AppBase] around mutations inside transaction.Run, delegates all persistence to app.Adapter, and publishes domain events (AppCreate, AppUpdate, AppDelete) via eventbus.Publisher after successful writes.

## Patterns

**Validate-then-delegate** — Every public method calls input.Validate() first. On error it wraps with models.NewGenericValidationError and returns early before any adapter or hook call. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**transaction.Run wraps all mutating operations** — CreateApp, UpdateApp, UninstallApp, UpdateAppStatus all wrap their adapter calls in transaction.Run. Hook firing and event publishing happen inside the closure so a hook error atomically rolls back the DB write. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.App, error) { s.hooks.PreUpdate → adapter.UpdateApp → s.hooks.PostUpdate → publisher.Publish })`)
**ServiceHooks[AppBase] fired inside transaction closure** — Pre* hooks fire before the adapter call; Post* hooks fire after. A hook error rolls back the DB write atomically. CreateApp has only PostCreate; UpdateApp has PreUpdate+PostUpdate; UninstallApp has PreDelete+PostDelete. (`s.hooks.PreDelete(ctx, &existingBase) → adapter.UninstallApp → s.hooks.PostDelete — all inside transaction.Run`)
**Pre-fetch existing app before transaction for hook payload** — UpdateApp and UninstallApp fetch the existing app via adapter.GetApp before opening the transaction. The pre-fetched value provides the hook's 'before' snapshot without requiring a lock. (`existingApp, err := s.adapter.GetApp(ctx, input.AppID)  // before transaction.Run`)
**Domain event published inside transaction closure** — CreateApp, UpdateApp, UpdateAppStatus, and UninstallApp each publish an event (NewAppCreateEvent, NewAppUpdateEvent, NewAppDeleteEvent) via s.publisher.Publish inside the transaction closure. (`event := app.NewAppCreateEvent(ctx, appBase); if err := s.publisher.Publish(ctx, event); err != nil { return app.AppBase{}, err }`)
**Config struct with Validate() + compile-time assertions per file** — Service is constructed via New(Config) where Config.Validate() rejects nil dependencies. Each file declares var _ app.AppService = (*Service)(nil) at the top to catch incomplete method sets at compile time. (`var _ app.AppService = (*Service)(nil) // in app.go, customer.go, marketplace.go`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct definition, Config/Validate, New() constructor, RegisterHooks. Holds adapter, publisher, and hooks (ServiceHookRegistry[AppBase]). | No requestValidators field — pre-mutation cross-domain guards are out of scope for the app domain. |
| `app.go` | CreateApp, GetApp, UpdateApp, ListApps, UninstallApp, UpdateAppStatus — validation, hook fan-out inside transaction.Run, and event publishing for each mutating operation. | UpdateApp pre-fetches existing app before the transaction for the hook payload. If AppConfigUpdate is provided, UpdateApp calls updatedApp.UpdateAppConfig then re-fetches via adapter.GetApp inside the same transaction. |
| `app_hook_test.go` | Integration tests for ServiceHooks: PostCreate fires on CreateApp; Pre/PostUpdate fire on UpdateApp; Pre/PostDelete fire on UninstallApp; failing PostCreate rolls back the DB write. | Uses apptestutils.NewTestEnv with RegisterSandboxFactory:true to get GetApp/UpdateApp/UninstallApp working without a billing service. |
| `customer.go` | ListCustomerData, EnsureCustomer, DeleteCustomer — all thin pass-throughs to adapter with no event publishing. | No events are emitted for customer data mutations — intentional; customer data changes are app-internal. |
| `marketplace.go` | RegisterMarketplaceListing, GetMarketplaceListing, ListMarketplaceListings, Install* methods, OAuth2 methods — all pass-throughs with input validation. | RegisterMarketplaceListing does NOT emit an event — it mutates the in-memory registry on the adapter and is called at startup, not per-request. |

## Anti-Patterns

- Bypassing input.Validate() before adapter calls — validation must happen at the service layer.
- Adding Ent queries directly to service methods — all persistence must go through app.Adapter.
- Publishing events inside the adapter — event publishing is the service layer's responsibility only.
- Registering hooks from inside a domain package constructor — always register from app/common to avoid import cycles.

## Decisions

- **Service delegates all persistence to app.Adapter without additional business logic** — Business logic lives in concrete app implementations (appsandbox, appstripe) and their factories; the service is purely an orchestration and validation boundary.
- **Events published inside transaction.Run, not after commit** — Watermill Kafka publish is not transactional with Postgres; publishing inside transaction.Run means a failing publish rolls back the write, preventing phantom events on rollback.

## Example: UpdateApp: validate, pre-fetch, transact, hook, config update, re-fetch, post-hook, publish event

```
func (s *Service) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }
	existingApp, err := s.adapter.GetApp(ctx, input.AppID)
	if err != nil { return nil, err }
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.App, error) {
		existingBase := existingApp.GetAppBase()
		if err := s.hooks.PreUpdate(ctx, &existingBase); err != nil { return nil, err }
		updatedApp, err := s.adapter.UpdateApp(ctx, input)
		if err != nil { return nil, err }
		updatedBase := updatedApp.GetAppBase()
		if err := s.hooks.PostUpdate(ctx, &updatedBase); err != nil { return nil, err }
		event, err := app.NewAppUpdateEvent(ctx, updatedApp)
		if err != nil { return nil, err }
		return updatedApp, s.publisher.Publish(ctx, event)
	})
// ...
```

<!-- archie:ai-end -->
