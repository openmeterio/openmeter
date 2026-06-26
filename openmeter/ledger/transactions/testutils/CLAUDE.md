# testutils

<!-- archie:ai-start -->

> Test-only helpers that provide concrete, field-settable implementations of the ledger package's input interfaces (EntryInput, TransactionInput, TransactionGroupInput) so tests can construct arbitrary ledger transaction inputs without the production builders.

## Patterns

**Interface conformance via compile-time assertion** — Every struct asserts it satisfies the corresponding ledger interface with a blank var declaration, so a signature drift in the ledger interface breaks compilation here immediately. (`var _ ledger.EntryInput = (*AnyEntryInput)(nil)`)
**Value-field + getter-method mirror** — Each interface method is backed by an exported `...Value` struct field; the method just returns the field, letting tests set inputs declaratively while still satisfying the interface. (`AnyEntryInput{AmountValue: ...} with func (a *AnyEntryInput) Amount() alpacadecimal.Decimal { return a.AmountValue }`)
**Pointer-receiver methods** — All methods use pointer receivers and assertions use `(*Type)(nil)`; pass `&AnyEntryInput{...}` to consumers expecting the interface. (`func (a *AnyTransactionInput) BookedAt() time.Time { return a.BookedAtValue }`)
**lo.Map for slice-of-pointer to slice-of-interface widening** — Collection getters convert `[]*AnyEntryInput` to `[]ledger.EntryInput` via lo.Map rather than manual loops, matching repo samber/lo conventions. (`lo.Map(a.EntryInputsValues, func(e *AnyEntryInput, _ int) ledger.EntryInput { return e })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `anytransaction.go` | Sole file: defines AnyEntryInput, AnyTransactionInput, AnyTransactionGroupInput plus the AsGroupInput convenience that wraps a single transaction into a group input. | AsGroupInput always wraps exactly one transaction; do not assume it merges multiple. Keep the `var _ ledger.X = (*AnyX)(nil)` assertions in sync when adding fields. |

## Anti-Patterns

- Importing this testutils package from non-test production code — it only exists to satisfy ledger interfaces in tests.
- Adding business logic to the getters; they must remain trivial field returns so the structs stay pure test fixtures.
- Dropping a `var _ ledger.X = (*AnyX)(nil)` assertion — without it, interface drift goes undetected.

## Decisions

- **Mirror each ledger input interface with an `Any...Input` struct whose every method returns a settable exported field.** — Production input builders enforce invariants; tests need to inject arbitrary or invalid inputs to exercise validation and edge cases, so a plain field-backed implementation is required.

## Example: Build a ledger transaction group input fixture for a test

```
import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

entry := &testutils.AnyEntryInput{
	Address:          addr, // ledger.PostingAddress
	AmountValue:      alpacadecimal.NewFromInt(100),
	IdentityKeyValue: "grant-1",
}
tx := &testutils.AnyTransactionInput{
	BookedAtValue:     clock.Now(),
	EntryInputsValues: []*testutils.AnyEntryInput{entry},
// ...
```

<!-- archie:ai-end -->
