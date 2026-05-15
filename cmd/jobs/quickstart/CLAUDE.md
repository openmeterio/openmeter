# quickstart

<!-- archie:ai-start -->

> Cobra sub-command group for docker-compose quickstart helpers; provides a 'cron' command that schedules billing lifecycle polling loops using gocron. Explicitly not for production — billing-worker handles these flows via Kafka/Watermill in production.

## Patterns

**gocron-based polling loops for billing lifecycle** — cronjobs.go uses gocron.NewScheduler to schedule BillingSubscriptionReconciler.All, BillingCollector.All, BillingAutoAdvancer.All at fixed intervals. Scheduler starts via s.Start() then blocks on cmd.Context().Done(). (`s.NewJob(gocron.DurationJob(time.Minute), gocron.NewTask(func() { internal.App.BillingCollector.All(cmd.Context(), namespaces, nil, batchSize) }))`)
**Nil-guard optional internal.App services before scheduling** — ChargesAutoAdvancer may be nil when charges/credits are disabled; always check nil before scheduling its job. (`if internal.App.ChargesAutoAdvancer != nil { s.NewJob(gocron.DurationJob(time.Minute), gocron.NewTask(func() { ... })) }`)
**Block on context cancellation then shutdown scheduler** — The cron RunE blocks on <-cmd.Context().Done() and calls s.Shutdown() before returning, ensuring graceful stop of all scheduled jobs. (`<-cmd.Context().Done(); if err := s.Shutdown(); err != nil { return err }`)
**Composition of other job packages via init()** — quickstart.go adds sub-commands from other job packages (e.g. billing/subscriptionsync) via Cmd.AddCommand in init(). The package acts as a meta-group aggregating related operational commands. (`func init() { Cmd.AddCommand(subscriptionsync.Cmd) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `quickstart.go` | Defines the 'quickstart' parent command and composes sub-commands from other job packages via init(). | Explicitly documented as not for production. Do not add production-critical logic here. |
| `cronjobs.go` | Implements 'cron' sub-command using gocron for billing lifecycle automation in quickstart environments. | Hardcodes namespace 'default' and batchSize=10 — unsuitable for multi-tenant production use. Any new cron job must nil-guard optional internal.App services before scheduling. |

## Anti-Patterns

- Using this package's gocron patterns in production workers — billing-worker uses Kafka/Watermill for these flows
- Hardcoding namespaces for production deployments
- Not nil-checking optional internal.App services (e.g. ChargesAutoAdvancer) before scheduling
- Adding production-critical billing logic to quickstart commands

## Decisions

- **Quickstart cron uses gocron instead of Watermill/Kafka for simplicity.** — The quickstart docker-compose environment may not have a full Kafka topology; gocron provides polling-based advancement without Kafka dependencies, making the quickstart self-contained.

<!-- archie:ai-end -->
