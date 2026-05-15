# benthos

<!-- archie:ai-start -->

> Orchestration layer for the Benthos/Redpanda Connect collector binary: wires together custom BatchInput plugins (input/), Bloblang functions (bloblang/), shared utilities (internal/), leader-election infrastructure (services/), and deployable YAML pipeline presets (presets/). No Go source lives here directly — this folder defines the full plugin surface and runtime config that cmd/benthos-collector blank-imports to activate.

## Patterns

**Blank-import activation** — All plugin packages (input/, bloblang/) register themselves via Go init(). cmd/benthos-collector must blank-import each package; forgetting an import silently omits the plugin at runtime with no error. (`import _ "github.com/openmeterio/openmeter/collector/benthos/input"`)
**Sub-package per concern** — input/ = BatchInput plugins only, bloblang/ = Bloblang function registrations only, internal/ = shared utilities (no plugin logic), services/ = cross-cutting infrastructure (leader election), presets/ = pure YAML configs. Do not mix concerns across these boundaries. (`// new input plugin: collector/benthos/input/myplugin.go; new Bloblang fn: collector/benthos/bloblang/myfn.go`)
**Leader-election gate on replicated work** — Inputs that must run on only one replica poll leaderelection.IsLeader(res) from service.Resources in Connect/ReadBatch. Absent key defaults to true so single-replica deployments need no election service. (`if !leaderelection.IsLeader(res) { return nil, component.ErrNotConnected }`)
**init()-based plugin registration** — Every plugin file calls service.RegisterBatchInput (or the Bloblang equivalent) inside an init() function. Registration must happen before process start; calling outside init() causes the plugin to be unavailable at pipeline construction time. (`func init() { _ = service.RegisterBatchInput("my_plugin", MyInputConfig(), newMyInput) }`)
**Environment-variable substitution in YAML presets** — All secrets and endpoint URLs in presets/ use ${ENV_VAR} Benthos interpolation — never hardcoded values. This allows config files to be shipped without recompiling the binary. (`url: ${OPENMETER_URL}/api/v1/events`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector/benthos/input/*.go` | One BatchInput plugin per file; each registered in init() via service.RegisterBatchInput. Provides Connect/ReadBatch/Close lifecycle. | Holding in.mu across network calls; returning errors from ReadBatch for transient per-resource failures (halts pipeline); omitting a dedicated XxxInputConfig() factory function. |
| `collector/benthos/bloblang/parse_resource.go` | Registers custom Bloblang functions via init(); must be blank-imported to activate. Side-effect-only — exports nothing. | Registering duplicate function names across files (startup panic); performing I/O inside the returned closure (run on every invocation); exporting any symbols. |
| `collector/benthos/internal/shutdown/signaller.go` | Two-tier graceful shutdown primitive (CloseAtLeisure / CloseNow) used by all plugins. | Forgetting ShutdownComplete() in a component goroutine — orchestrators block on HasClosedChan indefinitely. |
| `collector/benthos/internal/message/transaction.go` | Delivery-guaranteed message routing via Transaction/Ack pattern; buffered chan<-error (cap 1) for ack path. | Calling Ack more than once — buffered channel send is not idempotent and will block or panic. |
| `collector/benthos/services/leaderelection/service.go` | Kubernetes lease-based leader election; exposes state via service.Resources generic map under IsLeaderKey. | Reusing a LeaderElector instance across election cycles (single-use in client-go); calling Start with context.Background() (breaks shutdown cancellation). |
| `collector/benthos/presets/http-server/config.yaml` | Receive-validate-forward CloudEvent ingestion pipeline preset; no transformation logic. | Adding enrichment logic here; using generic http_client output instead of the custom openmeter plugin. |
| `collector/benthos/presets/kubernetes-pod-exec-time/config.yaml` | Kubernetes billing pipeline deriving duration_seconds from schedule_interval metadata at mapping time. | Omitting .number(0) on resource_quantity() calls — pods without resource requests produce nil mapping errors. |

## Anti-Patterns

- Registering a plugin outside init() — Benthos requires init-time registration before the process starts
- Adding business logic or plugin registrations to internal/ — it is a pure utilities layer with no plugin surface
- Reading IsLeaderKey directly via res.GetGeneric instead of using leaderelection.IsLeader(res) — misses the absent-key=true default for single-replica deployments
- Exporting symbols from collector/benthos/bloblang — this package is side-effect-only and must export nothing
- Returning a hard error from ReadBatch for transient per-resource failures — non-fatal errors must be logged and skipped; a returned error halts the entire Benthos pipeline

## Decisions

- **Sub-package per concern (input, bloblang, internal, services, presets) rather than a flat package** — Keeps plugin registration, shared utilities, infrastructure services, and YAML configs independently evolvable and prevents accidental coupling between Benthos plugin lifecycle and utility code.
- **Leader state stored in service.Resources generic map with absent-key defaulting to true** — Single-replica deployments work without configuring a leader-election service; service.Resources is the only cross-plugin shared-state mechanism Benthos exposes, so no alternative channel is available.
- **Presets are pure YAML with no Go code** — Pipeline configs can be updated and shipped without recompiling the binary; environment-variable substitution handles all deployment-specific values.

## Example: Registering a new BatchInput plugin that gates work on leader election

```
// collector/benthos/input/myplugin.go
package input

import (
	"context"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/openmeterio/openmeter/collector/benthos/services/leaderelection"
)

func init() {
	err := service.RegisterBatchInput("my_plugin", MyPluginInputConfig(),
		func(conf *service.ParsedConfig, res *service.Resources) (service.BatchInput, error) {
			return newMyPluginInput(conf, res)
		})
// ...
```

<!-- archie:ai-end -->
