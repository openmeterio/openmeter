# services

<!-- archie:ai-start -->

> Benthos service-layer packages providing cross-cutting infrastructure to input/output plugins without being plugins themselves. Currently one sub-package, leaderelection, which implements Kubernetes lease-based leader election and exposes leader state via service.Resources so plugins gate work to a single active replica.

## Patterns

**Config struct as CLI-to-service handoff** — CLI flags are declared as package-level constants in flags.go and parsed into a Config struct; Service.New accepts only Config (never a cli.Context), keeping the service testable without a CLI. (`cfg := leaderelection.Config{LeaseDuration: 15*time.Second}; svc := leaderelection.New(cfg, logger)`)
**IsLeader via service.Resources generic map** — Leader state is stored under IsLeaderKey in the Benthos service.Resources generic map; plugins call leaderelection.IsLeader(res) where an absent key defaults to true (single-replica safe). Never read IsLeaderKey directly. (`if !leaderelection.IsLeader(in.resources) { return emptyBatch, noopAck, nil }`)
**Infinite retry loop re-creates LeaderElector per cycle** — Service.Start runs an infinite loop creating a new LeaderElector per cycle; client-go electors are single-use and must never be reused across cycles. (`for { elector, _ := leaderelection.NewLeaderElector(cfg); elector.Run(ctx); if ctx.Err() != nil { return } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `leaderelection/flags.go` | Declares CLI flag name constants and GetLeaderElectionCLIOpts; import to add leader election flags to a Benthos CLI app. | Always reference flag names via the constants, not raw strings — they are the contract between CLI parsing and service.go. |
| `leaderelection/service.go` | Core leader election: constructs client-go LeaderElector, runs it in a goroutine, updates service.Resources on acquire/release, exposes IsLeader(). | logging.SetupKlog must run before any client-go code; ReleaseOnCancel:true means leadership releases on ctx cancel — tie Start ctx to process lifecycle; leaseHealthCheckTimeout defaults to LeaseDuration when zero. |

## Anti-Patterns

- Reading IsLeaderKey directly via res.GetGeneric instead of IsLeader(res) — the helper applies the absent-key=true default for single-replica deployments.
- Reusing a LeaderElector instance across election cycles — client-go electors are single-use; recreate per cycle inside the retry loop.
- Calling leaderelection.Service.Start with context.Background() — breaks cancellation on Benthos shutdown.
- Skipping logging.SetupKlog when adding another client-go-based service — klog bypasses the structured Benthos logger.
- Reading CLI flag names as raw string literals instead of the declared constants in flags.go.

## Decisions

- **Leader state stored in service.Resources generic map, not a channel or atomic.** — service.Resources is already injected by Benthos into every plugin; using it avoids a separate DI step and keeps state accessible to all plugins without coupling them to leaderelection internals.
- **Absent IsLeaderKey defaults to true (act as leader when no election service runs).** — Single-replica deployments that do not configure leader election must not have their plugins silently no-op; default-true preserves backwards compatibility.

<!-- archie:ai-end -->
