# streams

<!-- archie:ai-start -->

> Benthos/Redpanda Connect pipeline configs for seeding demo CloudEvents into OpenMeter's ingest API. Each YAML file is a self-contained stream definition that generates synthetic events at a configurable interval and POSTs them as CloudEvents JSON to /api/v1/events.

## Patterns

**CloudEvents 1.0 envelope** — Every generated event must include id (uuid_v4()), specversion ("1.0"), type, source, subject, and time at the root level. Domain-specific fields go inside data: {}. (`root = {"id": uuid_v4(), "specversion": "1.0", "type": $event_type, "source": $source, "subject": $subject, "time": $time, "data": {...}}`)
**Environment-variable parameterisation with defaults** — All tuneable values use ${ENV_VAR:default} syntax — never hardcode intervals, counts, URLs, or tokens. Standard vars: SEEDER_INTERVAL (default 100ms), SEEDER_SUBJECT_COUNT (default 1000), OPENMETER_BASE_URL, OPENMETER_TOKEN. (`interval: "${SEEDER_INTERVAL:100ms}"`)
**Subject pool via modulo randomisation** — Subjects are selected by random_int(seed: timestamp_unix_nano()) % $subject_count, formatted as "demo-subject-%d". This produces a bounded, repeatable subject space driven by SEEDER_SUBJECT_COUNT. (`let subject = "demo-subject-%d".format(random_int(seed: timestamp_unix_nano()) % $subject_count)`)
**Dual-output switch: HTTP primary + optional stdout log** — Output is always a switch with two cases: (1) unconditional http_client POST to /api/v1/events with Content-Type: application/cloudevents+json and Bearer auth, continue: true; (2) conditional stdout when SEEDER_LOG=true. (`output:
  switch:
    cases:
      - check: ""
        continue: true
        output:
          http_client: ...
      - check: '"${SEEDER_LOG:false}" == "true"'
        output:
          stdout: ...`)
**Infinite generation with count: 0** — All streams use count: 0 (run forever) under input.generate. Do not set a finite count for seed streams. (`input:
  generate:
    count: 0`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api-requests.yaml` | Generates HTTP-request events (type: "request") with method and path data fields. Demonstrates the minimal CloudEvents + data structure. | data fields are method and path — meter definitions that consume this stream must match those exact field names. |
| `prompt-tokens.yaml` | Generates LLM token events (type: "prompt") with tokens, model, and type (input/output) data fields. Models list must stay aligned with llmcost price definitions. | Model strings here are compared against llmcost price records; mismatches silently produce unresolved costs. |
| `workload-runtime.yaml` | Generates compute-workload events (type: "workload") with duration_seconds, region, zone, and instance_type data fields. | zone is constructed as region+suffix (e.g. "us-east-1a"), not an independent random pick — meter group-by on zone implicitly includes region. |

## Anti-Patterns

- Hardcoding OPENMETER_TOKEN or OPENMETER_BASE_URL — always use ${VAR:default}
- Adding a non-CloudEvents-1.0 envelope (missing specversion, id, or time at root)
- Setting count > 0 — seed streams must run indefinitely
- Putting dimensional attributes outside the data: {} block (they belong in data, not at CloudEvent root)
- Using a fixed random seed instead of timestamp_unix_nano() — produces non-random sequences

## Decisions

- **All three files share identical input.generate + output.switch structure with only the mapping body varying.** — Uniformity lets the seed.yaml orchestrator run them identically; operators only need to learn one config shape.
- **Subject space is bounded by SEEDER_SUBJECT_COUNT rather than fully random UUIDs.** — Bounded subjects produce realistic per-subject meter aggregations in ClickHouse and allow demo dashboards to show meaningful per-customer breakdowns.

## Example: Add a new seed stream for storage events

```
input:
  generate:
    count: 0
    interval: "${SEEDER_INTERVAL:100ms}"
    mapping: |
      let subject_count = ${SEEDER_SUBJECT_COUNT:1000}
      let event_type = "storage"
      let source = "demo-org-seeder"
      let subject = "demo-subject-%d".format(random_int(seed: timestamp_unix_nano()) % $subject_count)
      let time = now()
      let bytes = random_int(seed: timestamp_unix_nano(), min: 1024, max: 1048576)
      root = {
        "id": uuid_v4(),
        "specversion": "1.0",
        "type": $event_type,
// ...
```

<!-- archie:ai-end -->
