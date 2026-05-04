---
name: charges
description: Work with OpenMeter billing charges, including the root charges facade, charge meta queries, charge creation and advancement, usage-based lifecycle state machines, realization runs, and charges test setup. Use when modifying `openmeter/billing/charges/...` or charge-related tests.
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Charges

Guidance for working with OpenMeter billing charges.

This skill describes the charges package generically. Lifecycle state machines exist for both usage-based and flat-fee credit-only branches. All three charge types (usage-based, flat-fee, credit-purchase) follow the same structural pattern: `ChargeBase`/`Charge` with `Realizations`, own `Status` type, `status_detailed` DB column, and composite adapter interfaces.

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
- `openmeter/billing/charges/creditpurchase/`
- `openmeter/billing/charges/creditpurchase/service/`
- `openmeter/billing/charges/creditpurchase/adapter/`
- `openmeter/billing/charges/service/invoicable_test.go`
- `openmeter/billing/charges/service/advance_test.go`

## Current Design

`openmeter/billing/charges` is the root facade for charge operations.

For charge-owned detailed lines, the shared invoice-agnostic base belongs in `openmeter/billing/models/stddetailedline`. Prefer using `type DetailedLine = stddetailedline.Base` in charge packages and keep ownership implicit through containment in the parent aggregate (`flatfee.Realizations` or `usagebased.RealizationRun`) rather than duplicating `charge_id` / `run_id` fields in the domain type. Reuse the shared detailed-line base mapping and create helpers instead of duplicating common field assembly in charge adapters.

Charge-backed invoicing no longer relies on a charges-side `InvoicePendingLines(...)` wrapper. Billing owns invoice creation and dispatches gathering lines by `billing.LineEngineType`, while charge packages provide charge-specific line engines where needed.

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
- type-specific service subpackages may own reusable realization mechanics when the parent service becomes too broad; keep state-machine decisions in the lifecycle files and move only mechanical operations such as rating snapshots, realization persistence, credit allocation/correction, and realization lineage persistence into these helpers
- invoice-backed charges must not reach the meta `final` state until their payment lifecycle is fully settled; if a charge is waiting on invoice payment authorization or settlement, keep it in an `active.*` detailed status instead of collapsing to `final`

Important types:

- `charges.AdvanceChargesInput` identifies the customer whose charges should advance
- `meta.Charge` and `meta.ChargeID` define the shared charge identity and type
- `charges.Charge` wraps concrete charge variants
- `flatfee.Intent` carries `AmountBeforeProration`, `ProRating`, `SettlementMode`, `PaymentTerm`, `InvoiceAt` — immutable inputs provided by the caller
- `flatfee.ChargeBase` stores the persisted flat-fee charge row: current `Status` plus durable `State`
- `flatfee.State` currently tracks:
  - `AmountAfterProration`
  - `AdvanceAfter`
  - `FeatureID`
- `flatfee.Realizations` stores expanded, non-base data loaded from child tables:
  - `CreditRealizations`
  - `AccruedUsage`
  - `Payment`
  - `DetailedLines`
- `flatfee.Intent.CalculateAmountAfterProration()` computes the prorated amount from `AmountBeforeProration`, `ServicePeriod/FullServicePeriod` ratio, and `ProRating` config, with currency-precision rounding
- Charge-backed targets do not use invoice-style semantic proration or empty-period filtering; the charge stack materializes and prorates state itself, and the flat fee charge is responsible for omitting empty lines
- `usagebased.Intent` carries `FeatureKey`, `Price`, `SettlementMode`, `InvoiceAt`, and `ServicePeriod`
- `usagebased.ChargeBase` stores the current `Status` and `State`
- `usagebased.State` currently tracks:
  - `CurrentRealizationRunID`
  - `AdvanceAfter`
- `usagebased.RealizationRunBase` stores:
  - `Type`
  - `StoredAtLT`
  - `ServicePeriodTo`
  - `MeteredQuantity`
  - `Totals`
- `usagebased.RealizationRun` can expand:
  - `DetailedLines`

Detailed-line expansion rules:

- `meta.ExpandDetailedLines` is not standalone for charge reads; it requires `meta.ExpandRealizations`
- usage-based detailed lines live under `RealizationRun.DetailedLines`
- flat-fee detailed lines also live under `Charge.Realizations.DetailedLines`, not on the root charge
- `mo.None()` means detailed lines were not expanded; present options mean expanded data, even when the underlying slice is nil/empty

