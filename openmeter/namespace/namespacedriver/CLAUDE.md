# namespacedriver

<!-- archie:ai-start -->

> Provides the NamespaceDecoder abstraction for extracting namespace context from HTTP requests. The only implementation is StaticNamespaceDecoder, which always returns a fixed string — used in self-hosted deployments where every request belongs to the default namespace.

## Patterns

**NamespaceDecoder interface** — New namespace-resolution strategies must implement GetNamespace(ctx context.Context) (string, bool) — returning false signals 'namespace not found' and lets callers reject the request. (`type StaticNamespaceDecoder string
func (d StaticNamespaceDecoder) GetNamespace(ctx context.Context) (string, bool) { return string(d), true }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `decoder.go` | Defines NamespaceDecoder interface and its sole StaticNamespaceDecoder implementation. Entire package is a single file — keep it that way unless adding a dynamic decoder. | Do not add business logic or imports of openmeter domain packages here; this file must remain import-free to avoid cycles with callers like openmeter/server/router. |

## Anti-Patterns

- Importing domain packages (billing, customer, entitlement) — this package sits at the infrastructure/transport boundary and must stay dependency-free
- Adding state or initialization logic to StaticNamespaceDecoder — it is intentionally a plain string type
- Returning true with an empty string from GetNamespace — callers treat empty namespace as valid and will silently scope queries to an unintended tenant

## Decisions

- **StaticNamespaceDecoder is a named string type rather than a struct** — Avoids constructor boilerplate for the common case where the namespace is a config value; the string itself IS the decoder.

<!-- archie:ai-end -->
