# http

<!-- archie:ai-start -->

> Tiny conversion layer that maps the domain types models.Annotations and models.Metadata to/from their generated api.* counterparts. Exists so HTTP handlers (e.g. openmeter/meter/httphandler) translate annotation/metadata bags without redefining the cast at every call site.

## Patterns

**FromX / AsX directional naming** — Domain->API conversion is named From* (returns *api.T); API->domain conversion is named As* (returns models.T). Follow this paired naming for any new type added here. (`func FromAnnotations(a models.Annotations) *api.Annotations; func AsAnnotations(a *api.Annotations) models.Annotations`)
**Identical underlying type cast, not field copy** — models.Annotations and api.Annotations share the same underlying type, so conversion is a direct Go type conversion `(api.Annotations)(x)` wrapped with lo.ToPtr — never a manual field-by-field build. (`return lo.ToPtr((api.Annotations)(annotations))`)
**Nil-guard on pointer inputs** — As* functions take a pointer and must return nil (the zero map) when the input pointer is nil before dereferencing. (`if annotations == nil { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `annotation.go` | FromAnnotations / AsAnnotations converting between models.Annotations and *api.Annotations. | AsAnnotations must nil-check before dereferencing *annotations or it panics on a nil pointer. |
| `metadata.go` | FromMetadata / AsMetadata converting between models.Metadata and *api.Metadata. | Same nil-guard requirement as annotation.go; the two files are structurally identical and should stay in lockstep. |

## Anti-Patterns

- Manually copying map entries instead of a single underlying-type conversion — breaks the assumption that the domain and api types are byte-identical.
- Omitting the nil pointer guard in As* functions, causing a nil-dereference panic.
- Using ad-hoc names (ConvertAnnotations, ToModel) instead of the From*/As* convention used here and in the go-types-conversion skill.

## Decisions

- **Keep these casts in a shared pkg/models/http package rather than inline in each handler.** — Annotations/Metadata appear across many HTTP handlers; centralizing the trivial cast keeps the From*/As* naming consistent and avoids re-deriving the type-conversion at each site.

## Example: Adding a new annotation/metadata-style conversion in this package.

```
import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromMetadata(metadata models.Metadata) *api.Metadata {
	return lo.ToPtr((api.Metadata)(metadata))
}

func AsMetadata(metadata *api.Metadata) models.Metadata {
	if metadata == nil {
		return nil
	}
// ...
```

<!-- archie:ai-end -->
