# Billing Test Patterns

## Suite Hierarchy

```
billingtest.BaseSuite              (test/billing/suite.go)
    ↳ billingtest.SubscriptionMixin  (test/billing/subscription_suite.go)
        ↳ subscriptionsync.SuiteBase (worker/subscriptionsync/service/suitebase_test.go)
```

Embed the lowest suite that gives you what you need. Most billing service tests only need `BaseSuite`. Tests involving subscription creation need `SubscriptionMixin`. Tests for the sync algorithm need `SuiteBase`.

## BaseSuite (`test/billing/suite.go`)

Sets up a full in-process stack with a real PostgreSQL database:

1. `testutils.InitPostgresDB(t)` — real Postgres test DB (requires `POSTGRES_HOST=127.0.0.1`)
2. Atlas migrations unless `TEST_DISABLE_ATLAS` is set (falls back to Ent auto-create)
3. Full billing service + adapter chain
4. `ForegroundAdvancementStrategy` — state machine runs synchronously in tests (no async events)
5. `MockStreamingConnector` for meter queries
6. `invoicecalc.NewMockableCalculator` for overriding pricing calculations
7. Sandbox app factory + customer/subject sync hooks

**Key exposed fields:**
```go
BillingService  billing.Service
BillingAdapter  billing.Adapter
MockStreamingConnector *streamingtestutils.MockStreamingConnector
MeterAdapter    *metermock.MockRepository
CustomerService customer.Service
FeatureService  feature.FeatureConnector
```

**Namespace isolation:**
```go
ns := s.GetUniqueNamespace("my-test")  // returns "my-test-{ulid}"
```
Always use unique namespaces to isolate test data between test cases.

## SubscriptionMixin (`test/billing/subscription_suite.go`)

Call `mixin.SetupSuite(t, deps)` from your suite's `SetupSuite`. Adds:
- `PlanService`, `SubscriptionService`, `SubscriptionAddonService`, `SubscriptionWorkflowService`
- `EntitlementConnector` (full entitlement/credit/grant stack for metered entitlements)

**Access via:**
```go
s.PlanService
s.SubscriptionService
s.SubscriptionWorkflowService
```

## SuiteBase for Sync Tests (`worker/subscriptionsync/service/suitebase_test.go`)

Embeds both `BaseSuite` and `SubscriptionMixin`. Also provides:
- `SubscriptionSyncService subscriptionsync.Service`
- `SubscriptionSyncAdapter subscriptionsync.Adapter`

**Per-test setup** (`BeforeTest`):
```go
// Creates fresh per-test state:
ns := getUniqueTestNamespace(suiteName, testName)
s.InstallSandboxApp(t, ns)
s.ProvisionBillingProfile(t, ns)
// Creates test meter + feature
// Creates test customer
```

**Per-test teardown** (`AfterTest`):
```go
clock.UnFreeze()
s.MockStreamingConnector.Reset()
// resets feature flags on the service
```

## Provisioning Helpers

### `InstallSandboxApp(t, ns)`
Required before any invoice operations. Installs the sandbox invoicing app in the namespace.

### `ProvisionBillingProfile(t, ns, opts...)`
Creates a billing profile with option functions:

```go
s.ProvisionBillingProfile(t, ns,
    billingtest.WithProgressiveBilling(),
    billingtest.WithCollectionInterval(isodate.MustParse("P1D")),
    billingtest.WithManualApproval(),
    billingtest.WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
        p.WorkflowConfig.Tax.Enabled = true
    }),
)
```

Default: auto-advance, monthly collection, immediate approval.

### Subscription Creation Helpers (on `SuiteBase`)

```go
// Create from explicit phase definitions
sub, err := s.createSubscriptionFromPlanPhases([]subscriptiontestutils.CreatePhasesInput{...})

// Create from a full plan input
sub, err := s.createSubscriptionFromPlan(plan.CreatePlanInput{...})
```

## Gathering Invoice Helpers (on `SuiteBase`)

```go
// Assert exactly 1 gathering invoice exists and return it with lines expanded
gi := s.gatheringInvoice(ctx, ns, customerID)

// Assert no gathering invoice exists
s.expectNoGatheringInvoice(ctx, ns, customerID)

// Verify lines on an invoice
s.expectLines(invoice, subscriptionID, []expectedLine{
    {PhaseKey: "default", ItemKey: "api-calls", ...},
})
```

### `recurringLineMatcher{PhaseKey, ItemKey, Version, PeriodMin, PeriodMax}`
Generates expected `ChildUniqueReferenceID` strings for a range of billing periods:
```go
matcher := recurringLineMatcher{
    PhaseKey:  "default",
    ItemKey:   "api-calls",
    Version:   0,
    PeriodMin: 0,
    PeriodMax: 2,
}
// Generates: {subID}/default/api-calls/v[0]/period[0], /period[1], /period[2]
```

## MockStreamingConnector

```go
// Set meter values returned by queries
s.MockStreamingConnector.AddSimpleEvent(meterSlug, value, at)

// Or set a fixed return value for all queries
s.MockStreamingConnector.SetDefaultMeter(meterSlug, value)

// Reset all values (called in AfterTest)
s.MockStreamingConnector.Reset()
```

## MockableInvoiceCalculator

Override the invoice calculator for a single test:
```go
s.BillingService.GetInvoiceCalculator().(*invoicecalc.MockableCalculator).
    SetupMock(func(inv *billing.StandardInvoice) {
        // modify the invoice directly before totals are calculated
    })
defer s.BillingService.GetInvoiceCalculator().(*invoicecalc.MockableCalculator).Reset()
```

## Clock Control

```go
clock.SetTime(t1)           // advance clock without freezing
clock.FreezeTime(t)         // freeze at specific time (t is *testing.T for cleanup)
clock.UnFreeze()            // manual unfreeze (also called in AfterTest)
clock.ResetTime()           // reset to wall clock time
```

Always call `clock.UnFreeze()` in `AfterTest` or use `clock.FreezeTime(t)` which registers cleanup automatically.

## Progressive Billing Test Helpers

```go
// Enable progressive billing on the billing profile
s.enableProgressiveBilling()

// Set feature flags on the service directly (bypasses profile)
s.enableProrating()
```

## `RemoveMetaForCompare()`

Use before `require.Equal` comparisons to strip DB-only fields:
```go
expected.RemoveMetaForCompare()
actual.RemoveMetaForCompare()
s.Equal(expected, actual)
```

Available on both `StandardInvoice` and `StandardLine`. Strips: `DBState`, `DetailedLines`, IDs, timestamps.

## Running Billing Tests

```bash
# All billing tests (requires postgres)
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/billing/...

# Just sync algorithm tests
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/billing/worker/subscriptionsync/service/...

# Just charges tests
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/billing/charges/...

# Full billing integration tests (test/ package)
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./test/billing/...
```

Skip atlas migrations (faster, uses Ent auto-create):
```bash
TEST_DISABLE_ATLAS=true POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/billing/...
```
