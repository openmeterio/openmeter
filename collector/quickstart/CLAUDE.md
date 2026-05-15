# quickstart

<!-- archie:ai-start -->

> Local demo environment for the Benthos collector: a Docker Compose stack that runs openmeter, the custom-plugin collector, and a synthetic CloudEvent seeder. Contains only YAML configs and one docker-compose.yaml — no Go source. Role is quickstart demonstration, not production deployment or reusable pipeline template.

## Patterns

**Environment-variable-driven configuration** — All URLs, tokens, and tunable parameters use ${ENV_VAR} Benthos interpolation or Docker Compose environment: blocks — never hardcoded values. (`environment:
  OPENMETER_URL: http://openmeter:8888`)
**Layered config loading** — The collector service loads config.yaml first (bootstrap: HTTP API, metrics, cache), then overlays resources/*.yaml (shared infrastructure) and streams/*.yaml (processors). Stream processors and cache declarations must NOT go in the top-level config.yaml. (`command: ["--config", "/etc/collector/config.yaml", "--resources", "/etc/collector/resources/*.yaml", "streams", "--no-api", "/etc/collector/streams/*.yaml"]`)
**Healthcheck-gated depends_on** — Each service declares a wget healthcheck; downstream services use condition: service_healthy to ensure openmeter is ready before collector starts, and collector before seeder. (`depends_on:
  openmeter:
    condition: service_healthy`)
**Switch output with DEBUG stdout branch** — Output configs use a switch with a DEBUG env-var branch that short-circuits to stdout for local inspection without altering the real output path. (`switch:
  - check: '${SEEDER_LOG:false}' == 'true'
    output:
      stdout: {}`)
**Include base stack rather than duplicate** — docker-compose.yaml includes ../../quickstart/docker-compose.yaml and only adds collector and seeder services, reusing the canonical openmeter stack definition. (`include:
  - ../../quickstart/docker-compose.yaml`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector/quickstart/docker-compose.yaml` | Extends ../../quickstart/docker-compose.yaml and adds collector + seeder services with volume mounts binding local YAML configs. | OPENMETER_TOKEN is absent from the default collector environment — add it if the target OpenMeter instance requires auth. |
| `collector/quickstart/collector/config.yaml` | Bootstrap config only: HTTP management API, Prometheus metrics, optional OTel tracer stub. Stream logic lives in streams/. | Adding stream processors here — they belong in streams/; renaming dedupe_cache label without updating all cache: references in stream processors silently bypasses deduplication. |
| `collector/quickstart/collector/resources/dedupe-cache.yaml` | In-memory deduplication cache with 1h TTL shared by stream processors via the dedupe_cache label. | Switching to Redis requires updating both this file and all cache: label references in stream processor configs. |
| `collector/quickstart/seeder/config.yaml` | Synthetic CloudEvent generator: generate input → Bloblang mapping → http_client output to collector ingest endpoint. Demo-only. | Removing specversion or id fields from the generated CloudEvent breaks ingest API validation; do not add business logic here. |

## Anti-Patterns

- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of environment variable interpolation
- Adding transformation or enrichment logic to seeder/config.yaml — it is a synthetic demo generator, not a reusable pipeline template
- Adding stream processors or cache declarations to collector/config.yaml top-level instead of streams/ and resources/ sub-files
- Using generic http_client output instead of the custom openmeter output plugin in collector stream output configs
- Enabling debug_endpoints: true outside local development — exposes internal Benthos state

## Decisions

- **Include ../../quickstart/docker-compose.yaml rather than duplicating service definitions** — Reuses the canonical openmeter stack; collector quickstart only adds the collector and seeder services on top, avoiding drift between the two compose files.
- **In-memory dedupe cache instead of Redis** — Reduces quickstart dependencies to zero external services beyond openmeter itself; acceptable for demo volumes where cross-restart deduplication is not required.

<!-- archie:ai-end -->
