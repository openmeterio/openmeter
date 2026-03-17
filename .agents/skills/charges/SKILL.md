---
name: charges
description: Work with OpenMeter billing charges, including the root charges facade, charge meta queries, charge creation and advancement, usage-based lifecycle state machines, realization runs, and charges test setup. Use when modifying `openmeter/billing/charges/...` or charge-related tests.
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Charges

Guidance for working with OpenMeter billing charges.

This skill describes the charges package generically, but most lifecycle detail currently lives in the usage-based branch. Ignore inconsistencies or patterns from the flat-fee and credit-purchase branches unless the user explicitly asks to unify behavior across charge types.

## Scope

Primary packages:

- `openmeter/billing/charges/`
- `openmeter/billing/charges/service/`
- `openmeter/billing/charges/meta/`
- `openmeter/billing/charges/lock/`
- `openmeter/billing/charges/usagebased/`
- `openmeter/billing/charges/usagebased/service/`
- `openmeter/billing/charges/usagebased/adapter/`
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

Current support is intentionally narrow:

- `charges.AdvanceCharges(...)` currently only advances usage-based charges
- `usagebased.Service.AdvanceCharge(...)` only supports `CreditOnly`
- `CreditThenInvoice` usage-based advance is deliberately rejected with a not-implemented error
- `AdvanceCharge(...)` returns `*usagebased.Charge`
  - `nil` means no advancement happened
  - non-`nil` means at least one state transition occurred

## Create + Auto-Advance Flow

`charges.Create(...)` runs in two phases:

1. **Transaction phase** — creates all charge records (flat-fee, usage-based, credit-purchase) and any gathering invoice lines
2. **Post-create auto-advance** — `autoAdvanceCreatedCharges(...)` runs outside the transaction so that creation is persisted even if advancing fails (a worker can retry later)

`autoAdvanceCreatedCharges(...)` (`charges/service/create.go`):
- Iterates over the created charges directly by type (no `chargesByType` helper)
- Collects unique customer IDs that have credit-only usage-based charges
- Calls `s.AdvanceCharges(...)` (the facade) once per unique customer
- Merges advanced charges back into the result by charge ID

This means a newly created credit-only usage-based charge that is eligible for immediate activation will be returned as `active` (or further along) from `Create(...)` itself.

## Root Charges Advance Flow

The current root-facade advance flow is:

1. `charges.AdvanceCharges(...)` lists non-final charge metas for the customer
2. It filters to `meta.ChargeTypeUsageBased`
3. It resolves the merged customer profile with expanded customer details
4. It loads usage-based charges with realizations expanded
5. It resolves feature meters once per batch
6. It calls `usagebased.Service.AdvanceCharge(...)` per charge

Key package responsibilities:

- `charges/service/advance.go`
  - customer-scoped orchestration
  - preloads customer/profile + feature context
  - skips unsupported charge types for now
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
   - `FinalizeRealizationRun(...)` updates the run and allocations
6. `active.final_realization.completed -> final`
   - clears `AdvanceAfter`

`AdvanceUntilStateStable(...)` loops until the machine can no longer fire `TriggerNext`.

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
- for credit-only usage-based charges, `Create(...)` itself may return an already-advanced charge — assert the returned charge's status, do not assume it will be `created`

Test suite teardown:

- `BaseSuite.TearDownTest()` (capital D — testify calls this automatically between tests) resets `FlatFeeTestHandler`, `CreditPurchaseTestHandler`, `UsageBasedTestHandler`, and `MockStreamingConnector`
- Use `TearDownTest` (capital D) in all sub-suites; `TeardownTest` (lowercase d) is **not** called by testify
- `MockStreamingConnector` events are shared across all tests in the suite — always rely on `TearDownTest` to reset them rather than `defer`

Billing-profile test gotcha:

- `ProvisionBillingProfile(...)` supports multiple edit options
- when asserting collection timing, verify the created profile is default and, if needed, compare it to `GetDefaultProfile(...)`

## Running Tests

For direct package runs, use the repo env and Postgres:

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
- preserve the current root rule that `AdvanceCharges(...)` only advances supported types
- keep meta status and type-specific status in sync

When changing usage-based charges:

- confirm whether the change belongs in the facade, usage-based service, state machine, or adapter
- preserve the `nil means noop` contract for `AdvanceCharge(...)`
- preserve merged-profile based collection-period resolution
- keep `CollectionEnd` persisted on realization runs
- keep the `stored_at < cutoff` behavior explicit in tests
- update lifecycle tests if late-event visibility changes
- ignore flat-fee and credit-purchase design differences unless the user explicitly asks to unify them

## Adapter Gotchas

- `resolveFeatureMeters(ctx, namespace, charges)` takes an explicit `namespace` argument — do not access `charges[0].Namespace` directly (panics on empty slice)
- `GetByMetas` re-orders output to match input order; use `lo.KeyBy` (not `lo.GroupBy`) when building an intermediate lookup map — `GroupBy` produces `map[K][]V` and requires `[0]` indexing, `KeyBy` gives `map[K]V` directly
- `refetchCharge` in the state machine is a known interim pattern — the preferred direction is in-memory charge updates after adapter writes; avoid adding new `refetchCharge` calls without discussion
- `buildCreateUsageBasedCharge` is a builder chain — do not call the same setter twice (Ent builder chains accept duplicate `.SetX` calls silently, the last one wins)
