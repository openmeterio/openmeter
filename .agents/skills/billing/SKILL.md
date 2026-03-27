---
name: billing
description: Work with the OpenMeter billing package. Use this skill whenever touching invoice lifecycle, billing profiles, customer overrides, invoice line items, gathering invoices, standard invoices, the invoice state machine, billing validation issues, billing-subscription sync, the billing worker, invoice calculation, rating/pricing engine, or tax config on billing objects. Also use when writing or debugging billing integration tests (BaseSuite, SubscriptionMixin), billing adapter (Ent queries), billing HTTP handlers, or the subscription→billing sync algorithm. Trigger this skill for any file under `openmeter/billing/`, `openmeter/billing/worker/`, `openmeter/billing/service/`, `openmeter/billing/adapter/`, `openmeter/billing/rating/`, `test/billing/`, or `cmd/billing-worker/`.
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Billing

Guidance for working with the OpenMeter billing package (`openmeter/billing/`).

The charges subpackage has its own `/charges` skill — use it when touching `openmeter/billing/charges/`. This skill covers everything else in billing.

## Package Map

```
openmeter/billing/           # Domain types + service/adapter interfaces (no business logic here)
openmeter/billing/service/   # Service implementation + invoice state machine
openmeter/billing/adapter/   # Ent ORM persistence layer
openmeter/billing/httpdriver/ # HTTP handlers
openmeter/billing/rating/    # Pricing calculation engine (tiered, graduated, flat, dynamic)
openmeter/billing/models/totals/ # Shared Totals struct
openmeter/billing/validators/ # Subscription/customer pre-action hook validators
openmeter/billing/worker/    # Watermill event handlers + cron jobs
openmeter/billing/worker/subscriptionsync/ # Subscription→billing sync algorithm
openmeter/billing/worker/advance/  # Batch auto-advance cron
openmeter/billing/worker/collect/  # Gathering invoice collection cron
openmeter/billing/worker/asyncadvance/ # Event-driven advance handler
test/billing/                # Shared test suite base (BaseSuite, SubscriptionMixin)
```

## Core Type Patterns

### Union Types (Invoice, InvoiceLine)

`Invoice`, `InvoiceLine`, and `Charge` all use the same **private discriminated union** pattern:

```go
// Private type tag, private concrete pointer fields
type Invoice struct { t InvoiceType; std *StandardInvoice; gathering *GatheringInvoice }

// Always construct via generic constructor
inv := billing.NewInvoice[billing.StandardInvoice](std)

// Access via typed methods (return value + error)
std, err := inv.AsStandardInvoice()
gi, err := inv.AsGatheringInvoice()
```

Never construct `Invoice{}` directly. The same pattern applies to `InvoiceLine`.

### DBState on Lines

`StandardLine.DBState *StandardLine` stores the version as loaded from the DB. The adapter diffs this against the current state to determine minimal writes. Call `line.SaveDBSnapshot()` to capture the current state as `DBState` before modifying the line in a service method.

### `mo.Option` on Lines Collection

`StandardInvoice.Lines` is `StandardInvoiceLines`, which wraps `mo.Option[StandardLines]`. Absent means not loaded/expanded; present-but-empty means loaded but no lines. Use `.IsPresent()` / `.OrEmpty()` carefully — do not confuse nil with empty.

### ChildUniqueReferenceID (Idempotent Upserts)

`DetailedLine` and `GatheringLine` carry `ChildUniqueReferenceID string` for idempotent upserts. When recalculating pricing, new detailed lines (without IDs) are matched to existing DB rows via this field through `StandardLine.DetailedLinesWithIDReuse()`, avoiding unnecessary delete/re-create cycles.

### InvoiceAt vs Period vs CollectionAt

- `Period`: when the service was actually rendered (usage window)
- `InvoiceAt`: when the line should appear on an invoice (may be delayed)
- `CollectionAt`: when the invoice entered collection (drives due-date calculation)

These are distinct fields and must not be conflated.

## Service / Adapter Pattern

