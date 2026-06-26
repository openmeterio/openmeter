# subtract

<!-- archie:ai-start -->

> Low-level arithmetic primitive shared by the delta and periodpreserving rating engines: SubtractRatedRunDetails subtracts previously-rated detailed lines from currently-rated detailed lines and returns the remainder, preserving each output line's source service period. It does NOT query meters, call billing rating, allocate credits, or decide invoice periods.

## Patterns

**SubtractRatedRunDetails(current, previous, generator, opts...) is the only entry point** — Takes two usagebased.DetailedLines slices plus a required UniqueReferenceIDGenerator. Validates inputs, validates currency, groups by key, subtracts, then optionally validates output uniqueness. Returns an error if the generator is nil. (`out, err := subtract.SubtractRatedRunDetails(current, previous, generator)`)
**Match by calculation key + decimal-equal PerUnitAmount, never by string** — detailedLineKey is {PricerReferenceID(ReferenceID), Category, PaymentTerm}. Within a key, lines are matched by PerUnitAmount.Equal(...) — deliberately not a string map key — so repricing (same component, different unit amount) emits a current line plus a reversal rather than a bogus quantity delta. (`if !existingLine.PerUnitAmount.Equal(line.PerUnitAmount) { continue }`)
**ChildUniqueReferenceID is identity, not arithmetic key — generated via the 3-method interface** — Output child refs come from UniqueReferenceIDGenerator: CurrentOnly (unmatched current), MatchedDelta (matched current+previous), PreviousOnlyReversal (unmatched previous). PricerReferenceID — not ChildUniqueReferenceID — is the arithmetic key. (`childUniqueReferenceID, err := generator.PreviousOnlyReversal(previousLine)`)
**Drop zero-total output lines** — isZeroDetailedLine checks Totals.IsZero(); current-only, matched-delta, and reversal outputs are only emitted when totals are non-zero. (`if !isZeroDetailedLine(line) { out = append(out, line) }`)
**Strict currency validation before arithmetic** — validateCurrencyForSubtract: each side may contain at most one currency; if both sides have one, they must match. Otherwise it errors before any subtraction. (`if currentCurrencies[0] != previousCurrencies[0] { return fmt.Errorf("...currency mismatch...") }`)
**Accept negative quantities; uniqueness validation toggleable** — Negative quantities are valid correction inputs — validateDetailedLinesForSubtract clones and normalizes the sign only for validation. validateUniqueChildReferenceIDs runs by default but is skipped via WithUniqueReferenceIDValidationIgnored() for intermediate callers (periodpreserving epoch subtraction). (`subtract.WithUniqueReferenceIDValidationIgnored()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subtract.go` | The full algorithm: detailedLineKey, options/Option, SubtractRatedRunDetails, currency/line validation, group-and-sum (sumDetailedLinesByKey, appendOrSumDetailedLineByPerUnitAmount), and per-key subtraction. | Matching is PerUnitAmount.Equal, never a stringified key. Output is sorted by PerUnitAmount ascending and zero-total lines are dropped. Do not move currency/period decisions here — this package is meter/credit/invoice agnostic. |
| `uniquereferenceid.go` | UniqueReferenceIDGenerator interface (CurrentOnly/MatchedDelta/PreviousOnlyReversal) plus NewMockUniqueReferenceIDGenerator for tests. | This is the contract delta and periodpreserving implement. The mock's PreviousOnlyReversal encodes '#reversal:category=...:payment_term=...:per_unit_amount=...' — test expectations depend on that exact format. |
| `subtract_test.go` | Table-driven scenarios (TestUnitPriceSubtract, TestMinimumCommitmentSubtract, TestGraduatedTieredSubtract) using real productcatalog prices via billingratingservice.New(). | Tests assert exact reversal child-ref strings and rounding-driven zero/negative deltas (e.g. delta dropped when rounding makes it zero). Changing match or rounding semantics breaks these. |
| `README.md` | Matching model, algorithm, and Warning that this is a primitive, not a final invoice rater. | Honor the Warning — never expose this as a standalone production rating algorithm. |

## Anti-Patterns

- Using ChildUniqueReferenceID as the arithmetic match key (it is persistence identity only; PricerReferenceID + price-shape fields are the key).
- Stringifying PerUnitAmount for map matching instead of decimal equality — breaks repricing correctness.
- Querying meters, calling billing rating, allocating credits, or deciding invoice periods here — this package is intentionally meter/credit/invoice agnostic.
- Emitting zero-total output lines or rejecting negative input quantities (corrections are valid).
- Skipping currency validation, or merging lines across mismatched currencies.

## Decisions

- **Compare PerUnitAmount with decimal equality rather than via a string map key.** — Avoids unsafe decimal-string comparisons and keeps repricing explicit: a re-rated component emits a reversal of the old price plus the new current line.
- **Generate output ChildUniqueReferenceID through a pluggable UniqueReferenceIDGenerator instead of computing it inline.** — Different engines need different persistence/correction identities (delta correction refs vs period-stamped refs) over the same arithmetic core.
- **Make output-uniqueness validation opt-out via WithUniqueReferenceIDValidationIgnored.** — Period-preserving rating runs intermediate subtractions whose duplicate child refs are resolved later by period-stamping, so uniqueness cannot be enforced there.

## Example: Subtract previous from current with a custom reference-ID generator

```
out, err := subtract.SubtractRatedRunDetails(
    current,  // usagebased.DetailedLines (e.g. 8 units @ $10)
    previous, // usagebased.DetailedLines (e.g. 5 units @ $10)
    generator, // implements subtract.UniqueReferenceIDGenerator
)
// matched same-price -> 3 units @ $10 = $30 via generator.MatchedDelta;
// repriced -> +current line (CurrentOnly) and -reversal (PreviousOnlyReversal)
```

<!-- archie:ai-end -->
