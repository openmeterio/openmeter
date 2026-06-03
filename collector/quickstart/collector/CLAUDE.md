# collector

<!-- archie:ai-start -->

> Top-level Benthos runtime configuration for the quickstart collector. config.yaml is the minimal server bootstrap (HTTP management API on 4195, Prometheus metrics, optional OTel tracer stub) that Benthos loads first; the actual pipeline lives in streams/ (input+output) and shared infra in resources/ (the dedupe cache).

## Patterns

**Layered config: server vs streams vs resources** — config.yaml owns only server-level concerns (http.address, metrics, tracer); stream processors live in streams/ and shared caches in resources/ — never merge these into config.yaml. (`config.yaml: http.address + metrics.prometheus; streams/input.yaml: processors; resources/dedupe-cache.yaml: cache declaration`)
**Inproc channel couples the two streams** — streams/input.yaml publishes to inproc: openmeter and streams/output.yaml reads from the same named channel; both names must stay in sync or events are silently dropped. (`input.yaml output: inproc: openmeter
output.yaml input: inproc: openmeter`)
**Prometheus metrics always enabled, single backend** — The top-level config always exposes metrics: prometheus: {}; any replacement must stay scrape-compatible and Benthos supports only one metrics sink. (`metrics:
  prometheus: {}`)
**OTel tracer is an opt-in commented stub** — The tracer block is commented out; enabling it means uncommenting and setting address — do not add a second tracer block. (`#tracer:
#  open_telemetry_collector:
#    grpc:
#      - address: <host>:4317`)
**Secrets via ${ENV_VAR} interpolation** — OPENMETER_URL and OPENMETER_TOKEN are referenced via ${ENV_VAR} in stream configs, never hardcoded. (`url: ${OPENMETER_URL}
headers:
  Authorization: Bearer ${OPENMETER_TOKEN}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.yaml` | Benthos server bootstrap: HTTP management API (port 4195), Prometheus metrics, optional OTel tracer stub. | debug_endpoints: false in production (true exposes internal Benthos state); changing the HTTP address requires matching docker-compose port mappings. |
| `resources/dedupe-cache.yaml` | Declares the named in-memory dedupe_cache (1h TTL) used by stream processors for CloudEvent deduplication. | Renaming dedupe_cache breaks all cache: references in streams/input.yaml with no compile-time error — silent dedup bypass. |
| `streams/input.yaml` | HTTP CloudEvent receiver: Content-Type switch, xxhash64 dedup against dedupe_cache, schema validation, inproc publish to openmeter, sync HTTP response. | Removing the fan_out sync_response branch silences HTTP status codes; bypassing the Content-Type switch sends unnormalized bytes to dedup, corrupting deduplication. |
| `streams/output.yaml` | Inproc consumer: reads from openmeter channel, buffers to SQLite for durability, forwards batches to the OpenMeter API. | Renaming inproc: openmeter without matching input.yaml drops all events; removing the SQLite buffer loses events on restart. |

## Anti-Patterns

- Adding stream processors or cache declarations directly in config.yaml — they belong in streams/ and resources/
- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of ${ENV_VAR} interpolation
- Renaming the dedupe_cache label without updating all cache: references — silent dedup bypass with no error
- Enabling debug_endpoints: true outside local development — exposes internal Benthos state
- Adding a second metrics backend alongside prometheus — Benthos supports one metrics sink

## Decisions

- **config.yaml is a minimal bootstrap; streams and resources load as separate files** — Keeps the HTTP/metrics layer stable and independently reviewable while stream logic evolves separately.
- **In-memory dedup cache with 1h TTL instead of Redis** — The quickstart must run with zero external dependencies beyond the OpenMeter API; Redis would add a docker-compose service.
- **Two-stream split (input + output) via a named inproc channel** — Decouples HTTP ingestion latency from batch output delivery; the SQLite buffer in output gives durability across restarts without blocking the HTTP receiver.

<!-- archie:ai-end -->
