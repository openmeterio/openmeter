# sink-worker

<!-- archie:ai-start -->

> main.go entrypoint for the sink-worker binary, which sinks Kafka usage events into ClickHouse. Builds a lean Wire Application (no Migrator/Runner mixins) and runs app.Sink plus the telemetry server in a hand-built run.Group.

## Patterns

**Cancelable root context** — Unlike the other binaries, main() uses ctx, cancel := context.WithCancel(context.Background()) with defer cancel(), passed into Sink.Run(ctx). (`ctx, cancel := context.WithCancel(context.Background()); defer cancel()`)
**Minimal Application, no DB migration** — Application embeds only common.GlobalInitializer and exposes Sink, Streaming, TopicProvisioner, TopicResolver, FlushHandler, TelemetryServer; there is no Migrate step before running. (`Sink *sink.Sink; FlushHandler flushhandler.FlushEventHandler`)
**Sink + telemetry run.Group** — group.Add(app.Sink.Run/Close), group.Add(TelemetryServer.ListenAndServe/Shutdown), SignalHandler; group.Run(run.WithReverseShutdownOrder()) with lo.ErrorsAs[*run.SignalError] for signal detection. (`group.Add(func() error { return app.Sink.Run(ctx) }, func(err error) { _ = app.Sink.Close() })`)
**Wire provider subset for sinking** — wire.Build uses FieldsOf(...,"Sink"), common.Sink, common.SinkWorkerProvisionTopics, common.KafkaNamespaceResolver, common.WatermillNoPublisher (publisher-less), common.Streaming. (`common.WatermillNoPublisher`)
**Local NewLogger helper** — A binary-local NewLogger builds a slogmulti pipe with otelslog middleware (marked TODO: use the primary logger). (`slogmulti.Pipe(otelslog.ResourceMiddleware(res), otelslog.NewHandler)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Bootstrap, run Sink + telemetry server in run.Group; exits non-zero on non-signal/non-ErrServerClosed errors. | No app.Migrate call here; ctx is cancelable and feeds Sink.Run, so propagate it rather than context.Background(). |
| `wire.go` | Sink provider subset + local metadata/NewLogger helpers. | common.WatermillNoPublisher is intentional (sink only consumes); do not swap in a publisher provider. |
| `wire_gen.go` | Generated injector; DO NOT EDIT. | Regenerate via make generate. |
| `version.go` | ldflags version metadata. | Identical to other binaries. |

## Anti-Patterns

- Editing wire_gen.go instead of wire.go
- Adding a Migrator/Runner mixin or DB migration step the sink does not need
- Replacing context.WithCancel root ctx with context.Background()
- Wiring a Watermill publisher into a consume-only sink

## Decisions

- **sink-worker omits the Migrator and uses a cancelable context** — It only sinks events into ClickHouse and owns no Postgres schema; an explicit cancelable ctx drives clean Sink shutdown.

<!-- archie:ai-end -->
