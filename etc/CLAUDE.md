# etc

<!-- archie:ai-start -->

> Runtime configuration and dev-tooling assets for OpenMeter — not application source code. Houses seed-data pipelines (etc/seed) and a local OpenAPI proxy config (etc/wiretap) used exclusively for development, testing, and demos.

## Patterns

**Environment-variable parameterisation with defaults** — All runtime values (tokens, URLs, counts) are injected via ${VAR:default} so configs work out-of-the-box without manual edits and stay overridable in CI. (`${OPENMETER_BASE_URL:http://localhost:8888}`)
**Benthos/Redpanda Connect pipeline skeleton** — Each seed stream follows the identical input.generate + bloblang mapping + output.switch shape; only the bloblang body differs between streams. (`input: { generate: { count: 0, interval: ... } } / pipeline: { processors: [{ bloblang: ... }] } / output: { switch: [...] }`)
**CloudEvents 1.0 envelope** — Every generated event must include specversion, id (UUID), type, source, subject, time, and datacontenttype. Dimensional attributes go inside data: {}, not at the envelope root. (`{ specversion: '1.0', id: uuid_v4(), type: 'api-requests', source: 'seeder', subject: ..., time: now(), data: { tokens: ... } }`)
**Contract binding to generated v1 OpenAPI spec** — etc/wiretap points contract at the generated api/openapi.yaml (v1 spec), never v3; Docker redirects use host.docker.internal, not localhost. (`contract: ../../api/openapi.yaml / redirectURL: http://host.docker.internal:8888`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `etc/seed/seed.yaml` | Benthos pipeline entrypoint: imports all stream configs from etc/seed/streams/ and wires observability. | Adding new streams requires a corresponding entry here; a missing import silently skips that stream. |
| `etc/seed/streams/` | Per-domain generators (api-requests, prompt-tokens, workload-runtime) reusing the seed skeleton. | Copy the skeleton verbatim for new event types; deviating from the input.generate + output.switch shape breaks the pipeline, and dimensions must stay inside data: {}. |
| `etc/seed/observability.yaml` | Shared Benthos metrics/tracing config included by seed.yaml. | Do not duplicate in individual stream files. |
| `etc/wiretap/config.docker.yaml` | Wiretap proxy config for Docker-local v1 API contract validation. | Must target api/openapi.yaml (v1), not api/v3/openapi.yaml; redirectURL must use host.docker.internal. |

## Anti-Patterns

- Hardcoding OPENMETER_TOKEN or OPENMETER_BASE_URL — always use ${VAR:default}.
- Setting count > 0 in generate — seed streams must run indefinitely (count: 0 means infinite).
- Putting dimensional attributes outside data: {} — they belong in data, not at the CloudEvent root.
- Pointing the wiretap contract at api/v3/openapi.yaml — wiretap here validates v1 traffic only.
- Using localhost instead of host.docker.internal for redirectURL in the wiretap Docker config.

## Decisions

- **Seed streams share an identical Benthos skeleton with only the bloblang mapping varying.** — Keeps all streams structurally uniform so operators can read, copy, and modify any stream without learning a new pipeline shape.
- **Subject space bounded by SEEDER_SUBJECT_COUNT modulo rather than purely random UUIDs.** — Produces a realistic bounded set of subjects (mimicking real tenants) instead of ever-growing cardinality that would skew meter aggregations in demos.

<!-- archie:ai-end -->
