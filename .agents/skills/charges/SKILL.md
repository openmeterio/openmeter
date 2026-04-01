---
name: charges
description: Work with OpenMeter billing charges, including the root charges facade, charge meta queries, charge creation and advancement, usage-based lifecycle state machines, realization runs, and charges test setup. Use when modifying `openmeter/billing/charges/...` or charge-related tests.
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Charges

Guidance for working with OpenMeter billing charges.

This skill describes the charges package generically. Lifecycle state machines exist for both usage-based and flat-fee credit-only branches. The credit-purchase branch follows a different pattern.

## Scope

Primary packages:

- `openmeter/billing/charges/`
- `openmeter/billing/charges/service/`
- `openmeter/billing/charges/meta/`
- `openmeter/billing/charges/lock/`
- `openmeter/billing/charges/usagebased/`
- `openmeter/billing/charges/usagebased/service/`
- `openmeter/billing/charges/usagebased/adapter/`
- `openmeter/billing/charges/flatfee/`
- `openmeter/billing/charges/flatfee/service/`
- `openmeter/billing/charges/flatfee/adapter/`
- `openmeter/billing/charges/service/invoicable_test.go`
- `openmeter/billing/charges/service/advance_test.go`

## Current Design

`openmeter/billing/charges` is the root facade for charge operations.

Important layers:

- `charges.Service`
  - public entrypoint for `Create(...)`, `GetByID(...)`, `GetByIDs(...)`, `AdvanceCharges(...)`
- `charges/meta`
  - shared charge metadata, charge type, short status, and common query surface
- `charges/service`
  - orchestration across charge types
- `charges/usagebased`
  - the deepest current lifecycle implementation

The generic rule is:

- the root charges package owns cross-type orchestration
- type-specific packages own type-specific lifecycle and persistence
- `AdvanceCharges(...)` is a facade method, not the state machine itself

Important types:

- `charges.AdvanceChargesInput` identifies the customer whose charges should advance
- `meta.Charge` and `meta.ChargeID` define the shared charge identity and type
- `charges.Charge` wraps concrete charge variants
- `flatfee.Intent` carries `AmountBeforeProration`, `ProRating`, `SettlementMode`, `PaymentTerm`, `InvoiceAt` — immutable inputs provided by the caller
- `flatfee.State` carries `AmountAfterProration`, `AdvanceAfter`, `CreditRealizations`, `AccruedUsage`, `Payment` — computed/mutable state persisted in DB
- `flatfee.Intent.CalculateAmountAfterProration()` computes the prorated amount from `AmountBeforeProration`, `ServicePeriod/FullServicePeriod` ratio, and `ProRating` config, with currency-precision rounding
- Charge-backed targets do not use invoice-style semantic proration or empty-period filtering; the charge stack materializes and prorates state itself, and the flat fee charge is responsible for omitting empty lines
- `usagebased.Intent` carries `FeatureKey`, `Price`, `SettlementMode`, `InvoiceAt`, and `ServicePeriod`
- `usagebased.ChargeBase` stores the current `Status` and `State`
- `usagebased.State` currently tracks:
  - `CurrentRealizationRunID`
  - `AdvanceAfter`
- `usagebased.RealizationRunBase` stores:
  - `Type`
  - `AsOf`
  - `CollectionEnd`
  - `MeterValue`
  - `Totals`

## Supported Behavior

- `charges.AdvanceCharges(...)` advances both usage-based and flat-fee credit-only charges
- `usagebased.Service.AdvanceCharge(...)` only supports `CreditOnly`
- `flatfee.Service.AdvanceCharge(...)` only supports `CreditOnly`
- `CreditThenInvoice` usage-based advance is deliberately rejected with a not-implemented error
- Both `AdvanceCharge(...)` methods return `*Charge` (nil means noop, non-nil means at least one transition)

## Create + Auto-Advance Flow

`charges.Create(...)` runs in two phases:

1. **Transaction phase** — creates all charge records (flat-fee, usage-based, credit-purchase) and any gathering invoice lines
2. **Post-create auto-advance** — `autoAdvanceCreatedCharges(...)` runs outside the transaction so that creation is persisted even if advancing fails (a worker can retry later)

