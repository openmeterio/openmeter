# app

<!-- archie:ai-start -->

> Integration tests for the openmeter/app domain: the app marketplace registry plus per-app-type test suites (Stripe, custom-invoicing). The root files test marketplace listing/lookup; the constraint is that all app services are assembled directly from package constructors against a real Postgres DB, never through app/common DI wiring.

## Patterns

**TestEnv interface + testEnv struct** — Each suite gets a TestEnv interface exposing Adapter()/App()/Close(); NewTestEnv wires a real Postgres driver, runs migrations, and constructs adapters+services, returning a closerFunc that closes the ent + postgres drivers. (`env, err := NewTestEnv(t, ctx); defer env.Close()`)
**Direct constructor wiring** — Services are built from <pkg>adapter.New + <pkg>service.New (customer, secret, app, appstripe, billing) — not from app/common. Each New call returns (svc, err) checked with fmt.Errorf wrapping or require.NoError. (`appAdapter, err := appadapter.New(appadapter.Config{Client: entClient}); appService, err := appservice.New(appservice.Config{Adapter: appAdapter, Publisher: publisher})`)
**Real Postgres via testutils.InitPostgresDB + migrate** — InitPostgresDB(t) yields an EntDriver; migrate.New(MigrateOptions{Migrations: migrate.OMMigrationsConfig}) then migrator.Up() creates the schema before adapters are built. (`driver := testutils.InitPostgresDB(t); migrator, _ := migrate.New(...); migrator.Up()`)
**Validated Init input structs** — Helper assemblers like InitBillingService take an input struct (InitBillingServiceInput) with a Validate() error method checking required deps (DBClient, CustomerService, AppService) before construction. (`func (i InitBillingServiceInput) Validate() error { if i.DBClient == nil { return fmt.Errorf("db client is required") } ... }`)
**Per-suite namespace via ulid** — Suites generate an isolated namespace string with ulid.Make().String() in a setupNamespace(t) helper so tests do not collide. (`s.namespace = ulid.Make().String()`)
**Mock streaming/eventbus injected** — eventbus.NewMock(t) supplies the Publisher and streamingtestutils.NewMockStreamingConnector(t) the streaming connector; meteradapter is the mockadapter bound to the ent client via SetDBClient. (`publisher := eventbus.NewMock(t); mockStreamingConnector := streamingtestutils.NewMockStreamingConnector(t)`)
**Marketplace asserted against canonical listing** — Marketplace tests compare GetMarketplaceListing/ListMarketplaceListings output to appstripe.StripeMarketplaceListing and assert not-found via models.IsGenericNotFoundError for unknown AppType. (`require.Equal(t, expectedListing.Name, listing.Name); require.True(t, models.IsGenericNotFoundError(err))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | Builds the shared TestEnv (Postgres + migrations + customer/secret/app/appstripe/billing services) and the InitBillingService helper. The appstripe service is constructed for side effects (registration) and its return value discarded. | appstripeservice.New is called with _ = (return discarded); needs WebhookURLGenerator (NewBaseURLWebhookURLGenerator) and a BillingService. Forgetting migrator.Up() leaves an empty schema. |
| `marketplace.go` | AppHandlerTestSuite with TestGetMarketplaceListing / TestListMarketplaceListings; setupNamespace seeds a ulid namespace. TestType = app.AppTypeStripe. | Expects exactly one marketplace listing (list.TotalCount == 1); adding marketplace apps breaks these counts. |
| `app_test.go` | Top-level TestApp entrypoint that builds env via NewTestEnv and runs the Marketplace subtests through AppHandlerTestSuite. | Uses context.WithCancel(context.Background()) at the test root and defers cancel + env.Close(). |

## Anti-Patterns

- Importing app/common or production DI wiring instead of constructing adapters/services directly — creates test-only import cycles.
- Skipping migrator.Up() or reusing a non-isolated namespace, causing cross-test data bleed.
- Asserting marketplace results against hand-built listings instead of appstripe.StripeMarketplaceListing.
- Matching errors by string instead of typed predicates like models.IsGenericNotFoundError.
- Leaking the ent/postgres drivers by not deferring env.Close()/closerFunc().

## Decisions

- **Assemble app/customer/secret/billing services from package constructors against a real Postgres DB rather than app/common wiring.** — Keeps test deps narrow and avoids import cycles while still exercising real adapter SQL and the marketplace registry.
- **Construct the appstripe service for its registration side effect and discard the handle.** — The marketplace listing is registered at service construction; the test only needs it present in the registry, not a direct reference.

## Example: Standing up the app TestEnv with real Postgres and constructor-wired services

```
driver := testutils.InitPostgresDB(t)
entClient := driver.EntDriver.Client()
migrator, _ := migrate.New(migrate.MigrateOptions{ConnectionString: driver.URL, Migrations: migrate.OMMigrationsConfig, Logger: testutils.NewLogger(t)})
_ = migrator.Up()
appAdapter, _ := appadapter.New(appadapter.Config{Client: entClient})
appService, _ := appservice.New(appservice.Config{Adapter: appAdapter, Publisher: eventbus.NewMock(t)})
```

<!-- archie:ai-end -->
