# entitlement

<!-- archie:ai-start -->

> Feature entitlement management across three sub-types (metered, boolean, static) with scheduling, override, and supersede lifecycle operations. The metered sub-type drives credit grant burn-down via the credit engine and ClickHouse usage queries; the service layer composes all sub-types behind a single entitlement.Service interface.

## Patterns

**Sub-type connector dispatch via Parser** — entitlement.Service dispatches to the correct sub-type connector (metered, boolean, static) using the sub-type discriminator on Entitlement.EntitlementType. The Parser singleton in driver/parser.go converts between wire types and domain types. (`// driver/parser.go: Parser.ToAPIGeneric dispatches on EntitlementType to call the correct sub-type serializer`)
**ServiceHooks[Entitlement] fan-out** — entitlement.Service embeds models.ServiceHooks[Entitlement]. Other packages register hooks via Service.RegisterHooks() to react to entitlement lifecycle without creating circular imports. (`// service/service.go: s.hooks.PostCreate(ctx, entitlement) after creation`)
**transaction.Run for composite mutations** — All operations touching multiple tables (ResetEntitlementUsage, CreateEntitlement with grants, DeleteEntitlement with cleanup) run inside transaction.Run / transaction.RunWithNoValue to ensure atomicity. (`// metered/reset.go: transaction.Run wraps EndCurrentUsagePeriod + balance snapshot + event publish`)
**Distributed lock before per-customer operations** — entitlement.Service acquires a pg_advisory_xact_lock via lockr.Locker before operations that modify multiple entitlement rows for the same customer, preventing race conditions in concurrent reset/grant flows. (`// service/lock.go: s.locker.LockForTX(ctx, customerID) — always inside an active transaction`)
**MeasureUsageFromInput wrapper — never set ts directly** — MeasureUsageFromInput is a value wrapper (not a plain time.Time) to distinguish NOW vs CURRENT_PERIOD_START enums. Always use FromTime or FromEnum; never assign the ts field directly. (`mu := &MeasureUsageFromInput{}; mu.FromEnum(MeasureUsageFromCurrentPeriodStart, currPeriod, now)`)
**EntitlementResetEventV3 from new code only** — After any balance change (reset, grant, delete), the metered connector publishes a snapshot event. New code must use EntitlementResetEventV3 from entitlement/metered/events.go; never emit the deprecated v1 variant. (`// metered/events.go: NewEntitlementResetEventV3 constructor — never entitlement.SnapshotEvent literal`)
**ValidateUniqueConstraint before scheduled entitlement creation** — When scheduling or superseding entitlements, ValidateUniqueConstraint(ents) must be called to ensure no two entitlements for the same feature+customer overlap in their active cadence. (`// uniqueness.go: ValidateUniqueConstraint builds a SortedCadenceList and returns UniquenessConstraintError on overlap`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `connector.go` | entitlement.Service interface with all 13 lifecycle methods plus models.ServiceHooks[Entitlement] embed. Also defines ListEntitlementsParams. | GetEntitlementOfCustomerAt resolves by ID first, then by FeatureKey+CustomerID — never pass raw subject keys; always resolve to customer.ID first. |
| `entitlement.go` | Entitlement domain struct (GenericProperties + sub-type fields), CreateEntitlementInputs, and all validation logic including UsagePeriod minimum 1-hour check. | MeasureUsageFromInput is a wrapper type — FromTime / FromEnum are the only valid constructors. ActiveTo before ActiveFrom returns a Validate() error. |
| `repository.go` | EntitlementRepo interface — the persistence boundary. Intentionally separate from entitlement.Service so validators can depend on it without importing the full service graph. | EntitlementRepo embeds entutils.TxCreator; adapters must implement the TxUser+TxCreator triad. |
| `errors.go` | Typed domain errors (AlreadyExistsError, NotFoundError, WrongTypeError, InvalidValueError, ForbiddenError). AlreadyExistsError wraps GenericConflictError (409). | Use typed error constructors — plain fmt.Errorf falls through to 500 in the HTTP encoder. |
| `events.go` | EntitlementCreatedEventV2 and EntitlementDeletedEventV2 CloudEvent types with pinned v2 versions and NewEntitlement*EventPayloadV2 constructors. | mapEntitlementToV2Literal must be used for all new event construction — never build entitlementEventV2EntitlementLiteral with a struct literal. |
| `uniqueness.go` | ValidateUniqueConstraint and UniquenessConstraintError — validates no two entitlements for the same feature+customer overlap in their active cadence. | Relies on models.NewSortedCadenceList; entitlements where ActiveFromTime == ActiveToTime are filtered out before overlap detection. |

## Anti-Patterns

- Calling sub-type connector methods directly from HTTP handlers — always go through entitlement.Service.
- Constructing CreateEntitlementInputs with ActiveTo set but ActiveFrom not set — Validate() returns an error.
- Adding DB queries directly to entitlement.Service struct — all persistence goes through EntitlementRepo.
- Using time.Now() instead of clock.Now() in service or adapter methods — breaks test determinism with frozen clocks.
- Emitting EntitlementResetEvent v1 from new code — use EntitlementResetEventV3 from entitlement/metered/events.go.

## Decisions

- **Three independent sub-type connectors (metered, boolean, static) composed behind a single entitlement.Service orchestrator.** — Sub-types have radically different value semantics (credit burn-down vs on/off vs JSON config); keeping them separate prevents the service layer from accumulating type switches.
- **Balance worker subscribes to three topics (system, ingest, balance-worker) via a RecalculateEvent intermediary on the balance-worker topic.** — The two-stage approach (lifecycle event → RecalculateEvent → snapshot) enables deduplication and filter-before-recalculate discipline, preventing redundant ClickHouse queries on burst events.
- **EntitlementRepo is kept separate from entitlement.Service interface.** — Allows entitlement/validators/customer to depend on the lighter EntitlementRepo without pulling in the full service graph, avoiding circular imports.

## Example: Creating a metered entitlement with initial grant

```
import (
    "github.com/openmeterio/openmeter/openmeter/entitlement"
    "github.com/openmeterio/openmeter/openmeter/credit"
)

mu := &entitlement.MeasureUsageFromInput{}
_ = mu.FromEnum(entitlement.MeasureUsageFromCurrentPeriodStart, currPeriod, clock.Now())

ent, err := svc.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
    Namespace:       ns,
    FeatureKey:      &featureKey,
    EntitlementType: entitlement.EntitlementTypeMetered,
    UsageAttribution: streaming.CustomerUsageAttribution{CustomerID: customerID},
    MeasureUsageFrom: mu,
    IssueAfterReset: lo.ToPtr(100.0),
// ...
```

<!-- archie:ai-end -->
