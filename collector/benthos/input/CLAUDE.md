# input

<!-- archie:ai-start -->

> Houses all custom Benthos BatchInput plugin implementations (kubernetes_resources, otel_log, prometheus, run_ai, schedule). Each file registers exactly one plugin via service.RegisterBatchInput in an init() function; plugins are activated by blank-importing this package.

## Patterns

**init-based plugin registration** — Every plugin calls service.RegisterBatchInput inside an init() function. If registration fails, it panics. The plugin name string is its public Benthos identifier. (`func init() { err := service.RegisterBatchInput("kubernetes_resources", kubernetesResourcesInputConfig(), func(conf *service.ParsedConfig, mgr *service.Resources) (service.BatchInput, error) { return newKubernetesResourcesInput(conf, mgr) }); if err != nil { panic(err) } }`)
**ConfigSpec factory + constructor separation** — Each plugin has a dedicated XxxInputConfig() *service.ConfigSpec function declaring all fields, and a separate newXxxInput() constructor that reads fields via conf.FieldXxx. Never mix schema declaration and construction. (`func prometheusInputConfig() *service.ConfigSpec { return service.NewConfigSpec().Fields(...) }
func newPrometheusInput(conf *service.ParsedConfig, res *service.Resources) (*prometheusInput, error) { url, _ := conf.FieldString(fieldPrometheusURL); ... }`)
**leaderelection guard in ReadBatch** — Polling inputs (kubernetes_resources, prometheus, run_ai) call leaderelection.IsLeader(in.resources) before emitting data. Non-leaders return an empty batch immediately with a noop AckFunc. (`if !leaderelection.IsLeader(in.resources) { return batch, func(context.Context, error) error { return nil }, nil }`)
**nack-restore in AckFunc** — Scheduler-based inputs move items from store to processing before returning. The AckFunc restores items to store on nack (err != nil) so they are retried on the next ReadBatch call. (`return batch, func(ctx context.Context, err error) error { if err != nil { in.mu.Lock(); defer in.mu.Unlock(); for t := range processing { in.store[t] = processing[t] } }; return nil }, nil`)
**gocron scheduler with leader-aware goroutine** — Scheduler-based inputs create a gocron.Scheduler in the constructor, add a CronJob in Connect, then run a goroutine that calls scheduler.Start()/StopJobs() based on leader state polled every 1 second. (`go func() { running := false; for { select { case <-ctx.Done(): if running { _ = in.scheduler.StopJobs() }; return; case <-time.After(1 * time.Second): if leaderelection.IsLeader(in.resources) && !running { in.scheduler.Start(); running = true } else if !leaderelection.IsLeader(in.resources) && running { _ = in.scheduler.StopJobs(); running = false } } } }()`)
**message metadata decoration** — All inputs set msg.MetaSet() keys (scrape_time, scrape_interval, resource_type, etc.) on each service.Message so downstream Bloblang mappings can read them via meta(). (`msg.MetaSet("scrape_time", t.Format(time.RFC3339))
msg.MetaSet("scrape_interval", in.interval.String())`)
**graceful shutdown via cancel + done channel** — Long-lived background goroutines (kubernetes manager, otel gRPC server) are stopped via a context cancel and a done channel closed when the goroutine exits. Close() selects on <-in.done or <-ctx.Done(). (`in.cancel(); select { case <-in.done: in.logger.Info("manager exited"); case <-ctx.Done(): return ctx.Err() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `kubernetes.go` | Registers kubernetes_resources BatchInput. Uses controller-runtime manager with a cache-backed client. Leader-gates ReadBatch. | Cache must sync via WaitForCacheSync in Connect before ReadBatch emits data. Manager runs in a goroutine with cancel/done pattern — omitting done channel leaks the goroutine. |
| `otel_log.go` | Registers otel_log BatchInput backed by a gRPC server (OTLP Logs exporter). Bridges gRPC Export calls to ReadBatch via internal message.Transaction channel. | Uses shutdown.Signaller from collector/benthos/internal/shutdown — do not replace with a plain cancel. Export blocks until ack received or timeout; slow downstream triggers gRPC DeadlineExceeded upstream. |
| `prometheus.go` | Registers prometheus BatchInput that executes PromQL queries on a gocron schedule and buffers results in an in-memory map[time.Time][]QueryResult keyed by scrape time. | store guarded by mu — always hold the lock when reading or writing. Interval computed from gap between two consecutive cron occurrences, not from a simple duration field. |
| `run_ai.go` | Registers run_ai BatchInput that scrapes Run:ai workload/pod metrics via the collector/benthos/input/runai sub-package. Same gocron scheduler + leader guard + nack-restore pattern as prometheus.go. | runai.Service handles metric chunking internally (max 9 per call). pageSize validated 100-500 in constructor. timingMetrics/resourceTypeMetrics wired from RegisterBatchInput closure, not constructor. |
| `schedule.go` | Registers schedule BatchInput that wraps a child input and gates ReadBatch behind a time.Ticker. Decorates messages with schedule_time and schedule_interval metadata. | Uses conf.FieldInput to obtain *service.OwnedInput — child must be closed in Close(). Ticker fires non-blocking (default: case) so ReadBatch returns empty batch when interval has not elapsed. |

## Anti-Patterns

- Registering a new plugin outside init() — the Benthos plugin registry requires init-time registration before the process starts.
- Checking leaderelection.IsLeader inside the gocron task rather than the Connect goroutine — the scheduler must be started/stopped at the goroutine level, not per-task.
- Holding in.mu across a network call or ReadBatch return — the mutex guards only the in-memory store; IO must happen outside the lock.
- Returning an error from ReadBatch for transient per-resource failures — non-fatal errors should be logged and skipped; a returned error halts the Benthos pipeline.
- Defining a new input without a dedicated XxxInputConfig() function — mixing field declarations inside the RegisterBatchInput closure makes the spec untestable.

## Decisions

- **All plugins implement service.BatchInput (not service.Input) so they can return multiple messages per ReadBatch call.** — Polling sources (Kubernetes, Prometheus, Run:ai) naturally produce batches; BatchInput avoids a wrapper and exposes a single AckFunc for the whole batch.
- **Leader election state is polled from service.Resources generic map rather than a dedicated channel.** — service.Resources is already injected into every plugin by the Benthos framework; storing state there avoids a separate DI wire and keeps leader state accessible across plugins without coupling them.
- **Scheduler-based inputs buffer results in an in-memory map[time.Time][]T and drain on ReadBatch rather than sending results directly.** — Benthos calls ReadBatch in a tight loop; decoupling the cron-fired scrape from ReadBatch prevents blocking the scrape goroutine on downstream backpressure.

## Example: Minimal scheduled BatchInput with leader guard, nack-restore, and metadata decoration

```
// Connect: register cron job + leader-aware goroutine
func (in *myInput) Connect(ctx context.Context) error {
	_, err := in.scheduler.NewJob(gocron.CronJob(in.schedule, true), gocron.NewTask(func(ctx context.Context) error {
		in.mu.Lock(); in.store[time.Now()] = in.fetchResults(ctx); in.mu.Unlock(); return nil
	}), gocron.WithContext(ctx))
	if err != nil { return err }
	go func() {
		running := false
		for { select {
		case <-ctx.Done(): if running { _ = in.scheduler.StopJobs() }; return
		case <-time.After(1 * time.Second):
			if leaderelection.IsLeader(in.resources) && !running { in.scheduler.Start(); running = true }
			if !leaderelection.IsLeader(in.resources) && running { _ = in.scheduler.StopJobs(); running = false }
		} }
	}()
// ...
```

<!-- archie:ai-end -->
