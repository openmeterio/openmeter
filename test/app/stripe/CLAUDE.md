# stripe

<!-- archie:ai-start -->

> Integration tests for the Stripe app (openmeter/app/stripe): app lifecycle (install/get/update/uninstall), customer-data validation, and Stripe invoice sync including credit/usage-based progressive billing. Stripe API access is fully mocked via testify mocks so no live Stripe calls occur.

## Patterns

**Two test harness styles coexist** — App-handler tests use a hand-built TestEnv (NewTestEnv in testenv.go) returning a TestEnv interface; invoice tests embed billingtest.BaseSuite (StripeInvoiceTestSuite) and wire the stripe adapter/service in SetupSuite. Match the style of the file you extend. (`type StripeInvoiceTestSuite struct { billingtest.BaseSuite; AppStripeService appstripe.Service; Fixture *Fixture; StripeAppClient *StripeAppClientMock; Charges charges.Service; LedgerResolver *ledgerresolvers.AccountResolver }`)
**Stripe clients are testify mocks injected via factory** — StripeClientMock and StripeAppClientMock (stripe_mock.go) implement stripeclient interfaces; they are injected through appstripeadapter.Config.StripeClientFactory / StripeAppClientFactory closures that ignore config and return the mock. Each mock method calls input.Validate() before c.Called(...). (`StripeAppClientFactory: func(config stripeclient.StripeAppClientConfig) (stripeclient.StripeAppClient, error) { return stripeAppClient, nil }`)
**Set expectations with On(...).Return(...) then Restore()** — Program Stripe behavior per scenario via s.Env.StripeAppClient().On("GetCustomer", id).Return(...). Mocks expose Restore() which truncates ExpectedCalls; defer it (or TearDownTest calls it) so expectations do not leak between subtests. (`s.Env.StripeAppClient().On("GetCustomer", newStripeCustomerID).Return(stripeclient.StripeCustomer{StripeCustomerID: newStripeCustomerID}, nil)
defer s.Env.StripeAppClient().Restore()`)
**Fixture builds app + customer + customer-data** — Fixture (fixture.go) centralizes setup: setupApp (InstallMarketplaceListingWithAPIKey with app.AppTypeStripe), setupCustomer, setupAppCustomerData (default cus_123), and setupAppWithCustomer chaining all three. Random stripe account ids via getStripeAccountId(). (`testApp, customer, customerData, err := s.Env.Fixture().setupAppWithCustomer(ctx, s.namespace)`)
**Assert via typed error predicates** — Error outcomes are checked with domain predicates not string matching: app.IsAppNotFoundError, app.IsAppCustomerPreConditionError, app.IsAppProviderPreConditionError, models.IsGenericConflictError. (`require.True(t, app.IsAppCustomerPreConditionError(err))`)
**Invoice sync asserts Stripe line items by description/amount/metadata** — expectStripeInvoiceCreate and expectStripeInvoiceAddLines (invoice_credits_test.go) program CreateInvoice/AddInvoiceLines with mock.MatchedBy, keying expected items by Description and asserting Amount (cents), Metadata om_line_type and non-empty om_line_id. (`s.expectStripeInvoiceAddLines("stripe-partial-invoice-id", []expectedStripeInvoiceItem{{Amount: 500, Description: "...usage in period (5 x $1)", Type: "line"}, {Amount: -500, Description: "credits applied for ...", Type: "credit"}})`)
**Ledger-backed charges wired in SetupSuite** — Credit/usage scenarios build the charges stack via chargestestutils.NewServices with ledger handlers (NewCreditPurchaseHandler, NewFlatFeeHandler, NewUsageBasedHandler) from ledgertestutils.InitDeps, and create ledger-backed customers via LedgerResolver.EnsureBusinessAccounts/CreateCustomerAccounts. (`s.Charges = chargeStack.ChargesService`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | NewTestEnv builds an isolated Postgres-backed TestEnv (customer/app/billing/appstripe services) with mock Stripe clients and a MockSecretService; returns a TestEnv interface with accessors and Close(). | Uses testutils.InitPostgresDB + entClient.Schema.Create; closerFunc closes ent and PG drivers. Env is shared across subtests (see 'TODO: do not share env between tests'). |
| `app_test.go` | Top-level TestAppStripe entrypoint that constructs the env once and runs AppHandlerTestSuite subtests (Create/Get/Update/Uninstall/CustomerData/Validate/CheckoutSession/PortalSession/UpdateAPIKey). | Single env reused for all subtests; mock expectations must be Restore()d between them. |
| `appstripe.go` | AppHandlerTestSuite methods: TestCreate, TestGet, TestUpdate, TestUninstall, TestCustomerData, TestCustomerValidate, plus TestStripeAPIKey constant and setupNamespace (ULID per test). | Long mock dance in TestCustomerData: Restore() and re-program On(...) between sub-scenarios; pre-condition errors distinguish customer vs provider failures. |
| `fixture.go` | Fixture struct and setupApp/setupCustomer/setupAppCustomerData/setupAppWithCustomer; defaultStripeCustomerID=cus_123; getStripeAccountId() random acct_ ids. | setupApp programs GetAccount + SetupWebhook mocks and defers Restore() — calling it twice in one scenario re-stacks expectations. |
| `invoice_test.go` | StripeInvoiceTestSuite + SetupSuite wiring (secret/appstripe adapter+service, ledger deps, charges stack, fixture); TestComplexInvoice covering multi-line UBP invoicing (flat/tiered/AI) with tax codes. | SetupSuite uses slog.Default() (acceptable in tests only). MockStreamingConnector seeded with out-of-period 0 events to baseline; defer Reset(). |
| `invoice_credits_test.go` | Progressive-billing credit-then-invoice test driving charges.Service.Create (creditpurchase + usagebased intents) and asserting Stripe line/credit items; defines expectStripeInvoiceCreate/AddLines and intent builders. | Uses clock.FreezeTime/UnFreeze around fixed dates; RemoveCircularReferences() before UpsertStandardInvoice; settles credit purchase via HandleCreditPurchaseExternalPaymentStateTransition (Authorized then Settled). |
| `stripe_mock.go` | StripeClientMock and StripeAppClientMock implementing stripeclient.StripeClient/StripeAppClient; Restore() truncates ExpectedCalls; AddInvoiceLines/UpdateInvoiceLines stable-sort inputs for deterministic matching. | Most methods call input.Validate() before Called(); DeleteInvoice returns args.Error(1) (index 1) — a quirk vs the usual Error(0). |
| `secret_mock.go` | MockSecretService wrapping a real secretservice with EnableMock/DisableMock toggle; delegates to original when mock disabled, otherwise records Called() and validates input. | Implements secret.SecretService (compile-time assert var _); validation runs even in mock mode and returns models.NewGenericValidationError. |

## Anti-Patterns

- Issuing real Stripe API calls — always go through StripeClientMock/StripeAppClientMock factories.
- Leaving mock expectations un-Restore()d between subtests; the env/mocks are shared and will bleed expectations.
- Matching Stripe invoice errors by string instead of typed predicates (app.IsApp*Error, models.IsGenericConflictError).
- Forgetting clock.UnFreeze() / MockStreamingConnector.Reset() defers in invoice tests, leaking frozen time or events.
- Hand-building app/customer/billing services in invoice tests instead of reusing BaseSuite + Fixture wiring.

## Decisions

- **Stripe access is abstracted behind stripeclient interfaces and injected via factory closures so tests swap in mocks.** — Lets integration tests exercise the full adapter/service stack against real Postgres while keeping Stripe deterministic and offline.
- **Invoice/credit tests reuse billingtest.BaseSuite and the ledger/charges test utilities rather than the app/common DI layer.** — Keeps test dependencies built from concrete package constructors, avoiding test-only import cycles per repo testing guidance.
- **Stripe line-item assertions key on Description with metadata (om_line_type, om_line_id) and cent amounts via mock.MatchedBy.** — Line ordering is non-deterministic, so matching by stable description/metadata is more robust than positional assertions.

## Example: Assert a synced Stripe invoice contains the expected usage line and credit line

```
s.expectStripeInvoiceCreate(stripeApp.GetID(), cust.GetID(), partialInvoice.ID, customerData.StripeCustomerID, "stripe-partial-invoice-id")
s.expectStripeInvoiceAddLines("stripe-partial-invoice-id", []expectedStripeInvoiceItem{
	{Amount: 500, Description: "usage-based-progressive-credit-then-invoice: usage in period (5 x $1)", Type: "line"},
	{Amount: -500, Description: "credits applied for usage-based-progressive-credit-then-invoice: usage in period", Type: "credit"},
})

stripePartialInvoice := lo.Must(partialInvoice.RemoveCircularReferences())
upsertResult, err := stripeInvoicingApp.UpsertStandardInvoice(ctx, stripePartialInvoice)
s.NoError(err)
s.StripeAppClient.AssertExpectations(s.T())
```

<!-- archie:ai-end -->