## Billing Line Engines

Charge-backed gathering lines must carry the correct billing line engine when they are created.

Current engine values:

- `billing.LineEngineTypeChargeFlatFee`
- `billing.LineEngineTypeChargeUsageBased`
- `billing.LineEngineTypeChargeCreditPurchase`

Current implementations:

- flat fee line engine: `openmeter/billing/charges/flatfee/service/lineengine.go`
- credit purchase line engine: `openmeter/billing/charges/creditpurchase/lineengine`
- usage-based line engine: `openmeter/billing/charges/usagebased/service/lineengine.go`

Important rules:

- do not rely on billing to infer a charge-backed engine from `ChargeID`
- `billing/service.CreatePendingInvoiceLines(...)` rejects charge-backed gathering lines with empty `Engine`
- production wiring must register charge line engines through `billing.Service.RegisterLineEngine(...)`
- tests that temporarily add engines can remove them again through `billing.Service.DeregisterLineEngine(...)`; use the public registry API instead of mutating billing internals from non-service packages
- charge test setups must also register those engines explicitly; keep this in `openmeter/billing/charges/testutils`
- if a charge create path stamps a new `LineEngineType`, app wiring and charge test wiring must register a matching implementation in the same change
- usage-based exposes its billing line engine from `usagebased.Service.GetLineEngine()`; register that returned engine instead of reusing the service type directly
- flat-fee now follows the same pattern: `flatfee.Service.GetLineEngine()` returns the engine owned by the service package
- because flat-fee owns its line engine, `flatfee/service.New(...)` requires a `rating.Service`; forgetting that dependency breaks app/test wiring with `rating service cannot be null`
- for usage-based realization creation, validate at the run-creation boundary (`usagebased/service/run.CreateRatedRunInput.Validate`) that `Charge.State.CurrentRealizationRunID` is nil before creating a new run; keep the line-engine-side early return too so `InvoicePendingLines` fails with the charge-specific validation error at the billing boundary. In both places, key the guard off `CurrentRealizationRunID`, not a specific status prefix such as `partial_invoice`
- usage-based payment handling is intentionally different from flat-fee and credit-purchase: the usage-based state machine owns realization only, while the usage-based line engine/run service records payment authorization/settlement directly on historical runs and only re-enters the state machine through an aggregate trigger (for example `all_payments_settled`) once all invoiced runs on the charge are settled. Do not apply this rule generically to flat-fee or credit-purchase; those charge types may still keep payment states inside their own state machines.
- usage-based invoice branches should read in this order: `started -> waiting_for_collection -> processing -> issuing -> completed`, then auto-advance out of the branch. Keep `invoice_issued` as the boundary between `processing` and `issuing`, run `FinalizeInvoiceRun(...)` from the `issuing` state, and let `completed` be the last branch-local status before `next` returns a partial invoice to `active` or moves a final invoice to `active.awaiting_payment_settlement`
- when adding or renaming usage-based detailed statuses, remember that `status_detailed` is an Ent enum for `ChargeUsageBased`; run `make generate` so the generated enum validators and migrate schema include the new values before trusting state-machine changes

Operational consequence:

- adding a new charge engine enum without a registered implementation causes invoice collection to fail when billing resolves the line engine
- a schema migration defaulting old persisted rows to `invoicing` is not enough for charge-backed lines; existing persisted gathering lines may need a backfill if they should route to a charge engine after rollout

Current shared contract details:

- line-engine transport structs in `openmeter/billing/lineengine.go` are validated at the billing callsite before invoking the engine, and returned lines/results are validated after the call
- `OnCollectionCompleted(...)` takes `billing.OnCollectionCompletedInput` and returns updated `billing.StandardLines`
- collection-time engines must preserve the exact line ID set they were given; billing validates that returned line IDs match input line IDs exactly and then merges the returned lines back into the invoice
- `CalculateLines(...)` returns updated `billing.StandardLines`; billing treats this as a pure recalculation boundary, validates exact line ID preservation, and merges the returned lines back into the invoice instead of relying on in-place mutation
- `CalculateLines(...)` no longer takes `context.Context`; if a future charge engine needs context-aware recalculation, propagate that need deliberately through the contract instead of using `context.Background()` as a workaround
- `SplitGatheringLine(...)` takes a concrete `billing.GatheringLine` plus `SplitAt` and returns only the split line fragments; the billing caller owns fetching the current line from the gathering invoice and merging `PreSplitAtLine` / optional `PostSplitAtLine` back into the invoice aggregate
- optional split outputs use pointers; `SplitGatheringLineResult.PostSplitAtLine` is `*billing.GatheringLine`
- use `billing.ValidateStandardLineIDsMatchExactly(...)` when a charge-side test or helper needs to assert that returned standard-line identities are preserved across a line-engine boundary
- collection-time errors should be normalized through `billing.NewLineEngineValidationError(...)` instead of rebuilding the validation-issue wrapper at each billing callsite
- charge line engines are only responsible for invoice-backed charge flows such as `credit_then_invoice`; they are not the execution path for `credit_only` settlement mode
- because of that boundary, it is acceptable for a charge line engine to return an error when invoked with `credit_only` settlement mode; treat that as a lifecycle misuse rather than adding `credit_only` behavior to the engine

