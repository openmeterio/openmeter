# quickstart

<!-- archie:ai-start -->

> Local demo environment for the Benthos collector: a Docker Compose stack running openmeter, the custom-plugin collector, and a synthetic CloudEvent seeder. Contains only YAML configs and one docker-compose.yaml — its role is quickstart demonstration, not production deployment or a reusable pipeline template.

## Patterns

**Env-var-driven configuration** — All URLs, tokens, and tunables use ${ENV_VAR} Benthos interpolation or Docker Compose environment: blocks — never hardcoded. (`environment:
  OPENMETER_URL: http://openmeter:8888`)
**Layered config loading** — The collector loads config.yaml first (bootstrap: HTTP API, metrics, cache), then overlays resources/*.yaml (shared infra) and streams/*.yaml (processors). Stream processors and cache declarations must NOT live in top-level config.yaml. (`command: ["--config", "/etc/collector/config.yaml", "--resources", "/etc/collector/resources/*.yaml", "streams", "--no-api", "/etc/collector/streams/*.yaml"]`)
**Healthcheck-gated depends_on** — Each service declares a wget healthcheck; downstream services use condition: service_healthy so openmeter is ready before collector, and collector before seeder. (`depends_on:
  openmeter:
    condition: service_healthy`)
**Switch output with DEBUG stdout branch** — Output configs use a switch with a DEBUG env-var branch that short-circuits to stdout for inspection without altering the real output path. (`switch:
  - check: '${SEEDER_LOG:false}' == 'true'
    output:
      stdout: {}`)
**Include base stack rather than duplicate** — docker-compose.yaml includes ../../quickstart/docker-compose.yaml and only adds collector + seeder, reusing the canonical openmeter stack. (`include:
  - ../../quickstart/docker-compose.yaml`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `docker-compose.yaml` | Extends ../../quickstart/docker-compose.yaml and adds collector + seeder services with bind-mounted local YAML configs. | OPENMETER_TOKEN is absent from the default collector environment — add it if the target OpenMeter instance requires auth. |
| `collector/config.yaml` | Bootstrap only: HTTP management API, Prometheus metrics, optional OTel tracer stub. Stream logic lives in streams/. | Adding stream processors here; renaming the dedupe_cache label without updating all cache: references silently bypasses deduplication. |
| `collector/resources/dedupe-cache.yaml` | In-memory deduplication cache with 1h TTL shared via the dedupe_cache label. | Switching to Redis requires updating this file and every cache: label reference in stream processors. |
| `seeder/config.yaml` | Synthetic CloudEvent generator: generate input -> Bloblang mapping -> http_client output. Demo-only. | Removing specversion or id fields breaks ingest API validation; do not add business logic here. |

## Anti-Patterns

- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of env-var interpolation.
- Adding transformation or enrichment logic to seeder/config.yaml — it is a synthetic demo generator.
- Adding stream processors or cache declarations to collector/config.yaml instead of streams/ and resources/.
- Using the generic http_client output instead of the custom openmeter output plugin in collector stream output.
- Enabling debug_endpoints: true outside local development — exposes internal Benthos state.

## Decisions

- **Include ../../quickstart/docker-compose.yaml rather than duplicating service definitions.** — Reuses the canonical openmeter stack; the collector quickstart only adds collector and seeder on top, avoiding drift between the two compose files.
- **In-memory dedupe cache instead of Redis.** — Reduces quickstart dependencies to zero external services beyond openmeter; acceptable for demo volumes where cross-restart deduplication is not required.

<!-- archie:ai-end -->
