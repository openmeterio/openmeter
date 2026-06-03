# benthos

<!-- archie:ai-start -->

> Orchestration layer for the Benthos/Redpanda Connect collector binary: defines the full custom plugin surface (input/, bloblang/), shared utilities (internal/), leader-election infrastructure (services/), and deployable YAML presets (presets/). No Go source lives directly here — cmd/benthos-collector blank-imports the child packages to activate them.

## Patterns

**Blank-import activation** — Every plugin package registers itself via Go init(); cmd/benthos-collector must blank-import each one. A forgotten import silently omits the plugin at runtime with no error. (`import _ "github.com/openmeterio/openmeter/collector/benthos/input"`)
**Sub-package per concern** — input/ holds only BatchInput plugins, bloblang/ only Bloblang function registrations, internal/ only shared utilities (no plugin logic), services/ only cross-cutting infrastructure, presets/ only YAML. Never mix these boundaries. (`new input plugin -> input/myplugin.go; new Bloblang fn -> bloblang/myfn.go`)
**init()-based plugin registration** — Each plugin file calls service.RegisterBatchInput (or the Bloblang equivalent) inside init(); registration must happen before process start. (`func init() { _ = service.RegisterBatchInput("my_plugin", MyInputConfig(), newMyInput) }`)
**Leader-election gate on replicated work** — Inputs that must run on one replica poll leaderelection.IsLeader(res) from service.Resources in Connect/ReadBatch; absent key defaults to true so single-replica deployments need no election service. (`if !leaderelection.IsLeader(res) { return nil, component.ErrNotConnected }`)
**Env-var substitution in YAML presets** — All secrets and endpoint URLs in presets/ use ${ENV_VAR} Benthos interpolation — never hardcoded, so configs ship without recompiling. (`url: ${OPENMETER_URL}/api/v1/events`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `input/*.go` | One BatchInput plugin per file, registered in init() via service.RegisterBatchInput; provides Connect/ReadBatch/Close lifecycle. | Holding in.mu across network calls; returning errors from ReadBatch for transient per-resource failures (halts pipeline); omitting a dedicated XxxInputConfig() factory. |
| `bloblang/parse_resource.go` | Registers custom Bloblang functions via init(); side-effect-only, exports nothing. | Duplicate function names across files (startup panic); I/O inside the returned closure (runs every invocation); exporting any symbols. |
| `internal/shutdown/signaller.go` | Two-tier graceful-shutdown primitive (CloseAtLeisure / CloseNow) used by all plugins. | Forgetting ShutdownComplete() in a component goroutine — orchestrators block on HasClosedChan indefinitely. |
| `internal/message/transaction.go` | Delivery-guaranteed routing via Transaction/Ack; buffered chan<-error (cap 1) for the ack path. | Calling Ack more than once — the buffered send is not idempotent and blocks or panics. |
| `services/leaderelection/service.go` | Kubernetes lease-based leader election; exposes state via service.Resources generic map under IsLeaderKey. | Reusing a single-use client-go LeaderElector across cycles; calling Start with context.Background() (breaks shutdown cancellation). |
| `presets/http-server/config.yaml` | Receive-validate-forward CloudEvent ingestion preset; no transformation logic. | Adding enrichment logic; using generic http_client output instead of the custom openmeter plugin. |

## Anti-Patterns

- Registering a plugin outside init() — Benthos requires init-time registration before process start.
- Adding business logic or plugin registrations to internal/ — it is a pure utilities layer.
- Reading IsLeaderKey directly via res.GetGeneric instead of leaderelection.IsLeader(res) — misses the absent-key=true default.
- Exporting symbols from bloblang/ — it is side-effect-only and must export nothing.
- Returning a hard error from ReadBatch for transient per-resource failures — log and skip; a returned error halts the pipeline.

## Decisions

- **Sub-package per concern (input, bloblang, internal, services, presets) rather than a flat package.** — Keeps plugin registration, utilities, infrastructure, and YAML configs independently evolvable and prevents accidental coupling between plugin lifecycle and utility code.
- **Leader state stored in service.Resources generic map with absent-key defaulting to true.** — Single-replica deployments work without an election service, and service.Resources is the only cross-plugin shared-state mechanism Benthos exposes.
- **Presets are pure YAML with no Go code.** — Pipeline configs can be updated and shipped without recompiling; env-var substitution handles all deployment-specific values.

## Example: Registering a new BatchInput plugin that gates work on leader election

```
// collector/benthos/input/myplugin.go
package input

import (
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"
)

func init() {
	_ = service.RegisterBatchInput("my_plugin", MyPluginInputConfig(),
		func(conf *service.ParsedConfig, res *service.Resources) (service.BatchInput, error) {
			return newMyPluginInput(conf, res)
		})
}
```

<!-- archie:ai-end -->