`autoAdvanceCreatedCharges(...)` (`charges/service/create.go`):
- Iterates over the created charges using a type switch (`ChargeTypeUsageBased`, `ChargeTypeFlatFee`)
- Collects unique customer IDs that have credit-only charges of either type
- Calls `s.AdvanceCharges(...)` (the facade) once per unique customer
- Merges advanced charges back into the result by charge ID

This means a newly created credit-only charge (usage-based or flat fee) that is eligible for immediate activation will be returned as `active` (or `final`) from `Create(...)` itself.

## Root Charges Advance Flow

The root-facade advance flow is:

1. `charges.AdvanceCharges(...)` lists non-final charge metas for the customer
2. It partitions charges by type using `chargesByType(...)`
3. Early return if no usage-based and no flat-fee charges
4. For flat-fee credit-only charges: calls `flatfee.Service.AdvanceCharge(...)` per charge (no customer override or feature meters needed)
5. For usage-based charges: resolves merged customer profile, feature meters, then calls `usagebased.Service.AdvanceCharge(...)` per charge

Key package responsibilities:

- `charges/service/advance.go`
  - customer-scoped orchestration
  - preloads customer/profile + feature context for usage-based only
  - flat-fee credit-only charges are self-contained (fixed amount, no meters)
- `charges/meta/adapter`
  - lists non-final charges for a customer
- `charges/lock`
  - provides `NewChargeKey(...)` for charge-scoped locking

## Usage-Based Advance Flow

Usage-based advance currently does:

1. Validates:
   - `ChargeID`
   - expanded customer in `CustomerOverrideWithDetails`
   - valid merged profile
   - resolved feature meter
2. Takes a charge-scoped lock using:
   - `charges/lock.NewChargeKey(...)`
   - `*lockr.Locker`
3. Reloads the charge with realizations expanded
4. Routes by settlement mode
5. Builds `NewCreditsOnlyStateMachine(...)`
6. Calls `AdvanceUntilStateStable(...)`

This is the main place where charge lifecycle logic exists today.

## Credits-Only State Machine

The credits-only lifecycle is implemented in `usagebased/service/creditsonly.go` and `usagebased/service/statemachine.go`.

Relevant statuses:

- `created`
- `active`
- `active.final_realization.started`
- `active.final_realization.waiting_for_collection`
- `active.final_realization.processing`
- `active.final_realization.completed`
- `final`

High-level transitions:

1. `created -> active`
   - guarded by `IsInsideServicePeriod()`
   - sets `AdvanceAfter` to service-period start while waiting
2. `active -> active.final_realization.started`
   - guarded by `IsAfterServicePeriod()`
   - sets `AdvanceAfter` to service-period end while waiting
3. `active.final_realization.started -> active.final_realization.waiting_for_collection`
   - `StartFinalRealizationRun(...)` creates the realization run
4. `active.final_realization.waiting_for_collection -> active.final_realization.processing`
   - guarded by `IsAfterCollectionPeriod(...)`
5. `active.final_realization.processing -> active.final_realization.completed`
   - `FinalizeRealizationRun(...)` re-rates usage, computes delta vs initial run totals, then:
     - positive delta → allocates additional credits via `allocateCredits`
     - negative delta → corrects existing allocations via `Realizations.Correct()` with handler callback `OnCreditsOnlyUsageAccruedCorrection`
     - zero delta → no-op
6. `active.final_realization.completed -> final`
   - clears `AdvanceAfter`

`AdvanceUntilStateStable(...)` loops until the machine can no longer fire `TriggerNext`.

## Flat Fee Credits-Only State Machine

The flat fee credits-only lifecycle is implemented in `flatfee/service/creditsonly.go` and `flatfee/service/triggers.go`. Types are in `flatfee/statemachine.go`.

Statuses (much simpler than usage-based — no collection period):

- `created`
- `active`
- `final`

Transitions:

1. `created -> active`
   - guarded by `IsAfterInvoiceAt()` (`clock.Now() >= charge.Intent.InvoiceAt`)
   - sets `AdvanceAfter` to `InvoiceAt` while waiting
2. `active -> final`
   - unconditional (fires immediately after `active`)
   - `AllocateCredits(...)` calls `handler.OnCreditsOnlyUsageAccrued(...)` with `State.AmountAfterProration`
   - validates credit allocations sum equals amount
   - persists credit realizations via `adapter.CreateCreditAllocations(...)`
   - clears `AdvanceAfter` on entering `final`

