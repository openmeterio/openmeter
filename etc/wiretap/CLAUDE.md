# wiretap

<!-- archie:ai-start -->

> Holds the Docker-targeted configuration for wiretap, a reverse-proxy / API contract-validation tool that fronts the OpenMeter API and checks live traffic against the OpenAPI spec. Its sole constraint: paths and hostnames are written for execution inside a container, not the host.

## Patterns

**Docker-host bridge for redirect target** — redirectURL points at host.docker.internal so wiretap running in a container can forward to a service listening on the developer's host machine. (`redirectURL: http://host.docker.internal:8888`)
**Container-absolute contract path** — contract references the OpenAPI spec by an in-container absolute path (mounted volume), not a repo-relative path. (`contract: /usr/local/src/openmeter/api/openapi.yaml`)
**Distinct ports per wiretap function** — Separate ports are assigned for the proxy (port), monitor UI (monitorPort), and websocket (webSocketPort) so they do not collide. (`port: 19090
monitorPort: 19091
webSocketPort: 19092`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.docker.yaml` | wiretap runtime config for the dockerized variant: proxy redirect target, the three service ports, and the OpenAPI contract location used for request/response validation. | contract is an in-container path (/usr/local/src/openmeter/api/openapi.yaml) that must match the volume mount; if openapi.yaml moves or is unmounted, validation breaks. redirectURL uses host.docker.internal, which only resolves from inside a container. |

## Anti-Patterns

- Changing contract to a repo-relative or host path — wiretap runs in-container and resolves it against the container filesystem / mount.
- Pointing redirectURL at localhost instead of host.docker.internal, which would loop back into the container rather than reach the host service.
- Reusing one port for proxy, monitor, and websocket — each wiretap function needs its own port.

## Decisions

- **Keep a docker-specific config file separate from any host config.** — The container needs host.docker.internal and absolute mounted paths that would not work when running wiretap directly on the host.

<!-- archie:ai-end -->