## Timestamp Normalization

Charge persistence assumes timestamp precision is bounded by streaming aggregation precision.

Rules:

- persisted charge timestamps must be truncated to `streaming.MinimumWindowSizeDuration`
- `meta.NormalizeTimestamp(...)` is the shared primitive; it also converts to UTC
- `meta.NormalizeClosedPeriod(...)` and `Intent.Normalized()` helpers are the domain-level normalization entrypoints
- normalize intent timestamps before validation and before any derived calculation that depends on durations or boundaries
- flat-fee proration must use normalized periods, otherwise sub-second inputs can change `AmountAfterProration`
- for usage-based lifecycle timestamps (`AdvanceAfter`, `StoredAtLT`, `ServicePeriodTo`), normalize the computed timestamp before persisting it or handing it to downstream persistence callbacks

Important timestamp surfaces:

- `meta.Intent.ServicePeriod`
- `meta.Intent.FullServicePeriod`
- `meta.Intent.BillingPeriod`
- `flatfee.Intent.InvoiceAt`
- `usagebased.Intent.InvoiceAt`
- `flatfee.State.AdvanceAfter`
- `usagebased.State.AdvanceAfter`
- `usagebased.CreateRealizationRunInput.StoredAtLT`
- `usagebased.CreateRealizationRunInput.ServicePeriodTo`
- `usagebased.UpdateRealizationRunInput.StoredAtLT`

Placement guidance:

- prefer domain-side normalization when constructing or mutating intents and state (`Intent.Normalized()`, state-machine transition logic, temporary patch remap)
- keep a persistence backstop in shared write helpers such as `charges/models/chargemeta`
- in adapters, normalize at the actual write setter (`SetInvoiceAt(...)`, `SetStoredAtLt(...)`, `SetServicePeriodTo(...)`, `SetOrClearAdvanceAfter(...)`) rather than rewriting the whole input object at the top of the adapter method
- do not add redundant `.UTC()` calls after `meta.NormalizeTimestamp(...)`; the helper already returns UTC

## Currency Normalization

Charge lifecycle code owns currency rounding.

Rules:

- all currency amounts that enter or leave the charge lifecycle must be rounded with the charge currency calculator
- normalize charge-domain totals and inputs in charge code where charges own the amount
- ledger-backed allocation realizations may be stored exactly as returned by ledger-owned handlers for credits-only flows
- correction request and correction creation outputs are normalized in the shared `creditrealization.Realizations.Correct(...)` / `CorrectAll(...)` path
- zero-valued corrections are allowed after rounding; treat them as a no-op instead of an error

Important money surfaces:

- `creditpurchase.Intent.CreditAmount`
- `flatfee.Intent.AmountBeforeProration`
- `flatfee.State.AmountAfterProration`
- usage-based realization `Totals`
- `flatfee.OnAssignedToInvoiceInput.PreTaxTotalAmount`
- `flatfee.OnCreditsOnlyUsageAccruedInput.AmountToAllocate`
- `usagebased.CreditsOnlyUsageAccruedInput.AmountToAllocate`
- `creditrealization.CreateAllocationInputs`
- `creditrealization.CreateCorrectionInputs`

Placement guidance:

