# app

<!-- archie:ai-start -->

> Integration test root for app marketplace integrations — wires all services from raw package constructors against a real pgtestdb Postgres DB and validates app lifecycle, marketplace listings, and invoice sync without importing app/common or Wire provider sets. Its custominvoicing/ and stripe/ children own the per-provider invoice-sync and lifecycle suites.

## Patterns

**TestEnv interface + testEnv struct** — testenv.go defines a TestEnv interface (Adapter(), App(), Close()) with a compile-time assertion and a concrete testEnv. NewTestEnv wires every service via direct package constructors. (`type TestEnv interface { Adapter() app.Adapter; App() app.Service; Close() error }; var _ TestEnv = (*testEnv)(nil)`)
**migrate.Up() before service construction** — Each environment provisions Postgres via testutils.InitPostgresDB(t), runs migrator.Up() with migrate.OMMigrationsConfig, and defers migrator.CloseOrLogError() immediately. (`migrator, _ := migrate.New(migrate.MigrateOptions{ConnectionString: driver.URL, Migrations: migrate.OMMigrationsConfig}); if err := migrator.Up(); err != nil { t.Fatalf(...) }; defer migrator.CloseOrLogError()`)
**InitBillingService shared helper** — Billing construction is extracted into InitBillingService(t, ctx, InitBillingServiceInput) (validated via Input.Validate()) so app_test.go and stripe/invoice_test.go reuse the same wiring chain; it builds the entitlement registry, meter mockadapter, taxcode, lockr, and billingservice.New with ForegroundAdvancementStrategy. (`billingService, err := InitBillingService(t, ctx, InitBillingServiceInput{DBClient: entClient, CustomerService: customerService, AppService: appService})`)
**eventbus.NewMock(t) for the publisher** — All environments use eventbus.NewMock(t) instead of a real Kafka publisher — the only sanctioned publisher mock here. (`publisher := eventbus.NewMock(t)`)
**closerFunc closes entClient AND driver.EntDriver** — The testEnv closerFunc must close both entClient and driver.EntDriver, joining errors with errors.Join; missing either leaks a DB connection. (`closerFunc := func() error { var errs error; if err = entClient.Close(); err != nil { errs = errors.Join(errs, err) }; if err = driver.EntDriver.Close(); err != nil { errs = errors.Join(errs, err) }; return errs }`)
**setupNamespace(t) per test method** — AppHandlerTestSuite.setupNamespace(t) assigns a fresh ULID namespace at the top of every test method for full isolation; not-found assertions use models.IsGenericNotFoundError. (`func (s *AppHandlerTestSuite) TestGetMarketplaceListing(ctx context.Context, t *testing.T) { s.setupNamespace(t); service := s.Env.App(); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | Constructs the full app service graph; defines TestEnv, testEnv, NewTestEnv, and the InitBillingService helper + InitBillingServiceInput.Validate(). | Never import app/common; always defer migrator.CloseOrLogError(); closerFunc must close both entClient and driver.EntDriver; validate InitBillingServiceInput before constructing. |
| `marketplace.go` | AppHandlerTestSuite with TestGetMarketplaceListing / TestListMarketplaceListings methods. | Call setupNamespace(t) first in each method; use models.IsGenericNotFoundError for not-found, not string matching. |
| `app_test.go` | Entry point: creates TestEnv, defers Close(), runs AppHandlerTestSuite sub-tests. | Defer ctx cancel; report env.Close() errors via t.Errorf (not require.NoError) so cleanup completes. |
| `custominvoicing/` | Child: async invoice-sync protocol tests (draft→issuing→payment) embedding billingtest.BaseSuite. | Drive the state machine through BillingService/CustomInvoicingService only; defer MockStreamingConnector.Reset(). |
| `stripe/` | Child: Stripe install/CRUD + invoice-sync tests using mock Stripe clients (TestEnv interface and billingtest.BaseSuite architectures coexist). | defer stripeClient.Restore()/stripeAppClient.Restore(); stable-sort invoice lines before setting AddInvoiceLines expectations. |

## Anti-Patterns

- Importing app/common or Wire provider sets in testenv.go — wire from raw constructors (appadapter.New, appservice.New, etc.)
- Sharing a namespace string across AppHandlerTestSuite methods — call setupNamespace(t) per method
- Calling Stripe client methods directly instead of through app.Service or appstripe.Service
- Forgetting defer stripeClient.Restore()/stripeAppClient.Restore() in stripe sub-tests — leftover expectations poison later tests
- Adding invoice-line assertions without stable-sorting lines before setting mock expectations

## Decisions

- **InitBillingService is a shared helper, not inline construction** — app_test.go and stripe/invoice_test.go both need a billing.Service; extracting it avoids duplication and keeps testenv.go focused on app wiring.
- **Two parallel test architectures: TestEnv interface vs billingtest.BaseSuite embedding** — Simple marketplace tests need only App()/Adapter(); invoice-sync tests need the full billing suite machinery — both coexist.
- **Raw package constructors only, no app/common imports** — app/common imports all domain packages; importing it from tests creates cycles and forces every domain to compile for a single test.

## Example: Minimal TestEnv from raw constructors

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
	if err := migrator.Up(); err != nil { t.Fatalf("%v", err) }
	defer migrator.CloseOrLogError()
	// ... appadapter.New / appservice.New ...
// ...
```

<!-- archie:ai-end -->
