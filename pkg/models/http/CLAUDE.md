# http

<!-- archie:ai-start -->

> Thin bidirectional conversion layer between internal domain model types (models.Annotations, models.Metadata) and their generated API wire types (api.Annotations, api.Metadata). Acts as an isolated translation boundary so no domain package ever imports the generated api package directly.

## Patterns

**Direct Go type-cast conversion** — Internal and API types share the same underlying map type; conversions are pure Go type assertions, never field-by-field copies. Zero transformation logic lives here. (`return (models.Annotations)(*annotations)`)
**Pointer wrapping with lo.ToPtr for outbound direction** — FromX functions (domain -> API) return *api.T via lo.ToPtr to match optional/nullable pointer semantics in generated API types. (`func FromAnnotations(annotations models.Annotations) *api.Annotations { return lo.ToPtr((api.Annotations)(annotations)) }`)
**Nil-guard on inbound pointer conversion** — AsX functions (API -> domain) always check for nil before dereferencing the input pointer and return a nil domain value when the API field is absent. (`func AsAnnotations(annotations *api.Annotations) models.Annotations { if annotations == nil { return nil }; return (models.Annotations)(*annotations) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `annotation.go` | Converts models.Annotations <-> *api.Annotations. FromAnnotations wraps outbound; AsAnnotations unwraps inbound with nil-guard. | api.Annotations and models.Annotations are both map[string]interface{} aliases — any structural divergence between them would silently break the cast at runtime without a compile error. |
| `metadata.go` | Identical alias-cast pattern to annotation.go for models.Metadata <-> *api.Metadata. | Do not add transformation logic; if api.Metadata and models.Metadata ever diverge structurally, the cast will panic at runtime, not at compile time. |

## Anti-Patterns

- Adding business logic, field validation, or data transformation inside these helpers — they are pure structural casts.
- Importing from openmeter/* domain packages other than pkg/models — this package must remain a leaf with minimal imports.
- Returning a non-pointer value type from FromX helpers — callers expect *api.T to match generated optional fields.
- Skipping the nil check in AsX helpers — dereferencing a nil *api.T panics at runtime.
- Adding new types here that are not simple alias casts of the same underlying Go type — non-alias conversions belong in dedicated converter packages (e.g., Goverter-generated convert.gen.go).

## Decisions

- **Separate http sub-package under pkg/models instead of inline helpers in each httpdriver package** — Centralises the api <-> models translation boundary so all httpdriver packages share the same conversion, preventing drift between packages that handle annotations or metadata independently.
- **Use direct Go type casts instead of field-by-field mapping** — api.Annotations and models.Annotations are intentionally the same underlying map type; a cast fails at compile time if types diverge, whereas a copy would silently succeed with stale fields.

## Example: Adding a new shared alias-cast conversion helper (e.g., models.Labels <-> *api.Labels)

```
package http

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

// FromLabels converts a domain Labels to an API-wire pointer (outbound).
func FromLabels(labels models.Labels) *api.Labels {
	return lo.ToPtr((api.Labels)(labels))
}

// AsLabels converts an API-wire pointer to a domain Labels (inbound).
// ...
```

<!-- archie:ai-end -->