- normalize durable charge inputs in domain helpers such as `Intent.Normalized()`
- for flat fees, make sure `AmountAfterProration` is already rounded when calculated; adapters should persist it as-is
- when calling charge handlers that feed ledger-backed allocations/corrections, decide explicitly whether charges or ledger owns the returned amount; for current credits-only allocation flows, store ledger-returned allocation amounts as-is
- prefer shared normalization in `creditrealization` helpers for correction flows instead of repeating callback-local normalization at each callsite
- do not mutate whole intents inside adapters just to normalize currency; if an adapter needs a persistence backstop, keep it local to the `Set*` write
- when ledger logic derives monetary values from balances, entries, or its own calculations, round those values in ledger code as well; do not rely only on upstream callers

## Zero Invoice Accrual

Invoice accrual uses a non-negative, no-op-aware contract.

Rules:

- negative invoice-accrual amounts are invalid
- zero invoice-accrual amounts are valid no-ops
- positive invoice-accrual amounts must produce a non-empty ledger transaction group reference
- charges-side services should short-circuit zero before calling persistence that expects a real ledger transaction
- do not persist `invoicedusage.AccruedUsage` rows with an empty `LedgerTransaction.TransactionGroupID`

Current expected behavior:

- `usagebased.OnInvoiceUsageAccruedInput.Validate()` allows zero and rejects only negatives
- usage-based and flat-fee service/orchestration layers should skip invoice-accrual persistence when the invoice line total is zero
- ledger handlers may still defensively tolerate zero and return `ledgertransaction.GroupReference{}`
- when a service proceeds with non-zero invoice accrual, it must require a non-empty transaction group reference before storing accrued usage

## HTTP/API Conversion

Credit-purchase charges have an API/domain enum mismatch for promotional grants.

Rules:

- in the billing domain, promotional credit grants are represented as `creditpurchase.SettlementTypePromotional`
- in the v3 customer credits API, the same case is represented as `funding_method=none`
- v3 API responses for promotional grants must omit the `purchase` block entirely
- conversion code in `api/v3/handlers/customers/credits` must map this case explicitly instead of treating `promotional` as an unsupported settlement type

Important files:

- `api/v3/handlers/customers/credits/convert.go`
- `openmeter/billing/charges/creditpurchase/settlement.go`
- `openmeter/billing/creditgrant/service/service.go`
- `api/spec/packages/aip/src/customers/credits/grant.tsp`

## Realization Helper Subpackages

Use small type-specific realization helper subpackages to keep charge services and state machines from becoming kitchen-sink orchestration layers.

The purpose of these subpackages is to separate reusable realization mechanics from lifecycle decisions:

- state-machine files decide which trigger/status/action applies for a settlement mode
- realization helper packages execute reusable mechanics once that decision has already been made
- helper packages must not decide which trigger to fire, which status to enter, or whether a charge lifecycle event should advance

Naming should describe the charge-domain unit being manipulated rather than the current ledger operation. Prefer `realizations` for flat-fee helpers because flat fees have credit realizations today and will also support invoiced/payment realization flows. For usage-based charges, a `run` helper is appropriate when the helper owns realization-run mechanics such as rated run creation, run persistence, credit allocation/correction, and run credit-realization lineage.

Keep these helpers type-specific instead of forcing a generic cross-charge state machine. Flat-fee and usage-based lifecycles share some mechanics, but their durable state and lifecycle semantics differ: usage-based has realization runs, collection cutoffs, and `CurrentRealizationRunID`; flat-fee has charge-level realizations, proration, invoice hooks, and payment hooks.

When extracting helpers:

- inject only the dependencies the helper needs, usually adapter, handler, lineage, and any rating helper
- expose struct-returning methods for multi-value outcomes
- keep credit allocation/correction exactness explicit at the call site, because `credit_only` and `credit_then_invoice` can differ
- keep lineage persistence next to realization creation so allocation and correction paths do not duplicate lineage bookkeeping
- keep the parent service responsible for transactions, locking, and building settlement-mode-specific state machines

## Supported Behavior

- `charges.AdvanceCharges(...)` advances both usage-based and flat-fee credit-only charges
- `usagebased.Service.AdvanceCharge(...)` routes to the settlement-mode-specific state machine: `CreditOnly` uses the credits-only state machine and `CreditThenInvoice` uses `NewCreditThenInvoiceStateMachine(...)`
- `flatfee.Service.AdvanceCharge(...)` currently only supports `CreditOnly`; it does not have a `CreditThenInvoice` state machine path
- Both `AdvanceCharge(...)` methods return `*Charge` (nil means noop, non-nil means at least one transition)
- For invoice-settled flows, invoice creation and collection completion are still billing-triggered downstream events (`invoice_created`, `collection_completed`), not generic advance-loop transitions

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

