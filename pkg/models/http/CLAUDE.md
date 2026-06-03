# http

<!-- archie:ai-start -->

> Thin bidirectional conversion layer between internal domain model types (models.Annotations, models.Metadata) and their generated API wire types (api.Annotations, api.Metadata). Acts as an isolated translation boundary so no domain package imports the generated api package for these shared map types.

## Patterns

**Direct Go type-cast conversion** — Internal and API types share the same underlying map type; conversions are pure Go type assertions, never field-by-field copies. Zero transformation logic. (`return (models.Annotations)(*annotations)`)
**Pointer wrapping with lo.ToPtr outbound (FromX)** — FromX functions (domain -> API) return *api.T via lo.ToPtr to match optional/nullable pointer semantics in generated API types. (`func FromAnnotations(annotations models.Annotations) *api.Annotations { return lo.ToPtr((api.Annotations)(annotations)) }`)
**Nil-guard on inbound pointer conversion (AsX)** — AsX functions (API -> domain) always check for nil before dereferencing the input pointer and return a nil domain value when the API field is absent. (`func AsAnnotations(annotations *api.Annotations) models.Annotations { if annotations == nil { return nil }; return (models.Annotations)(*annotations) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `annotation.go` | Converts models.Annotations <-> *api.Annotations. FromAnnotations wraps outbound; AsAnnotations unwraps inbound with nil-guard. | api.Annotations and models.Annotations are both map alias types — any structural divergence would break the cast at runtime, not compile time. |
| `metadata.go` | Identical alias-cast pattern to annotation.go for models.Metadata <-> *api.Metadata (FromMetadata / AsMetadata). | Do not add transformation logic; if api.Metadata and models.Metadata diverge structurally, the cast fails at runtime, not at compile time. |

## Anti-Patterns

- Adding business logic, field validation, or data transformation inside these helpers — they are pure structural casts.
- Importing from openmeter/* domain packages other than pkg/models — this package must remain a leaf.
- Returning a non-pointer value type from FromX helpers — callers expect *api.T to match generated optional fields.
- Skipping the nil check in AsX helpers — dereferencing a nil *api.T panics at runtime.
- Adding types here that are not simple alias casts of the same underlying Go type — non-alias conversions belong in Goverter-generated converters.

## Decisions

- **Separate http sub-package under pkg/models instead of inline helpers in each httpdriver package.** — Centralises the api <-> models translation boundary so all httpdriver packages share one conversion, preventing drift between packages handling annotations or metadata.
- **Use direct Go type casts instead of field-by-field mapping.** — api.Annotations and models.Annotations are intentionally the same underlying map type; a cast fails at compile time if types diverge, whereas a copy would silently succeed with stale fields.

## Example: Adding a new shared alias-cast conversion helper (e.g., models.Labels <-> *api.Labels)

```
package http

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromLabels(labels models.Labels) *api.Labels {
	return lo.ToPtr((api.Labels)(labels))
}

func AsLabels(labels *api.Labels) models.Labels {
	if labels == nil {
// ...
```

<!-- archie:ai-end -->
