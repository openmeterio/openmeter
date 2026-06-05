# streams

<!-- archie:ai-start -->

> Benthos/Redpanda Connect stream configs that generate synthetic CloudEvents and POST them to the OpenMeter ingest API for local demo/seeding. Each file models one usage-metering scenario (HTTP API requests, LLM prompt tokens, cloud workload runtime).

## Patterns

**Generate-input + Bloblang mapping** — Each file uses input.generate with count:0 (run forever), interval from ${SEEDER_INTERVAL:100ms}, and a Bloblang `mapping` block that builds the event root. (`input:
  generate:
    count: 0
    interval: "${SEEDER_INTERVAL:100ms}"`)
**CloudEvents 1.0 envelope** — root must be a CloudEvent: id (uuid_v4()), specversion "1.0", type, source "demo-org-seeder", subject, time (now()), and a scenario-specific data object. (`root = { "id": uuid_v4(), "specversion": "1.0", "type": $event_type, "source": $source, ... }`)
**Subject fan-out via SEEDER_SUBJECT_COUNT** — subject is `demo-subject-%d` modulo ${SEEDER_SUBJECT_COUNT:1000} so events spread across a configurable number of synthetic customers. (`let subject = "demo-subject-%d".format(random_int(seed: timestamp_unix_nano()) % $subject_count)`)
**Random selection from literal lists** — Categorical fields (methods, routes, models, regions, instance_types) are chosen with `$list.index(random_int(seed: timestamp_unix_nano()) % $list.length())`. (`let model = $models.index(random_int(seed: timestamp_unix_nano()) % $models.length())`)
**Switch output: ingest + optional log** — output.switch has two cases: an always-true case (check: "") with continue:true that POSTs to the events endpoint, then a gated stdout case enabled by SEEDER_LOG. (`- check: ""
  continue: true
  output:
    http_client: { ... }
- check: '"${SEEDER_LOG:false}" == "true"'
  output:
    stdout: { codec: lines }`)
**HTTP client to /api/v1/events** — POST to ${OPENMETER_BASE_URL:http://127.0.0.1:8888}/api/v1/events with Content-Type application/cloudevents+json, Bearer ${OPENMETER_TOKEN:}, max_in_flight 256. (`http_client:
  url: ${OPENMETER_BASE_URL:http://127.0.0.1:8888}/api/v1/events
  verb: POST
  headers: { Content-Type: application/cloudevents+json, Authorization: "Bearer ${OPENMETER_TOKEN:}" }`)
**Env-var defaults for everything** — All tunables use ${VAR:default} syntax (SEEDER_INTERVAL, SEEDER_SUBJECT_COUNT, SEEDER_LOG, OPENMETER_BASE_URL, OPENMETER_TOKEN) so a stream runs with zero config. (`interval: "${SEEDER_INTERVAL:100ms}"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api-requests.yaml` | Generates `request` events with data.method (GET/POST) and data.path (one of /, /about, /contact, /pricing, /docs). | type must stay "request" to match the corresponding meter/event-type definition consuming it. |
| `prompt-tokens.yaml` | Generates `prompt` events with data.tokens (random_int min:100 max:1000), data.model (OpenAI model list), data.type (input/output). | data.tokens is the numeric value meters aggregate; keep it under the `tokens` key and named model list aligned with demo meters. |
| `workload-runtime.yaml` | Generates `workload` events with data.duration_seconds (random_int max:1000), data.region, data.zone (region+suffix), data.instance_type. | zone is composed as region+suffix via %s%s.format; duration_seconds has no min so 0 is possible. |

## Anti-Patterns

- Hardcoding the OpenMeter URL or token instead of using ${OPENMETER_BASE_URL}/${OPENMETER_TOKEN} env defaults.
- Omitting required CloudEvents envelope fields (id, specversion, type, source, subject, time) — ingest rejects malformed events.
- Sending Content-Type other than application/cloudevents+json for the JSON CloudEvent body.
- Dropping the continue:true on the ingest case, which would short-circuit the optional SEEDER_LOG stdout output.
- Setting generate.count to a non-zero value, which stops the stream after N events instead of seeding continuously.

## Decisions

- **Use Benthos/Redpanda Connect generate-input streams rather than a custom seeder binary.** — Declarative Bloblang config makes synthetic-event scenarios easy to author and tune via env vars without code.
- **One file per usage-metering scenario sharing an identical envelope + output structure.** — Each demonstrates a distinct meter type (requests, tokens, runtime) while keeping ingest wiring uniform and copy-pasteable for new scenarios.

## Example: Add a new seed stream: continuous CloudEvents generator POSTing to the ingest API

```
input:
  generate:
    count: 0
    interval: "${SEEDER_INTERVAL:100ms}"
    mapping: |
      let subject_count = ${SEEDER_SUBJECT_COUNT:1000}
      let subject = "demo-subject-%d".format(random_int(seed: timestamp_unix_nano()) % $subject_count)
      root = {
        "id": uuid_v4(),
        "specversion": "1.0",
        "type": "my_event",
        "source": "demo-org-seeder",
        "subject": $subject,
        "time": now(),
        "data": { "value": random_int(seed: timestamp_unix_nano(), max: 1000) },
// ...
```

<!-- archie:ai-end -->
