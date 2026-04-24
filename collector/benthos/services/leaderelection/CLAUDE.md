# leaderelection

<!-- archie:ai-start -->

> Implements Kubernetes lease-based leader election for the Benthos collector, ensuring only one replica acts as the active leader at a time. Exposes leader state via Benthos service.Resources generic key so plugins can gate work behind IsLeader().

## Patterns

**CLI flags declared as package-level constants** — Every tunable is a package-level const string (e.g. leaderElectionEnabledFlag) used in both flag definitions and ctx.Bool/String/Duration reads — never use raw string literals in two places. (`const leaderElectionEnabledFlag = "leader-election"; ctx.Bool(leaderElectionEnabledFlag)`)
**Config struct as the sole handoff between CLI and Service** — CLIOptCustomRunFlags callback populates a local Config value; CLIOptOnConfigParse passes it to NewService. The service never reads cli.Context directly. (`leaderElectionConfig = Config{Enabled: ctx.Bool(leaderElectionEnabledFlag), ...}; NewService(conf.Resources(), leaderElectionConfig)`)
**IsLeaderKey context-free state via service.Resources generic** — Leader state is communicated to other plugins through res.SetGeneric(IsLeaderKey, bool) — not via channel or context value. Plugins check with IsLeader(res). (`res.SetGeneric(IsLeaderKey, true) // in OnStartedLeading callback`)
**Start/Stop with mutex-guarded started flag** — Service.Start stores cancel func and sets started=true under mu.Lock; Stop calls cancel and resets started=false. Prevents double-start and ensures clean teardown. (`s.mu.Lock(); defer s.mu.Unlock(); if s.started { return fmt.Errorf(...) }`)
**Infinite retry loop inside goroutine in Start** — The goroutine re-creates LeaderElector after each le.Run returns (lease lost or error) and re-tries after LeaseRetryPeriod unless ctx is cancelled. New code that runs inside leader scope must not embed its own retry. (`for { le, _ := leaderelection.NewLeaderElector(lec); le.Run(ctx); if ctx.Err() != nil { return }; time.After(LeaseRetryPeriod) }`)
**klog routed through Benthos logger at construction time** — NewService calls logging.SetupKlog(logger) immediately so all client-go internal logs flow through the Benthos structured logger. Every new service using client-go must do the same. (`logging.SetupKlog(res.Logger().With("component", "leader election"))`)
**Default leaseHealthCheckTimeout derived from LeaseDuration when zero** — If LeaseHealthCheckTimeout is 0, it is set to LeaseDuration * 1.5 before constructing HealthzAdaptor. New code must not pass 0 to NewLeaderHealthzAdaptor. (`if leaseHealthCheckTimeout == 0 { leaseHealthCheckTimeout = cfg.LeaseDuration + cfg.LeaseDuration/2 }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `flags.go` | Declares all CLI flag name constants and the leaderElectionCLIFlags slice passed to service.CLIOptCustomRunFlags. Only place flag names are defined. | Adding a new tunable requires a const here AND a matching field in Config in service.go — missing either silently drops the value. |
| `service.go` | Contains Config, Service struct, NewService constructor, Start/Stop lifecycle, GetLeaderElectionCLIOpts wiring function, and IsLeader helper. All public API lives here. | IsLeader returns true when the key is absent (leader election disabled path) — do not change this semantic without updating every caller that relies on the no-leader-election=always-leader assumption. |

## Anti-Patterns

- Reading cli.Context flags by raw string literals instead of the declared const in flags.go
- Calling leaderelection.NewLeaderElector outside the retry loop in Start — the elector must be re-created on each lease cycle
- Using context.Background() instead of the ctx passed into Start/GetLeaderElectionCLIOpts — breaks cancellation on shutdown
- Skipping logging.SetupKlog when adding another client-go-based service — klog will bypass the structured logger
- Checking leader state by reading IsLeaderKey directly via res.GetGeneric instead of using IsLeader(res) — misses the absent-key=true default

## Decisions

- **Leader state stored in service.Resources generic map rather than a dedicated channel or atomic** — Benthos plugins receive *service.Resources from the framework; using SetGeneric/GetGeneric avoids threading a custom context or global variable through every plugin constructor.
- **ReleaseOnCancel: true in LeaderElectionConfig** — Ensures the Kubernetes lease is actively released when the context is cancelled (graceful shutdown), preventing other replicas from waiting out the full LeaseDuration.
- **Infinite retry loop re-creates LeaderElector per cycle** — LeaderElector is not reusable after Run returns; re-creating it on each cycle is the pattern mandated by client-go to correctly resume after a lost lease.

## Example: Wire leader election into a Benthos CLI application and gate a plugin on leadership

```
import (
    "context"
    "github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"
    "github.com/redpanda-data/benthos/v4/public/service"
)

func main() {
    ctx := context.Background()
    opts := leaderelection.GetLeaderElectionCLIOpts(ctx)
    svc := service.NewCLI(opts...)
    svc.Run()
}

// Inside a plugin handler:
func (p *myPlugin) Process(ctx context.Context, msg *service.Message) ([]*service.Message, error) {
// ...
```

<!-- archie:ai-end -->
