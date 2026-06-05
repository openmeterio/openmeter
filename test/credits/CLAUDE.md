# credits

<!-- archie:ai-start -->

> Integration test suite (package `credits`) for the ledger-backed credit + charges system, validating credit purchases, credit-then-invoice settlement, revenue recognition, breakage, and FBO/receivable balances. Its BaseSuite embeds test/billing.BaseSuite and layers the full ledger/charges stack on top.

## Patterns

**Embed billingtest.BaseSuite, extend in SetupSuite** — credits BaseSuite embeds billingtest.BaseSuite; its SetupSuite first calls s.BaseSuite.SetupSuite() then builds ledger/charges services from ledgertestutils.InitDeps(s.DBClient, logger) and chargestestutils.NewServices. (`func (s *BaseSuite) SetupSuite(){ s.BaseSuite.SetupSuite(); deps, err := ledgertestutils.InitDeps(s.DBClient, logger); ... }`)
**Build charges stack via chargestestutils.NewServices** — Charges, CreditPurchaseSvc, UsageBasedSvc come from chargestestutils.NewServices(t, Config{...}) wired with ledger charge handlers (NewFlatFeeHandler, NewCreditPurchaseHandler, NewUsageBasedHandler) and transactions.ResolverDependencies. (`stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{Client: s.DBClient, BillingService: s.BillingService, FlatFeeHandler: flatFeeHandler, CreditPurchaseHandler: creditPurchaseHandler, UsageBasedHandler: ...})`)
**Ledger-backed customers need explicit account provisioning** — Use CreateLedgerBackedCustomer(ns, key): it calls LedgerResolver.EnsureBusinessAccounts then CreateTestCustomer then LedgerResolver.CreateCustomerAccounts. A plain CreateTestCustomer has no ledger accounts. (`cust := s.CreateLedgerBackedCustomer(ns, "test")`)
**ChargeIntent via CreateMockChargeIntent** — Charge intents are assembled by CreateMockChargeIntent(CreateMockChargeIntentInput{...}) which validates inputs, derives invoiceAt from payment term, and builds flatfee.Intent or usagebased.Intent wrapped by charges.NewChargeIntent. Default settlement is CreditThenInvoiceSettlementMode. (`intent := s.CreateMockChargeIntent(CreateMockChargeIntentInput{Customer: cust.GetID(), Currency: USD, ServicePeriod: period, Price: productcatalog.NewPriceFrom(...)})`)
**Balance assertions via Must* helpers with mo.Option cost basis** — Balances are read through MustCustomerFBOBalance / MustCustomerReceivableBalance (+ ...WithPriority/AsOf/ForFeatures/ForTaxCode variants). Cost-basis arg is mo.Option[*alpacadecimal.Decimal]: mo.None()=all, mo.Some(nil)=nil-cost-basis route, mo.Some(&v)=one route. (`bal := s.MustCustomerFBOBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]())`)
**Reset streaming + clock in TearDownTest** — credits BaseSuite.TearDownTest calls s.MockStreamingConnector.Reset(), clock.UnFreeze(), clock.ResetTime(); per-test clock/streaming changes still need their own defers. (`func (s *BaseSuite) TearDownTest(){ s.MockStreamingConnector.Reset(); clock.UnFreeze(); clock.ResetTime() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `base.go` | credits BaseSuite (embeds billingtest.BaseSuite) with Charges, CreditPurchaseSvc, UsageBasedSvc, CustomerBalanceSvc, Ledger, BalanceQuerier, LedgerAccountService, LedgerResolver, BreakageService, RevenueRecognizer; helpers CreateMockChargeIntent, CreateLedgerBackedCustomer, Must*Balance | ledgertestutils.InitDeps returns HistoricalLedger used as both Ledger and BalanceQuerier; resolver vs account-service roles in transactions.ResolverDependencies are distinct (AccountService=ResolversService, AccountCatalog=AccountService). |
| `credit_then_invoice_test.go` | Exercises the credit-then-invoice settlement lifecycle end-to-end | Drive via Charges.Create/AdvanceCharges/ApplyPatches and CreditPurchaseSvc, not low-level ledger adapters. |
| `creditgrant_test.go` | Credit grant / purchase scenarios and resulting FBO balances | Assert balances with the mo.Option cost-basis conventions. |
| `rating_test.go` | Rating-backed charge fixtures (production rating path) | Prefer rating-backed fixtures over hand-built charges where the real path can express the scenario. |
| `sanity_lifecycle_test.go` | Broad lifecycle sanity checks across credit + charge phases | Late-arriving usage modeled via MockStreamingConnector with explicit StoredAt to exercise stored-at cutoff in finalization. |

## Anti-Patterns

- Calling ledger or charge adapters directly instead of Charges.Service / CreditPurchaseSvc / UsageBasedSvc.
- Using CreateTestCustomer for ledger scenarios — it lacks FBO/receivable accounts; use CreateLedgerBackedCustomer.
- Passing a raw decimal where a cost-basis mo.Option is expected (mo.None vs mo.Some(nil) vs mo.Some(&v) mean different balance routes).
- Re-wiring the billing stack instead of reusing the embedded billingtest.BaseSuite.SetupSuite().
- Forgetting per-test streaming reset / clock reset on top of the suite-level TearDownTest.

## Decisions

- **credits BaseSuite embeds the billing BaseSuite** — Credits/charges sit on top of billing invoices; reusing the billing stack avoids duplicating customer/profile/invoice wiring and keeps both domains consistent.
- **Ledger deps come from ledgertestutils.InitDeps + chargestestutils.NewServices** — Keeps test wiring aligned with the production ledger/charges construction while staying independent from app/common DI.

## Example: Set up a ledger-backed customer and create a charge intent

```
func (s *BaseSuite) example(ns string, period timeutil.ClosedPeriod) {
  cust := s.CreateLedgerBackedCustomer(ns, "test")
  intent := s.CreateMockChargeIntent(CreateMockChargeIntentInput{
    Customer: cust.GetID(), Currency: USD, ServicePeriod: period,
    Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
      Amount: alpacadecimal.NewFromInt(100), PaymentTerm: productcatalog.InAdvancePaymentTerm,
    }),
  })
  _ = intent
  bal := s.MustCustomerFBOBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal]())
  _ = bal
}
```

<!-- archie:ai-end -->