`billing.Service` is a **composite interface** of 10 sub-interfaces defined in `service.go`:
`ProfileService`, `CustomerOverrideService`, `InvoiceLineService`, `SplitLineGroupService`, `InvoiceService`, `StandardInvoiceService`, `GatheringInvoiceService`, `SequenceService`, `InvoiceAppService`, `LockableService`, `ConfigService`.

`billing.Adapter` mirrors this split but is closer to the DB. Implementation is in `adapter/`.

The `billingservice.Service` struct (in `service/service.go`) holds:
- adapter + external services: `customerService`, `appService`, `ratingService`, `featureService`, `meterService`, `streamingConnector`, `publisher`
- `invoiceCalculator` (mockable in tests)
- `standardInvoiceHooks []StandardInvoiceHook` (mutable, registered at startup)

### Advancement Strategy

`ForegroundAdvancementStrategy` runs the state machine synchronously (used in tests and the async-advance worker). `QueuedAdvancementStrategy` stops and queues async advancement (used in HTTP handlers). Controlled via `ConfigService.WithAdvancementStrategy()`.

The billing worker binary uses `ForegroundAdvancementStrategy` in the async-advance handler; the HTTP handlers use `QueuedAdvancementStrategy` (emit an event, return fast).

## Customer-Level Locking

Every invoice-mutating operation must call `transactionForInvoiceManipulation` which:
1. Calls `UpsertCustomerLock` **outside** any transaction (advisory lock record)
2. Wraps the operation in a DB transaction
3. Calls `LockCustomerForUpdate` **inside** the transaction (row-level lock)

This serializes all concurrent invoice operations for the same customer. Never bypass this pattern when writing new service methods that modify invoices.

## Invoice State Machine

Defined in `service/stdinvoicestate.go` using `github.com/qmuntal/stateless`. State machine instances are pooled via `sync.Pool`.

**Key states and flow:**

```
DraftCreated
  → DraftWaitingForCollection    (calculate invoice)
  → DraftCollecting              (guard: isReadyForCollection, or TriggerSnapshotQuantities)
  → DraftUpdating / DraftValidating
  → DraftInvalid                 (critical validation issue; TriggerRetry → DraftValidating)
  → DraftSyncing                 (OnActive: syncDraftInvoice → app.UpsertStandardInvoice)
  → DraftSyncFailed              (TriggerRetry → DraftValidating)
  → DraftManualApprovalNeeded    (if !autoAdvance; TriggerApprove → DraftReadyToIssue)
  → DraftWaitingAutoApproval     (if autoAdvance + shouldAutoAdvance → DraftReadyToIssue)
  → DraftReadyToIssue
  → IssuingSyncing               (OnActive: finalizeInvoice → app.FinalizeStandardInvoice)
  → Issued
  → PaymentProcessingPending / Paid / Overdue / Uncollectible / Voided

DeleteInProgress → DeleteSyncing → Deleted (TriggerFailed → DeleteFailed)
```

**Key guards:**
- `noCriticalValidationErrors`: blocks state transitions when any `ValidationIssue` with `Severity=critical` exists
- `shouldAutoAdvance`: checks `DraftUntil <= now` (auto-approval window has elapsed)
- `canIssuingSyncAdvance`: polls `InvoicingAppAsyncSyncer` if the app implements async sync

## Gathering vs Standard Invoices

**Gathering invoice**: one per customer per currency, never advances through states. Collects pending lines (from subscription sync). When lines become due (`CollectionAt <= now`), the `InvoiceCollector` cron calls `InvoicePendingLines` to move them into a new `StandardInvoice`. The gathering invoice is soft-deleted when it has no remaining lines.

**Standard invoice**: goes through the full state machine. Created from gathering lines by `CreateStandardInvoiceFromGatheringLines`.

**Line splitting (progressive billing)**: when a usage-based line must be billed mid-period, the original line gets `status=split` and two children are created. The parent's `SplitLineGroupID` connects them. `SplitLineHierarchy` carries all siblings for computing `GetPreviouslyBilledAmount()`.

## Validation Issues

Two-tier validation:

1. **Structural validation**: `.Validate()` on every type, returns `error`, used for input sanity.
2. **Domain validation issues**: `ValidationIssue{Severity, Code, Message, Component, Path}` — stored on the invoice for business-rule violations.

