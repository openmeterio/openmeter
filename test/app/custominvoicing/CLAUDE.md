# custominvoicing

<!-- archie:ai-start -->

> Integration tests for the custom invoicing app validating the async sync protocol (draft→issuing→payment) and invoice event JSON marshaling against a real Postgres DB via billingtest.BaseSuite. Primary constraint: all service interactions must go through BillingService and CustomInvoicingService — never adapter or Ent directly.

## Patterns

**Embed billingtest.BaseSuite** — All test suites embed billingtest.BaseSuite (not app/common wiring) to get BillingService, CustomerService, AppService, MockStreamingConnector wired from raw package constructors. (`type CustomInvoicingTestSuite struct { billingtest.BaseSuite }`)
**setupDefaultBillingProfile helper** — Each test scenario calls s.setupDefaultBillingProfile(ctx, namespace, config) to install the custom invoicing app, configure it, and provision the billing profile before any invoice operations. DraftPeriod is set to P0D so lines immediately become invoiceable. (`s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{EnableDraftSyncHook: true, EnableIssuingSyncHook: true})`)
**Drive state machine via service layer only** — Invoice lifecycle is driven exclusively through s.BillingService and s.CustomInvoicingService (SyncDraftInvoice, SyncIssuingInvoice, HandlePaymentTrigger) — never through adapter or Ent directly. (`s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{InvoiceID: invoice.GetInvoiceID(), UpsertInvoiceResults: upsertResults})`)
**Unique namespace per test function** — Every test function uses a unique string literal namespace to avoid cross-test DB contamination without teardown. (`namespace := "ns-custom-invoicing-flow"`)
**MockStreamingConnector with deferred Reset** — Usage data injected via s.MockStreamingConnector.AddSimpleEvent before invoicing; always defer s.MockStreamingConnector.Reset() to prevent stale events bleeding into subsequent tests. (`s.MockStreamingConnector.AddSimpleEvent("test", 100, periodStart.Add(time.Minute)); defer s.MockStreamingConnector.Reset()`)
**Suite.Run for sub-scenarios** — Related invoice lifecycle steps grouped with s.Run('description', func(){}) to share invoice state between steps and produce hierarchical test output. (`s.Run("invoice can be created", func() { invoices, err := s.BillingService.InvoicePendingLines(...) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `invocing_test.go` | Main integration test covering full invoice flow (hooks enabled) and payment-status-only flow (hooks disabled); also provides setupDefaultBillingProfile helper used by event_test.go. | DraftPeriod is set to P0D in setupDefaultBillingProfile so lines immediately become invoiceable — changing this requires adjusting AsOf in InvoicePendingLines or assertions will fail. The two test functions cover two distinct state machine paths: hooks-enabled produces draft.syncing; hooks-disabled produces payment_processing.pending directly. |
| `event_test.go` | Unit-level JSON round-trip test for billing.StandardInvoiceCreatedEvent — validates that app bases and appcustominvoicing.Meta survive marshal/unmarshal intact. | Uses context.Background() rather than t.Context() — acceptable here since no cancellation semantics are tested; new tests should prefer t.Context(). |

## Anti-Patterns

- Calling adapter or Ent methods directly instead of going through BillingService / CustomInvoicingService
- Importing app/common wiring into test setup — use raw package constructors like billingtest.BaseSuite
- Sharing namespace strings across test functions — always use a unique namespace per top-level test
- Forgetting defer s.MockStreamingConnector.Reset() — stale events bleed into subsequent tests
- Setting DraftPeriod to a non-zero duration without adjusting AsOf — lines won't be invoiceable

## Decisions

- **Tests embed billingtest.BaseSuite instead of constructing services manually** — Keeps test setup DRY and independent from app/common to avoid import cycles; BaseSuite provides all shared services from underlying constructors.
- **Both EnableDraftSyncHook=true and EnableDraftSyncHook=false flows are exercised in separate test functions** — Custom invoicing has two distinct state machine paths: full async sync (draft→issuing sync hooks) and payment-status-only (skips sync hooks, goes directly to payment_processing.pending); both paths need explicit coverage.

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
  synced, _ := s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{InvoiceID: invoices[0].GetInvoiceID()})
// ...
```

<!-- archie:ai-end -->
