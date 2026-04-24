# chargemeta

<!-- archie:ai-start -->

> Provides a reusable Ent mixin (chargemeta.Mixin) that embeds all common charge fields (customer_id, service/billing/full-service periods, status, currency, managed_by, subscription references, advance_after) into every charge entity. Also exports generic Create[T]/Update[T]/MapFromDB[T] functions parameterized over typed Creator/Updater/Getter interfaces so each charge type can share this logic without code duplication.

## Patterns

**RecursiveMixin composition** — Use entutils.RecursiveMixin[metaMixin] as the exported Mixin type alias so the charge entity schema gets AnnotationsMixin, ResourceMixin, plus all domain-specific fields in one embed. (`type Mixin = entutils.RecursiveMixin[metaMixin]`)
**Generic Create/Update/MapFromDB** — All three functions are type-parameterized over Creator[T]/Updater[T]/Getter[T] interfaces. A new charge entity passes its generated Ent creator/updater/getter and these functions handle all field mapping uniformly. (`chargemeta.Create[*entdb.UsageBasedChargeCreate](creator, in)`)
**UTC normalization on all timestamps** — Every time field assigned via Create or Update must call .UTC() before setting. MapFromDB calls .UTC() when reading back. Never store or return non-UTC times. (`SetServicePeriodFrom(in.Intent.ServicePeriod.From.UTC())`)
**Partial subscription reference** — All three subscription fields (subscription_id, subscription_phase_id, subscription_item_id) must be set atomically: either all three are non-nil, or all three are nil. MapFromDB enforces this by only populating SubscriptionReference when all three Getters return non-nil. (`if entity.GetSubscriptionID() != nil && entity.GetSubscriptionPhaseID() != nil && entity.GetSubscriptionItemID() != nil { ... }`)
**Unique reference index with partial condition** — The unique_reference_id uniqueness index is a partial index: WHERE unique_reference_id IS NOT NULL AND deleted_at IS NULL. Do not add a plain unique constraint on this field. (`index.Fields("namespace", "customer_id", "unique_reference_id").Annotations(entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL")).Unique()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixin.go` | Single file containing the entire package: Mixin type, Creator/Updater/Getter interfaces, CreateInput/UpdateInput structs, and Create/Update/MapFromDB generic functions. | Adding new fields here requires updating all three interfaces (Creator, Updater, Getter) and all three generic functions; missing any one breaks compilation for every charge entity that embeds this mixin. |

## Anti-Patterns

- Defining per-charge-type copy of these fields instead of embedding chargemeta.Mixin
- Storing timestamps without .UTC() normalization
- Setting only one or two of the three subscription fields
- Adding a non-partial unique index on unique_reference_id

## Decisions

- **Type-parameterized Create/Update/MapFromDB over interface constraints rather than concrete Ent types** — Each charge entity (usage-based, flat-fee, credit-purchase) generates its own Ent builder types; generic functions avoid a copy-paste of field-setting logic across all three.
- **intent.Normalized() + meta.NormalizeOptionalTimestamp called at the top of Create/Update** — Normalization must happen before validation so invalid raw input (e.g. unrounded advance_after) is coerced first; validation then only sees canonical values.

## Example: Embed chargemeta into a new charge entity schema

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"

type MyCharge struct {
    ent.Schema
}

func (MyCharge) Mixin() []ent.Mixin {
    return []ent.Mixin{
        chargemeta.Mixin{},
    }
}
```

<!-- archie:ai-end -->
