# currencyx

<!-- archie:ai-start -->

> Currency primitives layered on invopop/gobl: a Code (ISO 4217) type used directly in the Ent schema, a Calculator that resolves currency precision once, and largest-remainder proportional allocation of currency amounts across weighted or capped buckets.

## Patterns

**Resolve a Calculator once, reuse it** — Get a Calculator via Code(code).Calculator() so the gobl currency Def is resolved a single time; downstream rounding/allocation assumes Def is non-nil and valid. (`calc, err := currencyx.Code("USD").Calculator()`)
**Round to currency subunits** — Use Calculator.RoundToPrecision / IsRoundedToPrecision (driven by Def.Subunits) for all currency rounding — USD=2 decimals, JPY=0, etc. (`amt = calc.RoundToPrecision(amt)`)
**Largest-remainder allocation** — AllocateByWeight (dimensionless weights) and AllocateByAmount (amount buckets that double as caps) split an amount with floor + remainder distribution at currency precision; both omit zero allocations and accept an optional CompareKey tie-breaker for determinism. (`allocs, err := currencyx.AllocateByWeight(calc, input)`)
**Aggregate validation errors with errors.Join** — validateWeightedAllocationInput / validateAmountAllocationInput collect all issues into []error and return errors.Join, matching the project Validate() convention. (`return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `currency.go` | Code type (alias of gobl currency.Code) + Calculator with Validate/RoundToPrecision/IsRoundedToPrecision. | Calculator.RoundToPrecision/IsRoundedToPrecision dereference Def without nil-checking; only construct Calculator via Code.Calculator() (or validate first). The zero-value Calculator{} is invalid. |
| `allocation.go` | AllocateByWeight / AllocateByAmount and their input/result structs plus validators and currencyUnit helper. | Inputs must already be rounded to currency precision and amount non-negative; AllocateByAmount errors if it cannot distribute the remainder without exceeding item caps, and rejects amount > total item amount. |

## Anti-Patterns

- Constructing Calculator{} directly and calling RoundToPrecision — Def is nil and will panic.
- Rounding currency amounts with ad-hoc decimal.Round instead of Calculator.RoundToPrecision (ignores per-currency subunits like JPY).
- Passing unrounded amounts to AllocateBy* — validation rejects them.
- Returning on the first validation error instead of joining all issues.

## Decisions

- **Calculator caches the gobl currency.Def so precision is resolved once and assumed valid.** — Avoids repeated currency.Get lookups and error handling across the many billing/ledger call sites that round amounts.
- **Allocation uses the largest-remainder quota method with an optional deterministic CompareKey.** — Guarantees allocated parts sum exactly to the input amount at currency precision while keeping results stable/reproducible across runs.

## Example: Proportionally split a currency amount across weighted keys

```
import "github.com/openmeterio/openmeter/pkg/currencyx"

calc, _ := currencyx.Code("USD").Calculator()
allocs, err := currencyx.AllocateByWeight(calc, currencyx.WeightedAllocationInput[string]{
    Amount: amount, // must be rounded to precision
    Items: []currencyx.WeightedAllocationItem[string]{{Key: "A", Weight: w1}, {Key: "B", Weight: w2}},
})
```

<!-- archie:ai-end -->
