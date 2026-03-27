---
name: subscription
description: Work with the OpenMeter subscription package. Use when modifying subscription creation, editing, cancellation, plan changes, addons, the sync algorithm, patch system, spec model, workflow layer, or subscription-related tests. Trigger this skill whenever the task touches `openmeter/subscription/...`, subscription views, billing cadences, or the relationship between subscriptions, plans, entitlements, and addons.
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Subscription

Guidance for working with the OpenMeter subscription package (`openmeter/subscription/`).

## Package Layout

```
openmeter/subscription/
├── (root)              — Domain types, interfaces, patch system, spec model
├── service/            — Service implementation + sync algorithm
├── repo/               — Ent ORM-backed repositories
├── entitlement/        — EntitlementAdapter impl (bridges to entitlement service)
├── patch/              — Concrete Patch implementations (add/remove item/phase, stretch, unschedule)
├── workflow/           — High-level orchestration (CreateFromPlan, EditRunning, ChangeToPlan, Restore)
├── workflow/service/   — WorkflowService impl split into subscription.go + addon.go
├── addon/              — Subscription addon sub-system
├── addon/diff/         — Addon apply/restore to spec
├── addon/repo/         — Ent-backed addon repos
├── addon/service/      — Addon service impl
├── validators/         — Customer and subscription uniqueness validators (registered as Hooks)
├── hooks/annotations/  — AnnotationCleanupHook (maintains previous/superseding chain on delete)
└── testutils/          — Full integration test wiring, builders, mocks, comparison helpers
```

The public HTTP entry points live in `openmeter/productcatalog/subscription/http/`. The `PlanSubscriptionService` in `openmeter/productcatalog/subscription/service/` wraps the workflow service to add plan resolution, `StartingPhase`, `Migrate`, and `Change`.

DI wiring: `app/common/subscription.go` → `NewSubscriptionServices` assembles all repos, adapters, services, hooks, and returns `SubscriptionServiceWithWorkflow`.

## Core Domain Model

**Entity hierarchy:**

```
Subscription
└── SubscriptionPhase   (ordered by ActiveFrom; ActiveTo = next phase's ActiveFrom)
    └── SubscriptionItem[]  (grouped by Key; each key holds a version history as a slice)
        └── Entitlement     (optional, 1:1, when item has entitlement-bearing RateCard)
```

**`SubscriptionSpec`** is the "desired state" object. It is the primary thing patches and the sync engine operate on. Key fields:
- `Phases map[string]*SubscriptionPhaseSpec` — pointer values so patches mutate in-place
- `CreateSubscriptionPlanInput` — plan ref, billing cadence, pro-rating config
- `CreateSubscriptionCustomerInput` — customer, currency, timing, billing anchor

**`SubscriptionPhaseSpec`** — contains `StartAfter` (ISO duration from subscription start), and `ItemsByKey map[string][]*SubscriptionItemSpec`. Each key maps to an ordered slice of item versions.

**`SubscriptionItemSpec`** — contains `RateCard` and optional relative timing overrides (`ActiveFromOverrideRelativeToPhaseStart`, `ActiveToOverrideRelativeToPhaseStart` — stored as ISO durations, not timestamps).

**`SubscriptionView`** — the fully-hydrated read model. Built by `NewSubscriptionView()`. Contains `Subscription`, `Customer`, `Spec`, and `[]SubscriptionPhaseView` (each with items, entitlements, and feature data loaded in).

**Phase timing is implicit:** `SubscriptionPhase` stores only `ActiveFrom`. `ActiveTo` is derived as the next phase's `ActiveFrom` (or the subscription's `ActiveTo`). Calling `GetPhaseCadence()` must iterate all sorted phases to find the successor.

**Item versioning within a phase:** For a given `ItemKey`, `SubscriptionPhaseSpec.ItemsByKey[key]` is a slice. Index 0 is the first version, index 1 is a replacement, etc. The sync algorithm matches items by slice index. When `PatchAddItem` targets the current phase and a matching key already exists, it automatically sets the existing item's `ActiveTo` to the new item's `ActiveFrom`, creating a version history.

## Service Interface

```go
type Service interface {
    QueryService   // Get, GetView, List, ExpandViews
    CommandService // Create, Update, Delete, Cancel, Continue, UpdateAnnotations, RegisterHook
}
```

Every mutating operation:
1. Acquires a per-customer distributed lock: `customer/{id}/subscription` via `lockr.Locker`.
2. Runs inside a transaction via `transaction.Run()`.
3. Invokes Before*/After* hooks.
4. Publishes a domain event via `eventbus.Publisher`.

**`ExpandViews`** bulk-loads phases, items, entitlements, and features for multiple subscriptions — but only supports a single customer ID per call.

## Sync Algorithm (Core Engine)