`ToValidationIssues(err)` traverses the error tree unwrapping:
- `componentWrapper` → sets `Component`
- `fieldPrefixWrapper` → builds JSON path
- `ValidationIssue` → leaf node
- `errors.Join` trees → recurses

**Critical**: any unwrapped error at the root causes `ToValidationIssues` to return the original error (not converted to an issue). This distinguishes expected business rule violations from unexpected system errors.

Use:
- `ValidationWithComponent(ComponentName, err)` — tags which app produced the error
- `ValidationWithFieldPrefix(prefix, err)` — builds the JSON path (e.g. `"lines/0/price"`)
- `StandardInvoice.MergeValidationIssues(err, component)` — replaces all existing issues for that component (prevents stale accumulation on re-validation)

## Tax Handling

Tax config lives in `productcatalog.TaxConfig` (defined in the product catalog package).

Present on:
- `StandardLineBase.TaxConfig *productcatalog.TaxConfig`
- `GatheringLineBase.TaxConfig *productcatalog.TaxConfig`
- `DetailedLineBase.TaxConfig *productcatalog.TaxConfig`
- `InvoicingConfig.DefaultTaxConfig *productcatalog.TaxConfig` (invoice-level default)

**Tax merging**: `productcatalog.MergeTaxConfigs(override, base)` is used in `Profile.Merge()` and `StandardInvoice.GetLeafLinesWithConsolidatedTaxBehavior()`. The invoice-level default tax config is merged into leaf lines that don't have their own config.

**Workflow tax config** (`WorkflowTaxConfig`):
- `Enabled bool` — enables automatic tax calculation via the Tax app (e.g. Stripe Tax)
- `Enforced bool` — invoice fails if the app cannot compute tax

**Supplier tax code**: `SupplierContact.TaxCode *string` — on the billing profile's supplier contact.

## App Integration

`InvoicingApp` interface (implemented by Stripe, Sandbox, custom apps):
- `ValidateStandardInvoice` — called during `DraftSyncing`
- `UpsertStandardInvoice` — sync to external system
- `FinalizeStandardInvoice` — finalize + initiate payment collection
- `DeleteStandardInvoice` — remove from external system

Optional interfaces:
- `InvoicingAppAsyncSyncer` — `CanDraftSyncAdvance` / `CanIssuingSyncAdvance` (polling-based async sync)
- `InvoicingAppPostAdvanceHook` — `PostAdvanceStandardInvoiceHook` (post-transition callback)

`Profile.Apps *ProfileApps` references three apps by capability type: Tax, Invoicing, Payment.

## StandardInvoiceHook (charges integration)

`billing.StandardInvoiceHook` is a mutable slice on the service, populated at startup via `RegisterStandardInvoiceHooks`. The charges service registers itself this way. Hooks receive `PostCreate` / `PostUpdate` callbacks after invoice DB writes. Do not add billing logic here — use it only to notify other subsystems (like charges) of invoice state changes.

## Subscription → Billing Sync

See `references/subscription-sync.md` for full details. Key concepts:

**Entry point**: `subscriptionsync.Service.SynchronizeSubscriptionAndInvoiceCustomer` — called by the worker on subscription events and after a new invoice is issued (self-loop to fill the next period).

**Algorithm layers**:
1. **Persisted state** — load existing lines from DB for the subscription
2. **Target state** — compute what lines *should* exist (phase iterator + billing cadence)
3. **Reconciler** — diff (new / delete / upsert) + apply patches

**Line identification**: every line has a `ChildUniqueReferenceID`:
```
{subscriptionID}/{phaseKey}/{itemKey}/v[{version}]/period[{periodIndex}]
```

**Billing timing** (`GetInvoiceAt()`):
- Flat-fee in-advance → `BillingPeriod.Start`
- All other → `max(ServicePeriod.End, BillingPeriod.End)`

## Worker / Background Processing

