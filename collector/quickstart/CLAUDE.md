# quickstart

<!-- archie:ai-start -->

> Local demo environment for the Benthos collector: a Docker Compose stack that runs openmeter, the collector (with custom plugins), and a synthetic event seeder. Contains only YAML configs and one docker-compose.yaml — no Go source. Its role is quickstart demonstration, not production deployment.

## Patterns

**Environment-variable-driven configuration** — All URLs, tokens, and tunable parameters use ${ENV_VAR} Benthos interpolation or Docker Compose environment: blocks — never hardcoded. (`environment:
  OPENMETER_URL: http://openmeter:8888`)
**Layered config loading** — The collector service loads config.yaml first, then overlays resources/*.yaml and streams/*.yaml — base bootstrap stays minimal; stream processors and cache declarations live in their dedicated sub-files. (`command: ["--config", "/etc/collector/config.yaml", "--resources", "/etc/collector/resources/*.yaml", "streams", "--no-api", "/etc/collector/streams/*.yaml"]`)
**Healthcheck-gated depends_on** — Each service declares a wget healthcheck and downstream services use condition: service_healthy — ensures openmeter is ready before collector starts, and collector before seeder. (`depends_on:
  openmeter:
    condition: service_healthy`)
**Switch output with DEBUG stdout branch** — Output configs use a switch with a DEBUG env-var branch that short-circuits to stdout — allows local inspection without changing the real output path. (`switch:
  - check: '${SEEDER_LOG:false}' == 'true'
    output:
      stdout: {}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector/quickstart/docker-compose.yaml` | Extends ../../quickstart/docker-compose.yaml and adds collector + seeder services with volume mounts binding local YAML configs. | OPENMETER_TOKEN is absent from the default collector environment — add it if the target OpenMeter instance requires auth. |
| `collector/quickstart/collector/config.yaml` | Bootstrap config: HTTP management API, Prometheus metrics, optional OTel tracer stub, in-memory dedupe cache. | Adding stream processors here — they belong in streams/; renaming dedupe_cache label without updating all cache: references. |
| `collector/quickstart/collector/resources/dedupe-cache.yaml` | In-memory deduplication cache with 1h TTL shared by stream processors. | Switching to Redis cache requires updating both this file and any cache: label references in stream processors. |
| `collector/quickstart/seeder/config.yaml` | Synthetic CloudEvent generator: generate input → Bloblang mapping → http_client output to collector ingest endpoint. | Removing specversion or id fields breaks ingest API validation; this is demo-only — do not add business logic. |

## Anti-Patterns

- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of environment variable interpolation
- Adding transformation or enrichment logic to seeder/config.yaml — it is a synthetic demo generator only
- Using generic http_client output instead of the custom openmeter output plugin in collector streams
- Enabling debug_endpoints: true outside local development
- Adding stream processors or cache declarations to collector/config.yaml top-level instead of streams/ and resources/ sub-files

## Decisions

- **Include ../../quickstart/docker-compose.yaml rather than duplicating service definitions** — Reuses the canonical openmeter stack definition; collector quickstart only adds the collector and seeder services on top.
- **In-memory dedupe cache instead of Redis** — Reduces quickstart dependencies to zero external services beyond openmeter itself; acceptable for demo volumes.

<!-- archie:ai-end -->
