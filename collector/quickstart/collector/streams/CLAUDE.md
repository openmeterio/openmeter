# streams

<!-- archie:ai-start -->

> Defines the two-stream Benthos pipeline for the quickstart collector: input.yaml receives CloudEvents over HTTP, deduplicates and validates them, then passes to an inproc channel; output.yaml reads from that inproc channel, buffers to SQLite, and forwards to the OpenMeter API in batches.

## Patterns

**Inproc channel coupling between streams** — input.yaml writes validated events to `inproc: openmeter`; output.yaml reads from `inproc: openmeter`. The string label must match exactly across both files. (`# input.yaml output section
- inproc: openmeter
# output.yaml input section
input:
  inproc: openmeter`)
**Content-Type switch before dedup/validation** — A pipeline switch processor splits `application/cloudevents-batch+json` (unarchive json_array) from `application/cloudevents+json` (noop). Unknown Content-Types return HTTP 400 via sync_response and drop the message. (`processors:
  - switch:
      - check: meta("Content-Type").lowercase() == "application/cloudevents-batch+json"
        processors:
          - unarchive:
              format: json_array`)
**Sync response for HTTP status propagation** — HTTP response status is set via `meta http_response_status` and surfaced through the fan_out output's `sync_response: {}` branch. The address block echoes `${! meta("http_response_status").or("204") }`. (`sync_response:
  status: '${! meta("http_response_status").or("204") }'`)
**SQLite buffer for durable output** — output.yaml uses a SQLite buffer at `/var/lib/collector/buffer.sqlite` with a split post-processor to re-expand batched messages before forwarding. (`buffer:
  sqlite:
    path: /var/lib/collector/buffer.sqlite
    post_processors:
      - split: {}`)
**Environment variable configuration for output** — All deployment-specific values (OPENMETER_URL, OPENMETER_TOKEN, BATCH_SIZE, BATCH_PERIOD) are injected via environment variables with safe defaults. (`output:
  openmeter:
    url: "${OPENMETER_URL:https://openmeter.cloud}"
    token: "${OPENMETER_TOKEN:}"
    batching:
      count: ${BATCH_SIZE:1}
      period: ${BATCH_PERIOD:30s}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `input.yaml` | HTTP server entry point: receives CloudEvents, enforces Content-Type, deduplicates via dedupe_cache, validates against CloudEvents JSON schema, and fans out to inproc channel + sync_response. | The dedupe processor references `dedupe_cache` by label — that label must exist in resources/dedupe-cache.yaml. The catch block after json_schema validation returns 400 and drops invalid events; missing catch leaves schema errors unhandled. |
| `output.yaml` | Reads from inproc channel, buffers to SQLite for durability, then batches and forwards to the openmeter output plugin. | The `split: {}` post-processor is required to re-expand messages packed by the SQLite buffer; removing it sends unexpanded batch payloads to OpenMeter. |

## Anti-Patterns

- Renaming the `inproc: openmeter` label in either file without updating the other — breaks the inter-stream channel.
- Removing the fan_out sync_response branch — clients will never receive HTTP responses.
- Bypassing the Content-Type switch and sending raw bytes directly to dedupe — dedup hash would be computed on non-normalized payloads.
- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of using environment variable interpolation.
- Removing the SQLite buffer without an alternative durability mechanism — events lost on collector restart.

## Decisions

- **Two-file stream split (input + output) connected by inproc channel.** — Separates HTTP ingestion concerns (validation, dedup, sync response) from forwarding concerns (buffering, batching, auth), making each independently configurable.
- **xxhash64 content hash as dedupe key.** — Fast non-cryptographic hash over the full message content provides exact-once deduplication for retried CloudEvent payloads without requiring an event ID field.

<!-- archie:ai-end -->
