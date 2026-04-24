# quickstart

<!-- archie:ai-start -->

> Cobra sub-command group for docker-compose quickstart helpers; provides a 'cron' command that schedules billing and subscription sync loops using gocron, intended only for quickstart/demo environments.

## Patterns

**gocron-based polling loops for billing lifecycle** — cronjobs.go uses gocron.NewScheduler to schedule BillingSubscriptionReconciler.All, BillingCollector.All, BillingAutoAdvancer.All, and (conditionally) ChargesAutoAdvancer.All at fixed intervals. Context cancellation shuts down the scheduler. (`s.NewJob(gocron.DurationJob(time.Minute), gocron.NewTask(func() { internal.App.BillingCollector.All(cmd.Context(), namespaces, nil, batchSize) }))`)
**Optional feature guard via nil check** — ChargesAutoAdvancer may be nil when charges are disabled; code guards with if internal.App.ChargesAutoAdvancer != nil before scheduling. (`if internal.App.ChargesAutoAdvancer != nil { s.NewJob(...) }`)
**Block on context cancellation, then shutdown scheduler** — The cron command blocks on <-cmd.Context().Done() and calls s.Shutdown() before returning, ensuring graceful stop. (`<-cmd.Context().Done(); if err := s.Shutdown(); err != nil { return err }`)
**Composition of other job packages** — quickstart.go adds sub-commands from other job packages (e.g. billing/subscriptionsync) via init() — it acts as a meta-group aggregating related operational commands. (`func init() { Cmd.AddCommand(subscriptionsync.Cmd) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `quickstart.go` | Defines the 'quickstart' parent command and composes sub-commands from other job packages. | Explicitly documented as not for production; do not add production-critical logic here. |
| `cronjobs.go` | Implements 'cron' sub-command with gocron scheduler for billing lifecycle automation in quickstart environments. | Hardcodes namespace 'default' and small batchSize=10 — not suitable for multi-tenant production use. Any new cron job must null-guard optional services from internal.App before scheduling. |

## Anti-Patterns

- Using this package's cron patterns in production workers (billing-worker already handles these via Kafka/Watermill)
- Hardcoding namespaces inside quickstart cron tasks for production deployments
- Not null-checking optional internal.App services (e.g. ChargesAutoAdvancer) before scheduling

## Decisions

- **Quickstart cron uses gocron instead of Watermill/Kafka for simplicity.** — The quickstart docker-compose environment may not have a full Kafka topology; gocron provides polling-based advancement without Kafka dependencies.

<!-- archie:ai-end -->
