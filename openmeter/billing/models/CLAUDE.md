# models

<!-- archie:ai-start -->

> Organisational container for shared billing domain model sub-packages — each pairs a Go domain struct with its co-located Ent mixin so schema and type stay in sync. No source files live directly here; children are creditsapplied, externalid, stddetailedline, and totals.

## Patterns

**Ent mixin co-located with domain model** — Each sub-package defines the domain struct in model.go and the matching Ent mixin in mixin.go in the same package. (`// model.go: type Totals struct {...}
// mixin.go: func (Mixin) Fields() []ent.Field {...}`)
**Typed Setter/Getter generics over raw Ent mutations** — externalid and totals expose generic Creator[T]/Setter[T]/Getter so adapters never call individual Set* Ent methods. (`type Setter[T any] interface { SetTotals(T) *T }`)
**Currency-precision rounding before persistence** — Totals.RoundToPrecision(currency) must run before any DB write to prevent numeric drift; same discipline for creditsapplied sums. (`t = t.RoundToPrecision(currencyx.Calculator{Currency: currency})`)
**models.Validator on enums and structs** — Enum and struct types implement Validate() error; enums enforce a positive/non-negative invariant (creditsapplied positivity, totals non-negative). (`func (s Status) Validate() error { switch s { case ...: return nil; default: return fmt.Errorf("invalid status: %s", s) } }`)
**goderive for equality on stddetailedline.Base** — derived.gen.go is generated from generate.go annotations; never hand-edit the generated file. (`//go:generate go run github.com/awalterschulze/goderive .`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/models/totals/model.go` | Totals struct with Add, Sum, RoundToPrecision, Validate; non-negative invariant. | Adding a field requires updating Add, Sub, Neg, Equal, IsZero, RoundToPrecision, Validate, mixin Fields(), Setter[T], TotalsGetter, Set[T], and FromDB together. |
| `openmeter/billing/models/externalid/mixin.go` | Ent mixin for invoicing/payment external-ID columns plus generic Creator/Updater/Getter. | Use lo.EmptyableToPtr for nullable string columns — never store empty strings; update LineExternalIDs.Equal when adding a field. |
| `openmeter/billing/models/stddetailedline/model.go` | Base struct for flat-fee and usage-based detailed lines. | child_unique_reference_id has a DB CHECK and is the line upsert identity — never make it optional; update Creator[T] and DBGetter on field changes. |
| `openmeter/billing/models/creditsapplied/model.go` | CreditsApplied slice with Validate (positivity) and currency-aware Sum. | Accumulate via Sum with RoundToPrecision; use .Equal() not == on alpacadecimal.Decimal. |
| `openmeter/billing/models/stddetailedline/derived.gen.go` | Goderive-generated equality for Base. | Regenerate via go generate after any Base field change — never hand-edit. |

## Anti-Patterns

- Adding DB/Ent logic to model.go or domain logic to mixin.go — keep layers strictly separated
- Hand-editing derived.gen.go instead of regenerating via go generate
- Skipping RoundToPrecision before persisting or comparing Totals / CreditsApplied
- Storing empty strings in nullable external-ID columns instead of using lo.EmptyableToPtr
- Adding a new field to Totals/Base without updating every companion method and the mixin

## Decisions

- **Mixin and domain model co-located in the same sub-package rather than a shared mixin layer** — Keeps schema and Go representation in sync; a single file change updates both the DB column and its type.
- **goderive for equality rather than hand-written Equal on stddetailedline.Base** — Base has deeply nested pointer/slice fields; generated equality is complete and removes maintenance burden.

<!-- archie:ai-end -->