`sync()` in `service/sync.go` reconciles the current `SubscriptionView` against a new `SubscriptionSpec`.

Three passes:

1. **Mark dirty** — for each existing phase/item, look up the corresponding entry in the new spec by key (and index for items). If missing or entity inputs differ → mark as dirty (delete it). Dirtiness propagates downward via `touched map[SpecPath]bool` with parent-path inheritance: if `/phases/trial` is dirty, all its items are also dirty.

2. **Re-create dirty resources that still exist in new spec** — dirty phases/items that appear in the new spec are re-created.

3. **Create net-new resources** — phases/items in the new spec not present in the current view are created.

**Comparison is by value:** `CreateSubscriptionItemEntityInput.Equal()` uses `reflect.DeepEqual` but intentionally ignores `Metadata` and `Annotations`. Any difference in entity inputs — including a change to `ActiveToOverrideRelativeToPhaseStart` — causes a delete+recreate cycle for that item row and its linked entitlement.

**What happens when `PatchAddItem` replaces an existing item in the current phase:** The patch closes item[0] (sets its `ActiveTo`) and appends item[1]. In sync, Pass 1 detects that item[0]'s `ActiveTo` changed → soft-deletes item[0] and its entitlement. Pass 2 recreates item[0] with the closed window (and a new entitlement bounded to that window). Pass 3 creates item[1] as net-new (with its own entitlement starting from `editTime`). The phase entity is untouched.

**Entitlement comparison caveat (FIXME in code):** During comparison, the current entitlement is used for both old and new sides. This works because entitlement-relevant properties are captured in the item's `RateCard`, but it means the entitlement ID itself is never compared.

**`createItem` / `deleteItem`** (in `service/synchelpers.go`): item creation calls `EntitlementAdapter.ScheduleEntitlement` when the RateCard requires an entitlement. Item deletion calls `EntitlementAdapter.DeleteByItemID`. Phase deletion cascades to all items in the phase.

## Patch System

**`Patch` interface:**
```go
type Patch interface {
    AppliesToSpec  // ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error
    Validate() error
    Op() PatchOperation
    Path() SpecPath
}
```

**`SpecPath`** — a string path like `/phases/trial/items/seat/idx/0`. Valid depths:
- 2: phase — `/phases/{key}` — use for patches that target a whole phase (`PatchAddPhase`, `PatchRemovePhase`, `PatchStretchPhase`)
- 4: item — `/phases/{key}/items/{key}` — use for patches that target an item regardless of version (`PatchAddItem`, `PatchRemoveItem`, and any in-place mutation patch)
- 6: item version — `/phases/{key}/items/{key}/idx/{N}` — use only when a patch must address a specific version by index

**Do not use depth-6 for a patch that mutates in-place** (e.g., replacing a rate card without creating a new version). Depth-4 is correct for that — the patch logic selects the relevant version internally.

**`ApplyContext`** — carries `CurrentTime`. Patches use it to reject operations on past phases/items.

**Concrete patches** (`subscription/patch/`):

| Patch | What it does |
|---|---|
| `PatchAddItem` | Adds item to a phase; auto-closes previous version of same key in current phase |
| `PatchRemoveItem` | Removes item from a phase |
| `PatchAddPhase` | Adds a new future phase |
| `PatchRemovePhase` | Removes a future phase; `Shift: next|prev` moves subsequent phase start times |
| `PatchStretchPhase` | Extends/shrinks a phase by shifting all subsequent `StartAfter` values |
| `PatchUnscheduleEdit` | Removes a scheduled future edit |

Patches are applied via `SubscriptionSpec.ApplyMany(patches, actx)`. Addons also implement `AppliesToSpec` — they go through the same `ApplyTo` mechanism.

## Workflow Layer

`workflow.Service` (`openmeter/subscription/workflow/`) provides higher-level operations:

- **`CreateFromPlan`** — resolves timing, builds `SubscriptionSpec` from a `Plan`, calls `Service.Create`.
- **`EditRunning`** — validates no addons are on the subscription (edit is blocked if any exist), applies patches to current spec, calls `Service.Update`. Sets `OwnerSubscriptionSubSystem` and `UniquePatchID` annotations on added items.
- **`ChangeToPlan`** — cancels the current subscription, creates a new one with the new plan. Passes billing anchor through. Stores `previous`/`superseding` subscription IDs in annotations.
- **`Restore`** — (single-subscription mode only) deletes all future-scheduled subscriptions, calls `Continue` on the current one. Forbidden when `multi-subscription-enabled` feature flag is on.
- **`AddAddon`** / **`ChangeAddonQuantity`** — delegates to `AddonService`.

**Important:** `EditRunning` is blocked if the subscription has any active addons. Addon-bearing subscriptions must be modified through the addon workflow.

## Addon Sub-system

