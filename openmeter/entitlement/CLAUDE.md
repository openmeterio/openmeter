# entitlement

<!-- archie:ai-start -->

> Feature entitlement management across three sub-types (metered, boolean, static) with scheduling, override, and supersede lifecycle operations. The metered sub-type drives credit grant burn-down via the credit engine; the service layer composes all sub-types behind a single entitlement.Service interface.

## Patterns

**Sub-type connector dispatch via Parser** — entitlement.Service dispatches to the correct sub-type connector (metered, boolean, static) using the sub-type discriminator on Entitlement.EntitlementType. The Parser singleton in driver/parser.go is used by HTTP drivers to convert between wire types and domain types. (`entitlement/driver/parser.go: Parser.ToAPIGeneric dispatches on EntitlementType`)
**ServiceHooks[Entitlement] fan-out** — entitlement.Service embeds models.ServiceHooks[Entitlement]. Other packages (subscription hooks) register hooks via Service.RegisterHooks() to react to entitlement lifecycle without creating circular imports. (`entitlement/service/service.go: s.hooks.PostCreate(ctx, entitlement) after creation`)
**transaction.Run for composite mutations** — All operations that touch multiple tables (ResetEntitlementUsage, CreateEntitlement with grants, DeleteEntitlement with cleanup) run inside transaction.Run / transaction.RunWithNoValue to ensure atomicity. (`entitlement/metered/reset.go: transaction.Run wraps EndCurrentUsagePeriod + balance snapshot + event publish`)
**Distributed lock before per-customer operations** — entitlement.Service acquires a pg_advisory_xact_lock via lockr.Locker before operations that modify multiple entitlement rows for the same customer, preventing race conditions in concurrent reset/grant flows. (`entitlement/service/lock.go: s.locker.LockForTX(ctx, customerID)`)
**SnapshotEvent V2 for balance change notifications** — After any balance change (reset, grant, delete) the metered connector publishes entitlement.SnapshotEvent (v2) to the system Kafka topic. New code must emit V3 events from entitlement/metered/events.go; never create raw SnapshotEvent literals. (`entitlement/snapshot/event.go: NewSnapshotEvent constructor with CalculatedAt required`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/entitlement/connector.go` | entitlement.Service interface with all 13 lifecycle methods plus models.ServiceHooks[Entitlement] embed. Also defines ListEntitlementsParams and ListEntitlementsWithCustomerResult. | GetEntitlementOfCustomerAt resolves by ID first, then by FeatureKey+CustomerID — never pass raw subject keys; always resolve to customer.ID first. |
| `openmeter/entitlement/entitlement.go` | Entitlement domain struct (GenericProperties + sub-type fields), CreateEntitlementInputs, and all validation logic including UsagePeriod minimum 1-hour check. | MeasureUsageFromInput is a value wrapper (not a plain time.Time) to distinguish NOW vs CURRENT_PERIOD_START enums. Always use FromTime or FromEnum, never set ts directly. |
| `openmeter/entitlement/repository.go` | EntitlementRepo interface — the persistence boundary for the entitlement domain. | EntitlementRepo is intentionally kept separate from entitlement.Service so validators can depend on it without importing the full service graph. |
| `openmeter/entitlement/errors.go` | Typed domain errors (AlreadyExistsError, NotFoundError, InvalidValueError, etc.). | AlreadyExistsError wraps models.GenericConflictError — HTTP encoder maps it to 409. Use IsAlreadyExistsError helper for type checking. |

## Anti-Patterns

- Calling sub-type connector methods directly from HTTP handlers — always go through entitlement.Service.
- Constructing CreateEntitlementInputs with ActiveTo set but ActiveFrom not set — Validate() returns an error.
- Adding DB queries directly to entitlement.Service struct — all persistence goes through EntitlementRepo.
- Using time.Now() instead of clock.Now() in service or adapter methods — breaks test determinism with frozen clocks.
- Emitting EntitlementResetEvent (v1) from new code — use EntitlementResetEventV3 from entitlement/metered/events.go.

## Decisions

- **Three independent sub-type connectors (metered, boolean, static) composed behind a single entitlement.Service orchestrator.** — Sub-types have radically different value semantics (credit burn-down vs on/off vs JSON config); keeping them separate prevents the service layer from accumulating type switches.
- **Balance worker subscribes to three topics (system, ingest, balance-worker) through a RecalculateEvent intermediary.** — The two-stage approach (lifecycle event → RecalculateEvent → snapshot) enables deduplication and filter-before-recalculate discipline, preventing redundant ClickHouse queries on burst events.

<!-- archie:ai-end -->
