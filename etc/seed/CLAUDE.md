# seed

<!-- archie:ai-start -->

> Demo data seeding layer: Benthos/Redpanda Connect pipeline configs that generate synthetic CloudEvents and POST them to the OpenMeter ingest API. Contains the top-level seed.yaml (token usage events) and child streams/ with domain-specific event generators (api-requests, prompt-tokens, workload-runtime).

## Patterns

**CloudEvents 1.0 envelope** — Every generated event must have specversion, id (uuid_v4()), type, source, subject, and time at the root. Domain/dimensional attributes belong inside data: {}, never at envelope root. (`root = {"id": uuid_v4(), "specversion": "1.0", "type": $event_type, "source": $source, "subject": $subject, "time": $time, "data": {"model": $model, "tokens": $tokens}}`)
**Environment-variable parameterisation with defaults** — Every runtime value (URL, token, count, interval, subject count) uses ${VAR:default} syntax so files work out-of-the-box without env vars and are overridable in CI. (`url: ${OPENMETER_BASE_URL:http://127.0.0.1:8888}/api/v1/events
Authorization: "Bearer ${OPENMETER_TOKEN:}"`)
**Infinite generation with count: 0 default** — All seed streams use count: ${SEEDER_COUNT:0}. count: 0 means run indefinitely. count > 0 terminates the stream and breaks the seeder's purpose. (`input:
  generate:
    count: ${SEEDER_COUNT:0}
    interval: "${SEEDER_INTERVAL:50ms}"`)
**Subject pool via modulo randomisation with timestamp seed** — Subjects are drawn from a bounded pool (customer-0..N) using random_int(seed: timestamp_unix_nano()) % $max_subjects. Always seed from timestamp_unix_nano() — a fixed seed produces a repeating non-random sequence. (`let subject = "customer-%d".format(random_int(seed: timestamp_unix_nano()) % $max_subjects)`)
**Dual-output switch: HTTP primary + optional stdout log** — output.switch always has an unconditional HTTP POST case (continue: true) and a conditional stdout logger gated by SEEDER_LOG=true. Both cases are required in every stream. (`output:
  switch:
    cases:
      - check: ""
        continue: true
        output:
          http_client: ...
      - check: '"${SEEDER_LOG:false}" == "true"'
        output:
          stdout:
            codec: lines`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `seed.yaml` | Top-level seed stream generating 'tokens' CloudEvents with provider/model/type/count dimensions. Entry point for `benthos -c etc/seed/seed.yaml`. | count: ${SEEDER_COUNT:0} must stay 0-defaulted. The commented-out backdating block (ts_sub_iso8601) shows how to generate historical data — leave commented unless historical seeding is intentional. |
| `observability.yaml` | OTel tracing config for the Benthos seed process itself. Sends traces to local collector on :4317 with sampling disabled by default. | sampling.enabled defaults to false — enabling it in dev floods a local collector with no user value. |
| `streams/api-requests.yaml` | Child stream generating synthetic API request events. | Must follow identical input.generate + output.switch skeleton as seed.yaml; structural deviation breaks batch seeding. |
| `streams/prompt-tokens.yaml` | Child stream generating LLM prompt token events. | Token counts must be inside data: {} not at CloudEvent envelope root. |
| `streams/workload-runtime.yaml` | Child stream generating compute workload runtime events. | Same structural contract as other streams — any deviation breaks the batch seeder. |

## Anti-Patterns

- Hardcoding OPENMETER_TOKEN or OPENMETER_BASE_URL — always use ${VAR:default}
- Setting count > 0 — seed streams must run indefinitely (count: 0 means infinite)
- Putting dimensional attributes outside data: {} — they belong in data, not at CloudEvent envelope root
- Using a fixed random seed instead of timestamp_unix_nano() — produces a repeating non-random sequence
- Adding a non-CloudEvents-1.0 envelope (omitting specversion, id, or time breaks ingest API validation)

## Decisions

- **All streams share identical input.generate + output.switch skeleton; only the bloblang mapping body varies.** — Consistency makes adding new event types trivial and prevents structural drift that would break the Benthos pipeline runner.
- **Subject space bounded by SEEDER_SUBJECT_COUNT rather than purely random UUIDs.** — Bounded subjects produce realistic per-customer usage curves and allow meter aggregations to return non-trivial, non-sparse results in the demo UI.

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
