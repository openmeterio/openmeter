# entitlement

<!-- archie:ai-start -->

> Feature entitlement management across three independent sub-types (metered credit-grant burn-down, boolean on/off, static JSON) composed behind a single entitlement.Service. The metered path drives the credit engine + ClickHouse usage queries; the balanceworker/ sub-package recalculates balances off Kafka lifecycle events.

## Patterns

**Three sub-type connectors composed by a thin service orchestrator** — service/ dispatches each operation to metered/, boolean/, or static via the EntitlementType discriminator. Sub-types have radically different value semantics so the orchestrator stays free of credit logic. The Parser singleton in driver/parser.go converts wire<->domain types and is shared by v1 and v2 drivers and balanceworker. (`service/service.go routes by EntitlementType; driver/parser.go: Parser.ToAPIGeneric dispatches per sub-type`)
**EntitlementRepo kept separate from entitlement.Service** — The persistence boundary (repository.go) is intentionally a lighter interface than Service so validators/customer and hooks can depend on it without pulling the full service graph — avoiding import cycles. Adapters implement the TxUser+TxCreator triad. (`entitlement/validators/customer depends on entitlement.EntitlementRepo, not entitlement.Service`)
**transaction.Run + per-customer advisory lock for composite mutations** — Operations touching multiple rows (ResetEntitlementUsage, CreateEntitlement with grants, DeleteEntitlement) run in transaction.Run; service/lock.go acquires lockr.Locker.LockForTX(ctx, customerID) inside the open transaction before multi-row writes. (`metered/reset.go: transaction.Run wraps EndCurrentUsagePeriod + balance snapshot + event publish`)
**Versioned snapshot/lifecycle events, never struct literals or in-place version bumps** — Metered balance changes publish via NewEntitlementResetEventV3; lifecycle uses EntitlementCreatedEventV2/EntitlementDeletedEventV2; snapshot/ defines SnapshotEvent v2 (system topic). Always use constructors and call Validate(); add a new versioned struct rather than mutating an existing one. (`metered/events.go: NewEntitlementResetEventV3(...) — never an entitlement.SnapshotEvent literal`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | entitlement.Service interface (13 lifecycle methods) embedding models.ServiceHooks[Entitlement]; defines ListEntitlementsParams. | GetEntitlementOfCustomerAt resolves by ID then FeatureKey+CustomerID — resolve subject keys to customer.ID first; never pass raw subject keys to the service. |
| `entitlement.go` | Entitlement domain struct, CreateEntitlementInputs, and validation including the UsagePeriod minimum 1-hour check. | MeasureUsageFromInput is a wrapper type — only FromTime/FromEnum are valid constructors; ActiveTo set without ActiveFrom (or ActiveTo before ActiveFrom) fails Validate(). |
| `repository.go` | EntitlementRepo persistence interface, deliberately separate from Service. | Embeds entutils.TxCreator; adapters must implement the TxUser+TxCreator triad. |
| `uniqueness.go` | ValidateUniqueConstraint / UniquenessConstraintError — no two entitlements for the same feature+customer may overlap in active cadence. | Builds a models.SortedCadenceList; entitlements where ActiveFromTime == ActiveToTime are filtered out before overlap detection. |

## Anti-Patterns

- Calling sub-type connector methods (metered/boolean/static) directly from HTTP handlers — always go through entitlement.Service.
- Adding DB queries to entitlement.Service — all persistence goes through EntitlementRepo.
- Bypassing transaction.Run in ResetEntitlementUsage or calling lockr.LockForTX outside an open transaction — reset and event publish must be atomic and the lock correctly scoped.
- Emitting EntitlementResetEvent v1 from new code, or publishing a snapshot directly from a balanceworker lifecycle handler instead of via RecalculateEvent.
- Using time.Now() instead of clock.Now() — breaks test determinism with frozen clocks.

## Decisions

- **Three independent sub-type connectors composed behind a single Service orchestrator.** — Credit burn-down vs on/off vs JSON config have incompatible value semantics; keeping them separate prevents the service from accumulating type switches.
- **Balance worker subscribes to system + ingest + balance-worker topics via a RecalculateEvent intermediary on the balance-worker topic.** — The two-stage lifecycle->RecalculateEvent->snapshot flow enables dedup and filter-before-recalculate discipline, avoiding redundant ClickHouse queries on burst events.
- **EntitlementRepo is kept separate from entitlement.Service.** — Lets validators depend on the lighter repo interface without pulling in the full service graph, avoiding circular imports.

<!-- archie:ai-end -->
