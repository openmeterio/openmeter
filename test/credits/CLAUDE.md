# credits

<!-- archie:ai-start -->

> Integration tests for the credits/charges domain: credit grant lifecycle (invoice-funded, promotional, external), flat-fee/usage-based/credit-purchase charge advancement against a real ledger stack, and billing/charges rating logic. Extends test/billing.BaseSuite with ledger and charges service wiring.

## Patterns

**Extend BaseSuite with ledger + charges stack** — CreditsTestSuite embeds billingtest.BaseSuite and adds Charges, Ledger, LedgerAccountService, LedgerResolver, and RevenueRecognizer. SetupSuite calls s.BaseSuite.SetupSuite() first, then initialises the ledger via ledgertestutils.InitDeps and assembles the charges stack via chargestestutils.NewServices. (`type BaseSuite struct {
	billingtest.BaseSuite
	Charges charges.Service
	Ledger ledger.Ledger
	LedgerAccountService ledgeraccount.Service
	LedgerResolver *ledgerresolvers.AccountResolver
}`)
**CreateLedgerBackedCustomer for ledger-dependent tests** — Tests that require a customer with ledger accounts (FBO, receivable, etc.) call s.CreateLedgerBackedCustomer(ns, subjectKey) rather than BaseSuite.CreateTestCustomer. The helper calls LedgerResolver.EnsureBusinessAccounts then CreateCustomerAccounts. (`cust := s.CreateLedgerBackedCustomer(ns, "test-subject")`)
**chargestestutils.NewServices for charge stack construction** — The charges stack (usagebased, flatfee, creditpurchase handlers wired to ledgerchargeadapter) is always assembled via chargestestutils.NewServices, not ad-hoc inline. FlatFeeHandler, CreditPurchaseHandler, and UsageBasedHandler are injected from ledgerchargeadapter constructors. (`stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
	Client: s.DBClient, BillingService: s.BillingService,
	FlatFeeHandler: ledgerchargeadapter.NewFlatFeeHandler(...),
})`)
**Drive charge lifecycle via charges.Service, not adapters** — Tests call s.Charges.Create / AdvanceCharges / ApplyPatches rather than calling lower-level adapter methods directly, mirroring production billing-worker behaviour. CreateMockChargeIntent is the shared helper for building ChargeIntent values without struct literals. (`intent := s.CreateMockChargeIntent(CreateMockChargeIntentInput{Customer: cust.GetID(), Price: flatPrice, ...})
grants, err := s.Charges.Create(ctx, charges.CreateInput{Namespace: ns, Intents: charges.ChargeIntents{intent}})`)
**TearDownTest resets clock and streaming connector** — BaseSuite.TearDownTest calls s.MockStreamingConnector.Reset(), clock.UnFreeze(), and clock.ResetTime() automatically after each test. Individual test methods still need defer clock.ResetTime() if they call clock.SetTime after starting sub-tests. (`func (s *BaseSuite) TearDownTest() {
	s.MockStreamingConnector.Reset()
	clock.UnFreeze()
	clock.ResetTime()
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `base.go` | BaseSuite for the credits package: embeds billingtest.BaseSuite, initialises ledger deps via ledgertestutils.InitDeps, builds revenue recognizer, collector service, and the full charges stack via chargestestutils.NewServices. Exposes CreateLedgerBackedCustomer and CreateMockChargeIntent helpers. | SetupSuite order is critical: billingtest.BaseSuite.SetupSuite() must run before ledgertestutils.InitDeps to ensure DBClient is available. |
| `sanity_test.go / sanity_lifecycle_test.go` | Core charges lifecycle tests: AdvanceCharges, ApplyPatches, external payment settlement state transitions for flatfee, usagebased, and creditpurchase charge types. | Custom-invoicing app used (not sandbox) to verify async sync flows. Relies on clock.SetTime to control advancement windows. |
| `creditgrant_test.go` | Tests creditgrant.Service.Create for invoice-funded, promotional, and external settlement grants. Validates that invoice-funded grants produce standard invoices, promotional grants are immediately final, and external grants follow authorized→settled transitions. | Must use s.CreateLedgerBackedCustomer — plain CreateTestCustomer customers lack ledger accounts and cause resolver errors. |
| `rating_test.go` | Pure unit tests for charges billing calculator: verifies period resolution and billing period splitting without DB interaction. RatingTestSuite does not embed BaseSuite. | These tests are fast and DB-independent — do not pull in BaseSuite setup overhead. |

## Anti-Patterns

- Calling lower-level charge adapter methods directly instead of going through charges.Service.Create / AdvanceCharges / ApplyPatches.
- Using BaseSuite.CreateTestCustomer for tests that require ledger accounts — use s.CreateLedgerBackedCustomer instead.
- Constructing charges.ChargeIntent{} struct literals instead of charges.NewChargeIntent(flatfee.Intent{...}) — private discriminator stays zero-valued.
- Instantiating ledger adapters inline in test methods — always delegate to ledgertestutils.InitDeps.
- Using context.Background() instead of s.T().Context() in new tests.

## Decisions

- **Wire ledger and charges stack on top of existing billingtest.BaseSuite rather than creating a separate test environment.** — Reuses the billing, customer, feature, and streaming infrastructure already initialised in BaseSuite (~150-line setup chain), avoiding duplication and keeping charge tests co-located with billing tests.
- **Use chargestestutils.NewServices as the canonical charges stack builder in tests.** — Mirrors the production wiring in app/common/charges.go, ensuring tests exercise the same handler-to-ledger-adapter wiring that production code uses, including ledgerchargeadapter.NewFlatFeeHandler and related constructors.

## Example: Create a ledger-backed customer, build a flatfee charge intent, advance charges, and assert FBO balance.

```
cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
intent := s.CreateMockChargeIntent(CreateMockChargeIntentInput{
	Customer: cust.GetID(), Currency: USD,
	Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromFloat(100)}),
	ServicePeriod: period,
})
grants, err := s.Charges.Create(ctx, charges.CreateInput{Namespace: ns, Intents: charges.ChargeIntents{intent}})
s.NoError(err)
clock.SetTime(period.To.Add(time.Hour))
_, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Namespace: ns})
s.NoError(err)
balance := s.MustCustomerFBOBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]())
s.Equal(alpacadecimal.NewFromFloat(100), balance)
```

<!-- archie:ai-end -->
