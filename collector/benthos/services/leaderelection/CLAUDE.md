# leaderelection

<!-- archie:ai-start -->

> Kubernetes lease-based leader election for the Benthos collector, ensuring only one replica is the active leader. Exposes leader state via the service.Resources generic key so plugins can gate work behind IsLeader().

## Patterns

**CLI flags declared as package-level constants** — Every tunable is a package-level const string used in both flag definitions and ctx reads. Never use raw string literals in two places. (`const leaderElectionEnabledFlag = "leader-election"; ctx.Bool(leaderElectionEnabledFlag)`)
**Config struct as the sole CLI-to-Service handoff** — CLIOptCustomRunFlags populates a local Config; CLIOptOnConfigParse passes it to NewService. The service never reads cli.Context directly. (`leaderElectionConfig = Config{Enabled: ctx.Bool(leaderElectionEnabledFlag), ...}; NewService(conf.Resources(), leaderElectionConfig)`)
**IsLeaderKey state via service.Resources generic** — Leader state is communicated through res.SetGeneric(IsLeaderKey, bool) — not a channel or context value. Plugins check with IsLeader(res); absent key returns true (leader election disabled = always leader). (`res.SetGeneric(IsLeaderKey, true) // in OnStartedLeading callback`)
**Start/Stop with mutex-guarded started flag** — Service.Start stores the cancel func and sets started=true under mu.Lock; Stop calls cancel and resets started=false, preventing double-start. (`s.mu.Lock(); defer s.mu.Unlock(); if s.started { return fmt.Errorf("already started") }`)
**Infinite retry loop re-creates LeaderElector per cycle** — The goroutine re-creates LeaderElector after each le.Run returns (lease lost/error) and retries after LeaseRetryPeriod unless ctx is cancelled. LeaderElector is not reusable after Run — mandated by client-go. (`for { le, _ := leaderelection.NewLeaderElector(lec); le.Run(ctx); if ctx.Err() != nil { return }; time.After(cfg.LeaseRetryPeriod) }`)
**klog routed through Benthos logger at construction** — NewService calls logging.SetupKlog(logger) immediately so all client-go internal logs flow through the Benthos structured logger. Every client-go service must do the same. (`logging.SetupKlog(res.Logger().With("component", "leader election"))`)
**Default leaseHealthCheckTimeout derived from LeaseDuration** — If LeaseHealthCheckTimeout is 0, it is set to LeaseDuration * 1.5 before constructing HealthzAdaptor. Never pass 0 to NewLeaderHealthzAdaptor. (`if leaseHealthCheckTimeout == 0 { leaseHealthCheckTimeout = cfg.LeaseDuration + cfg.LeaseDuration/2 }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `flags.go` | Declares all CLI flag name constants and the leaderElectionCLIFlags slice passed to service.CLIOptCustomRunFlags. The only place flag names are defined. | Adding a tunable requires a const here AND a matching field in Config in service.go — missing either silently drops the value. |
| `service.go` | Config, Service struct, NewService, Start/Stop lifecycle, GetLeaderElectionCLIOpts wiring, and IsLeader helper. All public API lives here. | IsLeader returns true when the key is absent (leader election disabled) — do not change this semantic without updating every caller relying on no-leader-election=always-leader. |

## Anti-Patterns

- Reading cli.Context flags by raw string literal instead of the declared const in flags.go.
- Calling leaderelection.NewLeaderElector outside the retry loop in Start — the elector must be re-created each lease cycle.
- Using context.Background() instead of the ctx passed into Start/GetLeaderElectionCLIOpts — breaks shutdown cancellation.
- Skipping logging.SetupKlog when adding another client-go service — klog bypasses the structured logger.
- Reading IsLeaderKey directly via res.GetGeneric instead of IsLeader(res) — misses the absent-key=true default.

## Decisions

- **Leader state stored in service.Resources generic map rather than a channel or atomic.** — Benthos plugins receive *service.Resources from the framework; SetGeneric/GetGeneric avoids threading a custom context or global through every plugin constructor.
- **ReleaseOnCancel: true in LeaderElectionConfig.** — Actively releases the Kubernetes lease on context cancellation (graceful shutdown), preventing other replicas from waiting out the full LeaseDuration.
- **Infinite retry loop re-creates LeaderElector per cycle.** — LeaderElector is not reusable after Run returns; re-creating per cycle is the client-go pattern for resuming after a lost lease.

## Example: Wire leader election into a Benthos CLI and gate a plugin on leadership

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

// Inside a plugin handler: check leaderelection.IsLeader(res) before doing work.
```

<!-- archie:ai-end -->
