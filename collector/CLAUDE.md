# collector

<!-- archie:ai-start -->

> Houses the Benthos/Redpanda Connect collector plugin surface (collector/benthos) and local demo environment (collector/quickstart). collector/benthos defines all custom Go plugins (input, bloblang, internal utils, leader-election service) that cmd/benthos-collector imports via blank imports; no openmeter domain logic is wired here.

## Patterns

**Blank-import activation for plugin registration** — Benthos plugins must be registered in init() before process start. cmd/benthos-collector imports sub-packages as _ to trigger init-time registration. (`import _ "github.com/openmeterio/openmeter/collector/benthos/input"`)
**Leader-election gate on replicated work** — All work that must not run on every replica checks services/leaderelection.IsLeader(res). Absent key defaults to true (safe solo deployment). (`if !leaderelection.IsLeader(res) { return nil } // skip work on non-leader replicas`)
**Environment-variable substitution in YAML presets** — All YAML presets under presets/ and quickstart/ use ${VAR:default} substitution. No values are hardcoded. (`url: ${OPENMETER_URL:http://localhost:8888}`)
**Child-package specialisation** — input/, bloblang/, internal/, services/, presets/ each own exactly one concern. Cross-cutting utils go to internal/ only. (`collector/benthos/internal/shutdown/signaller.go — shutdown signal utility, not a plugin.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector/benthos/services/leaderelection/service.go` | Leader-election state stored in service.Resources generic map; IsLeader(res) helper reads it with absent-key=true default. | Never read IsLeaderKey directly via res.GetGeneric — use IsLeader(res) to preserve absent-key=true semantics for single-replica deployments. |
| `collector/benthos/bloblang/parse_resource.go` | Side-effect-only package: registers bloblang functions in init(). No exported symbols. | Do not export symbols from this package. Do not place business logic here. |
| `collector/benthos/internal/message/transaction.go` | Internal message transaction utility. Pure utilities layer — no plugin registrations. | Do not add plugin registrations to internal/. Keep it as a pure utility layer. |
| `collector/quickstart/docker-compose.yaml` | Demo stack that includes ../../quickstart/docker-compose.yaml rather than duplicating service definitions. | Do not duplicate service definitions. Use in-memory dedupe cache (not Redis) in quickstart. |

## Anti-Patterns

- Registering a plugin outside init() — Benthos requires init-time registration before process start
- Adding business logic or plugin registrations to internal/ — pure utilities layer only
- Reading IsLeaderKey directly via res.GetGeneric instead of using IsLeader(res) helper — misses absent-key=true default
- Exporting symbols from collector/benthos/bloblang — side-effect-only package
- Hardcoding OPENMETER_URL or OPENMETER_TOKEN in YAML configs instead of ${VAR:default} substitution

## Decisions

- **Sub-package per concern (input, bloblang, internal, services, presets) rather than a flat package.** — Benthos plugin types have distinct registration and lifecycle semantics; separating them prevents accidental cross-registration and clarifies what cmd/benthos-collector must blank-import.
- **No Wire DI or openmeter domain service instantiation in collector/.** — The Benthos framework provides its own component lifecycle; injecting openmeter Wire graphs would couple the collector to the full service dependency tree unnecessarily.

<!-- archie:ai-end -->
