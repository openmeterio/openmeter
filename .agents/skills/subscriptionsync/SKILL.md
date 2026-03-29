---
name: subscriptionsync
description: Work with the subscription sync bridge in `openmeter/billing/worker/subscriptionsync/...`. Use when modifying how subscription target state is reconciled into billing artifacts such as invoice lines, split-line groups, or charges; when changing persisted-state loading, reconciler patch routing, or subscription sync tests; and when reasoning about the bridge between subscription views and billing state.
---

# Subscription Sync

Guidance for working with `openmeter/billing/worker/subscriptionsync/`.

## What This Package Is

`subscriptionsync` is the bridge between the subscription domain and billing.

- Input: `subscription.SubscriptionView`
- Output: reconciled billing state
  - invoice lines / split-line groups
  - charges
  - persisted sync state

It does not own subscription editing rules and it does not own billing primitives. It translates subscription target state into billing-side operations.

## Package Layout

```
openmeter/billing/worker/subscriptionsync/
├── service/                         # orchestration entrypoint used by the worker
│   ├── service.go                   # Service struct, Config, FeatureFlags, constructor
│   ├── sync.go                      # SynchronizeSubscription + SynchronizeSubscriptionAndInvoiceCustomer
│   ├── reconcile.go                 # build persisted snapshot + target state + plan
│   ├── handlers.go                  # event handlers (HandleCancelledEvent, HandleInvoiceCreation)
│   ├── base_test.go                 # shared SuiteBase + SyncSuiteBase test harness
│   ├── sync_test.go                 # invoice-sync scenarios
│   ├── creditsonly_test.go          # credit-only charge scenarios
│   ├── syncbillinganchor_test.go    # billing anchor / alignment scenarios
│   ├── persistedstate/              # package-owned persisted snapshot abstractions
│   ├── targetstate/                 # expected billing/charge target generation
│   └── reconciler/                  # plan + apply layer
│       ├── reconciler.go            # Reconciler interface, Plan/Apply, diffItem, filterInScopeLinesForInvoiceSync
│       ├── patch.go                 # Patch interfaces, PatchCollection, patchCollectionRouter
│       ├── patchinvoice.go          # invoicePatchCollectionBase (shared invoice patch helpers)
│       ├── patchinvoiceline.go      # lineInvoicePatchCollection
│       ├── patchinvoicelinehierarchy.go # lineHierarchyPatchCollection
│       ├── patchcharge.go           # chargePatchCollection base + newChargeIntentBaseFromTargetState
│       ├── patchchargeflatfee.go    # flatFeeChargeCollection
│       ├── patchchargeusagebased.go # usageBasedChargeCollection
│       ├── patchhelpers.go          # shared patch utilities
│       ├── prorate.go               # semanticProrateDecision
│       ├── invoiceupdater/          # invoice line/group CRUD
│       └── chargeupdater/           # charge creation (or disabled no-op)
├── reconciler/                      # periodic reconciliation (batch re-sync of subscriptions)
├── adapter/                         # sync-state persistence (Ent-backed)
└── service.go                       # top-level interface/config (subscriptionsync.Service, subscriptionsync.Adapter)
```

## Core Flow

`Service.SynchronizeSubscription(...)` in `service/sync.go` is the main entrypoint.

High-level flow:
1. Load persisted sync state and persisted billing artifacts.
2. Build target state from the subscription view.
3. Reconcile target vs persisted.
4. Apply billing patches.
5. Persist sync state.

The important bridge boundaries are:
- `persistedstate`: current billing-side reality relevant to this subscription
- `targetstate`: expected billing-side reality derived from the subscription view
- `reconciler`: diff between the two, expressed as backend-specific patches

## Persisted State

`service/persistedstate` owns the billing-side read model used by sync.

Important rules:
- Do not leak raw `billing.LineOrHierarchy` through the rest of subscription sync.
- Use `persistedstate.Item` and the `ItemAs...` helpers instead.
- `Item.Type()` is package-owned and distinguishes:
  - `invoice.line`
  - `invoice.splitLineGroup`
  - `charge.flatFee`
  - `charge.usageBased`
- `State` contains:
  - `ByUniqueID map[string]Item`
  - `Invoices`

Charge loading notes:
- charges are optional and loaded only when `ChargesService` is configured
- persisted charges are merged into `State.ByUniqueID`; downstream sync code should not depend on a separate charge-only map
- persisted charge unique IDs must not overlap persisted invoice unique IDs
- credit-purchase charges tied to subscriptions are currently unsupported and should error

Invoice loading notes:
- only invoices referenced by the loaded persisted entities are fetched
- missing referenced invoices are treated as errors
- `Invoices.IsGatheringInvoice(...)` returns an error for unknown IDs

## Target State

`service/targetstate` converts a subscription view into expected billing/charge items.

Useful points:
- `StateItem.IsBillable()` is the first gate
- `StateItem.GetServicePeriod()` is the diff-level period source
- `StateItem.GetExpectedLine()` is invoice-specific rendering; keep direct billing assumptions isolated to places that really need invoice lines

For direct billing sync, target items that are not billable or do not render to an expected line are filtered before invoice diffing.

## Semantic Prorate Decision

`reconciler/prorate.go` contains `semanticProrateDecision(existing, target)`. For flat fee lines, it compares the existing per-unit amount and service period against the target. If either differs, it returns `ShouldProrate: true` with original/target amounts so the patch can update period and amount atomically. Non-flat-fee items always return `ShouldProrate: false` and fall through to the normal shrink/extend path.

The service-level `FeatureFlags` (`EnableFlatFeeInAdvanceProrating`, `EnableFlatFeeInArrearsProrating`) gate whether proration is applied during target state generation.

## Reconciler

