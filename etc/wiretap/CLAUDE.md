# wiretap

<!-- archie:ai-start -->

> Docker-specific configuration for Wiretap, an OpenAPI-aware HTTP proxy that intercepts and validates traffic against the OpenMeter v1 OpenAPI spec. Provides a local traffic monitor for development API contract validation.

## Patterns

**Contract binding to generated spec** — The `contract` field must always point to `api/openapi.yaml` (the generated v1 spec) — never a hand-written or v3 spec path. (`contract: /usr/local/src/openmeter/api/openapi.yaml`)
**Docker internal host redirect** — redirectURL uses `host.docker.internal` to forward requests to the host's running server — required for Docker network bridging in local dev. (`redirectURL: http://host.docker.internal:8888`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.docker.yaml` | Wiretap proxy config: routes incoming requests on port 19090 to the local API server and serves the monitor UI on 19091/19092. The `contract` path is volume-mounted from the repo root. | If `api/openapi.yaml` is regenerated via `make gen-api` and the mount path changes, this config must be updated. If the local server port changes from 8888, redirectURL must change too. |

## Anti-Patterns

- Pointing `contract` at `api/v3/openapi.yaml` — wiretap here validates v1 traffic only
- Using `localhost` instead of `host.docker.internal` for redirectURL inside Docker
- Hardcoding a port that conflicts with docker-compose service ports defined in docker-compose.yaml

## Decisions

- **Separate docker config file rather than inline docker-compose override** — Wiretap requires a dedicated YAML config file; isolating it in etc/wiretap keeps docker-compose.yaml clean and makes the proxy config independently editable.

<!-- archie:ai-end -->
