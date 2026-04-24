# seed

<!-- archie:ai-start -->

> Demo data seeding layer for OpenMeter. Contains Benthos/Redpanda Connect pipeline configurations that generate synthetic CloudEvents and POST them to the ingest API, plus a child `streams/` folder with domain-specific event generators (api-requests, prompt-tokens, workload-runtime).

## Patterns

**Environment-variable parameterisation with defaults** — Every runtime value (URL, token, count, interval, subject count) uses ${VAR:default} syntax so the file works out-of-the-box without env vars but is overridable in CI or production. (`url: ${OPENMETER_BASE_URL:http://127.0.0.1:8888}/api/v1/events`)
**CloudEvents 1.0 envelope** — Every generated event must include specversion, id (uuid_v4()), type, source, subject, time at root. Dimensional/domain attributes belong inside data: {}, not at the envelope root. (`root = {"id": uuid_v4(), "specversion": "1.0", "type": $event_type, "source": $source, "subject": $subject, "time": $time, "data": {"model": $model, "tokens": $tokens}}`)
**Infinite generation with count: 0** — All seed streams use count: ${SEEDER_COUNT:0} so they run indefinitely by default. count > 0 terminates the stream and breaks the seeder's purpose. (`input:
  generate:
    count: ${SEEDER_COUNT:0}
    interval: "${SEEDER_INTERVAL:50ms}"`)
**Subject pool via modulo randomisation with timestamp seed** — Subjects are drawn from a bounded pool (customer-0..N) using random_int(seed: timestamp_unix_nano()) % $max_subjects. Always seed from timestamp_unix_nano() — a fixed seed produces a non-random sequence. (`let subject = "customer-%d".format(random_int(seed: timestamp_unix_nano()) % $max_subjects)`)
**Dual-output switch: HTTP primary + optional stdout log** — output.switch always has at least two cases: an unconditional HTTP POST to the ingest API and a conditional stdout logger gated by SEEDER_LOG=true. (`output:
  switch:
    cases:
      - check: ""
        continue: true
        output:
          http_client: ...
      - check: '"${SEEDER_LOG:false}" == "true"'
        output:
          stdout: ...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `etc/seed/seed.yaml` | Top-level seed stream for token usage events. Generates 'tokens' CloudEvents with provider/model/type/count dimensions. Entry point for `benthos -c etc/seed/seed.yaml`. | count: ${SEEDER_COUNT:0} must stay 0-defaulted. The commented-out time backdating block shows how to generate historical data — keep it commented unless historical seeding is intentional. |
| `etc/seed/observability.yaml` | OTel tracing config for the Benthos seed process itself. Sends traces to a local collector on :4317 with sampling disabled by default. | sampling.enabled defaults to false — enabling it in dev adds no user value and can flood a local collector. |
| `etc/seed/streams/api-requests.yaml` | Generates synthetic API request events (child stream). | Must follow identical input.generate + output.switch structure as seed.yaml. |
| `etc/seed/streams/prompt-tokens.yaml` | Generates LLM prompt token events (child stream). | Token counts must be in data: {} not at envelope root. |
| `etc/seed/streams/workload-runtime.yaml` | Generates compute workload runtime events (child stream). | Same structural contract as other streams — deviation breaks batch seeding. |

## Anti-Patterns

- Hardcoding OPENMETER_TOKEN or OPENMETER_BASE_URL — always use ${VAR:default}
- Setting count > 0 — seed streams must run indefinitely
- Putting dimensional attributes outside data: {} (they belong in data, not at CloudEvent root)
- Using a fixed random seed instead of timestamp_unix_nano() — produces a repeating non-random sequence
- Adding a non-CloudEvents-1.0 envelope (omitting specversion, id, or time breaks the ingest API validation)

## Decisions

- **All streams share the identical input.generate + output.switch skeleton; only the bloblang mapping body varies.** — Consistency makes it trivial to add new event types without introducing structural drift that would break the Benthos pipeline runner.
- **Subject space is bounded by SEEDER_SUBJECT_COUNT rather than purely random UUIDs.** — Bounded subjects produce realistic per-customer usage curves and allow meter aggregations to produce non-trivial, non-sparse results in the demo UI.

## Example: Adding a new seed stream for a new event type

```
# etc/seed/streams/my-event.yaml
input:
  generate:
    count: ${SEEDER_COUNT:0}
    interval: "${SEEDER_INTERVAL:50ms}"
    mapping: |
      let max_subjects = ${SEEDER_MAX_SUBJECTS:20}
      let subject = "customer-%d".format(random_int(seed: timestamp_unix_nano()) % $max_subjects)
      root = {
        "id": uuid_v4(),
        "specversion": "1.0",
        "type": "my-event-type",
        "source": "my-source",
        "subject": $subject,
        "time": now().ts_format(),
// ...
```

<!-- archie:ai-end -->