For invoice-settled charges:

- flat fee and credit purchase creation now stamp the gathering line engine in their type-specific `Create(...)` flows
- usage-based charge creation also stamps `LineEngineTypeChargeUsageBased`; do not introduce this discriminator unless a corresponding billing engine exists or the path is intentionally blocked
- usage-based `IsLineBillableAsOf(...)` is currently billable only once `asOf >= resolved service period end`; keep the existing progressive-billing TODO in place when touching that logic
- for usage-based `credit_then_invoice`, `BuildStandardInvoiceLines(...)` is allowed to drive charge lifecycle transitions needed to create the invoice-backed realization run
- `OnCollectionCompleted(...)` is the single collection-time line-engine hook; do not reintroduce a generic shared `SnapshotLines()` abstraction for charge engines
- prefer explicit usage-based lifecycle triggers such as `invoice_created` and `collection_completed` over generic line-snapshot callbacks
- collection-time charge-engine failures must surface as invoice validation issues through the billing line-engine validation flow, not as raw invoice-state-machine failures

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
5. Builds the settlement-mode-specific state machine (`NewCreditsOnlyStateMachine(...)` or `NewCreditThenInvoiceStateMachine(...)`)
6. Calls `AdvanceUntilStateStable(...)`

This is the main place where charge lifecycle logic exists today.

State-machine organization rule:

- keep state-machine-specific methods and helpers in the state-machine file that owns them
- do not spread one charge state machine across multiple files unless the split is the standard `service/` or `adapter/` package boundary
- line-engine files may call into a state machine, but they should not define that state machine's transition handlers

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
- Uses `flatfee.Status` with only top-level states (not sub-statuses like `active.final_realization.*`)
- Persists only `flatfee.ChargeBase`; credit allocations / payment / accrued usage live in `flatfee.Realizations`

Service construction requires a `*lockr.Locker` (same as usage-based).

Handler interface: `OnCreditsOnlyUsageAccrued(ctx, OnCreditsOnlyUsageAccruedInput)` returns `creditrealization.CreateAllocationInputs`. The production implementation in `ledger/chargeadapter/flatfee.go` is stubbed as not-implemented; the test handler is in `charges/service/handlers_test.go`.

Flat fee credit-only charges start with `InitialStatus: flatfee.StatusCreated` (not `Active`). The invoiced path still starts as `flatfee.StatusActive`.

## Collection Period Semantics

The collection-period logic is central to this package.

Rules:

- `usagebased.InternalCollectionPeriod` is `1 minute`
- `StoredAtLT` is the exclusive stored-at query cap for the run (`stored_at < StoredAtLT`)
- `ServicePeriodTo` is the exclusive event-time upper bound for the run (`event_time < ServicePeriodTo`)
- final usage-based runs use the charge intent's service-period end as `ServicePeriodTo`
- final usage-based runs use the charge service-period end plus the billing profile collection interval as `StoredAtLT`
- partial invoice runs use the standard line period end as both `ServicePeriodTo` and `StoredAtLT`
- waiting logic must use the persisted run `StoredAtLT`, not a recomputed value
- `AdvanceAfterCollectionPeriodEnd(...)` sets `AdvanceAfter = StoredAtLT + InternalCollectionPeriod`
- `IsAfterCollectionPeriod(...)` checks `clock.Now() >= StoredAtLT + InternalCollectionPeriod`
- usage-based standard invoice lines should set `OverrideCollectionPeriodEnd = StoredAtLT + InternalCollectionPeriod` so invoice collection waits for the same internal buffer as the charge state machine

Final-run `StoredAtLT` currently uses:

- `CustomerOverride.MergedProfile.WorkflowConfig.Collection.Interval`
- added to `Charge.Intent.ServicePeriod.To`

Do not depend on a concrete customer-override record being present. The merged profile is the important input.

## Rating and Event Snapshot Semantics

Usage-based quantity is derived through `snapshotQuantity(...)`.

Important behavior:

- query window starts at the charge intent's service-period start
- query window ends at the run's `ServicePeriodTo`
- stored-at filtering uses `stored_at < cutoff`
- the cutoff is the run's `StoredAtLT`
- the service-period end is expected to behave as exclusive in lifecycle tests
- `GetDetailedRatingForUsage(...)` owns the current-run filtering rule: only realization runs with `ServicePeriodTo < input.ServicePeriodTo` are prior runs; a current run already present on the charge must be ignored rather than stripped by mutating the charge in the caller
- minimum commitment is final-only for usage-based snapshots; detailed rating ignores it when the current service-period end is before the charge intent service-period end
- realtime/current totals should ignore minimum commitment before the charge intent service-period end and include it at/after the service-period end

