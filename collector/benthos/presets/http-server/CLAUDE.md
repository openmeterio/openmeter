# http-server

<!-- archie:ai-start -->

> Benthos pipeline preset that receives CloudEvents via HTTP, validates them against the CloudEvents JSON schema, buffers to SQLite, and forwards batches to the OpenMeter ingest API. It is a self-contained operational config — no Go code, only YAML.

## Patterns

**sync_response via metadata** — HTTP status code and Content-Type for the synchronous response are set via Bloblang metadata assignments (`meta http_status_code`, `meta content_type`) before calling `sync_response: {}`. Never hardcode status in the sync_response block. (`mapping: |
  meta http_status_code = "400"
  meta content_type = "application/problem+json"`)
**catch-log-delete for validation failures** — After json_schema validation, a `catch` block logs the error, sends a 400 sync_response with RFC-7807 problem+json body, then drops the message with `root = deleted()`. All three steps must be present. (`- catch:
  - log: ...
  - mapping: root = { "type": "about:blank", ... }
  - sync_response: {}
  - mapping: root = deleted()`)
**SQLite buffer with post-processor split** — The buffer is always SQLite with a `split` post-processor capped at 100 to prevent oversized batches from reaching the output. Changing buffer type breaks the durability guarantee. (`buffer:
  sqlite:
    path: "${BUFFER_PATH:./buffer.db}"
    post_processors:
      - split:
          size: 100`)
**drop_on 400 at output** — The http_client output wraps with `drop_on` to discard permanently-invalid events (HTTP 400) rather than retrying them infinitely. This mirrors the input-side validation behaviour. (`drop_on:
  - 400`)
**counter metrics at input and output** — A `metric` processor increments `openmeter_event_received` on every ingested message and `openmeter_events_sent` inside the output batching processors. Both counters must be kept in sync when the pipeline is modified. (`- metric:
    type: counter
    name: openmeter_event_received
    value: 1`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.yaml` | Complete self-contained Benthos pipeline: input (HTTP server), CloudEvents schema validation, SQLite buffer, batched HTTP output to OpenMeter. | All configuration values are env-var substitutions with defaults (e.g. `${OPENMETER_URL:https://openmeter.cloud}`). Removing defaults will break deployments that rely on them. The `sync_response` in the input MUST come after the 204 meta assignment or callers receive no response. |

## Anti-Patterns

- Adding business logic (transformation, enrichment) inside the http-server preset — it is a receive-validate-forward pipeline only
- Using a non-SQLite buffer type without updating the post-processor split
- Hardcoding the OpenMeter URL or token instead of using environment variable substitution
- Removing the catch block after json_schema validation — unhandled validation errors will stall the pipeline

## Decisions

- **SQLite buffer for durability** — Ensures events are not lost if the output is temporarily unavailable, matching the at-least-once delivery guarantee of the broader OpenMeter ingest path.
- **Synchronous HTTP response before buffering** — Clients get immediate 204/400 feedback from the collector; the async buffer+forward path is invisible to the producer, keeping the ingest API contract intact.

<!-- archie:ai-end -->
