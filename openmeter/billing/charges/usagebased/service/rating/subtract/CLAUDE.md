# subtract

<!-- archie:ai-start -->

> Pure arithmetic primitive: subtracts previously-rated DetailedLines from currently-rated DetailedLines and returns the remaining lines. Does not query meters, call billing rating, allocate credits, or decide invoice periods. Shared subtraction backend for the delta and period-preserving engines.

## Patterns

**Match by calculation key, not ChildUniqueReferenceID** — Lines are grouped by (PricerReferenceID, Category, PaymentTerm); within a key, matched by PerUnitAmount via decimal equality. ChildUniqueReferenceID is a caller-assigned persistence identity, never a matching key. (`key := detailedLineKey{ReferenceID: line.PricerReferenceID, Category: line.Category, PaymentTerm: line.PaymentTerm}`)
**Repricing via PerUnitAmount mismatch** — When current and previous share a calculation key but differ in PerUnitAmount, emit the current line as CurrentOnly and the previous as PreviousOnlyReversal (negative) — a price change, not a quantity delta. (`findDetailedLineByPerUnitAmount uses decimal equality; unmatched previous lines become PreviousOnlyReversal`)
**Three generator callbacks for output identity** — Callers implement UniqueReferenceIDGenerator with CurrentOnly, MatchedDelta, PreviousOnlyReversal; each returns the ChildUniqueReferenceID for its output line. (`type UniqueReferenceIDGenerator interface { CurrentOnly(line) (string, error); MatchedDelta(current, previous line) (string, error); PreviousOnlyReversal(line) (string, error) }`)
**Zero-total line suppression** — Lines whose Totals.IsZero() are dropped — includes matched deltas where current==previous and reversals netting to zero. (`func isZeroDetailedLine(line usagebased.DetailedLine) bool { return line.Totals.IsZero() }`)
**WithUniqueReferenceIDValidationIgnored for intermediate use** — Pass this option for intermediate epoch arithmetic (period-preserving engine). Final output must keep validation enabled (the default). (`subtract.SubtractRatedRunDetails(curr, prev, gen, subtract.WithUniqueReferenceIDValidationIgnored())`)
**Deterministic key ordering in output** — Keys are sorted by (ReferenceID, Category, PaymentTerm); within a key, lines by PerUnitAmount ascending — stable output across runs. (`sort.Slice(keys, func(i, j int) bool { if keys[i].ReferenceID != keys[j].ReferenceID { return keys[i].ReferenceID < keys[j].ReferenceID } ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subtract.go` | Single exported function SubtractRatedRunDetails: validates inputs and currency consistency, groups by calculation key, matches by PerUnitAmount, delegates output identity to the generator, optionally validates uniqueness. | Negative quantities are valid inputs (existing reversals); validation clones and negates before calling Validate(). Do not add a sign check to inputs. Generator must be non-nil. |
| `uniquereferenceid.go` | Defines only the UniqueReferenceIDGenerator interface; implementations live in delta/ and periodpreserving/. | The interface must stay stable because both engines depend on it. |
| `subtract_test.go` | Table-driven tests covering unit/tiered/volume subtraction, rounding correction, no-delta suppression, and currency mismatch. | Tests call billingratingservice.New() (real billing-rating, not a stub); changes to billing-rating output shapes break these tests. |

## Anti-Patterns

- Using ChildUniqueReferenceID as the arithmetic matching key — it is a generator-assigned persistence identity.
- Comparing PerUnitAmount as a string map key — use decimal Equal to avoid treating 1.0 and 1.00 as different.
- Using SubtractRatedRunDetails as a standalone invoice rating algorithm — it is a low-level primitive needing correct cumulative, credit-stripped inputs.
- Suppressing zero-quantity lines without isZeroDetailedLine (Totals.IsZero()) — a line can have zero quantity but non-zero totals (discount-only lines).
- Skipping WithUniqueReferenceIDValidationIgnored() for intermediate epoch arithmetic — produces valid duplicate intermediate references before period stamping.

## Decisions

- **Match on (PricerReferenceID, Category, PaymentTerm) + PerUnitAmount, not ChildUniqueReferenceID.** — ChildUniqueReferenceID is output identity, not arithmetic identity; matching by it would hide repricing (same line, different price) as a price change.
- **UniqueReferenceIDGenerator is a caller-supplied strategy.** — Delta and period-preserving engines need different correction ID schemes; the strategy keeps subtract.go free of engine-specific ID formatting.

## Example: Implement a UniqueReferenceIDGenerator for a new engine

```
type myGenerator struct{}
func (myGenerator) CurrentOnly(line usagebased.DetailedLine) (string, error) { return line.ChildUniqueReferenceID, nil }
func (myGenerator) MatchedDelta(current, _ usagebased.DetailedLine) (string, error) { return current.ChildUniqueReferenceID, nil }
func (myGenerator) PreviousOnlyReversal(line usagebased.DetailedLine) (string, error) {
  if line.ID == "" { return "", fmt.Errorf("line id required for correction") }
  return fmt.Sprintf("%s#correction:detailed_line_id=%s", line.PricerReferenceID, line.ID), nil
}
```

<!-- archie:ai-end -->
