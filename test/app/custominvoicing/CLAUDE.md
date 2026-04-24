# custominvoicing

<!-- archie:ai-start -->

> Integration tests for the custom invoicing app — validates the async sync protocol (draft→issuing→payment) and invoice event JSON marshaling, exercising the full billing state machine against a real Postgres DB via billingtest.BaseSuite.

## Patterns

**Embed billingtest.BaseSuite** — All test suites embed billingtest.BaseSuite (not app/common wiring) to get BillingService, CustomerService, AppService, MockStreamingConnector, etc. wired from raw package constructors. (`type CustomInvoicingTestSuite struct { billingtest.BaseSuite }`)
**setupDefaultBillingProfile helper** — Each test scenario calls s.setupDefaultBillingProfile(ctx, namespace, config) to install the custom invoicing app, configure it, and provision the billing profile before any invoice operations. (`s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{EnableDraftSyncHook: true, EnableIssuingSyncHook: true})`)
**Drive state machine via service layer, not adapter** — Invoice lifecycle is driven through s.BillingService and s.CustomInvoicingService (SyncDraftInvoice, SyncIssuingInvoice, HandlePaymentTrigger) — never through adapter or Ent directly. (`s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{...})`)
**Unique namespace per test function** — Every test function uses a unique namespace string literal (e.g. 'ns-custom-invoicing-flow') to avoid cross-test DB contamination without needing teardown. (`namespace := "ns-custom-invoicing-flow"`)
**MockStreamingConnector for usage data** — Usage data is injected via s.MockStreamingConnector.AddSimpleEvent before invoicing; always defer s.MockStreamingConnector.Reset() to clean up. (`s.MockStreamingConnector.AddSimpleEvent("test", 100, periodStart.Add(time.Minute)); defer s.MockStreamingConnector.Reset()`)
**Suite.Run for sub-scenarios** — Related invoice lifecycle steps are grouped with s.Run('description', func(){...}) to produce hierarchical test output and share invoice state between steps. (`s.Run("invoice can be created", func() { invoices, err := s.BillingService.InvoicePendingLines(...) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `invocing_test.go` | Main integration test: CustomInvoicingTestSuite covers full invoice flow (hooks enabled) and payment-status-only flow (hooks disabled); also provides setupDefaultBillingProfile helper. | DraftPeriod is set to P0D in setupDefaultBillingProfile so lines immediately become invoiceable — change carefully or invoicing assertions will break. |
| `event_test.go` | Unit-level JSON round-trip test for billing.StandardInvoiceCreatedEvent — validates that app bases and appcustominvoicing.Meta survive marshal/unmarshal. | Uses context.Background() (not t.Context()) — acceptable here because no cancellation semantics are tested; new tests should prefer t.Context(). |

## Anti-Patterns

- Calling adapter or Ent methods directly instead of going through BillingService / CustomInvoicingService
- Importing app/common wiring into test setup — use raw package constructors like billingtest.BaseSuite
- Sharing namespace strings across test functions — always use a unique namespace per top-level test
- Forgetting defer s.MockStreamingConnector.Reset() — stale events bleed into subsequent tests
- Setting DraftPeriod to a non-zero duration without adjusting AsOf — lines won't be invoiceable

## Decisions

- **Tests embed billingtest.BaseSuite instead of constructing services manually** — Keeps test setup DRY and independent from app/common to avoid import cycles; BaseSuite provides all shared services from underlying constructors.
- **Hook configuration (EnableDraftSyncHook/EnableIssuingSyncHook) is exercised with both true and false to validate the two distinct invoice state paths** — Custom invoicing has two modes: full async sync (draft→issuing sync) and payment-status-only (skips sync hooks); both paths need explicit coverage.

## Example: Full invoice lifecycle with sync hooks enabled

```
import (
  appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
  "github.com/openmeterio/openmeter/openmeter/billing"
  billingtest "github.com/openmeterio/openmeter/test/billing"
)

type MyTestSuite struct { billingtest.BaseSuite }

func (s *MyTestSuite) TestFlow() {
  ctx := context.Background()
  namespace := "ns-my-unique-test"
  s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{EnableDraftSyncHook: true})
  invoices, _ := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{...})
  s.Equal(billing.StandardInvoiceStatusDraftSyncing, invoices[0].Status)
  synced, _ := s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{InvoiceID: invoices[0].GetInvoiceID(), ...})
// ...
```

<!-- archie:ai-end -->