Key differences from usage-based credits-only:

- No collection period, no two-phase realization
- Amount is computed at creation from `Intent.AmountBeforeProration` and stored in `State.AmountAfterProration`, no meter snapshot or rating
- No `FeatureMeter` or `CustomerOverride` needed
- Uses `meta.ChargeStatus` directly (not sub-statuses like `active.final_realization.*`)
- The flat fee adapter's `UpdateCharge(...)` takes a full `flatfee.Charge` (not `ChargeBase`)

Service construction requires a `*lockr.Locker` (same as usage-based).

Handler interface: `OnCreditsOnlyUsageAccrued(ctx, OnCreditsOnlyUsageAccruedInput)` returns `creditrealization.CreateAllocationInputs`. The production implementation in `ledger/chargeadapter/flatfee.go` is stubbed as not-implemented; the test handler is in `charges/service/handlers_test.go`.

Flat fee credit-only charges start with `InitialStatus: meta.ChargeStatusCreated` (not `Active`). The invoiced path still starts as `Active`.

## Collection Period Semantics

The collection-period logic is central to this package.

Rules:

- `usagebased.InternalCollectionPeriod` is `1 minute`
- `StartFinalRealizationRun(...)` computes `storedAtOffset = clock.Now() - InternalCollectionPeriod`
- the realization run persists `CollectionEnd`
- waiting logic must use the persisted run `CollectionEnd`, not a recomputed value
- `AdvanceAfterCollectionPeriodEnd(...)` sets `AdvanceAfter = CollectionEnd + InternalCollectionPeriod`
- `IsAfterCollectionPeriod(...)` checks `clock.Now() >= CollectionEnd + InternalCollectionPeriod`

`GetCollectionPeriodEnd(...)` currently uses:

- `CustomerOverride.MergedProfile.WorkflowConfig.Collection.Interval`
- added to `Charge.Intent.ServicePeriod.To`

Do not depend on a concrete customer-override record being present. The merged profile is the important input.

## Rating and Event Snapshot Semantics

Usage-based quantity is derived through `snapshotQuantity(...)`.

Important behavior:

- query window uses the charge service period
- stored-at filtering uses `stored_at < cutoff`
- the cutoff is the current `storedAtOffset`
- the service-period end is expected to behave as exclusive in lifecycle tests

This means late-arriving events can become eligible in later advances if their `stored_at` was previously too new but later falls before the next cutoff.

## Realization Runs

Realization runs are the persisted checkpoint for collection progress.

Important rules:

- the first final-realization advance creates a run
- `CollectionEnd` must be persisted on the run and mapped back into the domain model
- `CurrentRealizationRunID` points at the active run while waiting/finalizing
- finalization must clear `CurrentRealizationRunID`

Persistence gotcha:

- in `usagebased/adapter/charge.go`, use `SetOrClearCurrentRealizationRunID(...)`
- do not hand-roll separate `Set...` and `Clear...` branches unless there is a specific reason

## Status Persistence

Charge status persistence is split across:

- the shared meta charge row
- the charge-type-specific row

For usage-based charges:

When status changes:

- update the meta charge status to the short/meta status
- update the usage-based charge status to the full usage-based status

`usagebased.Status.ToMetaChargeStatus()` is the bridge between the full state-machine status and the root charge meta status.

## Testing Guidance

Key tests:

- `openmeter/billing/charges/service/advance_test.go`
- `openmeter/billing/charges/service/invoicable_test.go`

Use these conventions for lifecycle tests:

- validate root-facade behavior through `AdvanceCharges(...)` when testing orchestration
- validate persisted state through `mustGetChargeByID(...)`
- only use the direct `AdvanceCharges(...)` return as a secondary assertion
- if a returned charge is non-`nil`, at minimum match its status to the DB-loaded charge
- install usage-based handler callbacks only in the subtests that expect them (handler is reset in `TearDownTest`)
- use `streaming/testutils.WithStoredAt(...)` to simulate late events
- prefer `clock.FreezeTime(...)` for exact `AsOf` / `AllocateAt` assertions
- rely on the default billing profile unless the test explicitly needs customer-specific override behavior
- for credit-only charges (usage-based or flat fee), `Create(...)` itself may return an already-advanced charge — assert the returned charge's status, do not assume it will be `created`
- for flat fee credit-only tests, use `mustAdvanceFlatFeeCharges(...)` helper — it filters the advance result to flat fee charges only
- flat fee credit-only handler callbacks (`onCreditsOnlyUsageAccrued`) must return credit allocations that sum to the input `AmountToAllocate`

