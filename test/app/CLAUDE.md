# app

<!-- archie:ai-start -->

> Integration test root for app marketplace integrations (Stripe, custom invoicing) — wires all services from raw package constructors against a real pgtestdb Postgres DB and validates app lifecycle, invoice sync, and async invoice state protocols without importing app/common or Wire provider sets.

## Patterns

**TestEnv interface + testEnv struct** — Each sub-package defines a TestEnv interface exposing domain services (App(), Adapter(), Close()) and a concrete testEnv struct implementing it. NewTestEnv wires all services using direct package constructors only. (`type TestEnv interface { Adapter() app.Adapter; App() app.Service; Close() error }
var _ TestEnv = (*testEnv)(nil)`)
**InitBillingService shared helper** — Billing service construction is extracted into InitBillingService(t, ctx, InitBillingServiceInput) so multiple test files in the same package can reuse the constructor without duplicating the wiring chain. (`billingService, err := InitBillingService(t, ctx, InitBillingServiceInput{DBClient: entClient, CustomerService: customerService, AppService: appService})`)
**migrate.Up() before service construction** — Each test environment runs migrator.Up() on the pgtestdb-provisioned Postgres connection before constructing any services. Always defer migrator.CloseOrLogError() immediately after. (`migrator, err := migrate.New(migrate.MigrateOptions{ConnectionString: driver.URL, Migrations: migrate.OMMigrationsConfig})
_ = migrator.Up()
defer migrator.CloseOrLogError()`)
**setupNamespace(t) per test method** — AppHandlerTestSuite.setupNamespace(t) generates a fresh ULID namespace at the start of each test method to ensure full isolation. Must be called at the top of every test method before any service call. (`func (s *AppHandlerTestSuite) TestGetMarketplaceListing(ctx context.Context, t *testing.T) {
    s.setupNamespace(t)
    service := s.Env.App()
    // ...
}`)
**closerFunc closes both entClient and EntDriver** — The testEnv closerFunc must close both entClient.Close() and driver.EntDriver.Close(), joining errors with errors.Join. Missing either close leaks a DB connection. (`closerFunc := func() error {
    var errs error
    if err = entClient.Close(); err != nil { errs = errors.Join(errs, err) }
    if err = driver.EntDriver.Close(); err != nil { errs = errors.Join(errs, err) }
    return errs
}`)
**eventbus.NewMock(t) for publisher** — All test environments use eventbus.NewMock(t) instead of a real Kafka publisher. This is the only sanctioned publisher mock in this package. (`publisher := eventbus.NewMock(t)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `test/app/testenv.go` | Constructs the full service graph for app integration tests. Defines TestEnv interface, testEnv struct, NewTestEnv constructor, and InitBillingService helper. | Never import app/common here. Always defer migrator.CloseOrLogError(). closerFunc must close both entClient and driver.EntDriver. InitBillingServiceInput.Validate() must be called before constructing services. |
| `test/app/marketplace.go` | AppHandlerTestSuite with test methods for GetMarketplaceListing and ListMarketplaceListings. | Always call setupNamespace(t) at the top of each test method. Use models.IsGenericNotFoundError for not-found assertions, not require.Error with string matching. |
| `test/app/app_test.go` | Test entry point; creates TestEnv, defers Close(), then runs sub-tests via AppHandlerTestSuite. | ctx cancel must be deferred. env.Close() errors must be reported via t.Errorf, not require.NoError (to allow cleanup to complete). |

## Anti-Patterns

- Importing app/common or Wire provider sets in testenv.go — always wire from raw package constructors (appadapter.New, appservice.New, etc.)
- Sharing a namespace string across test methods in AppHandlerTestSuite — always call setupNamespace(t) per method
- Calling Stripe client methods directly instead of going through app.Service or appstripe.Service
- Forgetting defer stripeClient.Restore() / stripeAppClient.Restore() in stripe sub-tests — leftover mock expectations poison subsequent tests
- Adding invoice line assertions without stable-sorting lines before setting mock expectations — use StableSortInvoiceItemParams

## Decisions

- **InitBillingService is a shared helper rather than inline construction** — Multiple test files (app_test.go, stripe/invoice_test.go) need a billing.Service; extracting the constructor avoids duplication and keeps testenv.go focused on app-level wiring.
- **Two parallel test architectures: TestEnv interface vs billingtest.BaseSuite embedding** — Simple marketplace tests need only App() and Adapter(); invoice sync tests need the full billing suite machinery from billingtest.BaseSuite — the two patterns coexist without conflict.
- **Raw package constructors only, no app/common imports** — app/common imports all domain packages; importing it from test code creates import cycles and forces all domains to compile together for any single test.

## Example: Construct a minimal test environment from raw package constructors (testenv.go pattern)

```
import (
    appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
    appservice "github.com/openmeterio/openmeter/openmeter/app/service"
    "github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
    "github.com/openmeterio/openmeter/openmeter/testutils"
    "github.com/openmeterio/openmeter/tools/migrate"
)

func NewTestEnv(t *testing.T, ctx context.Context) (TestEnv, error) {
    publisher := eventbus.NewMock(t)
    driver := testutils.InitPostgresDB(t)
    entClient := driver.EntDriver.Client()
    migrator, _ := migrate.New(migrate.MigrateOptions{ConnectionString: driver.URL, Migrations: migrate.OMMigrationsConfig})
    _ = migrator.Up()
    defer migrator.CloseOrLogError()
// ...
```

<!-- archie:ai-end -->