This means late-arriving events can become eligible in later advances if their `stored_at` was previously too new but later falls before the next cutoff.

## Realization Runs

Realization runs are the persisted checkpoint for collection progress.

Important rules:

- the first final-realization advance creates a run
- `StoredAtLT`, `ServicePeriodTo`, and `MeteredQuantity` must be persisted on the run and mapped back into the domain model
- `CurrentRealizationRunID` points at the active run while waiting/finalizing
- finalization must clear `CurrentRealizationRunID`

Persistence gotcha:

- in `usagebased/adapter/charge.go`, use `SetOrClearCurrentRealizationRunID(...)`
- do not hand-roll separate `Set...` and `Clear...` branches unless there is a specific reason

## Status Persistence

Charge status persistence is split across:

- the shared meta charge row
- the charge-type-specific row

For all three charge types (usage-based, flat-fee, credit-purchase):

When status changes:

- update the meta charge status to the short/meta status
- update the type-specific charge `status_detailed` to the full type-specific status

`usagebased.Status.ToMetaChargeStatus()`, `flatfee.Status.ToMetaChargeStatus()`, and `creditpurchase.Status.ToMetaChargeStatus()` are the bridges between the full state-machine status and the root charge meta status.

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
- when testing stored-at cutoffs, remember the predicate is exclusive: an event with `stored_at == StoredAtLT` is excluded, and an event with `stored_at` before `StoredAtLT` is included
- when testing service-period cutoffs, remember the event-time window is half-open: an event with `event_time == ServicePeriodTo` is excluded
- prefer `streamingtestutils.NewMockStreamingConnector(...)` plus the real billing rating service when a usage-based rating test should exercise production quantity lookup, pricing, discounts, or commitments end-to-end
- prefer `clock.FreezeTime(...)` for exact `StoredAtLT` / `AllocateAt` assertions
- rely on the default billing profile unless the test explicitly needs customer-specific override behavior
- for credit-only charges (usage-based or flat fee), `Create(...)` itself may return an already-advanced charge — assert the returned charge's status, do not assume it will be `created`
- for flat fee credit-only tests, use `mustAdvanceFlatFeeCharges(...)` helper — it filters the advance result to flat fee charges only
- flat fee credit-only handler callbacks (`onCreditsOnlyUsageAccrued`) must return credit allocations that sum to the input `AmountToAllocate`
- when testing timestamp truncation, use sub-second fixtures and assert the persisted charge/run fields are second-aligned after create/advance
- `time.Time` fields on domain models are value typed; use `s.False(ts.IsZero())` instead of `s.NotNil(ts)` when asserting they are populated
- cover the temporary shrink/extend remap path as well; it synthesizes new intents and must normalize the replacement period ends before re-create

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
- keep `StoredAtLT`, `ServicePeriodTo`, and `MeteredQuantity` persisted on realization runs
- keep the `stored_at < cutoff` behavior explicit in tests
- update lifecycle tests if late-event visibility changes
When changing flat-fee charges:

- the invoiced path (CreditThenInvoice/InvoiceOnly) starts as `Active` and is driven by invoice lifecycle hooks
- the credit-only path starts as `Created` and is driven by the state machine — do not mix the two
- `AmountAfterProration` lives on `flatfee.State`, not `flatfee.Intent` — it is computed at creation via `Intent.CalculateAmountAfterProration()` and persisted on the base charge row. Callers must not provide it; they set `AmountBeforeProration`, `ServicePeriod`, `FullServicePeriod`, and `ProRating` on the Intent
- `IntentWithInitialStatus` carries `AmountAfterProration` alongside `InitialStatus` to pass the computed value from the service to the adapter at creation time
- `flatfee.State.AdvanceAfter` must be passed through `chargemeta.UpdateInput.AdvanceAfter` on every `UpdateCharge(...)` call
- `flatfee.Adapter.UpdateCharge(...)` takes `flatfee.ChargeBase` and persists only base-row fields; do not call it just because `flatfee.Realizations` changed
- `CreatePayment(...)`, `UpdatePayment(...)`, `CreateInvoicedUsage(...)`, and `CreateCreditAllocations(...)` already persist the realization-side rows; a follow-up `UpdateCharge(...)` is redundant unless base-row fields changed too
- `flatfee.Charge.Realizations` is expand-only data loaded from child tables; tests and service code should read payment/accrued-usage/credit-allocation state there, not from `flatfee.State`
- `charge_flat_fees.status_detailed` mirrors `status` today; schema changes or migrations that introduce new flat-fee statuses must keep both columns consistent through `ToMetaChargeStatus()`
- the `flatfee.Handler` interface has both invoiced-path methods and credits-only methods — implementors must satisfy all of them
- adding new `Handler` methods requires updating: `ledger/chargeadapter/flatfee.go`, `charges/service/handlers_test.go`
- the same applies to `usagebased.Handler` — new methods must be added to `UnimplementedHandler`, the ledger adapter (`ledger/chargeadapter/usagebased.go`), and the test handler
- the same applies to `creditpurchase.Handler` — new methods must be added to `ledger/chargeadapter/creditpurchase.go` and the test handler