Test suite teardown:

- `BaseSuite.TearDownTest()` (capital D — testify calls this automatically between tests) resets `FlatFeeTestHandler`, `CreditPurchaseTestHandler`, `UsageBasedTestHandler`, and `MockStreamingConnector`
- Use `TearDownTest` (capital D) in all sub-suites; `TeardownTest` (lowercase d) is **not** called by testify
- `MockStreamingConnector` events are shared across all tests in the suite — always rely on `TearDownTest` to reset them rather than `defer`

Billing-profile test gotcha:

- `ProvisionBillingProfile(...)` supports multiple edit options
- when asserting collection timing, verify the created profile is default and, if needed, compare it to `GetDefaultProfile(...)`

## Running Tests

For direct package runs, use the repo env and Postgres. Prefer direct command execution; do not wrap these in `sh -lc`, `bash -lc`, or similar helper shells when a direct invocation works.

```bash
POSTGRES_HOST=127.0.0.1 direnv exec . go test -run TestInvoicableCharges/TestUsageBasedCreditOnlyLifecycle -v ./openmeter/billing/charges/service
POSTGRES_HOST=127.0.0.1 direnv exec . go test ./openmeter/billing/charges/...
```

## Editing Checklist

When changing charges:

- decide whether the change belongs in:
  - the root facade
  - meta queries
  - charge locking
  - a type-specific package
- preserve the current root rule that `AdvanceCharges(...)` only advances supported types (usage-based and flat-fee credit-only)
- keep meta status and type-specific status in sync

When changing usage-based charges:

- confirm whether the change belongs in the facade, usage-based service, state machine, or adapter
- preserve the `nil means noop` contract for `AdvanceCharge(...)`
- preserve merged-profile based collection-period resolution
- keep `CollectionEnd` persisted on realization runs
- keep the `stored_at < cutoff` behavior explicit in tests
- update lifecycle tests if late-event visibility changes
When changing flat-fee charges:

- the invoiced path (CreditThenInvoice/InvoiceOnly) starts as `Active` and is driven by invoice lifecycle hooks
- the credit-only path starts as `Created` and is driven by the state machine — do not mix the two
- `AmountAfterProration` lives on `flatfee.State`, not `flatfee.Intent` — it is computed at creation via `Intent.CalculateAmountAfterProration()` and persisted. Callers must not provide it; they set `AmountBeforeProration`, `ServicePeriod`, `FullServicePeriod`, and `ProRating` on the Intent
- `IntentWithInitialStatus` carries `AmountAfterProration` alongside `InitialStatus` to pass the computed value from the service to the adapter at creation time
- `flatfee.State.AdvanceAfter` must be passed through `chargemeta.UpdateInput.AdvanceAfter` on every `UpdateCharge(...)` call
- `flatfee.Adapter.UpdateCharge(...)` takes the full `flatfee.Charge`, not a `ChargeBase` — extract `State.AdvanceAfter` when building `chargemeta.UpdateInput`
- the `flatfee.Handler` interface has both invoiced-path methods and credits-only methods — implementors must satisfy all of them
- adding new `Handler` methods requires updating: `ledger/chargeadapter/flatfee.go`, `test/credits/mockledger.go`, `charges/service/handlers_test.go`
- the same applies to `usagebased.Handler` — new methods must be added to `UnimplementedHandler`, the ledger adapter, and the test handler

Usage-based handler interface (`usagebased.Handler`):
- `OnCreditsOnlyUsageAccrued(ctx, CreditsOnlyUsageAccruedInput)` → `creditrealization.CreateAllocationInputs` — allocate credits for a realization run
- `OnCreditsOnlyUsageAccruedCorrection(ctx, CreditsOnlyUsageAccruedCorrectionInput)` → `creditrealization.CreateCorrectionInputs` — correct (partially revert) existing credit allocations when finalization discovers usage decreased
- `flatfee/service/service.go` Config requires a `*lockr.Locker` — when constructing in tests, create the locker before the flat fee service

## Credit Realization Model

The `creditrealization` package (`openmeter/billing/charges/models/creditrealization/`) defines the domain model for credit allocations and corrections (partial/full reverts).