**`SubscriptionAddon`** — an addon applied to a subscription, with a `timeutil.Timeline[SubscriptionAddonQuantity]` for quantity changes over time.

**`addon/diff`** — applies addon rate cards onto `SubscriptionSpec`:
- For each addon instance, the diff algorithm overlays rate cards on existing items or inserts new items.
- `GetRestores()` produces patches to undo addon effects.

**Quirk:** When an addon creates a new item (not a split of an existing one), feature key resolution happens at sync time. Multiple items with the same feature key may point to different feature instances.

## Hook System

**`SubscriptionCommandHook`** interface — `Before*` and `After*` callbacks:
- `BeforeCreate/BeforeContinue/BeforeUpdate/BeforeDelete`
- `AfterCreate/AfterUpdate/AfterCancel/AfterContinue`

Embed `NoOpSubscriptionCommandHook` to get default no-ops.

**Registered hooks (automatically wired in `app/common/subscription.go`):**

1. **`SubscriptionUniqueConstraintValidator`** — prevents overlapping subscriptions for the same customer; also prevents two items with the same feature key (entitlement or billable) from overlapping in time. Full overlap check is gated by `multi-subscription-enabled` feature flag in `BeforeUpdate`.
2. **`AnnotationCleanupHook`** — on `BeforeDelete`, updates `previous`/`superseding` annotation pointers in linked subscriptions to keep the chain intact when a middle subscription is deleted.

Register additional hooks via `Service.RegisterHook(hook)`.

## Annotations

Key annotation keys (`subscription/annotations.go`):

| Key | Meaning |
|---|---|
| `subscription.id` | Which subscription created an entitlement |
| `subscription.owner` | Owner subsystem (`"subscription"` or `"addon"`) |
| `subscription.previous.id` | The subscription this one replaced (set by ChangeToPlan) |
| `subscription.superseding.id` | The subscription that replaced this one |
| `subscription.entitlement.boolean.count` | Count of boolean entitlements |

Use `subscription.AnnotationParser` for typed access. `workflow.UniquePatchID` (in `workflow/annotations.go`) generates deterministic patch IDs to prevent duplicate item creation.

## Timing System

**`Timing`** struct — either `Custom *time.Time` or `Enum *TimingEnum`:
- `TimingImmediate` → `clock.Now()`
- `TimingNextBillingCycle` → end of current aligned billing period (requires `BillingCadence`)

**Validation rules (`timing.go`):**
- Update: must be in the future; cannot time-travel to a different phase
- Cancel: custom time must align with billing cadence if the subscription is aligned
- Create: custom time must not be more than 2 minutes in the past

**Aligned billing periods:** `SubscriptionSpec.GetAlignedBillingPeriodAt(at)` computes the billing period for a given time, accounting for phase boundaries and the billing anchor.

## State Machine

Uses the `stateless` library. States: `Inactive`, `Active`, `Canceled`, `Scheduled`.

```
Inactive  ──create──> Active
Active    ──cancel──> Canceled
Active    ──update──> Active  (reentry)
Canceled  ──continue──> Active
Scheduled ──delete──> (removed)
```

Each service method calls `CanTransitionOrErr` before proceeding.

## Billing Connection

`billing/charges/meta.SubscriptionReference` carries `{SubscriptionID, PhaseID, ItemID}`. The billing worker's subscription sync service (`billing/worker/subscriptionsync/`) reads subscription views and reconciles invoice lines via `billing.Service.GetLinesForSubscription`.

## Plan Bridge

`openmeter/productcatalog/subscription/` wraps the workflow service:
- `PlanSubscriptionService.Create` — plan resolution + `StartingPhase` support (zeroes out earlier phases)
- `Migrate` — switches to a different version of the same plan
- `Change` — switches to a different plan entirely
- HTTP handlers: `openmeter/productcatalog/subscription/http/`

The subscription package does NOT import the plan package directly — it uses the `Plan`, `PlanPhase`, `PlanRateCard` interfaces defined in `subscription/plan.go`.

## Multi-Subscription Feature Flag

`MultiSubscriptionEnabledFF = ffx.Feature("multi-subscription-enabled")` gates:
- Full overlap validation in `BeforeUpdate` (uniqueness check is relaxed when disabled)
- `Restore` is forbidden when enabled
- Default in tests: `false`

Set it in tests via `ffx.NewTestContextService(ffx.AccessConfig{...})`.

## Context Marker

`subscription.NewSubscriptionOperationContext(ctx)` marks the context as a subscription operation. `subscription.IsSubscriptionOperation(ctx)` checks it. Used to prevent recursive hook invocations when the customer validator internally queries the subscription service.

## Customer Currency Auto-Set

If a new subscription has billable items and the customer has no currency set, `Service.Create` automatically updates the customer's currency to the subscription's currency.

## Non-Obvious Pitfalls

