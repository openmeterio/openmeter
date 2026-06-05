# billing

<!-- archie:ai-start -->

> Integration test suite for the billing domain (package `billing`), exercising invoice lifecycle, profiles, customer overrides, tax config, line splitting, and subscription-to-billing sync against a real Postgres database. The shared `BaseSuite` constructs the full billing stack from concrete constructors (not app/common wiring) so tests run end-to-end through `BillingService`/`BillingAdapter`.

## Patterns

**Embed BaseSuite, run with testify/suite** — Every test file declares `type XxxTestSuite struct { BaseSuite }` and a `func TestXxx(t *testing.T) { suite.Run(t, new(XxxTestSuite)) }`. BaseSuite.SetupSuite wires the entire stack; do not re-wire services per test. (`type BillingAdapterTestSuite struct { BaseSuite }; func TestBillingAdapter(t *testing.T){ suite.Run(t, new(BillingAdapterTestSuite)) }`)
**Stack built from concrete constructors** — setupSuite calls billingservice.New, billingadapter.New, customerservice.New, appservice.New, taxcodeservice.New, meteradapter.New (mockadapter), streamingtestutils.NewMockStreamingConnector directly — never app/common DI. Add new deps the same way. (`billingService, err := billingservice.New(billingservice.Config{Adapter: billingAdapter, RatingService: billingratingservice.New(), CustomerService: s.CustomerService, AppService: s.AppService, TaxCodeService: taxCodeService, ...})`)
**Per-test namespace + provisioning sequence** — Each test creates an isolated namespace, then InstallSandboxApp(ns), CreateTestCustomer(ns, key), and ProvisionBillingProfile(ctx, ns, app.GetID(), opts...). Use s.GetUniqueNamespace(prefix) for ULID-suffixed uniqueness across parallel suites. (`sandboxApp := s.InstallSandboxApp(s.T(), ns); cust := s.CreateTestCustomer(ns, "test"); s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithProgressiveBilling())`)
**Drive behavior via BillingService, not the adapter** — Lifecycle is exercised through CreatePendingInvoiceLines, InvoicePendingLines, UpdateProfile, UpsertCustomerOverride, UpdateStandardInvoice (with EditFn). BillingAdapter is used directly only in adapter_test.go. (`res, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{Customer: cust.GetID(), Currency: currencyx.Code(currency.USD), Lines: []billing.GatheringLine{...}})`)
**Deterministic clock with paired reset** — Tests that depend on time use clock.SetTime(...) with `defer clock.ResetTime()` (or clock.FreezeTime + clock.UnFreeze). BaseSuite.TearDownTest also calls clock.UnFreeze/ResetTime as a safety net. (`clock.SetTime(lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z"))); defer clock.ResetTime()`)
**Mock streaming for usage** — Usage is injected via s.MockStreamingConnector.AddSimpleEvent(meterSlug, value, at) with explicit timestamps, and reset with defer s.MockStreamingConnector.Reset(). Meters are registered through s.MeterAdapter.ReplaceMeters. (`s.MockStreamingConnector.AddSimpleEvent(meterSlug, 100, now.Add(time.Minute)); defer s.MockStreamingConnector.Reset()`)
**Decimal assertions via InexactFloat64** — alpacadecimal.Decimal comparisons use require.Equal(t, float64(N), actual.InexactFloat64()); totals are checked with the requireTotals(t, expectedTotals{...}, line.Totals) helper. (`s.Equal(float64(100), detailedLine.PerUnitAmount.InexactFloat64())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `suite.go` | BaseSuite: TestDB/DBClient, BillingService, BillingAdapter, InvoiceCalculator (MockableInvoiceCalculator), CustomerService, AppService, SandboxApp, TaxCodeService; helpers InstallSandboxApp, CreateTestCustomer, GetUniqueNamespace, DebugDumpInvoice, CreateGatheringInvoice | setupSuite honors TEST_DISABLE_ATLAS (schema.Create vs atlas migrate). Hooks (subject/customer/entitlement) are registered here; missing a hook breaks cross-domain side effects. |
| `subscription_suite.go` | SubscriptionMixin + SubscriptionMixInDependencies: builds plan/subscription/addon/workflow/entitlement services via SetupSuite(t, deps). Get deps from BaseSuite.GetSubscriptionMixInDependencies() | deps.Validate() requires DBClient, FeatureRepo, FeatureService, CustomerService, MeterAdapter, MockStreamingConnector all non-nil. MultiSubscriptionEnabledFF is forced true here via ffx.NewStaticService. |
| `profile.go` | minimalCreateProfileInputTemplate(appID) — canonical CreateProfileInput with PT0S collection interval, AutoAdvance, AlignmentKindSubscription | Collection interval is PT0S (immediate) by design; collection tests must override it. |
| `adapter_test.go` | Direct BillingAdapter tests (CreateInvoice, detailed-line ID reuse via DetailedLinesWithIDReuse, ChildUniqueReferenceID) | Only file allowed to call BillingAdapter directly; constructs StandardLine/DetailedLine fixtures by hand. |
| `tax_test.go` | Tax config snapshotting through profile/override/invoice, line-splitting tax retention | Verifies normalized tax_code_id / tax_behavior DB columns via s.DBClient.BillingInvoiceLine.Query(), not just JSONB. |
| `taxcode_dual_write_test.go` | TaxCode FK dual-write/dual-read regression matrix (Group A profiles, Group B overrides) using assertTaxConfigHasStripeCode / assertInvoiceLineTaxCode | Stale-FK regressions: clearing Stripe must clear TaxCodeID; bare TaxCodeID without Stripe.Code is an intentional migration-path input that backfills Stripe.Code. |
| `ubpflatfee_test.go` | Usage-based flat-fee line creation, percentage discounts, validations | Uses billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{...}); ExpectJSONEqual on WithoutDBState() to compare lines ignoring DB-assigned fields. |
| `schemamigration_test.go` | Migration-oriented tests manipulating detailed child lines (markAllDetailedChildrenDeleted) | May force atlas via SetupSuiteOptions{ForceAtlas:true}; exercises real migration SQL. |

## Anti-Patterns

- Wiring billing services through app/common DI instead of the concrete *.New constructors used in suite.go (can create test-only import cycles).
- Calling BillingAdapter directly outside adapter_test.go instead of going through BillingService.
- Using clock.SetTime/FreezeTime without a paired defer clock.ResetTime()/UnFreeze() — leaks frozen time into later subtests.
- Reusing a fixed namespace string across parallel suites instead of GetUniqueNamespace; namespaces collide and tests flake.
- Asserting decimals with expected.Equal(actual) booleans instead of InexactFloat64() / requireTotals.

## Decisions

- **BaseSuite assembles the whole billing stack from package constructors** — Tests must run the real service/adapter/rating paths end-to-end against Postgres while staying independent of the application wiring layer to avoid import cycles.
- **Subscription dependencies are a separate SubscriptionMixin** — Not every billing test needs the heavyweight subscription/plan/entitlement stack; mixin keeps base setup lean and shares deps explicitly.
- **Streaming and meters are mocked, Postgres is real** — Usage and meter lookups are deterministic via MockStreamingConnector while invoice/line persistence is validated against the actual schema and migrations.

## Example: Provision a customer+profile and assert a usage-based flat-fee invoice line

```
func (s *UBPFlatFeeLineTestSuite) TestPendingLineCreation() {
  ns := "ns-ubpff"; ctx := context.Background()
  clock.SetTime(lo.Must(time.Parse(time.RFC3339, "2025-01-01T00:00:00Z"))); defer clock.ResetTime()
  sandboxApp := s.InstallSandboxApp(s.T(), ns)
  cust := s.CreateTestCustomer(ns, "test")
  s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), WithProgressiveBilling())
  res, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
    Customer: cust.GetID(), Currency: "USD",
    Lines: []billing.GatheringLine{billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
      Period: timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().Add(24*time.Hour)},
      InvoiceAt: clock.Now().Add(24*time.Hour), PerUnitAmount: alpacadecimal.NewFromInt(100),
      PaymentTerm: productcatalog.InArrearsPaymentTerm,
    })},
  })
  s.NoError(err); s.NotNil(res)
// ...
```

<!-- archie:ai-end -->
