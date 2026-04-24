# services

<!-- archie:ai-start -->

> Benthos service-layer packages that provide cross-cutting infrastructure to input/output plugins without being plugins themselves. Currently one sub-package: leaderelection, which implements Kubernetes lease-based leader election and exposes leader state via service.Resources so plugins can gate work to a single active replica.

## Patterns

**Config struct as CLI-to-service handoff** — CLI flags are declared as package-level constants in flags.go and parsed into a Config struct. Service.New accepts only Config — never a cli.Context. This makes the service testable without a CLI. (`cfg := leaderelection.Config{LeaseDuration: 15*time.Second, ...}
svc := leaderelection.New(cfg, logger)`)
**IsLeader via service.Resources generic map** — Leader state is stored under IsLeaderKey in the Benthos service.Resources generic map. Plugins call leaderelection.IsLeader(res) — absent key defaults to true (single-replica safe). Never read IsLeaderKey directly. (`if !leaderelection.IsLeader(in.resources) { return emptyBatch, noopAck, nil }`)
**Infinite retry loop in Start goroutine** — leaderelection.Service.Start runs an infinite loop that creates a new LeaderElector per cycle. The elector must be re-created each time — never reuse a LeaderElector across cycles. (`for { elector, _ := leaderelection.NewLeaderElector(cfg); elector.Run(ctx); if ctx.Err() != nil { return } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `services/leaderelection/flags.go` | Declares CLI flag name constants and GetLeaderElectionCLIOpts. Import this to add leader election flags to a Benthos CLI app. | Always reference flag names via the constants, not raw strings — the constants are the contract between the CLI and service.go. |
| `services/leaderelection/service.go` | Core leader election logic: constructs client-go LeaderElector, runs it in a goroutine, updates service.Resources on acquire/release, exposes IsLeader(). | logging.SetupKlog must be called here before any client-go code runs. ReleaseOnCancel:true means leadership is released when ctx is cancelled — Start's ctx must be tied to process lifecycle. leaseHealthCheckTimeout defaults to LeaseDuration when zero. |

## Anti-Patterns

- Reading IsLeaderKey directly via res.GetGeneric instead of using IsLeader(res) — the helper applies the absent-key=true default.
- Reusing a LeaderElector instance across election cycles — client-go electors are single-use; recreate per cycle.
- Calling leaderelection.Service.Start with context.Background() — breaks cancellation on Benthos shutdown.
- Skipping logging.SetupKlog when adding another client-go-based service — klog bypasses the structured logger.
- Reading CLI flag names as raw string literals instead of the declared constants in flags.go.

## Decisions

- **Leader state stored in service.Resources generic map, not a channel or atomic.** — service.Resources is already injected by Benthos into every plugin; using it avoids a separate DI step and keeps leader state accessible to all plugins without coupling them to the leaderelection package's internals.
- **Absent IsLeaderKey defaults to true (i.e., act as leader when no election service is running).** — Single-replica deployments that do not configure leader election must not have their plugins silently no-op; the default-true behavior preserves backwards compatibility.

<!-- archie:ai-end -->
