# subtract

<!-- archie:ai-start -->

> Pure arithmetic primitive: subtracts previously-rated DetailedLines from currently-rated DetailedLines and returns the remaining lines. Does not query meters, call billing rating, allocate credits, or decide invoice periods. Used as the shared subtraction backend by both the delta and period-preserving rating engines.

## Patterns

**Matching by calculation key, not ChildUniqueReferenceID** — Lines are matched by the (PricerReferenceID, Category, PaymentTerm) key. Within a key, matching is by PerUnitAmount using decimal equality. ChildUniqueReferenceID is a persistence identity assigned by the caller-supplied UniqueReferenceIDGenerator, not used for matching. (`key := detailedLineKey{ReferenceID: line.PricerReferenceID, Category: line.Category, PaymentTerm: line.PaymentTerm}`)
**Repricing via PerUnitAmount mismatch** — When current and previous lines share the same calculation key but differ in PerUnitAmount, the current line is emitted as CurrentOnly and the previous line as PreviousOnlyReversal (negative). This represents a price change, not a quantity delta. (`findDetailedLineByPerUnitAmount scans with decimal equality; if not found, generator.CurrentOnly is called for the current line and unmatched previous lines become PreviousOnlyReversal`)
**Three generator callbacks for output identity** — Callers implement UniqueReferenceIDGenerator with three methods: CurrentOnly (no matching previous), MatchedDelta (quantity delta of a matched pair), PreviousOnlyReversal (previous line absent from current). Each method determines the ChildUniqueReferenceID for its output line. (`type UniqueReferenceIDGenerator interface { CurrentOnly(line) (string, error); MatchedDelta(current, previous line) (string, error); PreviousOnlyReversal(line) (string, error) }`)
**Zero-total line suppression** — Lines whose Totals.IsZero() are silently dropped from output. This includes matched deltas where current and previous are identical and correction reversals that net to zero. (`func isZeroDetailedLine(line usagebased.DetailedLine) bool { return line.Totals.IsZero() }`)
**WithUniqueReferenceIDValidationIgnored for intermediate use** — Pass WithUniqueReferenceIDValidationIgnored() when SubtractRatedRunDetails is used for intermediate epoch arithmetic (period-preserving engine). Final output must always have validation enabled. (`subtract.SubtractRatedRunDetails(curr, prev, gen, subtract.WithUniqueReferenceIDValidationIgnored())`)
**Deterministic key ordering in output** — Keys are sorted (by ReferenceID, Category, PaymentTerm) before iteration so output order is stable across runs. Within a key, lines are sorted by PerUnitAmount ascending. (`sort.Slice(keys, func(i, j int) bool { if keys[i].ReferenceID != keys[j].ReferenceID { return keys[i].ReferenceID < keys[j].ReferenceID } ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subtract.go` | The single exported function SubtractRatedRunDetails. Validates inputs, groups by calculation key, matches by PerUnitAmount within key, delegates to UniqueReferenceIDGenerator for output identity, optionally validates ChildUniqueReferenceID uniqueness. | Negative quantities in input lines are accepted and valid (they represent existing reversals). Validation clones lines and negates quantity before calling Validate() so negative inputs pass. Do not add a sign check to inputs. |
| `uniquereferenceid.go` | Defines the UniqueReferenceIDGenerator interface. Implementations are in the consuming engines (delta/uniquereferenceid.go, periodpreserving/uniquereferenceid.go). | This file only defines the interface; implementations live in the engine packages. The interface must stay stable because both engines depend on it. |
| `subtract_test.go` | Table-driven unit tests covering unit price, tiered, and volume price subtraction scenarios including rounding correction, no-delta suppression, and currency mismatch validation. | Tests construct AlreadyBilledDetailedLines manually and call billingratingservice.New() to generate current lines — this exercises the real billing-rating service, not a stub. Changes to billing-rating output shapes will break these tests. |

## Anti-Patterns

- Do not use ChildUniqueReferenceID as the arithmetic matching key — it is a persistence identity assigned by the generator, not a semantic matching criterion.
- Do not compare PerUnitAmount as a string map key — use decimal equality (Equal) to avoid unsafe comparisons that treat 1.0 and 1.00 as different.
- Do not use SubtractRatedRunDetails as a standalone invoice rating algorithm — it is a low-level arithmetic primitive; callers must supply the correct cumulative context and credit-stripped inputs.
- Do not suppress zero-quantity lines without calling isZeroDetailedLine (which checks Totals.IsZero()) — a line can have zero quantity but non-zero totals (e.g., discount-only lines).
- Do not skip WithUniqueReferenceIDValidationIgnored() for intermediate epoch arithmetic — the period-preserving engine produces duplicate intermediate references that are valid before period stamping.

## Decisions

- **Match on (PricerReferenceID, Category, PaymentTerm) + PerUnitAmount, not ChildUniqueReferenceID.** — ChildUniqueReferenceID is the output identity, not the arithmetic identity. Matching by it would prevent the engine from detecting repricing (same semantic line, different price) as a price change.
- **UniqueReferenceIDGenerator is a caller-supplied strategy rather than a fixed function.** — Delta and period-preserving engines need different ChildUniqueReferenceID schemes for corrections. The strategy pattern keeps subtract.go free of engine-specific ID formatting logic.

## Example: Implement a UniqueReferenceIDGenerator for a new engine

```
type myGenerator struct{}
func (myGenerator) CurrentOnly(line usagebased.DetailedLine) (string, error) {
    return line.ChildUniqueReferenceID, nil
}
func (myGenerator) MatchedDelta(current, _ usagebased.DetailedLine) (string, error) {
    return current.ChildUniqueReferenceID, nil
}
func (myGenerator) PreviousOnlyReversal(line usagebased.DetailedLine) (string, error) {
    if line.ID == "" { return "", fmt.Errorf("line id required for correction") }
    return fmt.Sprintf("%s#correction:detailed_line_id=%s", line.PricerReferenceID, line.ID), nil
}
```

<!-- archie:ai-end -->
