# seed

<!-- archie:ai-start -->

> Demo data seeding layer: Benthos/Redpanda Connect pipeline YAML configs that generate synthetic CloudEvents and POST them to OpenMeter's ingest API (/api/v1/events). Holds the top-level seed.yaml (token usage events) and observability.yaml; the streams/ child holds domain-specific generators (api-requests, prompt-tokens, workload-runtime) that share one skeleton.

## Patterns

**CloudEvents 1.0 envelope** — Every generated event sets specversion, id (uuid_v4()), type, source, subject, and time at the root; dimensional attributes live inside data: {}, never at envelope root. (`root = {"id": uuid_v4(), "specversion": "1.0", "type": $event_type, "source": $source, "subject": $subject, "time": $time, "data": {"model": $model, "tokens": $tokens}}`)
**Env-var parameterisation with defaults** — Every runtime value uses ${VAR:default} so files run out-of-the-box yet stay overridable in CI. (`url: ${OPENMETER_BASE_URL:http://127.0.0.1:8888}/api/v1/events
Authorization: "Bearer ${OPENMETER_TOKEN:}"`)
**Infinite generation with count: 0** — Seed streams use count: ${SEEDER_COUNT:0}; 0 means run indefinitely. A positive count terminates the stream and defeats the seeder. (`input:
  generate:
    count: ${SEEDER_COUNT:0}
    interval: "${SEEDER_INTERVAL:50ms}"`)
**Subject pool via modulo randomisation seeded from time** — Subjects are drawn from a bounded pool (customer-0..N) via random_int(seed: timestamp_unix_nano()) % $max_subjects; always seed from timestamp_unix_nano() — a fixed seed repeats. (`let subject = "customer-%d".format(random_int(seed: timestamp_unix_nano()) % $max_subjects)`)
**Dual-output switch: HTTP primary + optional stdout** — output.switch always has an unconditional HTTP POST case (continue: true) and a conditional stdout logger gated by SEEDER_LOG=true; both cases appear in every stream. (`output:
  switch:
    cases:
      - check: ""
        continue: true
        output: { http_client: ... }
      - check: '"${SEEDER_LOG:false}" == "true"'
        output: { stdout: { codec: lines } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `seed.yaml` | Top-level stream generating 'tokens' CloudEvents (provider/model/type/count dimensions); entry point for `benthos -c etc/seed/seed.yaml`. | count: ${SEEDER_COUNT:0} must stay 0-defaulted. The commented ts_sub_iso8601 backdating block is for historical seeding — leave commented unless intentional. |
| `observability.yaml` | OTel tracing config for the Benthos seed process; sends traces to the local collector on :4317 with sampling disabled by default. | sampling.enabled defaults to false — enabling it in dev floods the local collector with no user value. |
| `streams/` | Child: per-domain generators (api-requests, prompt-tokens, workload-runtime) reusing the seed.yaml skeleton. | Each stream must keep the identical input.generate + output.switch structure; structural deviation breaks batch seeding, and dimensions must stay inside data: {}. |

## Anti-Patterns

- Hardcoding OPENMETER_TOKEN or OPENMETER_BASE_URL — always use ${VAR:default}
- Setting count > 0 — seed streams must run indefinitely (count: 0 means infinite)
- Putting dimensional attributes outside data: {} — they belong in data, not at the CloudEvent envelope root
- Using a fixed random seed instead of timestamp_unix_nano() — produces a repeating non-random sequence
- Emitting a non-CloudEvents-1.0 envelope (omitting specversion, id, or time) — breaks ingest API validation

## Decisions

- **All streams share one input.generate + output.switch skeleton; only the bloblang mapping body varies** — Consistency makes adding new event types trivial and prevents structural drift that would break the Benthos pipeline runner.
- **Subject space bounded by SEEDER_SUBJECT_COUNT rather than purely random UUIDs** — Bounded subjects produce realistic per-customer usage curves and let meter aggregations return non-sparse results in the demo UI.

## Example: Add a new seed stream for a new event type

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
