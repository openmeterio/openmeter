# billing

<!-- archie:ai-start -->

> Integration test suite for the billing domain — exercises invoice lifecycle, adapter persistence, collection flows, discounts, tax handling, schema migrations, and subscription sync against a real PostgreSQL database provisioned via pgtestdb and Atlas migrations.

## Patterns

**BaseSuite embedding** — Every test suite embeds BaseSuite (suite.go) which boots the full billing stack from raw package constructors — never from app/common. All shared service fields (BillingService, BillingAdapter, CustomerService, FeatureService, MeterAdapter, MockStreamingConnector, etc.) are initialised in BaseSuite.setupSuite. (`type InvoicingTestSuite struct { BaseSuite }
func TestInvoicing(t *testing.T) { suite.Run(t, new(InvoicingTestSuite)) }`)
**SubscriptionMixin for subscription-dependent tests** — Test suites that require plan/subscription/addon/entitlement/credit services embed SubscriptionMixin alongside BaseSuite and call SubscriptionMixin.SetupSuite(t, s.GetSubscriptionMixInDependencies()) in their own SetupSuite override. (`type SubscriptionTestSuite struct { BaseSuite; SubscriptionMixin; SubscriptionSyncService subscriptionsync.Service }`)
**Unique namespace per test** — Every test method uses a unique namespace string — either a hard-coded constant prefixed with 'ns-' unique per method, or s.GetUniqueNamespace(prefix) — to prevent cross-test FK violations on the shared database. (`namespace := s.GetUniqueNamespace("ns-schema-migration")`)
**clock.SetTime / clock.ResetTime for deterministic time** — Tests that depend on invoice_at / collection_at / period calculations freeze or advance pkg/clock globally. BaseSuite.TearDownTest always calls clock.UnFreeze() and clock.ResetTime(); individual tests must still defer clock.ResetTime() if they call clock.SetTime inline. (`clock.SetTime(periodStart); defer clock.ResetTime()`)
**MockStreamingConnector for usage events** — Usage events are injected via s.MockStreamingConnector.AddSimpleEvent(meterSlug, quantity, ts) or SetSimpleEvents. Always defer s.MockStreamingConnector.Reset() after injecting events to prevent bleed into subsequent sub-tests. (`s.MockStreamingConnector.AddSimpleEvent(meterSlug, 10, periodStart.Add(time.Minute))
defer s.MockStreamingConnector.Reset()`)
**Atlas migrations by default; TEST_DISABLE_ATLAS skips them** — setupSuite runs full golang-migrate Atlas migrations unless TEST_DISABLE_ATLAS env var is set. SchemaMigrationTestSuite forces Atlas by passing SetupSuiteOptions{ForceAtlas: true} explicitly to test schema-level migration correctness. (`func (s *SchemaMigrationTestSuite) SetupSuite() { s.BaseSuite.setupSuite(SetupSuiteOptions{ForceAtlas: true}) }`)
**t.Context() preferred in new test code** — New tests use ctx := s.T().Context() to tie context lifetime to the test harness. Legacy tests still use context.Background() — keep consistency within each file. (`ctx := s.T().Context()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `suite.go` | BaseSuite definition: wires billing, customer, feature, meter, app, taxcode, subject, sandbox, and custom-invoicing services from raw constructors. Provides test helpers: InstallSandboxApp, CreateTestCustomer, ProvisionBillingProfile, CreateDraftInvoice, DebugDumpInvoice. | Never import app/common — test-only import cycles. All cross-domain hooks (subject/customer, entitlement validator) must be registered here, not in individual test suites. |
| `subscription_suite.go` | SubscriptionMixin: builds plan/subscription/addon/entitlement/credit dependency graph from raw adapters. Exposes SetupEntitlements for the full credit/entitlement connector stack. | Requires SubscriptionMixInDependencies.Validate() to pass — all six fields must be non-nil. Uses eventbus.NewMock(t), not a real Kafka publisher. |
| `schemamigration_test.go` | Tests schema-level migration paths (e.g., schema level 1 → current). Uses s.BillingAdapter.SetInvoiceDefaultSchemaLevel and raw Ent queries to force legacy schema state, then validates migration read behaviour. | Do not use BillingService for setup here — the test must place rows that BillingService would normally reject. |
| `collection_test.go` | End-to-end tests for progressive billing collection: collection_at calculation, late-event windows, anchored alignment, flat-fee-only skipping collection. | Relies heavily on clock.SetTime at specific points and MockStreamingConnector events at specific StoredAt timestamps — event order and time advancement must match invoice state machine transitions. |
| `discount_test.go` | Tests discount correlation ID stability across invoice splits and unit-discount progressive billing across multiple invoice periods. | Calls s.MeterAdapter.ReplaceMeters and defers cleanup — meters must be replaced before features are created and cleaned up in defer to avoid test pollution. |
| `profile.go` | Shared profile fixture builder: minimalCreateProfileInputTemplate(appID) returns a sane default CreateProfileInput with AutoAdvance=true, DraftPeriod=P1D, CollectionInterval=PT0S. | CollectionInterval is set to PT0S (immediate) by default so general tests do not block on collection windows — tests that validate collection timing must override this. |

## Anti-Patterns

- Importing app/common in test files — causes import cycles; construct services from raw adapters as BaseSuite does.
- Using context.Background() when s.T().Context() is available — breaks test-scoped cancellation.
- Calling s.MockStreamingConnector.AddSimpleEvent without defer Reset() — events bleed into subsequent sub-tests.
- Hard-coding the same namespace constant across multiple test methods in a file — causes FK constraint violations on the shared database.
- Accessing BillingAdapter directly in tests when BillingService covers the same operation — bypasses state machine and validation logic.

## Decisions

- **Construct all services from raw package constructors, not app/common Wire providers.** — Avoids import cycles (app/common depends on openmeter packages; test/ packages are peers) and lets each test suite compose only the services it actually needs.
- **Run Atlas migrations by default (not ent.Schema.Create) in BaseSuite.** — Ensures tests exercise the same migration path as production, catching column/constraint drift that ent.Schema.Create would silently fix.
- **Use pkg/clock globally for deterministic time in tests.** — Invoice lifecycle depends on clock.Now() for collection_at, quantity_snapshoted_at, and invoice_at comparisons; freezing clock eliminates race conditions in time-sensitive assertions.

## Example: Typical test method: install sandbox app, provision billing profile, create customer, inject streaming events, advance clock, invoice pending lines.

```
namespace := "ns-my-feature"
ctx := s.T().Context()
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
