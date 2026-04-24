# benthos

<!-- archie:ai-start -->

> Orchestration layer for the Benthos/Redpanda Connect collector binary: wires together custom input plugins (input/), bloblang functions (bloblang/), internal utilities (internal/), leader-election service (services/), and YAML pipeline presets (presets/). No Go source lives here directly — this folder's role is to define the full plugin surface and runtime config that cmd/benthos-collector imports.

## Patterns

**Blank-import activation** — All plugin packages (input/, bloblang/, output/) register themselves via init(). cmd/benthos-collector must blank-import each package to activate it; forgetting an import silently omits the plugin at runtime. (`import _ "github.com/openmeterio/openmeter/collector/benthos/input"`)
**Child-package specialisation** — Each sub-folder owns a single concern: input/ = BatchInput plugins, bloblang/ = Bloblang functions, internal/ = shared utilities (no plugins), services/ = cross-cutting infrastructure, presets/ = pure YAML configs. Do not mix concerns across these boundaries. (`// adding a new input: create collector/benthos/input/myinput.go, register in init()`)
**Leader-election gate on all replicated work** — Inputs that must run on only one replica poll leaderelection.IsLeader(res) from service.Resources in Connect/ReadBatch — absent key defaults to true so single-replica setups work without an election service. (`if !leaderelection.IsLeader(res) { return nil, component.ErrNotConnected }`)
**Environment-variable substitution in YAML presets** — All secrets and endpoint URLs in presets/ use ${ENV_VAR} Benthos interpolation — never hardcoded values. (`url: ${OPENMETER_URL}/api/v1/events`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `collector/benthos/input/*.go` | One BatchInput plugin per file; registered in init() via service.RegisterBatchInput. | Holding in.mu across network calls; returning errors from ReadBatch for transient per-resource failures (halts pipeline); omitting XxxInputConfig() factory function. |
| `collector/benthos/bloblang/parse_resource.go` | Registers custom Bloblang functions via init(); must be blank-imported to activate. | Registering duplicate function names across files (panics); performing I/O inside the returned closure; exporting symbols (side-effect-only package). |
| `collector/benthos/internal/shutdown/signaller.go` | Two-tier graceful shutdown primitive used by all plugins. | Forgetting ShutdownComplete() in a component goroutine — orchestrators block on HasClosedChan indefinitely. |
| `collector/benthos/internal/message/transaction.go` | Delivery-guaranteed message routing via Transaction/Ack. | Calling Ack more than once — buffered channel send is not idempotent and will panic or block. |
| `collector/benthos/services/leaderelection/service.go` | Kubernetes lease-based leader election; exposes state via service.Resources generic map. | Reusing a LeaderElector instance across election cycles (client-go electors are single-use); calling Start with context.Background() (breaks shutdown). |
| `collector/benthos/presets/http-server/config.yaml` | Receive-validate-forward CloudEvent ingestion pipeline preset. | Adding transformation logic here; using generic http_client output instead of the custom openmeter plugin. |
| `collector/benthos/presets/kubernetes-pod-exec-time/config.yaml` | Kubernetes billing pipeline deriving duration_seconds from schedule_interval metadata. | Omitting .number(0) on resource_quantity() calls — pods without resource requests produce nil mapping errors. |

## Anti-Patterns

- Registering a plugin outside init() — Benthos requires init-time registration before process start
- Adding business logic or plugin registrations to internal/ — it is a pure utilities layer
- Placing new stream processors or cache declarations in presets/*/config.yaml top-level — those belong in streams/ and resources/ sub-files
- Reading IsLeaderKey directly via res.GetGeneric instead of using the IsLeader(res) helper (misses absent-key=true default)
- Exporting symbols from collector/benthos/bloblang — it is a side-effect-only package

## Decisions

- **Sub-package per concern (input, bloblang, internal, services, presets) rather than a flat package** — Keeps plugin registration, shared utilities, infrastructure services, and YAML configs independently evolvable and prevents accidental coupling between Benthos plugin lifecycle and utility code.
- **Leader state stored in service.Resources generic map, absent key defaults to true** — Single-replica deployments work without configuring a leader-election service; the map is the only cross-plugin shared-state mechanism Benthos exposes.
- **Presets are pure YAML with no Go code** — Pipeline configs can be updated and shipped without recompiling the binary; environment-variable substitution handles deployment-specific values.

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
