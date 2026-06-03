# service

<!-- archie:ai-start -->

> Orchestration layer implementing entitlement.Service by composing the three sub-type connectors (metered, boolean, static), the feature connector, customer service, and a distributed locker — routes each operation to the correct sub-type.

## Patterns

**ServiceConfig constructor** — NewEntitlementService takes a ServiceConfig struct with all deps explicit (Wire-built). No validation or logic in the constructor. (`service.NewEntitlementService(service.ServiceConfig{MeteredEntitlementConnector: meteredConn, EntitlementRepo: entRepo, Locker: locker})`)
**transaction.Run for composite mutations** — CreateEntitlement and OverrideEntitlement wrap ScheduleEntitlement plus the grant-issuance loop in transaction.Run(ctx, entitlementRepo, ...) for cross-connector atomicity. (`transaction.Run(ctx, c.entitlementRepo, func(ctx) (*entitlement.Entitlement, error) { ent, err := c.ScheduleEntitlement(ctx, input); ... })`)
**ServiceHookRegistry fan-out** — service embeds models.ServiceHookRegistry[entitlement.Entitlement] and exposes RegisterHooks; balance-worker registers hooks. Call hooks.PreUpdate before mutations. (`s.hooks.RegisterHooks(hooks...)`)
**registrybuilder for test setup** — Tests obtain a fully wired registry via registrybuilder.GetEntitlementRegistry(...) instead of importing app/common, avoiding cycles. (`registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{DatabaseClient: dbClient, StreamingConnector: streamingConnector})`)
**Sub-type dispatch via type switch** — ScheduleEntitlement switches on EntitlementType to call the right connector's BeforeCreate. Handlers must go through entitlement.Service, never connectors directly. (`switch input.EntitlementType { case entitlement.EntitlementTypeMetered: return c.meteredConnector.BeforeCreate(input, feat) }`)
**WithLock after tx opens** — lock.go WithLock acquires pg_advisory_xact_lock per entitlement/customer via lockr.Locker, requiring the active Postgres tx from transaction.Run. (`WithLock(ctx, locker, key, func(ctx) error { ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Concrete entitlement.Service dispatching operations to sub-type connectors. | CreateEntitlement delegates to ScheduleEntitlement then loops grants; if scheduling errors no grants are issued — keep this ordering. |
| `scheduling.go` | ScheduleEntitlement, SupersedeEntitlement, and time-window activation for scheduling future/expiring entitlements. | SupersedeEntitlement deletes the old and creates a new entitlement in the same transaction.Run closure. |
| `lock.go` | WithLock helper acquiring pg_advisory_xact_lock via lockr.Locker before mutation. | Locker requires an active Postgres tx — call after transaction.Run opens one, not before. |

## Anti-Patterns

- Importing app/common in test helpers — use registrybuilder.
- Calling sub-type connector methods directly from HTTP handlers.
- Adding DB queries to service.go — persistence goes through EntitlementRepo.
- Calling lockr.Locker.LockForTX outside an active transaction.Run closure.

## Decisions

- **Service is a thin orchestrator delegating to three independent sub-type connectors.** — Each entitlement type has different storage/computation needs; centralizing dispatch avoids routing logic scattered across HTTP handlers.

<!-- archie:ai-end -->
