# stripe

<!-- archie:ai-start -->

> Integration and unit tests for the Stripe app: covers app install/get/update/uninstall lifecycle, customer data CRUD, invoice sync to Stripe, and session creation — all using mock Stripe clients to avoid real API calls.

## Patterns

**TestEnv interface for app_test.go** — app_test.go and appstripe.go tests use a TestEnv interface (App, AppStripe, Billing, Customer, Fixture, Secret, StripeClient, StripeAppClient, Close) provisioned by NewTestEnv — services are wired manually without app/common. (`env, err := NewTestEnv(t, ctx); env.App().InstallMarketplaceListingWithAPIKey(...)`)
**StripeClientMock / StripeAppClientMock with Restore()** — Mock expectations are set per-test with .On(...).Return(...); call defer s.Env.StripeAppClient().Restore() or defer stripeClient.Restore() after each setup to clear expectations without recreating the mock. (`s.Env.StripeClient().On("GetAccount").Return(stripeclient.StripeAccount{...}, nil); defer s.Env.StripeClient().Restore()`)
**Embed billingtest.BaseSuite for invoice tests** — StripeInvoiceTestSuite embeds billingtest.BaseSuite and builds extra services (SecretService, AppStripeService) in SetupSuite using raw constructors, not app/common wiring. (`type StripeInvoiceTestSuite struct { billingtest.BaseSuite; AppStripeService appstripe.Service }`)
**Fixture for shared setup sequences** — fixture.go provides Fixture.setupApp, setupCustomer, setupAppWithCustomer to avoid duplicating Stripe mock setup and app installation across tests. (`testApp, customer, customerData, err := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)`)
**MockSecretService with passthrough mode** — MockSecretService wraps a real secretservice; EnableMock()/DisableMock() toggle mocked vs real behavior — tests that need real secret persistence call DisableMock (the default). (`secretService, _ := NewMockSecretService(); secretService.EnableMock(); secretService.On(...)`)
**Unique namespace per test via setupNamespace(t)** — Each test in AppHandlerTestSuite calls s.setupNamespace(t) which sets s.namespace = ulid.Make().String() to guarantee namespace isolation. (`func (s *AppHandlerTestSuite) TestCreate(ctx context.Context, t *testing.T) { s.setupNamespace(t); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | Constructs the complete test dependency graph for app_test.go: Postgres via testutils.InitPostgresDB, customer/app/secret/stripe services wired manually, closerFunc for cleanup. | Uses context.Background() in NewTestEnv — acceptable for env setup. New services added to the stripe test env must be added to both the testEnv struct and the TestEnv interface. |
| `stripe_mock.go` | Implements stripeclient.StripeClient and stripeclient.StripeAppClient with testify/mock; includes StableSortInvoiceItemParams/StableSortStripeInvoiceItemWithID to normalize line item order for deterministic mock matching. | Restore() clears c.ExpectedCalls[:0] — forgetting to call it leaks expectations across tests causing spurious failures. |
| `invoice_test.go` | Deep invoice sync tests: builds Stripe mock call sequences for CreateInvoice/AddInvoiceLines/FinalizeInvoice; verifies Stripe API call shapes match billing line structures. | Stripe mock calls must match exact input structs (including sorted line items) — use StableSortInvoiceItemParams in the mock to make AddInvoiceLines matching deterministic. |
| `fixture.go` | Shared setup helpers that install a Stripe app, create a customer, and attach customer data. Uses Restore() via defer after each setup call. | getStripeAccountId generates a random account ID — tests that call setupApp twice in the same test will have different account IDs unless the mock is configured to handle both. |
| `secret_mock.go` | Wraps real secretservice with optional mock interception; default is passthrough (mockEnabled=false). | The var _ secret.SecretService compile check enforces interface completeness — add new methods to both MockSecretService and the real service simultaneously. |

## Anti-Patterns

- Calling Stripe client methods directly in tests instead of going through app.Service or appstripe.Service
- Forgetting defer stripeClient.Restore() / stripeAppClient.Restore() — leftover mock expectations cause subsequent tests to fail
- Importing app/common in test env construction — always wire from raw package constructors (appadapter.New, appservice.New, etc.)
- Reusing the same namespace across multiple test functions in AppHandlerTestSuite — always call setupNamespace(t) at the start of each test method
- Adding invoice line assertions without stable-sorting lines — use StableSortInvoiceItemParams before setting mock expectations on AddInvoiceLines

## Decisions

- **Two parallel test architectures: TestEnv interface (app_test.go) vs billingtest.BaseSuite embedding (invoice_test.go)** — App lifecycle tests (install/update/uninstall) need a narrow env without full billing stack; invoice tests need the full billing stack from BaseSuite — different scopes justified different setups.
- **MockSecretService wraps real secretservice with Enable/Disable toggle rather than pure mock** — Most tests need real secret persistence (storing Stripe API keys); only error-path tests need mock control, so passthrough-by-default avoids brittle expectation setup in the happy path.
- **Stripe client mock sorts invoice line items before matching** — Billing line generation order is non-deterministic; stable sorting in the mock (StableSortInvoiceItemParams) ensures test assertions are order-independent without requiring production code to sort.

## Example: Install a Stripe app and verify it can be retrieved

```
import (
  "github.com/openmeterio/openmeter/openmeter/app"
  stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
)

func (s *AppHandlerTestSuite) TestGet(ctx context.Context, t *testing.T) {
  s.setupNamespace(t)
  s.Env.StripeClient().On("GetAccount").Return(stripeclient.StripeAccount{StripeAccountID: getStripeAccountId()}, nil)
  s.Env.StripeClient().On("SetupWebhook", mock.Anything).Return(stripeclient.StripeWebhookEndpoint{EndpointID: "we_123", Secret: "whsec_123"}, nil)
  defer s.Env.StripeClient().Restore()
  created, err := s.Env.App().InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
    InstallAppInput: app.InstallAppInput{MarketplaceListingID: app.MarketplaceListingID{Type: app.AppTypeStripe}, Namespace: s.namespace},
    APIKey: TestStripeAPIKey,
  })
  require.NoError(t, err)
// ...
```

<!-- archie:ai-end -->
