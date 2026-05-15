# service

<!-- archie:ai-start -->

> Orchestration layer implementing entitlement.Service by composing three sub-type connectors (metered, boolean, static), the feature connector, customer service, and a distributed locker — routes entitlement operations to the correct sub-type.

## Patterns

**ServiceConfig constructor pattern** — NewEntitlementService takes a ServiceConfig struct with all deps explicit. Wire constructs ServiceConfig from individual providers. Never add validation or logic to the constructor itself. (`service.NewEntitlementService(service.ServiceConfig{MeteredEntitlementConnector: meteredConn, EntitlementRepo: entRepo, Locker: locker, ...})`)
**transaction.Run for composite mutations** — CreateEntitlement and OverrideEntitlement wrap the full mutation (ScheduleEntitlement + grant issuance loop) in transaction.Run(ctx, entitlementRepo, ...) to ensure atomicity across sub-type connector calls. (`return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*entitlement.Entitlement, error) { ent, err := c.ScheduleEntitlement(ctx, input); ... })`)
**ServiceHookRegistry fan-out** — service embeds models.ServiceHookRegistry[entitlement.Entitlement] and exposes RegisterHooks. The balance-worker registers hooks to notify of balance changes. Call hooks.PreUpdate before mutations. (`s.hooks.RegisterHooks(hooks...)`)
**registrybuilder for test setup** — Tests use registrybuilder.GetEntitlementRegistry(EntitlementOptions{...}) to obtain a fully wired registry rather than importing app/common, avoiding import cycles. (`entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{DatabaseClient: dbClient, StreamingConnector: streamingConnector})`)
**Sub-type dispatch via type switch or direct connector** — CreateEntitlement delegates to ScheduleEntitlement which uses a switch on EntitlementType to call the right sub-type connector's BeforeCreate. Never route to connectors directly from HTTP handlers. (`switch input.EntitlementType { case entitlement.EntitlementTypeMetered: return c.meteredConnector.BeforeCreate(input, feat) ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Concrete entitlement.Service implementation dispatching all entitlement operations to the correct sub-type connector via type switch or direct call. | CreateEntitlement delegates to ScheduleEntitlement then loops over grants; if ScheduleEntitlement returns an error, no grants are issued. Keep this ordering. |
| `scheduling.go` | ScheduleEntitlement, SupersedeEntitlement, and time-window activation logic for scheduling future/expiring entitlements. | SupersedeEntitlement deletes the old entitlement and creates a new one in the same transaction — both steps must be inside the same transaction.Run closure. |
| `lock.go` | WithLock helper acquires pg_advisory_xact_lock per entitlement/customer using lockr.Locker before mutation. | Locker requires an active Postgres transaction — must be called after transaction.Run opens a tx, not before. |

## Anti-Patterns

- Importing app/common in test helpers — use registrybuilder instead.
- Calling sub-type connector methods directly from HTTP handlers — always go through entitlement.Service.
- Adding DB queries directly to service.go — all persistence goes through the EntitlementRepo interface.
- Calling lockr.Locker.LockForTX outside an active transaction.Run closure.

## Decisions

- **Service is a thin orchestrator that delegates to three independent sub-type connectors.** — Each entitlement type has fundamentally different storage and computation needs; centralizing dispatch in one service avoids scatter of routing logic across HTTP handlers.

<!-- archie:ai-end -->
