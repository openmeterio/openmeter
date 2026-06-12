# convert

<!-- archie:ai-start -->

> Generic pointer/value and time conversion helpers used heavily by API<->domain<->DB mapping code. Notably provides empty-aware pointer wrappers for maps and slices that lo.EmptyableToPtr handles incorrectly.

## Patterns

**Empty-aware container pointers** — Use MapToPointer / SliceToPointer to convert maps/slices to pointers, returning nil when empty — these exist specifically because lo.EmptyableToPtr mishandles maps and slices. (`field := convert.SliceToPointer(items) // nil if len==0`)
**Nil-safe dereference helpers** — SafeDeRef(ptr, fn) and DerefHeaderPtr guard nil before applying transforms; SafeToUTC/TimePtrIn wrap SafeDeRef for *time.Time conversions. (`utc := convert.SafeToUTC(t)`)
**Underlying-string conversions** — ToStringLike converts between distinct ~string types via pointers; StringerPtrToStringPtr turns a *Stringer into *string, both nil-preserving. (`s := convert.StringerPtrToStringPtr(code)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ptr.go` | Pointer helpers: ToPointer, MapToPointer, SliceToPointer, ToStringLike, SafeDeRef, SafeToUTC, DerefHeaderPtr, StringerPtrToStringPtr. | ToPointer duplicates lo.ToPtr; prefer lo.ToPtr in new code (AGENTS.md) and reserve this package for the empty-aware/nil-safe helpers lo lacks. |
| `time.go` | TimePtrIn converts a *time.Time into a given *time.Location, nil-preserving. | Returns nil for nil input — callers must handle the nil result. |

## Anti-Patterns

- Using lo.EmptyableToPtr on maps/slices — use MapToPointer/SliceToPointer which correctly return nil on empty.
- Adding a plain ToPointer call when lo.ToPtr already covers it (avoid local pointer-wrapper proliferation).

## Decisions

- **Maintain custom MapToPointer/SliceToPointer despite samber/lo availability.** — lo.EmptyableToPtr does not treat empty maps/slices as nil, which is required for omitempty-style API mapping.

## Example: Map a possibly-empty slice to an omittable API field

```
import "github.com/openmeterio/openmeter/pkg/convert"

out.Items = convert.SliceToPointer(domain.Items) // nil when empty
```

<!-- archie:ai-end -->
