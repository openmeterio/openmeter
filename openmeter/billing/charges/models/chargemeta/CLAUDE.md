# chargemeta

<!-- archie:ai-start -->

> Provides a reusable Ent mixin (chargemeta.Mixin) that injects all common charge fields (customer_id, service/billing/full-service periods, status, currency, managed_by, subscription references, advance_after, tax config) into every charge entity, plus generic Create[T]/Update[T]/MapFromDB[T] functions so each charge type shares identical field-mapping logic without duplication.

## Patterns

**RecursiveMixin composition** — Export Mixin as a type alias `type Mixin = entutils.RecursiveMixin[metaMixin]`; the inner metaMixin composes AnnotationsMixin + ResourceMixin. Never embed metaMixin directly. (`type Mixin = entutils.RecursiveMixin[metaMixin]`)
**Generic Create/Update/MapFromDB over interface constraints** — All three functions are type-parameterized over Creator[T]/Updater[T]/Getter[T] interfaces. Pass the Ent-generated builder/updater/getter; the function handles all field mapping uniformly. (`chargemeta.Create[*entdb.UsageBasedChargeCreate](creator, in)`)
**UTC normalization on all timestamps** — Every time field must call .UTC() before being set via Create or Update. MapFromDB also calls .UTC() when reading back. Never store or return non-UTC times. (`SetServicePeriodFrom(in.Intent.ServicePeriod.From.UTC())`)
**Atomic subscription reference triple** — All three subscription fields (subscription_id, subscription_phase_id, subscription_item_id) must be set atomically: either all non-nil or all nil. MapFromDB enforces this guard. (`if entity.GetSubscriptionID() != nil && entity.GetSubscriptionPhaseID() != nil && entity.GetSubscriptionItemID() != nil { ... }`)
**Partial unique index on unique_reference_id** — The unique_reference_id uniqueness constraint is a partial index `WHERE unique_reference_id IS NOT NULL AND deleted_at IS NULL`. Never add a plain UNIQUE constraint on this field. (`index.Fields("namespace", "customer_id", "unique_reference_id").Annotations(entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL")).Unique()`)
**Normalize before validate in Create/Update** — Call intent.Normalized() and meta.NormalizeOptionalTimestamp(advanceAfter) at the top of Create/Update before validation so raw input is coerced to canonical form first. (`in.Intent = in.Intent.Normalized()
in.AdvanceAfter = meta.NormalizeOptionalTimestamp(in.AdvanceAfter)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixin.go` | Single file containing the entire package: Mixin type alias, Creator/Updater/Getter interfaces, CreateInput/UpdateInput structs, and Create/Update/MapFromDB generic functions. | Adding a new field requires updating all three interfaces (Creator, Updater, Getter) and all three generic functions; missing any one breaks compilation for every charge entity that embeds this mixin. |

## Anti-Patterns

- Defining per-charge-type copies of common fields instead of embedding chargemeta.Mixin
- Storing timestamps without .UTC() normalization
- Setting only one or two of the three subscription reference fields
- Adding a non-partial unique index on unique_reference_id
- Calling Validate() before Normalized() — normalization must happen first

## Decisions

- **Type-parameterized Create/Update/MapFromDB over interface constraints rather than concrete Ent types** — Each charge entity (usage-based, flat-fee, credit-purchase) generates distinct Ent builder types; generic functions eliminate copy-paste of field-setting logic across all three.
- **intent.Normalized() + meta.NormalizeOptionalTimestamp called at the top of Create/Update** — Normalization before validation ensures invalid raw input (e.g. unrounded advance_after) is coerced first; validation then only sees canonical values.

## Example: Embed chargemeta into a new charge entity schema

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"

type MyCharge struct { ent.Schema }

func (MyCharge) Mixin() []ent.Mixin {
    return []ent.Mixin{chargemeta.Mixin{}}
}
```

<!-- archie:ai-end -->