Usage-based handler interface (`usagebased.Handler`):
- `OnCreditsOnlyUsageAccrued(ctx, CreditsOnlyUsageAccruedInput)` → `creditrealization.CreateAllocationInputs` — allocate credits for a realization run
- `OnCreditsOnlyUsageAccruedCorrection(ctx, CreditsOnlyUsageAccruedCorrectionInput)` → `creditrealization.CreateCorrectionInputs` — correct (partially revert) existing credit allocations when finalization discovers usage decreased

Credit purchase handler interface (`creditpurchase.Handler`):
- `OnPromotionalCreditPurchase(ctx, Charge)` → `ledgertransaction.GroupReference`
- `OnCreditPurchaseInitiated(ctx, Charge)` → `ledgertransaction.GroupReference`
- `OnCreditPurchasePaymentAuthorized(ctx, Charge)` → `ledgertransaction.GroupReference`
- `OnCreditPurchasePaymentSettled(ctx, Charge)` → `ledgertransaction.GroupReference`

`flatfee/service/service.go` Config requires a `*lockr.Locker` — when constructing in tests, create the locker before the flat fee service

When changing credit purchase charges:

- `creditpurchase.ChargeBase` stores base-row data: `ManagedResource`, `Intent`, `Status` (own `creditpurchase.Status` type); `State` exists but is an empty struct
- `creditpurchase.Charge` embeds `ChargeBase` + `Realizations` — all lifecycle outcomes live in `Realizations`, not `State`
- `creditpurchase.Realizations` holds `CreditGrantRealization`, `ExternalPaymentSettlement`, and `InvoiceSettlement` (all loaded from edge tables)
- `CreditGrantRealization` is stored in its own `charge_credit_purchase_credit_grants` table, not on the base row
- `creditpurchase.Status` mirrors the flatfee pattern: `StatusCreated`, `StatusActive`, `StatusFinal`, `StatusDeleted` with `ToMetaChargeStatus()` bridge
- `charge_credit_purchases.status_detailed` column mirrors `status` and is set via `SetStatusDetailed(...)` on create/update
- `UpdateCharge(ctx, ChargeBase) (ChargeBase, error)` only updates base-row fields — do not call it just because realization edges changed
- Realization edges are created/updated through dedicated adapter methods: `CreateCreditGrant`, `CreateExternalPayment`, `UpdateExternalPayment`, `CreateInvoicedPayment`, `UpdateInvoicedPayment`
- Adapter interface is composite: `ChargeAdapter` + `CreditGrantAdapter` + `ExternalPaymentAdapter` + `InvoicedPaymentAdapter`
- The `withExpands` helper in `creditpurchase/adapter/charge.go` adds `.WithCreditGrant().WithExternalPayment().WithInvoicedPayment()` to queries when `ExpandRealizations` is requested — use this helper instead of repeating the expand chain
- Service methods that only change edge data (e.g., `HandleExternalPaymentAuthorized`) update `charge.Realizations` in memory and return the full `Charge` without calling `UpdateCharge`
- Service methods that change status (e.g., `HandleExternalPaymentSettled`, `onPromotionalCreditPurchase`) call `UpdateCharge(ctx, charge.ChargeBase)` and merge the result back: `charge.ChargeBase = updatedBase`
- Credit grant creation must go through `adapter.CreateCreditGrant(...)` — do not write credit grant data through `UpdateCharge`

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
