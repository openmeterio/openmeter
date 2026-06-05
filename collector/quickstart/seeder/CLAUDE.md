# seeder

<!-- archie:ai-start -->

> Benthos (Redpanda Connect) stream config that generates synthetic CloudEvents usage data and POSTs them into OpenMeter for the quickstart demo. It is data/config only — no Go code; the behavior is entirely driven by the `benthos-collector` image consuming this `config.yaml`.

## Patterns

**Env-var-parameterized Benthos config** — Every tunable knob uses the `${VAR:default}` interpolation form so the same config works across compose/CI without edits. Counts, intervals, target URL, token, and logging are all env-driven. (`count: ${SEEDER_COUNT:0}; url: ${OPENMETER_URL:http://127.0.0.1:8888}/api/v1/events`)
**generate input + bloblang mapping emits CloudEvents** — The `generate` input's `mapping` Bloblang builds a CloudEvents 1.0 envelope into `root` with id/specversion/type/source/subject/time/data. New event shapes must keep the CloudEvents envelope fields intact. (`root = { "id": uuid_v4(), "specversion": "1.0", "type": $event_type, "source": $source, ... }`)
**switch output with continue-through HTTP + optional stdout** — Output is a `switch` whose first case (`check: ""`, `continue: true`) always POSTs to the events endpoint as `application/cloudevents+json`; a second case logs to stdout only when `SEEDER_LOG=true`. (`check: '"${SEEDER_LOG:false}" == "true"' -> stdout: { codec: lines }`)
**Backdated, bounded-random event fields** — `time` is randomized within the last 3 days (`ts_sub_iso8601("P3D")` + random seconds); `subject` is `customer-%d` modulo `SEEDER_MAX_SUBJECTS`. Keep new randomized fields seeded with `timestamp_unix_nano()` to avoid identical batches. (`let subject = "customer-%d".format(random_int(seed: timestamp_unix_nano()) % $max_subjects)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.yaml` | Whole-folder Benthos stream definition: http server on :4196 (matches the compose healthcheck `/ready`), generate input, output switch to the OpenMeter events API. | The HTTP port 4196 and `OPENMETER_URL` must stay aligned with collector/quickstart/docker-compose.yaml (seeder healthcheck hits :4196, and compose points OPENMETER_URL at the collector's :8889 ingest, not 8888). `max_in_flight: 1` serializes posts — raising it changes ingest ordering/load. |

## Anti-Patterns

- Hardcoding the OpenMeter URL, token, count, or interval instead of using `${VAR:default}` — breaks the env-driven compose/CI wiring.
- Changing the HTTP `address` port without updating the compose seeder healthcheck/port mapping (4196).
- Dropping required CloudEvents envelope fields (id/specversion/type/source/subject/time) — OpenMeter ingest expects valid CloudEvents.
- Sending a non-`application/cloudevents+json` Content-Type to /api/v1/events.

## Decisions

- **Use a second copy of the benthos-collector image (not custom Go) as the seeder.** — The quickstart only needs synthetic load; reusing the published image with a generate input avoids shipping a separate seeding binary.
- **Backdate event times across a 3-day window.** — Gives the quickstart dashboards/meters non-trivial historical data immediately rather than only live events.

<!-- archie:ai-end -->
