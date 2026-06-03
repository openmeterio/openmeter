# namespacedriver

<!-- archie:ai-start -->

> Provides the NamespaceDecoder abstraction for extracting namespace context from HTTP requests. The sole implementation, StaticNamespaceDecoder, always returns a fixed string — used in self-hosted single-tenant deployments where every request belongs to the default namespace.

## Patterns

**NamespaceDecoder interface contract** — All namespace-resolution strategies implement GetNamespace(ctx) (string, bool). Returning false signals 'namespace not found' and callers must reject; returning true with an empty string is a bug (callers treat empty as valid). (`type StaticNamespaceDecoder string; func (d StaticNamespaceDecoder) GetNamespace(ctx context.Context) (string, bool) { return string(d), true }`)
**Named string type for zero-boilerplate decoders** — StaticNamespaceDecoder is a plain named string type so config values cast directly without a constructor. Stateful decoders should be structs; pure-string resolvers follow the named-type pattern. (`decoder := namespacedriver.StaticNamespaceDecoder(cfg.Namespace)`)
**Zero-import leaf package** — decoder.go imports only 'context'. This package sits at the infrastructure/transport boundary and must stay free of openmeter domain imports to avoid cycles with callers like openmeter/server/router. (`import "context" // only stdlib allowed here`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `decoder.go` | Defines the NamespaceDecoder interface and StaticNamespaceDecoder. The entire package is one file. | Do not add business logic, state, or domain imports. Do not return true with an empty string from any GetNamespace implementation. |

## Anti-Patterns

- Importing openmeter domain packages (billing, customer, entitlement, meter) — creates import cycles with callers.
- Adding initialization logic or fields to StaticNamespaceDecoder — it is intentionally a plain named string.
- Returning (true, "") from GetNamespace — callers treat an empty namespace as valid and silently misscope queries.
- Placing multi-tenant namespace resolution logic here — dynamic lookup belongs in a separate decoder implementation.
- Adding HTTP middleware or Chi handler code — request plumbing belongs in openmeter/server.

## Decisions

- **StaticNamespaceDecoder is a named string type rather than a struct.** — Avoids constructor boilerplate for the common self-hosted case where the namespace is a plain config string.
- **NamespaceDecoder returns (string, bool) rather than (string, error).** — Namespace absence is a routing concern, not an error; bool cleanly signals 'not found' without forcing callers to inspect error types.

## Example: Implement a static namespace decoder from a config value

```
import "github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"

decoder := namespacedriver.StaticNamespaceDecoder(conf.Namespace.Default)

ns, ok := decoder.GetNamespace(ctx)
if !ok {
  http.Error(w, "namespace not found", http.StatusUnauthorized)
  return
}
```

<!-- archie:ai-end -->
