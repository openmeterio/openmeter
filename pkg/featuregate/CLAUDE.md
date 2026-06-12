# featuregate

<!-- archie:ai-start -->

> Minimal feature-gate abstraction: a one-method `Gate` interface plus a `Noop` implementation that always allows. It is the seam where real flag backends (LaunchDarkly, etc.) plug into namespace/flag boolean evaluation.

## Patterns

**Single-method Gate interface** — `Gate.EvaluateBool(namespace, flag string, defaultValue bool) (bool, error)` is the only contract; implementations live elsewhere and are injected (`type Gate interface { EvaluateBool(namespace, flag string, defaultValue bool) (bool, error) }`)
**Noop returns true (open by default)** — `NewNoop()` yields `Noop{}` whose `EvaluateBool` always returns `(true, nil)` — when gating is disabled, features are ON (`func (n Noop) EvaluateBool(string, string, bool) (bool, error) { return true, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `featuregate.go` | Defines `Gate`, `NewNoop()`, and the `Noop` struct | Noop ignores the supplied defaultValue and always returns true; do not assume the default is honored when wiring Noop in cmd/server / app/common |

## Anti-Patterns

- Adding business logic to Noop — it must stay a trivial always-true stub
- Widening the Gate interface here instead of composing additional interfaces at the call site

## Decisions

- **Keep the interface to a single EvaluateBool method** — Lets every consumer (server, billing-worker, jobs, productcatalog http) depend on a tiny seam and swap real vs noop backends without touching call sites

<!-- archie:ai-end -->
