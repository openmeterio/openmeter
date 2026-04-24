# input

<!-- archie:ai-start -->

> Houses all custom Benthos BatchInput plugin implementations (kubernetes_resources, otel_log, prometheus, run_ai, schedule). Each file registers one plugin via service.RegisterBatchInput in an init() function; plugins are activated simply by importing this package.

## Patterns

**init-based plugin registration** — Every input plugin calls service.RegisterBatchInput (or service.RegisterInput) inside an init() function. If registration fails, it panics. The plugin is activated by blank-importing this package. (`func init() { err := service.RegisterBatchInput("my_input", myInputConfig(), func(...) (service.BatchInput, error) { ... }); if err != nil { panic(err) } }`)
**ConfigSpec factory + constructor separation** — Each plugin has a dedicated XxxInputConfig() function returning *service.ConfigSpec with all fields declared, and a separate newXxxInput() constructor that reads those fields via conf.FieldXxx. Never mix schema and construction logic. (`func prometheusInputConfig() *service.ConfigSpec { return service.NewConfigSpec().Fields(...) }
func newPrometheusInput(conf *service.ParsedConfig, res *service.Resources) (*prometheusInput, error) { url, _ := conf.FieldString(fieldURL); ... }`)
**leaderelection guard in ReadBatch/Connect** — Polling inputs (kubernetes_resources, prometheus, run_ai) check leaderelection.IsLeader(in.resources) before emitting data or starting the scheduler goroutine. Non-leaders return an empty batch immediately. (`if !leaderelection.IsLeader(in.resources) { return batch, func(context.Context, error) error { return nil }, nil }`)
**nack-restore in AckFunc** — Scheduler-based inputs (prometheus, run_ai) move items from store to processing before returning them. The AckFunc restores items to store on nack (err != nil) so they are retried on the next ReadBatch call. (`processing[t] = results; delete(in.store, t)
return batch, func(ctx context.Context, err error) error { if err != nil { in.mu.Lock(); in.store[t] = processing[t]; in.mu.Unlock() }; return nil }, nil`)
**gocron scheduler with leader-aware goroutine** — Scheduler-based inputs (prometheus, run_ai) create a gocron.Scheduler in the constructor, add a CronJob in Connect, then run a goroutine that calls scheduler.Start()/StopJobs() based on leader state polled every 1 second. (`go func() { running := false; for { select { case <-ctx.Done(): ...; case <-time.After(1*time.Second): if leaderelection.IsLeader(...) && !running { in.scheduler.Start(); running = true } } } }()`)
**message metadata decoration** — All inputs set msg.MetaSet() keys (scrape_time, scrape_interval, resource_type, etc.) on each service.Message so downstream Bloblang mappings can read them via @metadata. (`msg.MetaSet("scrape_time", t.Format(time.RFC3339))
msg.MetaSet("scrape_interval", in.interval.String())`)
**graceful shutdown via cancel + done channel** — Long-lived background goroutines (kubernetes manager, otel gRPC server) are stopped via a context cancel and a done channel closed when the goroutine exits. Close() blocks on <-in.done or ctx.Done(). (`in.cancel(); select { case <-in.done: ...; case <-ctx.Done(): return ctx.Err() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `kubernetes.go` | Registers kubernetes_resources BatchInput. Uses controller-runtime manager with a cache-backed client. Checks leader state in ReadBatch. | Cache must sync before ReadBatch returns data — handled by WaitForCacheSync in Connect. Manager runs in a goroutine; cancel/done pattern is required for Close to not leak it. |
| `otel_log.go` | Registers otel_log BatchInput backed by a gRPC server (OTLP Logs exporter). Uses an internal message.Transaction channel to bridge gRPC Export calls to ReadBatch. | Uses internal shutdown.Signaller from collector/benthos/internal/shutdown — do not replace with a plain cancel. Export blocks until ack is received or timeout; slow downstream output will trigger deadline errors upstream. |
| `prometheus.go` | Registers prometheus BatchInput that executes PromQL queries on a gocron schedule and buffers results in an in-memory map keyed by scrape time. | store is guarded by mu — always hold lock when reading or writing. Interval is derived by computing the gap between next two cron occurrences, not from a simple duration field. |
| `run_ai.go` | Registers run_ai BatchInput that scrapes Run:ai workload/pod metrics via the collector/benthos/input/runai sub-package. Nearly identical scheduler pattern to prometheus.go. | runai.Service handles metric chunking internally (max 9 per call). pageSize is validated 100–500 in the constructor — reject invalid values early. TimingMetrics and resourceTypeMetrics are wired from the RegisterBatchInput closure, not from the constructor. |
| `schedule.go` | Registers schedule BatchInput that wraps a child input and gates ReadBatch behind a time.Ticker. Decorates messages with schedule_time and schedule_interval metadata. | Uses conf.FieldInput to obtain an *service.OwnedInput — the child must be closed in Close(). Ticker fires non-blocking (default: case) so ReadBatch returns an empty batch when the interval has not elapsed. |

## Anti-Patterns

- Registering a new plugin outside init() — the Benthos plugin registry requires init-time registration before the process starts.
- Calling leaderelection.IsLeader inside the gocron task rather than the Connect goroutine — the scheduler must be started/stopped at the goroutine level, not per-task.
- Holding in.mu across a network call or ReadBatch return — the mutex guards only the in-memory store; IO must happen outside the lock.
- Returning an error from ReadBatch for transient per-resource failures — non-fatal errors should be logged and skipped; a returned error will halt the Benthos pipeline.
- Defining a new input without a dedicated XxxInputConfig() function — mixing field declarations inside the RegisterBatchInput closure makes the spec untestable.

## Decisions

- **All plugins use service.BatchInput (not service.Input) so they can return multiple messages per ReadBatch call.** — Polling sources (Kubernetes, Prometheus, Run:ai) naturally produce batches; BatchInput avoids a wrapper and exposes a single AckFunc for the whole batch.
- **Leader election state is polled from service.Resources generic map rather than a dedicated channel.** — service.Resources is already injected into every plugin by the Benthos framework; storing state there avoids a separate DI wire and keeps leader state accessible across plugins without coupling them.
- **Scheduler-based inputs buffer results in an in-memory map[time.Time][]T and drain on ReadBatch rather than sending results directly.** — Benthos calls ReadBatch in a tight loop; decoupling the cron-fired scrape from ReadBatch prevents blocking the scrape goroutine on downstream backpressure.

## Example: Minimal scheduled BatchInput with leader guard, nack-restore, and metadata decoration

```
package input

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/redpanda-data/benthos/v4/public/service"

	"github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"
)

func myInputConfig() *service.ConfigSpec {
// ...
```

<!-- archie:ai-end -->
