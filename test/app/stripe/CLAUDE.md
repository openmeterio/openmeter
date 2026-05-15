# stripe

<!-- archie:ai-start -->

> Integration and unit tests for the Stripe app covering app install/get/update/uninstall lifecycle, customer data CRUD, invoice sync to Stripe, and session creation — all using mock Stripe clients to avoid real API calls. Primary constraint: never import app/common; wire all services manually from raw package constructors.

## Patterns

**TestEnv interface for app lifecycle tests** — app_test.go and appstripe.go use a TestEnv interface (App, AppStripe, Billing, Customer, Fixture, Secret, StripeClient, StripeAppClient, Close) provisioned by NewTestEnv — services wired manually without app/common. (`env, err := NewTestEnv(t, ctx); env.App().InstallMarketplaceListingWithAPIKey(...)`)
**StripeClientMock / StripeAppClientMock with Restore()** — Mock expectations are set per-test with .On(...).Return(...); always defer Restore() after each setup to clear ExpectedCalls[:0] without recreating the mock. (`s.Env.StripeClient().On("GetAccount").Return(stripeclient.StripeAccount{...}, nil); defer s.Env.StripeClient().Restore()`)
**Embed billingtest.BaseSuite for invoice tests** — StripeInvoiceTestSuite embeds billingtest.BaseSuite and builds extra services (SecretService, AppStripeService, Charges, LedgerResolver) in SetupSuite using raw constructors, not app/common wiring. (`type StripeInvoiceTestSuite struct { billingtest.BaseSuite; AppStripeService appstripe.Service; Fixture *Fixture; StripeAppClient *StripeAppClientMock; Charges charges.Service }`)
**Fixture for shared setup sequences** — fixture.go provides setupApp, setupCustomer, setupAppWithCustomer to avoid duplicating Stripe mock setup and app installation across tests. setupApp calls Restore() via defer after each setup. (`testApp, customer, customerData, err := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)`)
**Unique namespace per test via setupNamespace(t)** — Each test in AppHandlerTestSuite calls s.setupNamespace(t) which sets s.namespace = ulid.Make().String() to guarantee isolation. (`func (s *AppHandlerTestSuite) TestCreate(ctx context.Context, t *testing.T) { s.setupNamespace(t); ... }`)
**StableSortInvoiceItemParams for deterministic line matching** — StripeAppClientMock.AddInvoiceLines calls StableSortInvoiceItemParams to sort line items by Description before mock matching, making assertions order-independent. (`func (c *StripeAppClientMock) AddInvoiceLines(ctx context.Context, input stripeclient.AddInvoiceLinesInput) ... { c.StableSortInvoiceItemParams(input.Lines); args := c.Called(input) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | Constructs the complete test dependency graph for app_test.go: Postgres via testutils.InitPostgresDB, customer/app/secret/stripe services wired manually, closerFunc for cleanup. | New services added to the stripe test env must be added to both the testEnv struct and the TestEnv interface. Uses context.Background() in NewTestEnv — acceptable for env setup. |
| `stripe_mock.go` | Implements stripeclient.StripeClient and stripeclient.StripeAppClient with testify/mock; includes StableSortInvoiceItemParams/StableSortStripeInvoiceItemWithID to normalize line item order. | Restore() clears c.ExpectedCalls[:0] — forgetting to call it leaks expectations across tests causing spurious failures in subsequent tests. |
| `invoice_test.go` | Deep invoice sync tests with StripeInvoiceTestSuite embedding billingtest.BaseSuite; builds Stripe mock call sequences for CreateInvoice/AddInvoiceLines/FinalizeInvoice and verifies exact Stripe API call shapes. | Stripe mock calls must match exact input structs including sorted line items — use expectStripeInvoiceAddLines helper which matches by Description key rather than index order. Ledger-backed tests (invoice_credits_test.go) require createStripeLedgerBackedCustomer which calls EnsureBusinessAccounts and CreateCustomerAccounts before any charge creation. |
| `invoice_credits_test.go` | Tests progressive billing with credit allocation, covering full credit-then-invoice settlement mode across partial and final invoices synced to Stripe. | Uses charges.NewChargeIntent (not struct literal Charge{}) and mustSettleExternalCreditPurchase to advance credit purchase through Authorized→Settled states before usage charges are created. clock.FreezeTime must be set appropriately for each invoice advancement step. |
| `fixture.go` | Shared setup helpers for Stripe app install, customer creation, and customer data attachment. getStripeAccountId generates a random account ID each call. | Tests calling setupApp twice in the same test will get different Stripe account IDs — configure mock to handle both or use a single setupAppWithCustomer call. |
| `secret_mock.go` | MockSecretService wraps real secretservice with optional mock interception; mockEnabled=false (passthrough) by default. | var _ secret.SecretService = (*MockSecretService)(nil) compile check enforces interface completeness — add new methods to MockSecretService when secret.Service interface gains new methods. |

## Anti-Patterns

- Calling Stripe client methods directly in tests instead of going through app.Service or appstripe.Service
- Forgetting defer stripeClient.Restore() / stripeAppClient.Restore() — leftover mock expectations cause subsequent tests to fail
- Importing app/common in test env construction — always wire from raw package constructors (appadapter.New, appservice.New, etc.)
- Reusing the same namespace across multiple test functions in AppHandlerTestSuite — always call setupNamespace(t) at the start of each test method
- Adding invoice line assertions without stable-sorting lines — use expectStripeInvoiceAddLines or StableSortInvoiceItemParams before setting mock expectations on AddInvoiceLines

## Decisions

- **Two parallel test architectures: TestEnv interface (app_test.go) vs billingtest.BaseSuite embedding (invoice_test.go)** — App lifecycle tests need a narrow env without the full billing stack; invoice tests need BaseSuite for BillingService, MockStreamingConnector, and meter setup — different scopes justified different setups.
- **MockSecretService wraps real secretservice with Enable/Disable toggle rather than pure mock** — Most tests need real secret persistence (storing Stripe API keys); only error-path tests need mock control, so passthrough-by-default avoids brittle expectation setup in the happy path.
- **Stripe client mock sorts invoice line items before matching** — Billing line generation order is non-deterministic; stable sorting in the mock (StableSortInvoiceItemParams) makes test assertions order-independent without requiring production code to sort.

## Example: Install a Stripe app and verify it can be retrieved

```
import (
  "github.com/openmeterio/openmeter/openmeter/app"
  stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
)

func (s *AppHandlerTestSuite) TestGet(ctx context.Context, t *testing.T) {
  s.setupNamespace(t)
  testApp, err := s.Env.Fixture().setupApp(ctx, s.namespace)
  require.NoError(t, err)
  getApp, err := s.Env.App().GetApp(ctx, testApp.GetID())
  require.NoError(t, err)
  require.Equal(t, testApp.GetID(), getApp.GetID())
  // 404 path:
  _, err = s.Env.App().GetApp(ctx, app.AppID{Namespace: s.namespace, ID: "not_found"})
  require.True(t, app.IsAppNotFoundError(err))
// ...
```

<!-- archie:ai-end -->
