# namespacedriver

<!-- archie:ai-start -->

> Defines the NamespaceDecoder seam that every HTTP driver/handler uses to resolve the active tenant namespace from a request context, decoupling handlers from how the namespace is determined (static config today, auth-derived later).

## Patterns

**Single-method decoder interface** — NamespaceDecoder exposes exactly one method GetNamespace(ctx) (string, bool); the bool reports presence rather than returning an error. Handlers consume the interface, never a concrete type. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { /* reject */ }`)
**Depend on interface, inject concrete** — Downstream httpdrivers store a namespacedriver.NamespaceDecoder field; wiring (app/common/namespace.go, openmeter/server/router) supplies the concrete StaticNamespaceDecoder. (`namespaceDecoder namespacedriver.NamespaceDecoder // field type is the interface`)
**Context-only input** — Resolution takes only context.Context. The namespace is expected to be carried in/derived from ctx, not passed as request parameters. (`func (d StaticNamespaceDecoder) GetNamespace(ctx context.Context) (string, bool)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `decoder.go` | Declares the NamespaceDecoder interface and the StaticNamespaceDecoder string implementation that always returns its configured namespace with ok=true. | StaticNamespaceDecoder ignores ctx and never returns ok=false; do not rely on it to reject missing namespaces. Keep this package dependency-free (only stdlib context) since ~25 httpdrivers import it. |

## Anti-Patterns

- Adding domain, DB, or auth dependencies to this package — it must stay a leaf imported by every httpdriver without cycles.
- Returning an error instead of the (string, bool) contract from GetNamespace implementations.
- Having handlers branch on the concrete StaticNamespaceDecoder type instead of the NamespaceDecoder interface.
- Reading the namespace from request body/query params here rather than from context.

## Decisions

- **Model namespace resolution as a one-method interface in its own tiny package.** — Lets a static, config-driven namespace be swapped for an auth/tenant-derived one later without touching the many httpdrivers that depend on it.
- **Use (string, bool) instead of (string, error).** — Absence of a namespace is a normal control-flow case for handlers, not an exceptional one; the bool keeps call sites terse.

## Example: Resolving the namespace in an HTTP handler

```
import "github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"

type handler struct {
	namespaceDecoder namespacedriver.NamespaceDecoder
}

func (h *handler) resolveNamespace(ctx context.Context) (string, error) {
	ns, ok := h.namespaceDecoder.GetNamespace(ctx)
	if !ok {
		return "", fmt.Errorf("namespace not found")
	}
	return ns, nil
}
```

<!-- archie:ai-end -->
