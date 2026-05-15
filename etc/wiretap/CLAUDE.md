# wiretap

<!-- archie:ai-start -->

> Docker-specific configuration for Wiretap, an OpenAPI-aware HTTP proxy that intercepts and validates local development traffic against the generated OpenMeter v1 OpenAPI spec (api/openapi.yaml). Its sole constraint is that the contract path must always point to the generated v1 spec, never hand-written or v3 variants.

## Patterns

**Contract bound to generated v1 spec** — The `contract` field must always reference `api/openapi.yaml` (the oapi-codegen output from TypeSpec). Pointing at any other spec path silently validates against the wrong contract. (`contract: /usr/local/src/openmeter/api/openapi.yaml`)
**Docker internal host redirect** — redirectURL must use `host.docker.internal` to reach the host machine's running server from inside Docker. Using `localhost` resolves to the container itself and breaks proxying. (`redirectURL: http://host.docker.internal:8888`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.docker.yaml` | Wiretap proxy config: routes incoming requests on port 19090 to the local API server and serves the monitor UI on 19091/19092. The contract path is volume-mounted from the repo root. | If api/openapi.yaml is regenerated via `make gen-api` and the volume mount path changes, the contract field must be updated. If the local server port changes from 8888, redirectURL must change accordingly. Port conflicts with docker-compose.yaml services must be avoided. |

## Anti-Patterns

- Pointing `contract` at `api/v3/openapi.yaml` — this config validates v1 traffic only
- Using `localhost` instead of `host.docker.internal` for redirectURL inside Docker
- Hardcoding a port that conflicts with docker-compose service ports defined in docker-compose.yaml
- Hand-editing api/openapi.yaml to patch the contract — the file is generated; changes are lost on next `make gen-api`

## Decisions

- **Separate docker config file rather than inline docker-compose override** — Wiretap requires a dedicated YAML config file; isolating it in etc/wiretap keeps docker-compose.yaml clean and makes the proxy config independently editable without touching service definitions.

<!-- archie:ai-end -->
