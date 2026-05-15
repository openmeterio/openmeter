# streams

<!-- archie:ai-start -->

> Defines the two-stream Benthos pipeline for the quickstart collector: input.yaml handles HTTP ingestion (Content-Type switching, deduplication, schema validation, sync HTTP responses); output.yaml reads from the shared inproc channel, buffers to SQLite for durability, and forwards to the OpenMeter API in batches.

## Patterns

**Inproc channel coupling between streams** — input.yaml writes validated events to `inproc: openmeter`; output.yaml reads from `inproc: openmeter`. The label string must match exactly — mismatch silently drops all events. (`# input.yaml output section
- inproc: openmeter
# output.yaml input section
input:
  inproc: openmeter`)
**Content-Type switch before dedup/validation** — A pipeline switch processor splits `application/cloudevents-batch+json` (unarchive json_array) from `application/cloudevents+json` (noop). Unknown Content-Types return HTTP 400 via sync_response and drop the message using `mapping: root = deleted()`. (`processors:
  - switch:
      - check: meta("Content-Type").lowercase() == "application/cloudevents-batch+json"
        processors:
          - unarchive:
              format: json_array`)
**Sync response for HTTP status propagation** — HTTP response status is set via `meta http_response_status` and surfaced through the fan_out output's `sync_response: {}` branch. The address block echoes the meta value, defaulting to 204. (`sync_response:
  status: '${! meta("http_response_status").or("204") }'`)
**SQLite buffer for durable output** — output.yaml uses a SQLite buffer with a split post-processor to re-expand batched messages before forwarding to the openmeter output plugin. Removing `split: {}` sends unexpanded batch payloads. (`buffer:
  sqlite:
    path: /var/lib/collector/buffer.sqlite
    post_processors:
      - split: {}`)
**Environment variable configuration for output** — All deployment-specific values (OPENMETER_URL, OPENMETER_TOKEN, BATCH_SIZE, BATCH_PERIOD) are injected via environment variables with safe defaults. Never hardcode these values. (`output:
  openmeter:
    url: "${OPENMETER_URL:https://openmeter.cloud}"
    token: "${OPENMETER_TOKEN:}"
    batching:
      count: ${BATCH_SIZE:1}
      period: ${BATCH_PERIOD:30s}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `input.yaml` | HTTP server entry point: receives CloudEvents, enforces Content-Type, deduplicates via dedupe_cache, validates against CloudEvents JSON schema, and fans out to inproc channel + sync_response. | The dedupe processor references `dedupe_cache` by label — that label must exist in resources/dedupe-cache.yaml. The catch block after json_schema validation returns 400 and drops invalid events; removing catch leaves schema errors unhandled and events pass through. |
| `output.yaml` | Reads from inproc channel, buffers to SQLite for durability, then batches and forwards to the openmeter output plugin. | The `split: {}` post-processor is required to re-expand messages packed by the SQLite buffer; removing it sends unexpanded batch payloads to OpenMeter. |

## Anti-Patterns

- Renaming `inproc: openmeter` in either file without updating the other — breaks the inter-stream channel silently.
- Removing the fan_out sync_response branch — HTTP clients will never receive responses.
- Bypassing the Content-Type switch and sending raw bytes directly to dedupe — dedup hash computed on non-normalized payloads produces incorrect deduplication.
- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of using environment variable interpolation.
- Removing the SQLite buffer without an alternative durability mechanism — events lost on collector restart.

## Decisions

- **Two-file stream split (input + output) connected by inproc channel.** — Separates HTTP ingestion concerns (validation, dedup, sync response) from forwarding concerns (buffering, batching, auth), making each independently configurable.
- **xxhash64 content hash as dedupe key.** — Fast non-cryptographic hash over full message content provides exact-once deduplication for retried CloudEvent payloads without requiring a unique event ID field in the payload.

## Example: Full input pipeline: Content-Type switch → dedupe → schema validation → catch → fan_out with sync_response and inproc

```
# input.yaml
pipeline:
  processors:
    - switch:
        - check: meta("Content-Type").lowercase() == "application/cloudevents-batch+json"
          processors:
            - unarchive:
                format: json_array
        - check: meta("Content-Type").lowercase() == "application/cloudevents+json"
          processors:
            - noop: {}
        - check: ""
          processors:
            - mapping: |
                meta http_response_status = "400"
// ...
```

<!-- archie:ai-end -->
