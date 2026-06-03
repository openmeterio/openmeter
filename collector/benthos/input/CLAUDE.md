# input

<!-- archie:ai-start -->

> Houses all custom Benthos BatchInput plugin implementations (kubernetes_resources, otel_log, prometheus, run_ai, schedule) for the collector module. Each file registers exactly one plugin via service.RegisterBatchInput in an init() function; plugins are activated by blank-importing this package. The runai/ sub-package is the typed Run:ai HTTP client used by run_ai.go.

## Patterns

**init-based plugin registration** — Every plugin calls service.RegisterBatchInput inside init() and panics on registration failure; the name string is its public Benthos identifier. (`func init() { if err := service.RegisterBatchInput("kubernetes_resources", kubernetesResourcesInputConfig(), func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) { return newKubernetesResourcesInput(conf, mgr) }); err != nil { panic(err) } }`)
**ConfigSpec factory + constructor separation** — Each plugin has a dedicated XxxInputConfig() *service.ConfigSpec declaring fields and a separate newXxxInput() constructor reading them via conf.FieldXxx. Never mix schema declaration and construction. (`func prometheusInputConfig() *service.ConfigSpec { return service.NewConfigSpec().Fields(...) }
func newPrometheusInput(conf *service.ParsedConfig, res *service.Resources) (*prometheusInput, error) { url, _ := conf.FieldString(fieldPrometheusURL); ... }`)
**leaderelection guard in ReadBatch** — Polling inputs (kubernetes_resources, prometheus, run_ai) call leaderelection.IsLeader(in.resources) before emitting; non-leaders return an empty batch with a noop AckFunc. (`if !leaderelection.IsLeader(in.resources) { return batch, func(context.Context, error) error { return nil }, nil }`)
**nack-restore in AckFunc** — Scheduler-based inputs move items from store to processing before returning; the AckFunc restores them to store on nack (err != nil) so they retry next ReadBatch. (`return batch, func(ctx context.Context, err error) error { if err != nil { in.mu.Lock(); defer in.mu.Unlock(); for t := range processing { in.store[t] = processing[t] } }; return nil }, nil`)
**gocron scheduler with leader-aware goroutine** — Scheduler inputs create a gocron.Scheduler in the constructor, add a CronJob in Connect, then run a goroutine that Start()/StopJobs() based on leader state polled every 1 second. (`go func() { running := false; for { select { case <-ctx.Done(): if running { _ = in.scheduler.StopJobs() }; return; case <-time.After(time.Second): if leaderelection.IsLeader(in.resources) && !running { in.scheduler.Start(); running = true } else if !leaderelection.IsLeader(in.resources) && running { _ = in.scheduler.StopJobs(); running = false } } } }()`)
**message metadata decoration** — Inputs set msg.MetaSet() keys (scrape_time, scrape_interval, resource_type, ...) so downstream Bloblang mappings read them via meta(). (`msg.MetaSet("scrape_time", t.Format(time.RFC3339)); msg.MetaSet("scrape_interval", in.interval.String())`)
**graceful shutdown via cancel + done channel** — Long-lived background goroutines (kubernetes manager, otel gRPC server) stop via context cancel + a done channel closed on goroutine exit; Close() selects on <-in.done or <-ctx.Done(). (`in.cancel(); select { case <-in.done: in.logger.Info("manager exited"); case <-ctx.Done(): return ctx.Err() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `kubernetes.go` | Registers kubernetes_resources BatchInput using a controller-runtime manager with a cache-backed client; leader-gates ReadBatch. | Cache must sync via WaitForCacheSync in Connect before ReadBatch emits; manager runs in a goroutine with cancel/done — omitting done leaks the goroutine. |
| `otel_log.go` | Registers otel_log BatchInput backed by a gRPC OTLP Logs server, bridging Export to ReadBatch via internal message.Transaction. | Uses shutdown.Signaller from collector/benthos/internal/shutdown — do not replace with a plain cancel; Export blocks until ack/timeout. |
| `prometheus.go` | Registers prometheus BatchInput executing PromQL on a gocron schedule, buffering results in map[time.Time][]QueryResult. | store guarded by mu — hold the lock for read/write; interval is computed from the gap between two cron occurrences. |
| `run_ai.go` | Registers run_ai BatchInput scraping Run:ai metrics via the runai sub-package; same scheduler + leader + nack-restore pattern as prometheus. | runai.Service chunks metrics internally (max 9/call); pageSize validated 100-500; timing/resourceType metrics wired from the RegisterBatchInput closure, not the constructor. |
| `schedule.go` | Registers schedule BatchInput wrapping a child input and gating ReadBatch behind a time.Ticker; decorates messages with schedule_time/schedule_interval. | Uses conf.FieldInput to obtain *service.OwnedInput — the child must be closed in Close(); Ticker fires non-blocking so ReadBatch returns empty when the interval has not elapsed. |

## Anti-Patterns

- Registering a plugin outside init() — the Benthos registry requires init-time registration before process start.
- Checking leaderelection.IsLeader inside the gocron task rather than the Connect goroutine — the scheduler must be started/stopped at goroutine level.
- Holding in.mu across a network call or ReadBatch return — the mutex guards only the in-memory store; IO must happen outside the lock.
- Returning an error from ReadBatch for transient per-resource failures — log and skip; a returned error halts the pipeline.
- Defining an input without a dedicated XxxInputConfig() function — inline field declarations in the closure make the spec untestable.

## Decisions

- **All plugins implement service.BatchInput (not service.Input).** — Polling sources (Kubernetes, Prometheus, Run:ai) naturally produce batches; BatchInput exposes one AckFunc for the whole batch.
- **Leader state polled from service.Resources rather than a dedicated channel.** — service.Resources is already injected into every plugin; storing state there avoids a separate DI wire and keeps state accessible across plugins.
- **Scheduler inputs buffer results in an in-memory map and drain on ReadBatch.** — Benthos calls ReadBatch in a tight loop; decoupling the cron scrape from ReadBatch prevents blocking the scrape goroutine on downstream backpressure.

## Example: Minimal scheduled BatchInput with leader guard and metadata decoration

```
func (in *myInput) Connect(ctx context.Context) error {
  _, err := in.scheduler.NewJob(gocron.CronJob(in.schedule, true), gocron.NewTask(func(ctx context.Context) error {
    in.mu.Lock(); in.store[time.Now()] = in.fetchResults(ctx); in.mu.Unlock(); return nil
  }), gocron.WithContext(ctx))
  if err != nil { return err }
  go func() { running := false; for { select {
    case <-ctx.Done(): if running { _ = in.scheduler.StopJobs() }; return
    case <-time.After(time.Second):
      if leaderelection.IsLeader(in.resources) && !running { in.scheduler.Start(); running = true }
      if !leaderelection.IsLeader(in.resources) && running { _ = in.scheduler.StopJobs(); running = false }
  } } }()
  return nil
}
```

<!-- archie:ai-end -->
