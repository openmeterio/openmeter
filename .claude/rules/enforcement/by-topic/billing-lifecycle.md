# Enforcement: billing-lifecycle (3 rules)

Topic file. Loaded on demand when an agent works on something in the `billing-lifecycle` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-fsm-001` — Model invoice and charge lifecycle transitions through the qmuntal/stateless FSM, not ad-hoc status mutation

*source: `deep_scan`*

**Why:** Invoice and usage-based-charge lifecycles have guarded transitions with side effects (calculation, finalization, voiding) that must be auditable and persisted across requests. stateless.NewStateMachineWithExternalStorage stores the current state on the domain aggregate (invoice.status / charge.status), and states are Configure'd with Permit / PermitDynamic(guard) edges and OnActive side-effect callbacks. Mutating status directly or branching with if/switch scatters guards and makes illegal transitions possible; the FSM centralizes the legal edge set as a single source of truth.

### `dec-charge-idempotency-001` — Create charges with a unique_reference_id for idempotency and read across subtypes via the ChargesSearchV1 view

*source: `deep_scan`*

**Why:** Charge is a polymorphic parent row where exactly one of three subtype FK columns (flat_fee / credit_purchase / usage_based) points to the subtype table, with UNIQUE(namespace, unique_reference_id) WHERE unique_reference_id IS NOT NULL AND deleted_at IS NULL for idempotent creation (openmeter/ent/schema/charges.go:167). Reads go through the ChargesSearchV1 Postgres VIEW (UNION ALL of the three subtype tables). Creating a charge without unique_reference_id loses retry-safety; a single wide charges table or separate tables without a parent loses the idempotency key and unified search surface.

## Tradeoff Signals (warn)

### `tr-fsm-002` — Do not mutate invoice.status directly or add a status without a corresponding Permit edge

*source: `deep_scan`*

**Why:** External-storage state machines put the source-of-truth status on the aggregate row; the FSM definition and the persisted status must stay in sync, and every legal transition must be declared as a Permit edge. Directly assigning invoice.status, branching with if/switch on status instead of Permit edges, or using an in-memory NewStateMachine for a persisted aggregate undermines durability and auditability of transitions.
