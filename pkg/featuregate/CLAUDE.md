# featuregate

<!-- archie:ai-start -->

> Minimal namespace-scoped boolean feature-gate abstraction. Defines the Gate interface (EvaluateBool) and a Noop implementation that always returns true, allowing consumers to depend on the interface while gating logic is supplied elsewhere or disabled.

## Patterns

**Gate interface for all evaluations** — Consumers depend on the Gate interface and call EvaluateBool(namespace, flag, defaultValue) — never inspect a concrete gate type. (`enabled, err := gate.EvaluateBool(ns, "my-flag", false)`)
**Noop returns true (allow) by default** — NewNoop() returns a Noop whose EvaluateBool always returns (true, nil), enabling all gated paths when no real gate is wired. (`gate := featuregate.NewNoop()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `featuregate.go` | Sole file: Gate interface, Noop struct, NewNoop() constructor. | Noop.EvaluateBool ignores the defaultValue and always returns true — relying on the default param against a Noop gate is a no-op. |

## Anti-Patterns

- Type-asserting to the concrete Noop type instead of depending on the Gate interface.
- Assuming EvaluateBool honors defaultValue under Noop — Noop always returns true.

## Decisions

- **Noop defaults to enabled (returns true).** — Widely imported (server, app/common, billing worker, productcatalog); a permissive default lets features run when no gating backend is configured.

## Example: Depend on the Gate interface and evaluate a flag

```
import "github.com/openmeterio/openmeter/pkg/featuregate"

func NewHandler(gate featuregate.Gate) *Handler { return &Handler{gate: gate} }

func (h *Handler) do(ns string) error {
	enabled, err := h.gate.EvaluateBool(ns, "beta-path", false)
	if err != nil { return err }
	if enabled { /* new path */ }
	return nil
}
```

<!-- archie:ai-end -->
