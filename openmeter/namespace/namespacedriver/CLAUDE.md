# namespacedriver

<!-- archie:ai-start -->

> Provides the NamespaceDecoder abstraction for extracting namespace context from HTTP requests. The sole implementation, StaticNamespaceDecoder, always returns a fixed string — used in self-hosted single-tenant deployments where every request belongs to the default namespace.

## Patterns

**NamespaceDecoder interface contract** — All namespace-resolution strategies must implement GetNamespace(ctx context.Context) (string, bool). Returning false signals 'namespace not found'; callers must reject the request. Returning true with an empty string is a bug — callers treat empty namespace as valid. (`type StaticNamespaceDecoder string
func (d StaticNamespaceDecoder) GetNamespace(ctx context.Context) (string, bool) { return string(d), true }`)
**Named string type for zero-boilerplate decoders** — StaticNamespaceDecoder is a plain named string type so config values can be cast directly without a constructor. New decoders that require state should be structs; pure-string resolvers should follow the same named-type pattern. (`decoder := namespacedriver.StaticNamespaceDecoder(cfg.Namespace)`)
**Zero-import leaf package** — decoder.go imports only 'context'. This package sits at the infrastructure/transport boundary and must remain free of openmeter domain imports to avoid import cycles with callers such as openmeter/server/router. (`import "context" // only stdlib allowed here`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `decoder.go` | Defines the NamespaceDecoder interface and the StaticNamespaceDecoder implementation. The entire package is a single file — keep it that way unless a second decoder strategy with its own logic is added. | Do not add business logic, state, or domain package imports. Do not return true with an empty string from any GetNamespace implementation. |

## Anti-Patterns

- Importing openmeter domain packages (billing, customer, entitlement, meter) — creates import cycles with callers
- Adding initialization logic or fields to StaticNamespaceDecoder — it is intentionally a plain named string
- Returning (true, "") from GetNamespace — callers treat an empty namespace as valid and will silently misscope queries
- Placing multi-tenant namespace resolution logic here — dynamic namespace lookup belongs in a separate decoder implementation, not in this file
- Adding HTTP middleware or Chi handler code — request plumbing belongs in openmeter/server, not in this package

## Decisions

- **StaticNamespaceDecoder is a named string type rather than a struct** — Avoids constructor boilerplate for the common self-hosted case where the namespace is a plain config string; the string value itself is the decoder.
- **NamespaceDecoder returns (string, bool) rather than (string, error)** — Namespace absence is a routing concern, not an error condition; bool cleanly signals 'not found' without forcing callers to inspect error types.

## Example: Implementing a new static namespace decoder from a config value

```
import "github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"

// Cast the config string directly — no constructor needed.
decoder := namespacedriver.StaticNamespaceDecoder(conf.Namespace.Default)

// Callers use the interface:
ns, ok := decoder.GetNamespace(ctx)
if !ok {
    http.Error(w, "namespace not found", http.StatusUnauthorized)
    return
}
```

<!-- archie:ai-end -->
