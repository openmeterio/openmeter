# app

<!-- archie:ai-start -->

> Integration tests for app integrations (Stripe, custom invoicing) — validates marketplace listing lifecycle, invoice sync, and async invoice state protocols against a real Postgres DB without importing app/common wiring.

## Patterns

**TestEnv interface + testEnv struct** — Each sub-package defines a TestEnv interface exposing domain services (App(), Adapter(), etc.) and a concrete testEnv struct implementing it. NewTestEnv wires all services from raw package constructors (appadapter.New, appservice.New, etc.). (`type TestEnv interface { App() app.Service; Adapter() app.Adapter; Close() error }`)
**Suite struct with Env field** — Test logic is grouped in a *HandlerTestSuite struct embedding Env TestEnv and a namespace string. Each test method calls setupNamespace(t) at the top to get an isolated ULID namespace. (`type AppHandlerTestSuite struct { Env TestEnv; namespace string }`)
**Raw package constructors only** — testenv.go wires all services using direct package constructors (appadapter.New, billingadapter.New, billingservice.New, etc.) and never imports app/common or Wire provider sets. (`appAdapter, err := appadapter.New(appadapter.Config{Client: entClient})`)
**migrate.Up() in NewTestEnv** — Each test environment runs migrate.Up() on the pgtestdb-provisioned Postgres connection before constructing services, ensuring schema is current without relying on autoMigrate config. (`migrator.Up()`)
**InitBillingService helper function** — In test/app, billing service construction is extracted into InitBillingService(t, ctx, InitBillingServiceInput) to allow reuse across different test files in the same package. (`billingService, err := InitBillingService(t, ctx, InitBillingServiceInput{DBClient: entClient, CustomerService: customerService, AppService: appService})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `test/app/testenv.go` | Constructs the full service graph for app integration tests. Defines TestEnv interface, testEnv struct, NewTestEnv constructor, and InitBillingService helper. | Never import app/common here. Always call defer migrator.CloseOrLogError(). closerFunc must close both entClient and driver.EntDriver. |
| `test/app/marketplace.go` | AppHandlerTestSuite with test methods for GetMarketplaceListing and ListMarketplaceListings. | Always call setupNamespace(t) at the top of each test method. |
| `test/app/app_test.go` | Test entry point; creates TestEnv, defers Close(), then runs sub-tests via AppHandlerTestSuite. | ctx cancel must be deferred. env.Close() errors must be reported via t.Errorf. |

## Anti-Patterns

- Importing app/common or Wire provider sets in testenv.go — always wire from raw package constructors
- Sharing a namespace across test methods in AppHandlerTestSuite — always call setupNamespace(t) per method
- Calling Stripe client methods directly instead of going through app.Service or appstripe.Service
- Forgetting defer stripeClient.Restore() / stripeAppClient.Restore() in stripe sub-tests
- Adding invoice line assertions without stable-sorting before setting mock expectations

## Decisions

- **InitBillingService is a shared helper rather than inline construction** — Multiple test files (app_test.go, invoice_test.go) need a billing.Service; extracting the constructor avoids duplication and keeps testenv.go focused on the app-level wiring.
- **Two parallel test architectures: TestEnv interface (app_test.go) vs billingtest.BaseSuite embedding (stripe/invoice_test.go)** — Simple marketplace tests need only an App() and Adapter(), while invoice sync tests need the full billing suite machinery from billingtest.BaseSuite.

## Example: Wire a minimal test environment from raw constructors

```
import (
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

func NewTestEnv(t *testing.T, ctx context.Context) (TestEnv, error) {
	publisher := eventbus.NewMock(t)
	driver := testutils.InitPostgresDB(t)
	entClient := driver.EntDriver.Client()
	// run migrations
	migrator, _ := migrate.New(migrate.MigrateOptions{ConnectionString: driver.URL, Migrations: migrate.OMMigrationsConfig})
	_ = migrator.Up()
	defer migrator.CloseOrLogError()
	// wire services from raw constructors only
// ...
```

<!-- archie:ai-end -->