`service/reconciler` is intentionally split into:
- semantic diffing
- patch collection routing
- backend-specific apply

Current shape:
- invoice patches and charge patches are separate
- routing is based on persisted item type for existing entities
- default routing for new target items uses subscription settlement mode and rate-card type
- apply order is intentionally invoice-first, charge-second during the backend transition; this is not atomic across backends, and partial apply is acceptable because the invoice backend is being deprecated in favor of charges

Important routing rules:
- `GetCollectionFor(persistedItem)` routes by persisted item type (invoice line, split-line group, flat fee charge, usage-based charge)
- `ResolveDefaultCollection(targetItem)` routes new items (no persisted counterpart) by subscription settlement mode + price type:
  - `credit_only` + flat price -> `flatFeeChargeCollection`
  - `credit_only` + unit price -> `usageBasedChargeCollection`
  - everything else -> `lineCollection` (invoice lines)

The `filterInScopeLines` function gates which target items enter reconciliation. It filters out non-billable items for every backend, and only invoicing-backed targets are additionally gated on `GetExpectedLine()`. This runs before any diffing so absent targets naturally produce delete/no-op outcomes.

Current charge limitations:
- charge collections only support create
- delete/shrink/extend/prorate return unsupported errors

This is intentional. If a test expects credit-only cancellation to fail on delete, that is current behavior, not a bug in the test.

## Invoice vs Charge Semantics

Keep these separate:

- Invoice sync:
  - may need `GetExpectedLine()`
  - reasons about gathering vs standard invoices
  - updates invoice lines and split-line groups

- Charge sync:
  - provisions charge intents directly
  - does not go through invoice-line rendering
  - currently only create is supported
  - invoice-style semantic proration is skipped in `diffItem(...)`; charges own their own proration logic

Do not force charge behavior through invoice abstractions.

## Split-Line Groups

Split-line hierarchies are invoice-only persisted items.

Important detail:
- annotation semantics are taken from the last relevant child line
- hierarchy shrink logic must delete an emptied usage-based child instead of updating it to an empty non-billable window

If you see regressions around progressive billing cancellation, inspect the hierarchy shrink path first.

## Testing Guidance

Main service tests live in:
- `service/sync_test.go` for invoice-oriented scenarios
- `service/creditsonly_test.go` for charge-oriented `credit_only` scenarios
- `service/syncbillinganchor_test.go` for billing anchor / alignment scenarios
- `service/base_test.go` for shared setup and helpers

Test suite hierarchy:
- `SuiteBase` — base struct embedding `billingtest.BaseSuite` + `billingtest.SubscriptionMixin`. Handles service construction, namespace/customer/feature provisioning, and teardown.
- `SyncSuiteBase` — extends `SuiteBase` with sync-specific helpers: `gatheringInvoice(...)`, `createSubscriptionFromPlanAt(...)`, `expectLines(...)`, line matchers (`recurringLineMatcher`, `oneTimeLineMatcher`).

Use `setupChargesService(config)` on `SuiteBase` to rebuild the sync service with a charge-capable stack (replaces the default no-charges service).

For charge-backed sync tests:
- prefer `openmeter/billing/charges/testutils.NewMockHandlers()`
- these mocks are intentionally minimal but valid enough for charge creation/advancement
- do not query charge tables directly from sync tests; use `Charges.ListCharges(...)` with `SubscriptionIDs` and `ChargeTypes` filters to assert the end state through the public charges stack
- when asserting charge subscription phase IDs, derive the expected phase from the child unique reference ID and the loaded subscription view instead of hardcoding phase IDs in scenario data

Pattern for credit-only tests:
1. construct plan with `SettlementMode: productcatalog.CreditOnlySettlementMode`
2. create subscription through the plan workflow
3. sync to a future horizon
4. assert charges by unique reference IDs and exact periods

## Event Handlers

`service/handlers.go` contains two event-driven entrypoints:

- `HandleCancelledEvent`: triggered on subscription cancellation. Syncs up to the subscription's `ActiveTo` time. Skips pre-sync invoice creation to avoid creating invoices that would immediately change.
- `HandleInvoiceCreation`: triggered when a standard invoice is created. Finds affected subscriptions from the invoice lines and re-syncs each to backfill the gathering invoice.

## Periodic Reconciler

`reconciler/reconciler.go` (the top-level `reconciler` package, not `service/reconciler`) is the batch reconciliation component. It periodically re-syncs subscriptions to catch missed events.

Key methods:
- `ListSubscriptions(...)` — pages through active subscriptions with their sync states
- `ReconcileSubscription(...)` — fetches subscription view and calls `SynchronizeSubscription`
- `All(...)` — reconciles all eligible subscriptions, skipping those with no billables or whose `NextSyncAfter` is in the future (unless `Force` is set)

## Common Refactor Rules

- Keep `persistedstate` package-owned. It is the anti-corruption boundary.
- Keep `diffItem(...)` driven by semantic periods and target items, not by rendered invoice-line details unless required.
- Prefer adding narrow helpers over passing raw billing union types around.
- When adding new billing backends, route them through the reconciler collections instead of branching ad hoc in `sync.go`.

## Verification

When changing this package, the usual verification commands are:

```bash
nix develop --impure .#ci -c go vet ./...
nix develop --impure .#ci -c make lint-go
nix develop --impure .#ci -c env POSTGRES_HOST=127.0.0.1 go test -count=1 -tags dynamic ./openmeter/billing/worker/subscriptionsync/...
nix develop --impure .#ci -c env POSTGRES_HOST=127.0.0.1 go test -count=1 -tags dynamic ./test/billing
```

If the change touches charges provisioning behavior, also verify the relevant `openmeter/billing/charges/...` packages or suites.
