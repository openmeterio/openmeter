# streams

<!-- archie:ai-start -->

> Benthos/Redpanda Connect stream pipelines that form the quickstart collector's ingestion path: input.yaml exposes an HTTP CloudEvents endpoint that validates/dedupes events, output.yaml buffers them and ships to OpenMeter. The two streams are wired together via the `openmeter` inproc channel.

## Patterns

**Two streams bridged by inproc channel** — input.yaml's output writes to `inproc: openmeter`; output.yaml's input reads from `inproc: openmeter`. The named inproc channel is the contract between the ingestion and delivery streams — both names must match. (`# input.yaml output
- inproc: openmeter
# output.yaml input
input:
  inproc: openmeter`)
**Content-Type switch for CloudEvents formats** — Incoming requests are routed by a `switch` processor on `meta("Content-Type")`: `application/cloudevents-batch+json` is unarchived from a json_array, `application/cloudevents+json` passes through, anything else returns an RFC7807-style 400. (`- check: meta("Content-Type").lowercase() == "application/cloudevents-batch+json"
  processors:
    - unarchive:
        format: json_array`)
**RFC7807 error responses via mapping + sync_response** — Validation failures build an about:blank problem-details JSON root, set `meta http_response_status` to 400, emit a sync_response, then `root = deleted()` to drop the message from the pipeline. (`meta http_response_status = "400"
root = {"type":"about:blank","title":"Bad Request","status":400,"detail":...}`)
**Schema validation with catch** — A json_schema processor validates against file://./cloudevents.spec.json; a following `catch` block converts any schema error into a 400 problem-details sync_response and deletes the message. (`- json_schema:
    schema_path: "file://./cloudevents.spec.json"
- catch:
    - mapping: meta http_response_status = "400"`)
**Durable SQLite buffer before delivery** — output.yaml inserts a `buffer: sqlite` at /var/lib/collector/buffer.sqlite (with a split post_processor) so accepted events survive collector restarts before reaching the openmeter output. (`buffer:
  sqlite:
    path: /var/lib/collector/buffer.sqlite`)
**Env-driven output config** — The openmeter output URL/token/batching are parameterized via ${OPENMETER_URL:...}, ${OPENMETER_TOKEN:}, ${BATCH_SIZE:1}, ${BATCH_PERIOD:30s}; a DEBUG_INPUT switch case can tee events to stdout. (`output:
  openmeter:
    url: "${OPENMETER_URL:https://openmeter.cloud}"
    token: "${OPENMETER_TOKEN:}"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `input.yaml` | HTTP CloudEvents ingestion stream: http_server on :8889 path /api/v1/events, Content-Type routing, dedupe, json_schema validation, then fan-out to inproc `openmeter` with a null sync_response (204). | Relies on `dedupe_cache` resource (resources/dedupe-cache.yaml) and a `cloudevents.spec.json` file resolved relative to the working dir. The fan_out broker order matters: sync_response is sent before inproc delivery. http_response_status meta defaults to 204 on success. |
| `output.yaml` | Delivery stream: reads inproc `openmeter`, buffers to durable SQLite, batches, and posts to the OpenMeter ingest endpoint using the custom `openmeter` output plugin. | The `openmeter` output is a custom Benthos plugin registered by the collector binary, not a stock Benthos output — it only resolves when running the OpenMeter collector build. Buffer path must be a writable mounted volume in docker-compose. Empty OPENMETER_TOKEN default means auth must be supplied via env. |

## Anti-Patterns

- Changing the inproc channel name in one stream but not the other — events silently stop flowing between input.yaml and output.yaml.
- Removing the `cache:` dedupe reference or renaming the cache without updating resources/dedupe-cache.yaml.
- Assuming `output: openmeter` is a stock Benthos output — it is a custom collector plugin and won't exist in a vanilla Benthos/Connect binary.
- Pointing the SQLite buffer at a non-persistent path, defeating the at-least-once durability the buffer provides across restarts.
- Returning bare error strings instead of the RFC7807 problem-details shape the existing error mappings produce.

## Decisions

- **Ingestion and delivery split into two streams joined by an inproc channel.** — Decouples synchronous HTTP request handling (validate + ack fast with 204) from durable, batched, retryable delivery to OpenMeter, so the HTTP caller is not blocked on upstream availability.
- **Durable SQLite buffer sits between the two streams.** — Provides at-least-once delivery: accepted events are persisted locally and survive collector restarts before being batched to the OpenMeter API.
- **Validation errors are returned as RFC7807 problem-details with explicit http_response_status meta.** — Matches OpenMeter's API error contract so clients posting CloudEvents get consistent, parseable Bad Request responses.

## Example: Add a new Content-Type branch or validation step to the ingestion pipeline

```
pipeline:
  processors:
    - switch:
        - check: meta("Content-Type").lowercase() == "application/cloudevents+json"
          processors:
            - noop: {}
        - check: ""
          processors:
            - mapping: |
                meta http_response_status = "400"
                root = {"type":"about:blank","title":"Bad Request","status":400,
                        "detail":"unexpected Content-Type %s".format(meta("Content-Type"))}
            - sync_response: {}
            - mapping: "root = deleted()"
    - dedupe:
// ...
```

<!-- archie:ai-end -->