### Type hierarchy

- `CreateAllocationInput` — positive-amount allocation input (has `LineID`). Collection type: `CreateAllocationInputs`.
- `CreateCorrectionInput` — positive-amount correction request (has `CorrectsRealizationID`). Collection type: `CreateCorrectionInputs`.
- `CreateInput` — unified DB write input (used by both allocations and corrections). Has `Type` field (`TypeAllocation` or `TypeCorrection`). Collection type: `CreateInputs`.
- `Realization` — full model read from DB, embeds `CreateInput` + `NamespacedModel` + `ManagedModel` + `SortHint`.
- `Realizations` — slice of `Realization` with query/aggregation methods.

### Sign convention

- **Allocations** have **positive** `Amount` in `CreateInput` and DB.
- **Corrections** have **negative** `Amount` in `CreateInput` and DB (negated by `AsCreateInputs`).
- `CreateCorrectionInput.Amount` is always **positive** (the amount to correct). It gets negated when converting to `CreateInput` via `CreateCorrectionInputs.AsCreateInputs()`.
- `Realizations.Sum()` returns the net total (allocations minus corrections).
- `allocationsWithCorrections()` computes remaining amounts by calling `.Sub(corrections.Sum())` on each allocation. Since corrections have negative amounts in the DB, `corrections.Sum()` is negative, and `.Sub(negative)` correctly adds back — resulting in `remaining = allocation + |corrections|`. **This is wrong** — it makes remaining larger than the allocation. This is a known sign-convention risk: the `Sub` works correctly only if corrections are stored with positive amounts (old convention) or if the code is updated to use `.Add(corrections.Sum())` with the new negative convention.

### Correction flow

The full correction orchestration is `Realizations.Correct(amount, currency, callback)`:

1. `CreateCorrectionRequest(amount, currency)` — builds `CorrectionRequest` items in **reverse** creation order (latest allocation first)
2. `CorrectionRequest.ValidateWith(currency)` — validates the request items
3. Calls `callback(req)` — caller (ledger handler) maps request items to `CreateCorrectionInputs` with ledger transaction references
4. `CreateCorrectionInputs.ValidateWith(realizations, totalAmount, currency)` — validates corrections don't exceed remaining per-allocation amounts
5. `CreateCorrectionInputs.AsCreateInputs(realizations)` — maps to `[]CreateInput` with negated amounts, copies `ServicePeriod` from the corrected allocation

### Unit test patterns for creditrealization

Tests are in `correction_test.go` (same package, not `_test`). Reusable helpers:

- `allocationBuilder` — builds allocation `Realization` entries with auto-incrementing `SortHint` and configurable `CreatedAt`
- `correctionFor(allocation, amount)` — builds a correction `Realization` targeting a given allocation
- `correctionCallback(txGroupID)` — returns a `func(CorrectionRequest) (CreateCorrectionInputs, error)` for use with `Correct()`
- `correctionRequestAmounts(cr)` / `correctionRequestAllocationIDs(cr)` — extract slices for assertions
- `correctionInputsSum(inputs)` — sums `CreateCorrectionInputs` amounts
- `testCurrency(t)` — returns a USD `currencyx.Calculator`

Test structure follows the `rate_test` pattern: declarative test cases with `t.Run` subtests, shared helpers, no DB required.

## Adapter Gotchas

- `resolveFeatureMeters(ctx, namespace, charges)` takes an explicit `namespace` argument — do not access `charges[0].Namespace` directly (panics on empty slice)
- `GetByMetas` re-orders output to match input order; use `lo.KeyBy` (not `lo.GroupBy`) when building an intermediate lookup map — `GroupBy` produces `map[K][]V` and requires `[0]` indexing, `KeyBy` gives `map[K]V` directly
- `refetchCharge` in the state machine is a known interim pattern — the preferred direction is in-memory charge updates after adapter writes; avoid adding new `refetchCharge` calls without discussion
- `buildCreateUsageBasedCharge` is a builder chain — do not call the same setter twice (Ent builder chains accept duplicate `.SetX` calls silently, the last one wins)
- `currencyx.Calculator.IsRoundedToPrecision(amount)` is the preferred way to check if an amount is rounded to currency precision — use it instead of manual `RoundToPrecision(x).Equal(x)` patterns
