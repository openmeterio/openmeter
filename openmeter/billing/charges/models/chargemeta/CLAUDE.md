# chargemeta

<!-- archie:ai-start -->

> Value-type + Ent-schema-mixin package for the shared charge-meta of every charge type (service/billing/full-service periods, status, currency, managed-by, subscription refs, tax config). It is a leaf models package: it defines the reusable Ent mixin and generic Create/Update/Map helpers that each charge-type schema (flatfee, usagebased, creditpurchase) embeds.

## Patterns

**Reusable Ent mixin via RecursiveMixin** — The schema is exposed as `type Mixin = entutils.RecursiveMixin[metaMixin]` so charge-type schemas embed it rather than redefining fields. metaMixin.Mixin() pulls in entutils.AnnotationsMixin and entutils.ResourceMixin. (`type Mixin = entutils.RecursiveMixin[metaMixin]`)
**Generic builder interfaces over concrete Ent types** — Create/Update/MapFromDB are written against typed-generic Creator[T]/Updater[T]/Getter[T] interfaces (Set*/Get* methods) instead of concrete *entdb.XCreate, so one helper serves every charge-type table. (`func Create[T Creator[T]](creator Creator[T], in CreateInput) (T, error)`)
**Intent normalize+validate before persist** — Create normalizes the Intent (`in.Intent.Normalized()`) and AdvanceAfter (`meta.NormalizeOptionalTimestamp`) and calls `in.Intent.Validate()` before any Set* call, returning the zero T on error. (`if err := in.Intent.Validate(); err != nil { var empty T; return empty, err }`)
**All timestamps coerced to UTC** — Every period boundary is stored via `.UTC()` and optional times via convert.SafeToUTC / convert.TimePtrIn(..., time.UTC); MapFromDB reads them back with `.UTC()`. (`SetServicePeriodFrom(in.Intent.ServicePeriod.From.UTC())`)
**Subscription/tax fields are all-or-nothing pointers** — Subscription reference is reconstructed only when ID, PhaseID and ItemID are all non-nil; tax config only when TaxCodeID or Behavior is set. New nullable fields must follow this grouped-pointer reconstruction in MapFromDB. (`if entity.GetSubscriptionID() != nil && entity.GetSubscriptionPhaseID() != nil && entity.GetSubscriptionItemID() != nil { ... }`)
**Unique reference scoped + partial index** — unique_reference_id is unique per (namespace, customer_id) only where it is non-null and not soft-deleted, via entsql.IndexWhere. (`index.Fields("namespace","customer_id","unique_reference_id").Annotations(entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL")).Unique()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixin.go` | Sole file: Ent field/index definitions plus generic CreateInput/Create, UpdateInput/Update, Getter/MapFromDB helpers that translate between meta.Charge and any embedding charge table. | Adding a field requires updating the field list, Creator/Updater/Getter interfaces, AND Create/Update/MapFromDB together, or the per-charge-type adapters won't compile. Tax provider-specific fields (e.g. Stripe.Code) are intentionally NOT stored here — they are resolved at invoice snapshot time. |

## Anti-Patterns

- Calling Set* on a concrete *entdb.XCreate directly instead of going through the generic Create/Update helpers — bypasses Intent normalization and validation.
- Persisting non-UTC timestamps; every boundary must be coerced with .UTC()/convert.*ToUTC.
- Reconstructing a SubscriptionReference or TaxCodeConfig from a partial set of its pointer fields.
- Storing provider-specific tax fields here; only TaxCodeID (FK) and Behavior belong on charge tables.

## Decisions

- **Schema and mapping live in one generic package embedded by each charge type** — flatfee, usagebased and creditpurchase share identical meta columns; a single RecursiveMixin + generic helpers avoids per-table duplication and keeps the meta.Charge contract consistent.
- **Tax config stored as FK + behavior only** — Provider-specific codes change and are resolved at invoice snapshot time, so charge rows stay stable and provider-agnostic.

## Example: Mapping an Ent charge row back to the domain meta.Charge via the generic Getter

```
func MapFromDB[T Getter[T]](entity T) meta.Charge {
	var taxConfig *productcatalog.TaxCodeConfig
	if entity.GetTaxCodeID() != nil || entity.GetTaxBehavior() != nil {
		taxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: entity.GetTaxCodeID(), Behavior: entity.GetTaxBehavior()}
	}
	return meta.Charge{ /* ManagedResource + Intent{... ServicePeriod: timeutil.ClosedPeriod{From: entity.GetServicePeriodFrom().UTC()}, TaxConfig: taxConfig} */ }
}
```

<!-- archie:ai-end -->
