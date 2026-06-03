# cmd

<!-- archie:ai-start -->

> Thin launcher for the benthos-collector binary: a separate Go module whose main.go blank-imports the custom OpenMeter bloblang/input/output plugin packages and then calls service.RunCLI with leader-election CLI options. Contains no pipeline logic of its own.

## Patterns

**Blank-import plugins before RunCLI** — Custom plugin packages and Benthos/Connect component bundles are registered solely via blank imports (_) so their init() side-effects run before service.RunCLI is invoked. main() must not call plugin constructors directly. (`import (
	_ "github.com/openmeterio/openmeter/collector/benthos/bloblang"
	_ "github.com/openmeterio/openmeter/collector/benthos/input"
	_ "github.com/openmeterio/openmeter/collector/benthos/output"
	_ "github.com/redpanda-data/connect/public/bundle/free/v4"
)`)
**Cancellable root context wired into RunCLI** — main() builds a cancellable context.Background() and threads it into both service.RunCLI and leaderelection.GetLeaderElectionCLIOpts so shutdown propagates. (`ctx, cancel := context.WithCancel(context.Background()); defer cancel(); service.RunCLI(ctx, leaderelection.GetLeaderElectionCLIOpts(ctx)...)`)
**version provisioned by ldflags with init() fallback** — version is a package var set by ldflags at build time; init() defaults it to 'unknown' when absent. (`var version string; func init() { if version == "" { version = "unknown" } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | Entrypoint that blank-imports plugin packages and Benthos component bundles, then calls service.RunCLI with leader-election options. | New custom plugins (bloblang/input/output) must be added as blank imports here or they will not register. Do not add pipeline/business logic — it belongs in collector/benthos. |
| `version.go` | Holds the ldflags-provisioned version string and the init() fallback to 'unknown'. | Carries //nolint:gochecknoinits — keep the init() minimal. |

## Anti-Patterns

- Adding pipeline/business logic to main.go instead of collector/benthos plugin packages
- Registering a new plugin by calling its constructor rather than blank-importing the package
- Replacing the cancellable context with context.Background() passed directly to RunCLI without cancel propagation

## Decisions

- **Built as a separate Go module and Docker image (benthos-collector.Dockerfile, CGO_ENABLED=0).** — The collector is an independently deployable Redpanda Benthos/Connect service with a distinct dependency tree, kept out of the main module to avoid pulling Benthos into the core binaries.

## Example: The full collector launcher

```
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.RunCLI(ctx, leaderelection.GetLeaderElectionCLIOpts(ctx)...)
}
```

<!-- archie:ai-end -->
