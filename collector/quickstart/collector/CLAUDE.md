# collector

<!-- archie:ai-start -->

> Top-level Benthos runtime configuration for the quickstart collector: bootstraps the HTTP management API, Prometheus metrics endpoint, and optional OTel tracer stub. Acts as the entry-point config Benthos loads first; all stream logic (streams/) and shared infrastructure (resources/) are loaded on top of it.

## Patterns

**Separation of concerns across config layers** — config.yaml owns only server-level concerns (HTTP address, metrics, tracer). Stream processors live in streams/, shared caches in resources/. Never merge these concerns into config.yaml. (`config.yaml sets `http.address` and `metrics.prometheus`; streams/input.yaml owns all processors; resources/dedupe-cache.yaml owns the cache declaration.`)
**Prometheus metrics always enabled** — The top-level config always exposes `metrics: prometheus: {}`. Any replacement metrics block must remain scrape-compatible. Do not add a second metrics backend alongside prometheus. (`metrics:
  prometheus: {}`)
**OTel tracer as opt-in comment stub** — The tracer block is commented out. Enabling it requires uncommenting and setting `address`. Do not add a second tracer block or rename the existing one. (`#tracer:
#  open_telemetry_collector:
#    grpc:
#      - address: <host>:4317`)
**Inproc channel coupling between streams** — streams/input.yaml publishes to `inproc: openmeter`; streams/output.yaml reads from the same named channel. Both names must stay in sync — renaming one without the other silently drops all events. (`input.yaml output: inproc: openmeter
output.yaml input: inproc: openmeter`)
**Environment variable interpolation for secrets** — OPENMETER_URL and OPENMETER_TOKEN must always be referenced via `${ENV_VAR}` interpolation in stream configs, never hardcoded. (`url: ${OPENMETER_URL}
headers:
  Authorization: Bearer ${OPENMETER_TOKEN}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.yaml` | Benthos server bootstrap: HTTP management API (port 4195), Prometheus metrics endpoint, optional OTel tracer stub. | Enabling debug_endpoints in production exposes internal Benthos state. Changing the HTTP address requires matching updates in docker-compose port mappings. |
| `resources/dedupe-cache.yaml` | Declares the named in-memory cache `dedupe_cache` (1h TTL) used by stream processors for CloudEvent deduplication. | Renaming `dedupe_cache` breaks all `cache:` references in streams/input.yaml without a compile-time error — silent dedup bypass. |
| `streams/input.yaml` | HTTP CloudEvent receiver: Content-Type switch, xxhash64 dedup against dedupe_cache, schema validation, inproc publish to `openmeter` channel, sync HTTP response propagation. | Removing the fan_out sync_response branch silences HTTP response codes to callers. Bypassing the Content-Type switch sends unnormalized bytes to dedup, causing incorrect deduplication. |
| `streams/output.yaml` | Inproc consumer: reads from `openmeter` channel, buffers to SQLite for durability, forwards batches to the OpenMeter API. | Renaming `inproc: openmeter` without matching change in input.yaml drops all events. Removing the SQLite buffer loses events on collector restart. |

## Anti-Patterns

- Adding stream processors or cache declarations directly in config.yaml — those belong in streams/ and resources/ respectively.
- Hardcoding OPENMETER_URL or OPENMETER_TOKEN in any config file instead of using `${ENV_VAR}` interpolation.
- Renaming the `dedupe_cache` label without updating all `cache:` references in stream processors — silent dedup bypass with no error.
- Enabling `debug_endpoints: true` outside local development — exposes internal Benthos state.
- Adding a second metrics backend alongside prometheus — Benthos supports one metrics sink.

## Decisions

- **config.yaml is the minimal bootstrap; streams and resources are loaded as separate files.** — Keeps server-level config stable and independently reviewable; stream logic can evolve without touching the HTTP/metrics layer.
- **In-memory dedup cache with 1h TTL instead of Redis.** — Quickstart must run with zero external dependencies beyond the OpenMeter API; Redis would require an additional service in docker-compose.
- **Two-stream split (input + output) connected by named inproc channel.** — Decouples HTTP ingestion latency from batch output delivery; SQLite buffer in output stream provides durability across restarts without blocking the HTTP receiver.

<!-- archie:ai-end -->
