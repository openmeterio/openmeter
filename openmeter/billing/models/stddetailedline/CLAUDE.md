# stddetailedline

<!-- archie:ai-start -->

> Defines the Base struct for standard detailed billing lines (flat-fee and usage-based), the Ent mixin providing all DB columns, generic Creator/DBGetter interfaces for Ent mutations and mapping, and goderive-generated equality — the shared schema/model layer consumed by both Ent adapters and domain services.

## Patterns

**Composite Ent mixin via entutils.RecursiveMixin** — Mixin embeds entutils.RecursiveMixin[mixinBase] composing AnnotationsMixin, ResourceMixin, and totals.Mixin. New fields go in mixinBase.Fields(), not entity schemas. (`type Mixin struct { entutils.RecursiveMixin[mixinBase] }`)
**Generic Creator[T] for Ent create mutations** — Creator[T] composes externalid.LineExternalIDCreator[T], billingtotals.Setter[T], and primitive Set* methods; Create[T] atomically populates all fields. (`func Create[T Creator[T]](creator Creator[T], line Base) T`)
**DBGetter for DB-to-domain mapping** — FromDB[T DBGetter] reads all fields via the getter interface; all time.Time fields normalize to UTC via .In(time.UTC). (`func FromDB[T DBGetter](dbEntity T, taxConfig *productcatalog.TaxConfig) Base`)
**goderive equality — never hand-edit derived.gen.go** — Base.Equal delegates to deriveEqualBase; run go generate after adding fields. The generator uses .Equal() for decimal fields. (`func (l Base) Equal(other Base) bool { return deriveEqualBase(&l, &other) }`)
**Deep clone of pointer/slice fields** — Base.Clone deep-copies TaxConfig pointer and CreditsApplied slice; new pointer/slice fields must be deep-copied to avoid aliasing. (`if l.TaxConfig != nil { taxConfig := *l.TaxConfig; l.TaxConfig = &taxConfig }`)
**Enum Values()/Validate() with assertion** — Category implements models.Validator via slices.Contains against Values(); new enums follow the same pattern with var _ models.Validator = (*T)(nil). (`var _ models.Validator = (*Category)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Base struct, Category enum, Comparable interface, Compare, BackfillTaxConfig. | Adding a Base field requires updating Clone, Equal (goderive regen), DBGetter, Creator[T], FromDB, and mixin.go Fields() — all six together. |
| `mixin.go` | Ent mixin providing all DB fields, SQL CHECK constraints, and indexes. | child_unique_reference_id has a NOT EMPTY CHECK — never make it optional. credits_applied is jsonb-optional with a pointer DB getter. |
| `create.go` | Generic Create[T] populating Ent create mutations from a Base. | Must stay in sync with all Base fields; a missing field silently omits it from writes. time.Time values use .In(time.UTC). |
| `mapping.go` | DBGetter interface and FromDB generic mapping. | Apply time.UTC normalization to all time fields. GetCreditsApplied returns a pointer — dereference with lo.FromPtr. |
| `derived.gen.go` | Goderive-generated equality. DO NOT EDIT. | After adding decimal fields, verify regenerated code uses .Equal() not ==. |
| `generate.go` | go:generate directive for goderive. | Run after any Base change to regenerate derived.gen.go. |

## Anti-Patterns

- Hand-editing derived.gen.go — always regenerate via go generate.
- Adding DB/Ent logic to model.go or domain logic to mixin.go.
- Omitting time.UTC normalization in FromDB.
- Forgetting to update Creator[T] and DBGetter when adding a Base field.
- Making child_unique_reference_id optional — it has a DB CHECK and is the line upsert identity.

## Decisions

- **Base struct co-located with the Ent mixin rather than a separate domain package.** — Ent schema and domain model must stay lock-step; co-location makes drift visible and lets Creator/DBGetter share package types.
- **goderive for equality rather than hand-written Equal.** — Base has alpacadecimal.Decimal fields needing .Equal() not ==; goderive generates correct field-by-field equality kept in sync via go generate.

## Example: Add a string field to Base and wire it through all layers

```
// 1. model.go: add field to Base
type Base struct { /* ... */ NewField string `json:"newField"` }
// Update Clone() if pointer/slice.
// 2. mixin.go: add DB field
field.String("new_field").NotEmpty()
// 3. create.go: add to Creator[T] + Create[T]
type Creator[T any] interface { /* ... */ SetNewField(string) T }
// 4. mapping.go: add to DBGetter + FromDB; then go generate for derived.gen.go
```

<!-- archie:ai-end -->
