# billing

<!-- archie:ai-start -->

> Integration test suite for the billing domain — covers invoice lifecycle, adapter persistence, collection flows, discounts, tax handling, schema migrations, and subscription sync. All tests run against a real PostgreSQL database provisioned via pgtestdb/Atlas migrations.

## Patterns

**BaseSuite embedding** — Every test suite embeds BaseSuite (defined in suite.go), which boots the full billing stack from raw constructors (no app/common). All service fields (BillingService, BillingAdapter, CustomerService, FeatureService, MeterAdapter, MockStreamingConnector, etc.) are set in BaseSuite.setupSuite. (`type InvoicingTestSuite struct { BaseSuite }`)
**SubscriptionMixin for subscription-dependent tests** — Tests that need subscription, plan, entitlement, or subscription-sync services embed SubscriptionMixin alongside BaseSuite and call SubscriptionMixin.SetupSuite(t, s.GetSubscriptionMixInDependencies()) in their SetupSuite override. (`type SubscriptionTestSuite struct { BaseSuite; SubscriptionMixin; SubscriptionSyncService subscriptionsync.Service }`)
**Unique namespace per test** — Every test method uses a unique namespace string (either a hard-coded constant prefixed with 'ns-' or s.GetUniqueNamespace(prefix)) to prevent cross-test pollution on the shared database. (`namespace := s.GetUniqueNamespace("ns-schema-migration")`)
**clock.SetTime / clock.ResetTime for deterministic time** — Tests that depend on invoice_at / collection_at / period calculations freeze or advance pkg/clock. TearDownTest always calls clock.UnFreeze() and clock.ResetTime() via BaseSuite.TearDownTest. (`clock.SetTime(periodStart); defer clock.ResetTime()`)
**MockStreamingConnector for usage events** — Usage events are injected via s.MockStreamingConnector.AddSimpleEvent(meterSlug, quantity, ts) or SetSimpleEvents. Always call defer s.MockStreamingConnector.Reset() after injecting events to avoid cross-test bleed. (`s.MockStreamingConnector.AddSimpleEvent(meterSlug, 10, periodStart.Add(time.Minute)); defer s.MockStreamingConnector.Reset()`)
**Atlas migrations by default; ent.Schema.Create only when TEST_DISABLE_ATLAS is set** — setupSuite runs full golang-migrate Atlas migrations unless TEST_DISABLE_ATLAS is set. SchemaMigrationTestSuite forces Atlas with SetupSuiteOptions{ForceAtlas: true} to test schema-level migration logic. (`func (s *SchemaMigrationTestSuite) SetupSuite() { s.BaseSuite.setupSuite(SetupSuiteOptions{ForceAtlas: true}) }`)
**t.Context() preferred over context.Background()** — New tests use ctx := s.T().Context() (or ctx := t.Context() in sub-tests) to tie context lifetime to the test harness. Older tests still use context.Background() — keep consistency within a file. (`ctx := s.T().Context()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `suite.go` | BaseSuite definition: wires billing, customer, feature, meter, app, taxcode, subject, sandbox, and custom-invoicing services from raw constructors without app/common. Also defines test helpers: InstallSandboxApp, CreateTestCustomer, ProvisionBillingProfile, CreateDraftInvoice, DebugDumpInvoice. | Never import app/common here — test-only import cycles. Always register hooks (subject/customer cross-hooks, entitlement validator) in this file, not in individual test suites. |
| `subscription_suite.go` | SubscriptionMixin: builds the plan/subscription/addon/entitlement/credit dependency graph from raw adapters. Used by SubscriptionTestSuite. Exposes SetupEntitlements for the full credit/entitlement connector stack. | Requires SubscriptionMixInDependencies.Validate() to pass — all six fields must be non-nil. Uses eventbus.NewMock(t), not a real Kafka publisher. |
| `schemamigration_test.go` | Tests schema-level migration paths (e.g., schema level 1 → current). Directly manipulates Ent raw DB to simulate legacy rows and then validates that migration reads behave correctly. | Uses s.BillingAdapter.SetInvoiceDefaultSchemaLevel and raw Ent queries (billinginvoiceline, billingstandardinvoicedetailedline) to force schema state — do not use BillingService for setup here. |
| `collection_test.go` | End-to-end tests for progressive billing collection: collection_at calculation, late-event windows, anchored alignment, flat-fee-only skipping collection. | Relies heavily on clock.SetTime at specific points and MockStreamingConnector events at specific StoredAt timestamps. Event order and time advancement must match invoice state machine transitions. |
| `discount_test.go` | Tests discount correlation ID stability across invoice splits, unit discount progressive billing across multiple invoice periods, and discount quantity accounting. | Calls s.MeterAdapter.ReplaceMeters and defers cleanup. Meters must be replaced before features are created, and cleaned up in defer to avoid test pollution. |

## Anti-Patterns

- Importing app/common in test files — causes import cycles; construct services from raw adapters as BaseSuite does.
- Using context.Background() when s.T().Context() is available — breaks test-scoped cancellation.
- Calling s.MockStreamingConnector.AddSimpleEvent without defer Reset() — events bleed into subsequent subtests.
- Hard-coding the same namespace string across multiple test methods in a file — leads to FK constraint violations when tests share the database.
- Accessing BillingAdapter directly in tests when BillingService covers the same operation — bypasses state machine and validation logic.

## Decisions

- **Construct all services from raw package constructors, not app/common Wire providers.** — Avoids import cycles (app/common depends on openmeter packages; test/ packages are peers). Lets each test suite compose only the services it actually needs.
- **Run Atlas migrations by default (not ent.Schema.Create) in BaseSuite.** — Ensures tests exercise the same migration path as production, catching column/constraint drift that ent.Schema.Create would silently fix.
- **Use pkg/clock globally for deterministic time in tests.** — Invoice lifecycle depends on clock.Now() for collection_at, quantity_snapshoted_at, and invoice_at comparisons. Freezing clock in tests eliminates race conditions in time-sensitive assertions.

## Example: Typical suite setup: embed BaseSuite, install sandbox app, provision billing profile, create customer, inject streaming events, advance clock, invoice pending lines.

```
// In a test method:
namespace := "ns-my-feature"
ctx := context.Background()
sandboxApp := s.InstallSandboxApp(s.T(), namespace)
s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), WithProgressiveBilling())
cust := s.CreateTestCustomer(namespace, "test-subject")
s.MockStreamingConnector.AddSimpleEvent(meterSlug, 10, periodStart.Add(time.Minute))
defer s.MockStreamingConnector.Reset()
clock.SetTime(periodEnd.Add(time.Hour))
defer clock.ResetTime()
invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{Customer: cust.GetID()})
s.NoError(err)
```

<!-- archie:ai-end -->
