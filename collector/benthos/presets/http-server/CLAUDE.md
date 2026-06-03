# http-server

<!-- archie:ai-start -->

> Self-contained Benthos YAML pipeline preset that receives CloudEvents over HTTP, validates them against the CloudEvents JSON schema, buffers durably to SQLite, and forwards batches to the OpenMeter ingest API. No Go code — only YAML configuration with env-var substitution.

## Patterns

**sync_response via metadata** — HTTP status code and Content-Type for the synchronous response are set via Bloblang metadata (meta http_status_code, meta content_type) before sync_response: {}. The input reads these with .or() defaults so the block never hardcodes values. (`meta http_status_code = "400"
meta content_type = "application/problem+json"
- sync_response: {}`)
**catch-log-delete for validation failures** — After json_schema validation, a catch block must log the error, assign 400 status metadata, send an RFC-7807 problem+json sync_response, then drop the message with root = deleted(). All four steps are required. (`- catch:
  - log: {level: ERROR, message: "schema validation failed: ${!error()}"}
  - mapping: 'meta http_status_code = "400"'
  - sync_response: {}
  - mapping: root = deleted()`)
**SQLite buffer with split post-processor** — The buffer must be SQLite with a split post-processor capped at 100. Changing buffer type breaks at-least-once durability; removing split allows oversized batches to reach the output. (`buffer:
  sqlite:
    path: "${BUFFER_PATH:./buffer.db}"
    post_processors: [{split: {size: 100}}]`)
**drop_on 400 at output** — The http_client output is wrapped with drop_on: [400] to permanently discard events the OpenMeter API rejects as invalid, preventing infinite retry loops for structurally bad events. (`drop_on:
  output:
    http_client:
      drop_on: [400]`)
**counter metrics at input and output** — openmeter_event_received is incremented by a metric processor on every ingested message; openmeter_events_sent inside the output batching processors. Keep both in sync when modifying the pipeline. (`- metric:
    type: counter
    name: openmeter_event_received
    value: 1`)
**env-var substitution with defaults** — All operational parameters (URL, token, buffer path, log level) use ${VAR:default} substitution. Removing defaults breaks deployments that rely on them. (`url: "${OPENMETER_URL:https://openmeter.cloud}/api/v1/events"
Authorization: "Bearer ${OPENMETER_TOKEN:}"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.yaml` | Complete self-contained Benthos pipeline: HTTP input with CloudEvents schema validation, SQLite buffer, batched HTTP output to OpenMeter. The single deployable artifact for this preset. | The 204 meta http_status_code assignment and sync_response: {} must appear AFTER the validation catch block — reordering means callers receive no acknowledgement. The input's sync_response reads meta("http_status_code").or("204"), so both success and error branches must set the metadata first. |

## Anti-Patterns

- Adding transformation or enrichment logic — this preset is receive-validate-forward only
- Using a non-SQLite buffer without updating the split post-processor size
- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of using env-var substitution with defaults
- Removing the catch block after json_schema validation — unhandled validation errors stall the pipeline
- Placing sync_response: {} before the meta http_status_code assignment — callers receive 204 even for invalid events

## Decisions

- **SQLite buffer for durability.** — Ensures events are not lost if the OpenMeter output is temporarily unavailable, matching the at-least-once delivery guarantee of the broader ingest path.
- **Synchronous HTTP response before buffering.** — Clients receive immediate 204/400 feedback from the collector; the async buffer+forward path is invisible to the producer, keeping the ingest API contract intact.

<!-- archie:ai-end -->
