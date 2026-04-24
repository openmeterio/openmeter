# service

<!-- archie:ai-start -->

> Orchestration layer implementing entitlement.Service by composing three sub-type connectors (metered, boolean, static), the feature connector, customer service, and a distributed locker. Routes entitlement operations to the correct sub-type and manages transaction boundaries for composite mutations.

## Patterns

**ServiceConfig constructor pattern** — NewEntitlementService takes a ServiceConfig struct (all deps explicit). Wire constructs ServiceConfig from individual providers; never add logic to the constructor. (`entitlement.Service = service.NewEntitlementService(service.ServiceConfig{MeteredEntitlementConnector: ..., EntitlementRepo: ..., Locker: ...})`)
**transaction.Run for composite mutations** — CreateEntitlement and OverrideEntitlement wrap the full mutation (ScheduleEntitlement + CreateGrant loop) in transaction.Run to ensure atomicity. (`return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*entitlement.Entitlement, error) { ent, err := c.ScheduleEntitlement(ctx, input); ... })`)
**ServiceHookRegistry fan-out** — service embeds models.ServiceHookRegistry[entitlement.Entitlement] and exposes RegisterHooks. Hooks are called on mutations; balance-worker registers hooks to notify of balance changes. (`s.hooks.RegisterHooks(hooks...)`)
**registrybuilder for test setup** — Tests use registrybuilder.GetEntitlementRegistry(EntitlementOptions{...}) to obtain a fully wired registry rather than importing app/common, avoiding import cycles. (`entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{DatabaseClient: dbClient, StreamingConnector: streamingConnector, ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Concrete service implementation with all entitlement.Service methods. Dispatches to sub-type connectors via type switch or direct connector call. | CreateEntitlement delegates to ScheduleEntitlement then loops over grants — if ScheduleEntitlement returns an error, no grants are issued. |
| `scheduling.go` | ScheduleEntitlement, SupersedeEntitlement and the time-window activation logic for scheduling future/expiring entitlements. | SupersedeEntitlement deletes the old entitlement and creates a new one in the same transaction. |
| `lock.go` | WithLock helper that acquires pg_advisory_xact_lock per entitlement/customer using lockr.Locker before mutation. | Locker requires an active Postgres transaction; must be called after transaction.Run opens a tx. |

## Anti-Patterns

- Importing app/common in test helpers — use registrybuilder instead.
- Calling sub-type connector methods directly from HTTP handlers — always go through entitlement.Service.
- Adding DB queries directly to service.go — all persistence goes through the EntitlementRepo interface.

## Decisions

- **Service is a thin orchestrator that delegates to three independent sub-type connectors.** — Each entitlement type has fundamentally different storage and computation needs; centralizing dispatch in one service avoids scatter of routing logic across HTTP handlers.

<!-- archie:ai-end -->