- **Spec phases are pointer-valued**: `Phases map[string]*SubscriptionPhaseSpec` — patches mutate the pointer contents directly. Never replace the pointer with a new value in a patch; modify the pointed-to struct.
- **`ExpandViews` is single-customer only**: current implementation does not support multiple customers per call.
- **Phase timing is not stored on the phase entity**: `ActiveTo` must be derived by sorting all phases and finding the successor. Never assume you can read `ActiveTo` from a single phase row.
- **Item index = version history**: `ItemsByKey[key][0]` is the original version, `[1]` is a replacement, etc. This is not a multi-quantity concept — it is a time-ordered edit history.
- **Sync deletes and recreates changed items**: any diff in entity inputs causes a full delete+recreate cycle for that item, including re-scheduling the linked entitlement.
- **EditRunning is blocked by addons**: any attempt to call `EditRunning` on a subscription that has addons returns an error. Use the addon workflow instead.
- **`BillingAnchor` defaults to `activeFrom`**: normalized to UTC. Used for all aligned billing period calculations.

## Testing Guidance

- **`testutils/service.go` → `NewService()`** provides a full integration test wiring with a real Postgres DB (schema via migrations). Prefer this over mocks for any service-layer tests.
- **`testutils/builder.go`** — `SubscriptionSpecBuilder`, `PhaseBuilder`, `ItemBuilder` for constructing specs declaratively in tests.
- **`testutils/compare.go`** — deep comparison helpers that account for non-deterministic ordering in views.
- **`testutils/mock.go`** — mock implementations for adapters (useful for unit tests of patches and spec logic).
- Feature flag tests: use `ffx.NewTestContextService`.
- Unit tests for patches live in `subscription/patch/*_test.go`.
- Integration tests for the service live in `subscription/service/service_test.go` and `subscription/service/sync_test.go`.
- Addon service tests: `subscription/addon/service/*_test.go`.

**Addon test rate cards and entitlements (`testutils/addon.go`):**

| Constant | EntitlementTemplate | Notes |
|---|---|---|
| `ExampleAddonRateCard1` | none | Has feature key; `itemView.Entitlement` will be nil |
| `ExampleAddonRateCard2` | none | No feature key |
| `ExampleAddonRateCard3` | `BooleanEntitlementTemplate` | Use this when testing entitlement cadences |
| `ExampleAddonRateCard4` | none | Flat price on ExampleFeatureKey2 |

Use `ExampleAddonRateCard3` (or build a custom rate card with an `EntitlementTemplate`) when the test needs to assert `itemView.Entitlement != nil`.

**Identifying addon items vs plan items in a view:**

```go
// Plan-sourced items have OwnerSubscriptionSubSystem; addon items do not.
ownerSystems := subscription.AnnotationParser.ListOwnerSubSystems(item.Annotations)
isAddonItem := !lo.Contains(ownerSystems, subscription.OwnerSubscriptionSubSystem)
```

**Addon integration test wiring** (in `addon/service/` tests):

```go
dbDeps := subscriptiontestutils.SetupDBDeps(t)
defer dbDeps.Cleanup(t)
deps := subscriptiontestutils.NewService(t, dbDeps)
// or use the withDeps helper already defined in create_test.go
```

The `createPlanWithAddon` helper (available in addon service tests) creates features, plan, addon, links via `PlanAddon`, and publishes everything in one call.

## Running Tests

```bash
# All subscription tests
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/subscription/...

# Service integration tests only
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/subscription/service/...

# Addon service tests
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/subscription/addon/service/...

# Patch unit tests (no Postgres needed)
go test -v ./openmeter/subscription/patch/...
```

## Editing Checklist

When modifying subscription service behavior:
- Check whether the change belongs in: domain type, service, workflow, or plan bridge layer
- If touching mutating operations: ensure the per-customer lock is held and the operation runs in a transaction
- If adding new Before*/After* hook callbacks: add a no-op default to `NoOpSubscriptionCommandHook`
- If changing spec comparison: remember `Equal()` ignores `Metadata` and `Annotations` by design

When modifying the sync algorithm:
- Preserve the three-pass structure (mark dirty → recreate dirty → create new)
- Keep `touched` parent-path inheritance working correctly when adding new SpecPath depths
- The entitlement comparison FIXME is intentional — the entitlement ID is not compared

When modifying the patch system:
- New patches must implement `Patch` (which embeds `AppliesToSpec`) and validate against `ApplyContext.CurrentTime`
- Register new patches in any patch-routing code that switches on `PatchOperation`
- Test both valid and invalid (past-phase, missing-key) scenarios

When modifying addons:
- Keep `EditRunning` blocked when addons are present
- The diff/restore roundtrip must be symmetric: `Apply` followed by `GetRestores` must produce a spec equivalent to the original
- Feature key resolution happens at sync time for addon-created items — do not assume it happens at diff time
