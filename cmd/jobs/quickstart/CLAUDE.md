# quickstart

<!-- archie:ai-start -->

> Cobra sub-command group for docker-compose quickstart helpers; its 'cron' command schedules billing-lifecycle polling loops via gocron. Explicitly NOT for production — billing-worker drives these flows via Kafka/Watermill instead.

## Patterns

**gocron polling loops for billing lifecycle** — cronjobs.go uses gocron.NewScheduler to schedule BillingSubscriptionReconciler.All (hourly), BillingCollector.All and BillingAutoAdvancer.All (per-minute) on internal.App; jobs run via gocron.DurationJob + gocron.NewTask passing cmd.Context(). (`s.NewJob(gocron.DurationJob(time.Minute), gocron.NewTask(func() { internal.App.BillingCollector.All(cmd.Context(), namespaces, nil, batchSize) }))`)
**Nil-guard optional internal.App services** — ChargesAutoAdvancer may be nil when charges/credits are disabled; always nil-check before scheduling its job. (`if internal.App.ChargesAutoAdvancer != nil { s.NewJob(gocron.DurationJob(time.Minute), gocron.NewTask(func() { internal.App.ChargesAutoAdvancer.All(cmd.Context(), namespaces) })) }`)
**Block on context cancel then shutdown scheduler** — Cron RunE calls s.Start(), blocks on <-cmd.Context().Done(), then s.Shutdown() for graceful stop. (`<-cmd.Context().Done(); if err := s.Shutdown(); err != nil { return err }`)
**Meta-group composition via init()** — quickstart.go aggregates sub-commands from other job packages (subscriptionsync) and cronjobs.go adds Cron, all through Cmd.AddCommand in init(). (`func init() { Cmd.AddCommand(subscriptionsync.Cmd) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `quickstart.go` | Defines the 'quickstart' parent command and composes sub-commands (subscriptionsync) via init(). | Documented as not for production — do not add production-critical logic here. |
| `cronjobs.go` | Implements the 'cron' sub-command using gocron for billing-lifecycle automation in quickstart environments. | Hardcodes namespace 'default' and batchSize=10 (unsuitable for multi-tenant production). New cron jobs must nil-guard optional internal.App services. Job task errors are logged, not returned. |

## Anti-Patterns

- Using these gocron patterns in production workers — billing-worker uses Kafka/Watermill
- Hardcoding namespaces for production deployments
- Not nil-checking optional internal.App services (e.g. ChargesAutoAdvancer) before scheduling
- Adding production-critical billing logic to quickstart commands

## Decisions

- **Quickstart cron uses gocron instead of Watermill/Kafka** — The quickstart docker-compose environment may lack a full Kafka topology; gocron provides polling-based advancement without Kafka dependencies, keeping quickstart self-contained.

## Example: Add a nil-guarded scheduled billing job

```
import (
  "time"
  "github.com/go-co-op/gocron/v2"
  "github.com/openmeterio/openmeter/cmd/jobs/internal"
)
if internal.App.ChargesAutoAdvancer != nil {
  _, err = s.NewJob(gocron.DurationJob(time.Minute), gocron.NewTask(func() {
    if err := internal.App.ChargesAutoAdvancer.All(cmd.Context(), namespaces); err != nil {
      slog.Error("Error advancing charges", "error", err)
    }
  }))
  if err != nil { return err }
}
```

<!-- archie:ai-end -->
