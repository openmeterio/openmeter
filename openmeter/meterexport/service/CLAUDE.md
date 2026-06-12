# service

<!-- archie:ai-start -->

> Concrete implementation (package meterexportservice) of the meterexport.Service interface. It re-queries existing meter values and re-emits them as synthetic streaming.RawEvent records, used to backfill/replay meter data into a target meter via a streaming, channel-based pipeline.

## Patterns

**Config-validated constructor returning interface conformance** — New(Config) validates required deps via Config.validate() before returning *service; package asserts var _ meterexport.Service = (*service)(nil). (`func New(config Config) (*service, error) { if err := config.validate(); err != nil { return nil, err }; return &service{Config: config}, nil }`)
**Channel-based streaming with deferred close by producer** — Exported export methods and funnel own their output channels and close them via a top-of-function defer (close(resultCh); close(errCh)). The function returns an error only for start-up failures; runtime errors go to errCh. (`func (s *service) funnel(...) error { defer func(){ close(resultCh); close(errCh) }(); ... }`)
**Producer/consumer split via errgroup** — ExportSyntheticMeterData uses errgroup.WithContext: one g.Go consumes meter.MeterQueryRow from meterRowCh and converts to events, another g.Go runs funnel as producer. g.Wait() joins them. (`g, ctx := errgroup.WithContext(ctx); g.Go(consumer); g.Go(func() error { return s.funnel(...) }); g.Wait()`)
**Single context-cancellation report via sync.Once** — Because both funnel and consumer can observe ctx.Done(), sendCtxErr is guarded by sync.Once so exactly one context.Canceled/DeadlineExceeded reaches errCh. Consumer filters context errors out of meterRowErrCh and relies on sendCtxErr instead. (`var sendCtxErrOnce sync.Once; sendCtxErr := func(){ sendCtxErrOnce.Do(func(){ if err := ctx.Err(); err != nil { errCh <- err } }) }`)
**Validate-then-fetch shared helper** — validateAndGetMeter(config) centralizes config.Validate(), meter lookup via MeterService.GetMeterByIDOrSlug, aggregation guard (only Sum/Count), and TargetMeterDescriptor construction. GetTargetMeterDescriptor and ExportSyntheticMeterData both call it. (`m, descriptor, err := s.validateAndGetMeter(ctx, config)`)
**Batched time-window iteration capped at TARGET_ROWS_PER_QUERY** — funnel walks [from,to) in slices computed by iterateQueryTime, advancing windowSize.AddTo up to TARGET_ROWS_PER_QUERY (500) windows per QueryMeter call, clamping to params.To. (`queryTo, err := iterateQueryTime(queryFrom, params.queryParams.To, *params.queryParams.WindowSize)`)
**Iterator wrapper cancels underlying op on early break** — ExportSyntheticMeterDataIter validates upfront, then returns an iter.Seq2[streaming.RawEvent, error] backed by a cancellable context; a failed yield returns and defer cancel() stops the goroutine-driven export. (`ctx, cancel := context.WithCancel(ctx); defer cancel(); if !yield(event, nil) { return }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Config (EventSourceGroup, StreamingConnector, MeterService), service struct embedding Config, New constructor, and the meterexport.Service interface assertion. | All three Config fields are required; New returns error if any is missing. Keep the var _ meterexport.Service assertion satisfied when adding interface methods. |
| `syntheticdata.go` | Core export: validateAndGetMeter, GetTargetMeterDescriptor, ExportSyntheticMeterData (errgroup producer/consumer), createEventFromMeterRow, and SUM_VALUE_PROPERTY_KEY const. | Only Sum/Count aggregations are supported; both map to a SUM descriptor with ValueProperty 'value'. Event Source is fmt.Sprintf("%s:%s/%s", EventSourceGroup, ns, id). Do not break the sendCtxErr sync.Once guard or you'll emit duplicate context errors. |
| `funnel.go` | Producer: funnelParams + validation, funnel() streaming meter query loop, iterateQueryTime time-window stepper, TARGET_ROWS_PER_QUERY const. | funnelParams.validate rejects unsupported params (ClientID, FilterCustomer, FilterGroupBy) and requires From + WindowSize. funnel checks ctx.Err() each iteration and pushes to errCh then returns nil (no return error mid-stream). |
| `syntheticdata_iter.go` | Ergonomic iter.Seq2 wrapper over ExportSyntheticMeterData with early-break cancellation and result/error interleaving plus a startupErrCh. | Drains both resultCh and errCh on close to avoid dropping tail events/errors; nils out startupErrCh after close to stop reselecting it. |
| `service_test.go` | Table-driven tests using MockMeterService and testutils.NewMockStreamingConnector/AddSimpleEvent; covers SUM/COUNT export, unsupported-aggregation rejection, validation, not-found, context cancellation (exactly-one context.Canceled), and New validation. | Context-cancellation tests assert contextCanceledCount == 1 — regressions in sendCtxErr's sync.Once will fail here. Mock streaming events are matched by MeterSlug == meter.Key. |

## Anti-Patterns

- Closing resultCh/errCh anywhere other than the producer's top-level defer — double-close panics or leaks.
- Returning a runtime error from funnel/export mid-stream instead of sending it to errCh — callers consume errors off the channel, not the return value.
- Bypassing validateAndGetMeter and querying the meter directly — skips the Sum/Count aggregation guard and descriptor construction.
- Removing the sendCtxErr sync.Once guard or letting both funnel and consumer push ctx.Err() to errCh — produces duplicate context.Canceled reports.
- Adding support for ClientID/FilterCustomer/FilterGroupBy query params without removing the explicit rejects in validateUnsupportedParams.

## Decisions

- **Stream via channels + errgroup rather than collecting a slice.** — Exports can span large time ranges (funnel batches 500 windows per query); streaming bounds memory and lets callers backpressure or cancel mid-flight.
- **Normalize Sum and Count source meters to a single SUM target descriptor with value property 'value'.** — Both aggregations can be replayed as additive SUM events, so a synthetic target meter only needs one value-bearing property; other aggregations are rejected up front.
- **Provide both a channel API (ExportSyntheticMeterData) and an iter.Seq2 wrapper (ExportSyntheticMeterDataIter).** — The channel form integrates with worker pipelines; the iterator form gives ergonomic range-over-func consumption with automatic cancellation on early break.

## Example: Producer/consumer export with single-shot context error reporting

```
g, ctx := errgroup.WithContext(ctx)
var sendCtxErrOnce sync.Once
sendCtxErr := func() { sendCtxErrOnce.Do(func() { if err := ctx.Err(); err != nil { errCh <- err } }) }
g.Go(func() error {
  for {
    select {
    case <-ctx.Done(): sendCtxErr(); return nil
    case row, ok := <-meterRowCh:
      if !ok { sendCtxErr(); return nil }
      event, err := s.createEventFromMeterRow(m, row)
      if err != nil { errCh <- fmt.Errorf("create event from meter row: %w", err); continue }
      resultCh <- event
    }
  }
})
// ...
```

<!-- archie:ai-end -->
