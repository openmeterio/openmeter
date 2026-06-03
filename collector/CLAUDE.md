# collector

<!-- archie:ai-start -->

> Separate Go module (own go.mod, replace github.com/openmeterio/openmeter => ../) housing the Benthos/Redpanda Connect collector: the custom plugin surface (collector/benthos) activated by blank imports from cmd/benthos-collector, plus a local docker-compose demo (collector/quickstart). It is the production-ready ingestion pipeline that collects, transforms, buffers, and reliably delivers usage events into OpenMeter; no openmeter domain Wire graph is instantiated here.

## Patterns

**Blank-import plugin activation** — Benthos plugins must register in init() before process start; the launcher imports collector/benthos sub-packages as _ to trigger registration. Plugins are never wired by calling their constructors. (`import _ "github.com/openmeterio/openmeter/collector/benthos/input"`)
**Sub-package per concern** — collector/benthos splits into input/, bloblang/, internal/ (pure utils), services/ (leader-election), and presets/ (pure YAML), each owning exactly one concern; cross-cutting utilities go only to internal/. (`collector/benthos/internal/shutdown/signaller.go is a utility, not a plugin.`)
**Leader-election gate on replicated work** — Work that must not run on every replica checks leaderelection.IsLeader(res); leader state is stored in service.Resources generic map and an absent key defaults to true (safe single-replica deployment). (`if !leaderelection.IsLeader(res) { return nil } // skip on non-leader replicas`)
**Env-var substitution in YAML presets** — All preset and quickstart YAML uses ${VAR:default} interpolation; no endpoints, URLs, or tokens are hardcoded. (`url: ${OPENMETER_URL:http://localhost:8888}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector/cmd/main.go` | Thin launcher: blank-imports plugin packages then calls service.RunCLI with a cancellable root context and leader-election CLI options. | Add no pipeline logic here; register plugins via blank import; keep the cancellable context (do not pass a bare context.Background() to RunCLI). |
| `collector/benthos/services/leaderelection/service.go` | Leader-election state in service.Resources generic map with IsLeader(res) helper (absent-key=true default). | Read leadership only via IsLeader(res), never res.GetGeneric on IsLeaderKey, or the single-replica default is lost. |
| `collector/benthos/bloblang/parse_resource.go` | Side-effect-only package registering bloblang functions in init(); exports nothing. | Do not export symbols or place business logic here. |
| `collector/benthos/input/*.go` | Custom Benthos input plugins registered in init(). | Return transient per-resource failures by logging and skipping; a returned error from ReadBatch halts the whole pipeline. |
| `collector/quickstart/docker-compose.yaml` | Local demo stack that includes ../../quickstart/docker-compose.yaml rather than duplicating service definitions; uses in-memory dedupe (not Redis). | Use the custom openmeter output plugin (not generic http_client); keep debug_endpoints off outside local dev. |
| `collector/go.mod` | Separate module pinning Redpanda benthos/connect and a large transitive dependency set; replaces openmeter with the parent repo. | Built as a separate Docker image (benthos-collector.Dockerfile, CGO_ENABLED=0); dependency bumps are isolated from the root module. |

## Anti-Patterns

- Registering a plugin outside init() — Benthos requires init-time registration before process start.
- Adding business logic or plugin registrations to internal/ — it is a pure utilities layer.
- Reading IsLeaderKey directly via res.GetGeneric instead of leaderelection.IsLeader(res), missing the absent-key=true default.
- Exporting symbols from collector/benthos/bloblang — it is side-effect-only and must export nothing.
- Hardcoding OPENMETER_URL/OPENMETER_TOKEN instead of ${VAR:default} substitution, or wiring an openmeter Wire graph into the collector.

## Decisions

- **Sub-package per concern (input, bloblang, internal, services, presets) rather than a flat package.** — Benthos plugin types have distinct registration and lifecycle semantics; separation prevents accidental cross-registration and clarifies what cmd/benthos-collector must blank-import.
- **No Wire DI or openmeter domain service instantiation in collector/.** — The Benthos framework provides its own component lifecycle; injecting openmeter Wire graphs would couple the collector to the full service dependency tree unnecessarily.
- **Collector is its own Go module and Docker image.** — Isolates the heavy Redpanda Connect dependency tree (CGO_ENABLED=0 build) from the main module's build and dependency graph.

## Example: Launcher blank-imports plugins then runs the Benthos CLI

```
package main

import (
	"context"

	"github.com/redpanda-data/benthos/v4/public/service"

	_ "github.com/openmeterio/openmeter/collector/benthos/bloblang"
	_ "github.com/openmeterio/openmeter/collector/benthos/input"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.RunCLI(ctx)
// ...
```

<!-- archie:ai-end -->
