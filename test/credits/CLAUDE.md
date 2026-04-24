# credits

<!-- archie:ai-start -->

> Integration tests for the credits/charges domain: credit grant lifecycle (invoice-funded, promotional, external), flat-fee/usage-based/credit-purchase charge advancement against a real ledger stack, and billing/charges rating logic. Extends test/billing.BaseSuite with ledger and charges service wiring.

## Patterns

**Extend BaseSuite with ledger + charges stack** — CreditsTestSuite embeds billingtest.BaseSuite and adds Charges, Ledger, LedgerAccountService, and LedgerResolver. SetupSuite calls s.BaseSuite.SetupSuite() then initialises the ledger via ledgertestutils.InitDeps and chargestestutils.NewServices. (`type CreditsTestSuite struct { billingtest.BaseSuite; Charges charges.Service; Ledger ledger.Ledger; ... }`)
**createLedgerBackedCustomer for ledger-dependent tests** — Tests that require a customer with ledger accounts (FBO, receivable, etc.) use s.createLedgerBackedCustomer(ns, subjectKey) rather than BaseSuite.CreateTestCustomer — the helper also registers the account with the resolver. (`cust := s.createLedgerBackedCustomer(ns, "test-subject")`)
**chargestestutils.NewServices for charge stack construction** — The charges stack (usagebased, flatfee, creditpurchase handlers wired to ledgerchargeadapter) is always assembled via chargestestutils.NewServices, not ad-hoc. FlatFeeHandler, CreditPurchaseHandler, and UsageBasedHandler are injected from ledgerchargeadapter. (`stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{Client: s.DBClient, BillingService: s.BillingService, FlatFeeHandler: ledgerchargeadapter.NewFlatFeeHandler(...)})`)
**Drive charge lifecycle via charges.Service, not adapters** — Tests call s.Charges.Create / AdvanceCharges / ApplyPatches rather than calling lower-level adapter methods directly, mirroring production billing-worker behaviour. (`grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{...})`)
**Explicit SetupCustomInvoicing for custom-invoicing tests** — Tests that drive the custom-invoicing app call s.SetupCustomInvoicing(ns) to install the app and return a helper, rather than s.InstallSandboxApp. This matches the production custom-invoicing wiring. (`customInvoicing := s.SetupCustomInvoicing(ns)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `sanity_test.go / sanity_lifecycle_test.go` | Core charges lifecycle tests: AdvanceCharges, ApplyPatches, external payment settlement state transitions for flatfee, usagebased, and creditpurchase charge types. | Relies on clock.SetTime to control advancement windows. Custom-invoicing app used (not sandbox) to verify async sync flows. |
| `creditgrant_test.go` | Tests creditgrant.Service.Create for invoice-funded, promotional, and external settlement grants. Validates that invoice-funded grants produce standard invoices, promotional grants are immediately final, and external grants follow authorized→settled transitions. | Must create a ledger-backed customer (s.createLedgerBackedCustomer) not a plain test customer — otherwise ledger account resolution fails. |
| `rating_test.go` | Unit-level rating tests for the charges billing calculator: verifies period resolution and billing period splitting without DB interaction. | These tests do not embed BaseSuite; they are pure unit tests using RatingTestSuite which only needs clock and productcatalog types. |

## Anti-Patterns

- Calling lower-level charge adapter methods directly instead of going through charges.Service.
- Using BaseSuite.CreateTestCustomer for tests that require ledger accounts — use createLedgerBackedCustomer instead.
- Skipping defer clock.ResetTime() or defer s.MockStreamingConnector.Reset() in tests that set time or usage events.
- Instantiating ledger adapters inline in test methods — always delegate to ledgertestutils.InitDeps.
- Using context.Background() instead of t.Context() / s.T().Context() in new tests.

## Decisions

- **Wire ledger and charges stack on top of existing BaseSuite rather than creating a separate test environment.** — Reuses the billing, customer, feature, and streaming infrastructure already initialised in BaseSuite, avoiding duplication of the ~150-line setup chain.
- **Use chargestestutils.NewServices as the canonical charges stack builder in tests.** — Mirrors the production wiring in app/common/charges.go, ensuring tests exercise the same handler-to-ledger-adapter wiring that production code uses.

<!-- archie:ai-end -->
