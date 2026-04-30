# service

<!-- archie:ai-start -->

> Thin orchestration layer implementing app.Service — validates inputs, fires ServiceHooks[AppBase] around mutations, delegates persistence to app.Adapter, and publishes domain events (AppCreate, AppUpdate, AppDelete) via eventbus.Publisher after successful writes.

## Patterns

**Validate-then-delegate** — Every public method calls input.Validate() first; on error it wraps with models.NewGenericValidationError and returns early. Only after validation succeeds does it call the adapter. (`if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }`)
**transaction.Run wraps all state-changing operations** — All mutating methods (CreateApp, UpdateApp, UninstallApp, UpdateAppStatus) wrap their adapter calls in `transaction.Run(ctx, s.adapter, fn)`. Hook firing and event publishing happen inside the closure so a hook error atomically rolls back the DB write. (`service/app.go: transaction.Run → hooks.Pre* → adapter write → hooks.Post* → publisher.Publish`)
**ServiceHooks[AppBase] fired inside the transaction closure** — PostCreate, PreUpdate/PostUpdate, PreDelete/PostDelete are invoked inside the transaction.Run fn closure. A hook error rolls back the DB write atomically. Pre* hooks fire before the adapter call; Post* hooks fire after. (`app.go: s.hooks.PreDelete → adapter.UninstallApp → s.hooks.PostDelete, all inside the closure`)
**Pre-fetch for hook payload** — UpdateApp and UninstallApp fetch the existing app via adapter.GetApp before opening the transaction. The pre-fetched value provides the hook's "before" snapshot without requiring the lock; the adapter write proceeds inside transaction.Run. (`existingApp, err := s.adapter.GetApp(ctx, input.AppID)  // before transaction.Run`)
**Publish domain events after adapter writes** — CreateApp, UpdateApp, UpdateAppStatus, and UninstallApp each publish an event (NewAppCreateEvent, NewAppUpdateEvent, NewAppDeleteEvent) via s.publisher.Publish after the adapter call succeeds, still inside the transaction closure. (`event := app.NewAppCreateEvent(ctx, appBase); s.publisher.Publish(ctx, event)`)
**Config struct with Validate()** — Service is constructed via New(Config) where Config holds Adapter and Publisher and has a Validate() method. Callers cannot construct Service with nil dependencies. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Compile-time interface assertions per file** — Each file that implements a sub-interface declares var _ app.AppService = (*Service)(nil) at the top. This ensures method set completeness is caught at compile time per logical grouping. (`var _ app.AppService = (*Service)(nil) // in app.go, customer.go, marketplace.go`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct definition, Config/Validate, New() constructor, RegisterHooks. Holds adapter, publisher, and hooks (ServiceHookRegistry[AppBase]). | No requestValidators field — pre-mutation cross-domain guards are out of scope for the app domain (no consumer registers them). |
| `app.go` | CreateApp, GetApp, UpdateApp, ListApps, UninstallApp, UpdateAppStatus — validator checks, hook fan-out (Pre/Post around adapter call inside transaction.Run), and event publishing for each mutating operation. | UninstallApp uses input directly as AppID (UninstallAppInput = AppID type alias). UpdateApp pre-fetches existing app before the transaction for the hook payload. |
| `app_hook_test.go` | Integration tests: PostCreate fires on CreateApp; PreUpdate+PostUpdate fire on UpdateApp; PreDelete+PostDelete fire on UninstallApp; failing PostCreate rolls back the DB write; failing PreDelete rolls back UninstallApp. | Uses apptestutils.NewTestEnv with DBSchemaMigrate and RegisterSandboxFactory:true; no billing service needed since tests create AppTypeSandbox rows directly via CreateApp. |
| `customer.go` | ListCustomerData, EnsureCustomer, DeleteCustomer — all thin pass-throughs to adapter with no event publishing. | No events are emitted for customer data mutations — this is intentional; customer data changes are considered app-internal. |
| `marketplace.go` | RegisterMarketplaceListing, GetMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, InstallMarketplaceListing, OAuth2 methods — all pass-throughs with input validation. | RegisterMarketplaceListing does NOT emit an event — it mutates the in-memory registry on the adapter and is called at startup, not per-request. |

## Anti-Patterns

- Bypassing input.Validate() before adapter calls — validation must happen at the service layer so the adapter only sees valid inputs.
- Adding Ent queries directly to service methods — all persistence must go through app.Adapter.
- Publishing events inside the adapter — event publishing is the service layer's responsibility only.
- Adding new dependencies to Config without a corresponding nil check in Config.Validate().
- Registering hooks from inside a domain package constructor — always register from app/common to avoid import cycles with billing and subscription.

## Decisions

- **Service delegates all persistence to app.Adapter without business logic** — The app domain's business logic is in the concrete app implementations (appsandbox, appstripe) and their factories; the service is purely an orchestration and validation boundary.
- **Events published inside transaction.Run, not after commit** — Watermill Kafka publish is not transactional with Postgres; publishing inside transaction.Run means the event is sent before the DB commit, but a failing publish rolls back the write. This is preferable to phantom events on rollback.

## Example: UpdateApp: validate, pre-fetch, transact, hook, config update, re-fetch, post-hook, publish event

```
func (s *Service) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	if err := input.Validate(); err != nil { return nil, models.NewGenericValidationError(err) }
	existingApp, err := s.adapter.GetApp(ctx, input.AppID)  // pre-fetch for hook payload
	if err != nil { return nil, err }
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.App, error) {
		existingBase := existingApp.GetAppBase()
		if err := s.hooks.PreUpdate(ctx, &existingBase); err != nil { return nil, err }
		updatedApp, err := s.adapter.UpdateApp(ctx, input)
		if err != nil { return nil, err }
		// optional config update + re-fetch
		updatedBase := updatedApp.GetAppBase()
		if err := s.hooks.PostUpdate(ctx, &updatedBase); err != nil { return nil, err }
		event, err := app.NewAppUpdateEvent(ctx, updatedApp)
		if err != nil { return nil, err }
		return updatedApp, s.publisher.Publish(ctx, event)
	})
// ...
```

<!-- archie:ai-end -->
