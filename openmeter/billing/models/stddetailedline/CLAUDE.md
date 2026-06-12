# stddetailedline

<!-- archie:ai-start -->

> Domain model + Ent mixin for standard invoice detailed lines (the Base aggregate). Composes the totals, externalid, and creditsapplied value-objects with line metadata (currency, service period, quantity, per-unit amount, category, payment term) and provides generic DB Create/FromDB mapping plus goderive-generated equality.

## Patterns

**Composition over the shared value-object mixins** — Base embeds models.ManagedResource and totals.Totals/externalid.LineExternalIDs/creditsapplied.CreditsApplied; the Ent Mixin() composes ResourceMixin + AnnotationsMixin + totals.Mixin via entutils.RecursiveMixin. (`type Mixin struct { entutils.RecursiveMixin[mixinBase] }`)
**Generic Creator/DBGetter mapping interfaces** — Create[T Creator[T]] writes Base into any Ent builder satisfying Creator (which embeds externalid.LineExternalIDCreator + totals.Setter); FromDB[T DBGetter] reads it back, normalizing all times to UTC. (`func Create[T Creator[T]](creator Creator[T], line Base) T`)
**Aggregated Validate collecting errors with errors.Join** — Base.Validate appends per-field wrapped errors (fmt.Errorf("category: %w", err)) into errs and returns errors.Join; supports functional ValidateOption (IgnoreQuantityChecks). (`errs = append(errs, fmt.Errorf("service period: %w", err)); return errors.Join(errs...)`)
**goderive equality, not reflect.DeepEqual** — Base.Equal delegates to deriveEqualBase in derived.gen.go (regenerated via go:generate goderive); decimal/period fields compared with their Equal methods, not ==. (`func (l Base) Equal(other Base) bool { return deriveEqualBase(&l, &other) }`)
**Deprecated tax fields retained on the mixin** — tax_config/tax_code_id/tax_behavior are .Deprecated(...) because detailed lines now inherit tax from the parent standard line; kept until the rollout migration completes. (`field.JSON("tax_config", productcatalog.TaxConfig{}).Deprecated("detailed lines inherit tax configuration from their parent standard line")`)
**UTC normalization at the DB boundary** — Create and FromDB call .In(time.UTC) on every time.Time (service period, created/updated/deleted); CategoryRegular and InAdvancePaymentTerm are schema defaults. (`SetServicePeriodStart(line.ServicePeriod.From.In(time.UTC))`)
**Index-aware Compare ordering** — Compare[T Comparable] orders by Index (nil sorts last), then CreatedAt, then ID — used to deterministically sort detailed lines. (`if a.GetIndex() == nil && b.GetIndex() != nil { return 1 }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `model.go` | Base aggregate, Category enum + Values()/Validate, ValidateOption, Clone, Equal, Compare/Comparable. | Clone only deep-copies CreditsApplied (rest are value types); Quantity may legitimately be negative-checked unless IgnoreQuantityChecks is passed (UBB persists quantity only at issue time). |
| `mixin.go` | Ent schema fields/indexes/checks (currency varchar(3) immutable, numeric decimals, credits_applied JSONB, child_unique_reference_id non-empty check). | Editing fields requires make generate then atlas migrate diff. Several tax fields are .Deprecated — do not reuse them for new tax logic. |
| `mapping.go` | DBGetter interface and FromDB[T] mapper; embeds externalid.LineExternalIDGetter + totals.TotalsGetter. | CreditsApplied comes from a *CreditsApplied pointer via lo.FromPtr; all times normalized to UTC. |
| `create.go` | Creator[T] interface and Create[T] builder writer. | Must stay in sync with mixin Fields() and the Base struct; chains externalid.CreateLineExternalID and billingtotals.Set at the end. |
| `derived.gen.go` | goderive-generated deriveEqualBase / element equality. | Generated — DO NOT EDIT. Add fields to Base then run go generate; equality omits any field not regenerated. |
| `generate.go` | go:generate directive invoking goderive. | Required for derived.gen.go regeneration. |

## Anti-Patterns

- Editing derived.gen.go by hand instead of re-running goderive after changing Base.
- Reaching for the deprecated tax_config/tax_code_id/tax_behavior fields — tax is inherited from the parent standard line.
- Comparing Base or its decimal/period fields with == or reflect.DeepEqual instead of Equal/deriveEqualBase.
- Returning on the first validation failure instead of collecting into errs and errors.Join.
- Adding a Base field without updating mixin.go Fields(), create.go Creator, mapping.go DBGetter, and the goderive output together.

## Decisions

- **Detailed-line persistence is expressed via generic Creator/DBGetter interfaces over Ent builders rather than concrete types.** — Several Ent entities (billingstandardinvoicedetailedline, chargeflatfeerundetailedline, chargeusagebasedrundetailedline) share the same line shape; generics let one Create/FromDB serve all builders.
- **Equality is goderive-generated, not reflection-based.** — alpacadecimal and ClosedPeriod need semantic Equal comparison; reflect.DeepEqual would give wrong results on decimals.

## Example: Persist a detailed line into any Ent builder via the generic Create helper

```
import (
  "github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
  billingtotals "github.com/openmeterio/openmeter/openmeter/billing/models/totals"
)

func Create[T Creator[T]](creator Creator[T], line Base) T {
  create := creator.
    SetName(line.Name).
    SetCurrency(line.Currency).
    SetServicePeriodStart(line.ServicePeriod.From.In(time.UTC)).
    SetServicePeriodEnd(line.ServicePeriod.To.In(time.UTC)).
    SetQuantity(line.Quantity).
    SetPerUnitAmount(line.PerUnitAmount).
    SetCategory(line.Category).
    SetChildUniqueReferenceID(line.ChildUniqueReferenceID)
// ...
```

<!-- archie:ai-end -->