**Events handled** (Watermill, single Kafka topic):
- `subscription.Created/Updated/Continued/Cancelled` → `SynchronizeSubscriptionAndInvoiceCustomer`
- `subscription.SubscriptionSyncEvent` → `HandleSubscriptionSyncEvent` (self-loop after invoice issued)
- `billing.AdvanceStandardInvoiceEvent` → `asyncAdvanceHandler.Handle`
- `billing.StandardInvoiceCreatedEvent` → `HandleInvoiceCreation` (re-sync subscriptions referenced in new invoice)

**Cron jobs**:
- `AutoAdvancer.All` — batch advance `DraftWaitingAutoApproval` and `DraftWaitingForCollection` invoices + stuck invoices
- `InvoiceCollector.All` — batch move gathering lines to standard invoices when `collection_at <= now`

**Advancement strategy in worker**: `asyncadvance.Handler` uses `ForegroundAdvancementStrategy` (prevents infinite event loops). `AutoAdvancer` also uses foreground.

## Rating / Pricing Engine

Located in `openmeter/billing/rating/`. Pricing types:
- `flat` — flat rate (generates a single `DetailedLine`)
- `unit` — per-unit pricing
- `tieredvolume` — tiered volume
- `tieredgraduated` — graduated tiered (cannot be split mid-period; splitting would produce incorrect amounts)
- `dynamic` — dynamic pricing

All pricing types implement `GenerateDetailedLines(StandardLineAccessor) GenerateDetailedLinesResult`, producing `[]DetailedLine` that carry `PerUnitAmount`, `Quantity`, `TaxConfig`, `AmountDiscounts`.

## Totals

`totals.Totals` (in `billing/models/totals/model.go`) is present on both invoices and lines:
```
Amount              — pre-discount, pre-tax gross
ChargesTotal        — additional charges
DiscountsTotal      — sum of all discounts
TaxesInclusiveTotal / TaxesExclusiveTotal / TaxesTotal
CreditsTotal        — prepaid credits applied (pre-tax)
Total = Amount + ChargesTotal + TaxesExclusive - DiscountsTotal - CreditsTotal
```

## Testing

See `references/testing.md` for full test patterns. Key points:

**`BaseSuite`** (`test/billing/suite.go`):
- Real Postgres DB + Ent client + Atlas migrations
- `ForegroundAdvancementStrategy` (synchronous state machine)
- `MockStreamingConnector` for meter queries
- `invoicecalc.MockableInvoiceCalculator` for overriding invoice calculations
- `GetUniqueNamespace(prefix)` — ULID-based namespace isolation per test

**`SubscriptionMixin`** (`test/billing/subscription_suite.go`):
- Adds plan/subscription/addon/entitlement stack on top of `BaseSuite`
- Embed this when tests need subscription wiring

**`SuiteBase`** for subscription sync tests (`worker/subscriptionsync/service/suitebase_test.go`):
- Embeds both `BaseSuite` + `SubscriptionMixin`
- Adds `subscriptionsync.Service`
- `BeforeTest`: creates unique namespace, installs sandbox app, provisions billing profile, creates meter+feature+customer

**Provisioning helpers**:
- `ProvisionBillingProfile(opts...)` — takes option functions: `WithProgressiveBilling()`, `WithCollectionInterval(period)`, `WithManualApproval()`, `WithBillingProfileEditFn(fn)`
- `InstallSandboxApp` — required before any invoice operations

## Key Files Reference

