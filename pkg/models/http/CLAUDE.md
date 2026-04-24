# http

<!-- archie:ai-start -->

> Provides thin bidirectional conversion helpers between internal domain model types (models.Annotations, models.Metadata) and their generated API wire types (api.Annotations, api.Metadata). Acts as a translation boundary layer so domain packages never import the generated api package directly.

## Patterns

**Direct type-cast conversion** — Internal and API types share the same underlying map type, so conversions are simple Go type assertions wrapped with nil-guards for pointer inputs. No field mapping or transformation logic belongs here. (`return (models.Annotations)(*annotations)`)
**Pointer wrapping with lo.ToPtr for outbound direction** — FromX functions (domain -> API) return a pointer to the converted value using lo.ToPtr, matching the optional/nullable pointer semantics used in the generated API types. (`return lo.ToPtr((api.Annotations)(annotations))`)
**Nil-guard on inbound pointer conversion** — AsX functions (API -> domain) check for nil before dereferencing the pointer and return nil domain value when the API field is absent. (`if annotations == nil { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `annotation.go` | Converts models.Annotations <-> *api.Annotations; FromAnnotations wraps outbound, AsAnnotations unwraps inbound. | api.Annotations and models.Annotations are both map[string]interface{} aliases — do not add transformation logic; keep as pure casts. |
| `metadata.go` | Identical pattern to annotation.go for models.Metadata <-> *api.Metadata. | Same alias-cast pattern; any structural divergence between api.Metadata and models.Metadata would silently break here at runtime. |

## Anti-Patterns

- Adding business logic or field validation inside these conversion helpers — they are pure structural translations.
- Importing from openmeter/* domain packages other than pkg/models — this package must stay a leaf with minimal imports.
- Returning a non-pointer (value type) from FromX helpers — callers expect *api.T to match generated optional fields.
- Skipping the nil check in AsX helpers — dereferencing a nil *api.T panics at runtime.
- Adding new types here that are not simple alias casts of the same underlying Go type.

## Decisions

- **Separate http sub-package under pkg/models instead of inline helpers in each httpdriver** — Centralises the api <-> models boundary in one place so all httpdriver packages share the same conversion, preventing drift between packages that handle annotations or metadata.
- **Use direct Go type casts instead of field-by-field mapping** — api.Annotations and models.Annotations are intentionally the same underlying map type; a cast is safer than a copy because it will fail to compile if the underlying types diverge.

## Example: Adding a new shared model <-> API conversion helper for a new alias type

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
