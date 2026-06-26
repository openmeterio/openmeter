# models

<!-- archie:ai-start -->

> The codebase's most-depended-on utility package (229 in-edges): shared domain primitives (ManagedModel/NamespacedModel/CadencedModel/Metadata/Annotations/Percentage), the canonical error taxonomy (Generic*Error + Validate() aggregation), and the ValidationIssue/FieldDescriptor system that maps Go error trees into RFC-7807 API problems. Almost every domain package embeds these types, so signatures here are effectively frozen public API.

## Patterns

**Generic error constructor + Is* checker pairs** — Each error class is a struct wrapping `err error` with `Error()`/`Unwrap()`, plus a `New<Name>Error` constructor and an `Is<Name>Error(err) bool` using `errors.As`, asserting `var _ GenericError = (*X)(nil)`. New error categories must follow this triad. (`NewGenericValidationError(err) / IsGenericValidationError(err) in errors.go`)
**Validate() aggregates with errors.Join then NewNillableGenericValidationError** — Validate() collects into `var errs []error`, wraps field context with `fmt.Errorf("field: %w", err)`, and returns `models.NewNillableGenericValidationError(errors.Join(errs...))` (nil-safe). NamespacedIDOrKey.Validate is the in-package example. (`return NewNillableGenericValidationError(errors.Join(errs...))`)
**ValidationIssue built only through options** — ValidationIssue has all-private fields; construct via NewValidationIssue/NewValidationError/NewValidationWarning and refine with immutable `.With*` methods (WithField, WithAttr, WithSeverity) which Clone first. Never set fields directly outside the package. (`NewValidationError(code, msg).WithField(NewFieldSelector("x"))`)
**Error-tree to ValidationIssues mapping** — AsValidationIssues walks an error tree: componentWrapper/fieldPrefixedWrapper add context, leaf ValidationIssues are collected, and unknown leaves under a wrapper become critical issues. Use ErrorWithFieldPrefix / ErrorWithComponent to attach path/component context that survives the walk. (`AsValidationIssues(errIn) in validationissue.go`)
**Cadenced time-interval model (inclusive-from, exclusive-to)** — CadencedModel{ActiveFrom, *ActiveTo} is the standard active-period type; ActiveTo nil means open-ended, ActiveTo==ActiveFrom means never active. Implement CadenceComparable.GetCadence() to use CadenceList overlap/continuity helpers. (`CadencedModel.IsActiveAt(t) — from inclusive, to exclusive`)
**Immutable map-types with Clone/Merge/Equal** — Metadata (map[string]string) and Annotations (map[string]interface{}) provide Clone (deep for Annotations via brunoga/deep, shallow maps.Clone for Metadata), Merge (right wins, returns new map), and Equal. Treat them as value types; mutate copies, not inputs. (`Annotations.Merge clones via deep.Copy so inputs are never mutated`)
**Marker-interface sealing** — Cadenced and Metadatad use unexported marker methods (cadenced()/annotated() returning private marker types) so only in-package types can satisfy them. Don't implement these externally; embed CadencedModel/MetadataModel instead. (`type cadencedMarker bool; func (c CadencedModel) cadenced() cadencedMarker`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errors.go` | Generic error taxonomy (NotFound/Conflict/Forbidden/Validation/NotImplemented/...) + ErrorWithFieldPrefix/ErrorWithComponent + ErrorSeverity enum. | Severity is inverted: lower numeric value = MORE severe (Critical=0, Warning=1); WithSeverityOrHigher filters with `<=`. |
| `validationissue.go` | ValidationIssue value type, option functions, ValidationIssues slice, AsValidationIssues/EncodeValidationIssues error-tree mapping. | ValidationIssue is not comparable (map attributes) so it implements Is() in addition to Unwrap(); WithAttr panics on nil/non-comparable keys. |
| `model.go` | ManagedModel/NamespacedModel/ManagedResource/ManagedUniqueResource/VersionedModel base structs embedded across all domains; Address/CountryCode. | NewManagedResource forces all timestamps to .UTC(); IsDeleted uses clock.Now() (mockable in tests). |
| `fielddescriptor.go` | Tree-backed field path builder producing both human String() and JSONPath() for ValidationIssue.field; uses pkg/treex. | With*/WithPrefix return clones (value receiver + ShallowClone); the original argument is never mutated — chain the return value. |
| `fieldexpression.go` | FieldExpression impls: FieldAttrValue, MultiFieldAttrValue, FieldArrIndex, WildCard — render as [key=value] / JSONPath [?(@.key=='value')]. | valueString() emits "!UNSUPPORTED" for value types other than numbers/string/Stringer. |
| `cadence.go / cadencelist.go` | CadencedModel period type and CadenceList[T] for overlap/sort/continuity validation over CadenceComparable items. | GetOverlaps/IsContinuous assume a SORTED list (NewSortedCadenceList); TODOs flag intent to migrate to timeutil.OpenPeriod. |
| `annotation.go / metadata.go / attributes.go` | Map-backed value types: Annotations (deep clone), Metadata (string map), Attributes (any keys, AsStringMap for serialization). | Annotations.GetInt rejects non-whole floats; Metadata.Merge returns nil when both sides empty. |
| `servicehook.go` | Generic ServiceHookRegistry[T] / ServiceHook[T] lifecycle hooks (Pre/Post Create/Update/Delete) + NoopServiceHook. | Registry uses a per-instance context-value loop guard so re-entrant hook calls short-circuit silently; register via RegisterHooks, never mutate .hooks. |

## Anti-Patterns

- Returning on the first validation failure instead of collecting into errs and returning NewNillableGenericValidationError(errors.Join(...)).
- Setting ValidationIssue fields directly or building one without the option constructors — fields are private and With* must Clone.
- Treating ErrorSeverity numerically backwards (Critical is the lowest value, not the highest).
- Mutating an Annotations/Metadata/Attributes argument in place instead of using Clone/Merge which return fresh maps.
- Implementing the Cadenced/Metadatad marker interfaces externally rather than embedding CadencedModel/MetadataModel.

## Decisions

- **Centralize the Generic*Error taxonomy + Is* helpers here** — Lets every layer map domain errors to HTTP status codes uniformly without per-package error types, keeping the v3 apierrors mapping table small.
- **Model field paths as a treex tree behind FieldDescriptor** — One builder yields both a readable string and a valid RFC-9535 JSONPath, so ValidationIssue.field is machine-queryable in API responses and prefixing nested errors composes cleanly.
- **ValidationIssue is immutable value-with-options** — Allows safe sharing, errors.Is comparison via Equal, and prefix/component enrichment while walking arbitrary error trees in AsValidationIssues.

## Example: Standard Validate() returning aggregated, field-prefixed validation issues

```
func (x Input) Validate() error {
	var errs []error
	if x.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if err := x.Inner.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("inner: %w", err))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

<!-- archie:ai-end -->