| File | What it defines |
|---|---|
| `billing/service.go` | All `Service` sub-interfaces |
| `billing/adapter.go` | All `Adapter` sub-interfaces |
| `billing/stdinvoice.go` | `StandardInvoice`, `StandardInvoiceStatus`, `StandardInvoiceLines` |
| `billing/stdinvoicestate.go` | Trigger constants (`TriggerNext`, `TriggerApprove`, etc.), `StandardInvoiceOperation` |
| `billing/stdinvoiceline.go` | `StandardLine`, `StandardLineBase`, `UsageBasedLine`, `NewFlatFeeLine()` |
| `billing/invoiceline.go` | `GenericInvoiceLine` interface, `Period`, `InvoiceLineManagedBy` |
| `billing/gatheringinvoice.go` | `GatheringInvoice`, `GatheringLine`, `GatheringLineBase` |
| `billing/invoicelinesplitgroup.go` | `SplitLineGroup`, `SplitLineHierarchy`, split-line math |
| `billing/profile.go` | `BaseProfile`, `Profile`, `WorkflowConfig`, `InvoicingConfig`, `CollectionConfig` |
| `billing/validationissue.go` | `ValidationIssue`, `ToValidationIssues()`, `ValidationWithComponent()`, `ValidationWithFieldPrefix()` |
| `billing/errors.go` | All domain error sentinels (`ErrInvoiceNotFound`, etc.) |
| `billing/app.go` | `InvoicingApp` interface, `UpsertResults`, `FinalizeStandardInvoiceResult` |
| `billing/discount.go` | `Discounts`, `PercentageDiscount`, `UsageDiscount`, `MaximumSpendDiscount` |
| `billing/annotations.go` | `AnnotationSubscriptionSyncIgnore`, `AnnotationSubscriptionSyncForceContinuousLines` |
| `billing/serviceconfig.go` | `AdvancementStrategy` type and constants |
| `billing/service/stdinvoicestate.go` | `InvoiceStateMachine` struct, full state machine wiring |
| `billing/service/invoicecalc/calculator.go` | `Calculator` interface, `MockableInvoiceCalculator` |
| `billing/models/totals/model.go` | `totals.Totals` struct |
| `billing/adapter/stdinvoicelines.go` | `GetLinesForSubscription` DB query (line 755) |
| `billing/worker/worker.go` | Watermill event handler wiring |
| `billing/worker/subscriptionsync/service/sync.go` | Main sync algorithm |
| `billing/worker/subscriptionsync/service/targetstate/phaseiterator.go` | Billing cadence loop + `GetInvoiceAt()` |
| `billing/worker/subscriptionsync/service/reconciler/reconciler.go` | Diff algorithm |
| `test/billing/suite.go` | `BaseSuite` definition |
| `test/billing/subscription_suite.go` | `SubscriptionMixin` definition |

## Non-Obvious Gotchas

- **`toValidationIssues` swallows nothing**: an error that isn't wrapped in a `ValidationIssue` or `componentWrapper` at the leaf level will cause the whole call to return the original error (not a `[]ValidationIssue`). Always wrap business rule violations before passing to `MergeValidationIssues`.

- **Graduated tiered pricing cannot be split**: splitting a graduated line mid-period produces incorrect totals because earlier tiers become "already consumed." The rating engine returns an error for this case. Use continuous (non-split) lines for graduated pricing.

- **`mo.Option` absent ≠ empty**: an absent `StandardInvoice.Lines` means "not requested/loaded" and must not be treated as "no lines." Always check `.IsPresent()` before calling `.OrEmpty()`.

- **Namespace lockdown**: `WithLockedNamespaces([]string)` on `ConfigService` blocks invoice advancement for those namespaces (used during migrations). Returns `ErrNamespaceLocked`. Don't bypass this in tests.

- **State machine pooling**: `InvoiceStateMachine` instances use `sync.Pool`. The pool resets all fields after use. Do not hold references to state machine instances across operations.

- **Schema levels**: `StandardInvoice.SchemaLevel` enables gradual schema migration. New code should always set and respect the schema level when reading/writing invoice data.

- **Worker uses `BackgroundAdvancementStrategy`**: the billing-worker binary uses async advancement (events), but the `asyncadvance.Handler` within it uses `ForegroundAdvancementStrategy` to prevent event loops. Tests always use `ForegroundAdvancementStrategy`.

- **`RemoveMetaForCompare()`**: both `StandardInvoice` and `StandardLine` have this method that strips DB-only fields for test assertions. Use it before `require.Equal` comparisons.

- **Hook registration is mutable**: `RegisterStandardInvoiceHooks` appends to a slice on the billing service. The charges service self-registers at `New()`. In tests, the hook is registered once per suite (not per test) — reset handler function fields in `TearDownTest()` rather than re-registering.

## References

- `references/subscription-sync.md` — detailed subscription→billing sync algorithm, phase iterator, reconciler
- `references/testing.md` — full test setup patterns, suite helpers, clock control
