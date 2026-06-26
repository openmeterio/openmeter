# leaderelection

<!-- archie:ai-start -->

> Kubernetes-based leader-election service for the standalone Benthos/Redpanda Connect collector binary. Exposes CLI flags and a Benthos CLIOpt that lets exactly one collector instance act as leader (e.g. so only the leader runs singleton inputs/jobs), tracking leadership via the Benthos Resources generic store.

## Patterns

**Wire leadership into Benthos via CLIOptFunc** — Leadership is integrated by returning []service.CLIOptFunc from GetLeaderElectionCLIOpts: CLIOptCustomRunFlags registers leaderElectionCLIFlags and parses them into a Config, then CLIOptOnConfigParse constructs NewService and starts it. Add new collector-wide leadership wiring here, not in input/output plugins. (`service.CLIOptCustomRunFlags(leaderElectionCLIFlags, func(ctx *cli.Context) error { leaderElectionConfig = Config{Enabled: ctx.Bool(leaderElectionEnabledFlag), ...}; return nil })`)
**Leadership state lives in Resources generic store** — Leader status is published with res.SetGeneric(IsLeaderKey, true/false) in the LeaderCallbacks and read with res.GetGeneric(IsLeaderKey). IsLeaderKey is a typed genericKey constant. Consumers must call IsLeader(res) rather than reading the generic key directly. (`OnStartedLeading: func(ctx context.Context){ res.SetGeneric(IsLeaderKey, true) }`)
**Fail-open IsLeader semantics** — IsLeader(res) returns true when IsLeaderKey is absent (leader election disabled => everyone is leader), and false when present-but-not-a-bool. Preserve this fail-open-when-unset / fail-closed-when-corrupt contract so disabling leader election doesn't silently stop all processing. (`leader, ok := res.GetGeneric(IsLeaderKey); if !ok { return true }`)
**Flags carry env-var fallbacks and K8s downward-API defaults** — Every cli.Flag in leaderElectionCLIFlags sets EnvVars (e.g. LEASE_LOCK_NAMESPACE plus K8S_* downward-API names) and durations have sane defaults (LeaseDuration 15s, RenewDeadline 10s, RetryPeriod 2s). Identity defaults to os.Hostname(). New flags must follow the same EnvVars + Value convention. (`&cli.StringFlag{Name: leaseLockIdentityFlag, EnvVars: []string{"K8S_POD_NAME", "LEASE_LOCK_IDENTITY"}, Value: hostname}`)
**Self-healing run loop with cancel guard** — Start() launches a goroutine that recreates a NewLeaderElector and calls le.Run(ctx) in a for-loop, retrying after LeaseRetryPeriod unless ctx is cancelled. mu + started guard against double-start; Stop() calls s.cancel(). ReleaseOnCancel is true so leadership is released on shutdown. (`for { le, _ := leaderelection.NewLeaderElector(*s.leaderElectionConfig); le.Run(ctx); if ctx.Err() != nil { return } ... }`)
**Validate required lock fields only when enabled** — In CLIOptOnConfigParse the service short-circuits when !Enabled; when enabled it requires LeaseLockNamespace and LeaseLockName before constructing the service. Keep validation gated on Enabled so disabled deployments don't need K8s config. (`if !leaderElectionConfig.Enabled { return nil }; if leaderElectionConfig.LeaseLockNamespace == "" { return fmt.Errorf(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Config, Service, NewService (builds K8s client + LeaseLock + LeaderElectionConfig), Start/Stop, the GetLeaderElectionCLIOpts entrypoint, and the IsLeader(res) helper. | NewService reads kubeconfig via controller-runtime config.GetConfig() and routes klog through logging.SetupKlog(logger) — it only works in-cluster or with a valid kubeconfig. LeaseHealthCheckTimeout==0 is auto-set to LeaseDuration*1.5. Don't read IsLeaderKey directly; use IsLeader. |
| `flags.go` | Declares the leaderElection* flag-name constants, the hostname default, and leaderElectionCLIFlags ([]cli.Flag) consumed by CLIOptCustomRunFlags. | Flag names are string constants used in both files — keep flags.go constants and service.go ctx.Bool/String/Duration lookups in sync. hostname is captured once at package init via os.Hostname(). |

## Anti-Patterns

- Reading res.GetGeneric(IsLeaderKey) directly instead of calling IsLeader(res), losing the fail-open-when-unset semantics.
- Calling le.Run / NewLeaderElector outside the guarded Start() goroutine, bypassing the mu/started double-start guard and retry loop.
- Adding a flag without EnvVars and a default Value, breaking the K8s downward-API / env-driven deployment convention.
- Setting ReleaseOnCancel=false or removing s.cancel() in Stop(), which leaves the lease held after shutdown and blocks failover.
- Importing this package into root-module code — it belongs to the independent collector go.mod and pulls in k8s.io/client-go.

## Decisions

- **Leadership is exposed through the Benthos Resources generic store (SetGeneric/GetGeneric) rather than a shared Go variable.** — Lets Benthos plugins (input/output/bloblang) in other packages query leadership via *service.Resources without a direct import dependency on this service.
- **Integration is via service.CLIOptFunc (CLIOptCustomRunFlags + CLIOptOnConfigParse) instead of a Benthos processor/plugin.** — Leader election is a process-wide concern parsed once at CLI startup, so it hooks the Redpanda Connect CLI lifecycle rather than per-stream config.
- **IsLeader fails open (returns true) when the generic key is unset.** — When leader election is disabled the collector must still process events, so absence of leadership state means 'act as leader'.

## Example: Gating singleton work on leadership inside a Benthos plugin

```
import "github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"

if !leaderelection.IsLeader(res) {
    // not the leader (and leader election is enabled): skip singleton work
    return nil
}
```

<!-- archie:ai-end -->
