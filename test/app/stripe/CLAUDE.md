# stripe

<!-- archie:ai-start -->

> Integration and unit tests for the Stripe app covering install/get/update/uninstall, customer data CRUD, invoice sync to Stripe, and session creation — all using mock Stripe clients to avoid real API calls. Never import app/common; wire all services manually from raw constructors.

## Patterns

**TestEnv interface for app lifecycle tests** — app_test.go and appstripe.go use a TestEnv interface (App, AppStripe, Billing, Customer, Fixture, Secret, StripeClient, StripeAppClient, Close) provisioned by NewTestEnv — services wired without app/common. (`env, _ := NewTestEnv(t, ctx); env.App().InstallMarketplaceListingWithAPIKey(...)`)
**Mock clients with Restore()** — Set mock expectations per-test with .On(...).Return(...); always defer Restore() to clear ExpectedCalls without recreating the mock. (`s.Env.StripeClient().On("GetAccount").Return(stripeclient.StripeAccount{...}, nil); defer s.Env.StripeClient().Restore()`)
**Embed billingtest.BaseSuite for invoice tests** — StripeInvoiceTestSuite embeds billingtest.BaseSuite and builds extra services (SecretService, AppStripeService, Charges, LedgerResolver) in SetupSuite from raw constructors. (`type StripeInvoiceTestSuite struct { billingtest.BaseSuite; AppStripeService appstripe.Service; Fixture *Fixture; StripeAppClient *StripeAppClientMock; Charges charges.Service }`)
**Fixture for shared setup sequences** — fixture.go provides setupApp / setupCustomer / setupAppWithCustomer to avoid duplicating Stripe mock setup; setupApp defers Restore(). (`testApp, customer, customerData, _ := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)`)
**Unique namespace per test via setupNamespace(t)** — Each AppHandlerTestSuite test calls s.setupNamespace(t) which sets s.namespace = ulid.Make().String(). (`func (s *AppHandlerTestSuite) TestCreate(ctx context.Context, t *testing.T) { s.setupNamespace(t); ... }`)
**StableSort invoice lines for deterministic matching** — StripeAppClientMock.AddInvoiceLines calls StableSortInvoiceItemParams to sort line items by Description before mock matching, making assertions order-independent. (`func (c *StripeAppClientMock) AddInvoiceLines(ctx, input) ... { c.StableSortInvoiceItemParams(input.Lines); args := c.Called(input) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | Constructs the full test dependency graph for app_test.go: Postgres via testutils.InitPostgresDB, customer/app/secret/stripe services wired manually, closerFunc for cleanup. | New services must be added to both the testEnv struct and the TestEnv interface. Uses context.Background() in NewTestEnv — acceptable for env setup. |
| `stripe_mock.go` | Implements stripeclient.StripeClient and StripeAppClient with testify/mock; includes StableSortInvoiceItemParams/StableSortStripeInvoiceItemWithID. | Restore() clears c.ExpectedCalls[:0] — forgetting it leaks expectations across tests, causing spurious failures. |
| `invoice_test.go` | Deep invoice sync tests with StripeInvoiceTestSuite; builds Stripe mock call sequences for CreateInvoice/AddInvoiceLines/FinalizeInvoice and verifies exact call shapes. | Stripe calls must match exact input structs including sorted lines — use expectStripeInvoiceAddLines (matches by Description, not index order). |
| `invoice_credits_test.go` | Tests progressive billing with credit allocation across partial and final invoices synced to Stripe (credit-then-invoice settlement). | Uses charges.NewChargeIntent (not struct literal Charge{}) and mustSettleExternalCreditPurchase to advance credit purchase Authorized→Settled. clock.FreezeTime must be set per advancement step; createStripeLedgerBackedCustomer calls EnsureBusinessAccounts/CreateCustomerAccounts first. |
| `fixture.go` | Shared setup helpers for app install, customer creation, customer data attachment; getStripeAccountId returns a random ID each call. | Calling setupApp twice in one test yields different Stripe account IDs — configure the mock for both or use a single setupAppWithCustomer call. |
| `secret_mock.go` | MockSecretService wraps real secretservice with optional interception; mockEnabled=false (passthrough) by default. | var _ secret.SecretService = (*MockSecretService)(nil) compile check — add new methods when secret.Service gains them. |

## Anti-Patterns

- Calling Stripe client methods directly instead of through app.Service or appstripe.Service.
- Forgetting defer stripeClient.Restore() / stripeAppClient.Restore() — leftover expectations fail subsequent tests.
- Importing app/common in test env construction — wire from raw constructors (appadapter.New, appservice.New, etc.).
- Reusing the same namespace across AppHandlerTestSuite test functions — call setupNamespace(t) at the start of each test method.
- Adding invoice line assertions without stable-sorting — use expectStripeInvoiceAddLines or StableSortInvoiceItemParams before setting AddInvoiceLines expectations.

## Decisions

- **Two parallel test architectures: TestEnv interface (app_test.go) vs billingtest.BaseSuite embedding (invoice_test.go).** — App lifecycle tests need a narrow env; invoice tests need BaseSuite for BillingService, MockStreamingConnector, and meter setup — different scopes.
- **MockSecretService wraps real secretservice with Enable/Disable toggle rather than a pure mock.** — Most tests need real secret persistence (Stripe API keys); only error-path tests need mock control, so passthrough-by-default avoids brittle happy-path expectations.
- **The Stripe client mock sorts invoice line items before matching.** — Billing line generation order is non-deterministic; stable sorting in the mock makes assertions order-independent without sorting in production code.

## Example: Install a Stripe app and verify it can be retrieved

```
import (
  "github.com/openmeterio/openmeter/openmeter/app"
  stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
)

func (s *AppHandlerTestSuite) TestGet(ctx context.Context, t *testing.T) {
  s.setupNamespace(t)
  testApp, _ := s.Env.Fixture().setupApp(ctx, s.namespace)
  getApp, _ := s.Env.App().GetApp(ctx, testApp.GetID())
  require.Equal(t, testApp.GetID(), getApp.GetID())
  _, err := s.Env.App().GetApp(ctx, app.AppID{Namespace: s.namespace, ID: "not_found"})
  require.True(t, app.IsAppNotFoundError(err))
}
```

<!-- archie:ai-end -->
