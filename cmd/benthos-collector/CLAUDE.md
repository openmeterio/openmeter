# benthos-collector

<!-- archie:ai-start -->

> Binary entrypoint for the Redpanda Benthos/Connect collector: registers custom OpenMeter bloblang, input, and output plugins via blank imports, then delegates entirely to service.RunCLI with optional leader-election CLI opts. No Wire DI — no domain services are wired here.

## Patterns

**Plugin registration via blank imports** — All OpenMeter Benthos extensions (bloblang functions, custom inputs, custom outputs) are registered by blank-importing their packages before calling service.RunCLI. (`import _ "github.com/openmeterio/openmeter/collector/benthos/bloblang"
import _ "github.com/openmeterio/openmeter/collector/benthos/input"
import _ "github.com/openmeterio/openmeter/collector/benthos/output"`)
**Minimal main — delegate to Benthos RunCLI** — main() creates a cancellable context, appends leaderelection CLI opts, and calls service.RunCLI. All pipeline logic is in YAML config files, not Go. (`service.RunCLI(ctx, leaderelection.GetLeaderElectionCLIOpts(ctx)...)`)
**Simplified version init (no runtime/debug)** — version.go only sets version = "unknown" if unset via ldflags; no vcs revision tracking unlike other binaries. (`func init() { if version == "" { version = "unknown" } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Entry point: registers plugins, starts Benthos CLI with leader election opts. | New plugins must be blank-imported here to be registered before RunCLI is called; plugins not imported will silently not appear in the Benthos component registry. |

## Anti-Patterns

- Adding Wire DI or domain service instantiation — all pipeline logic belongs in collector/benthos
- Adding Kafka consumer/producer Go code directly — use Benthos YAML pipeline configs instead
- Forgetting blank imports when adding new bloblang/input/output plugins in collector/benthos/*

## Decisions

- **No Wire DI: this binary is a thin launcher for the Benthos framework, not a Go service.** — Benthos manages its own plugin/component registry and YAML-driven pipeline wiring; injecting Go DI would duplicate what Bentmill already provides.

<!-- archie:ai-end -->
