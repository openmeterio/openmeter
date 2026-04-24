# testutils

<!-- archie:ai-start -->

> Provides concrete test-only implementations of ledger.EntryInput, ledger.TransactionInput, and ledger.TransactionGroupInput interfaces so tests can construct arbitrary ledger postings without depending on production adapters or the database.

## Patterns

**Interface compliance assertion at compile time** — Each test struct declares a `var _ <Interface> = (*Struct)(nil)` blank-identifier assertion to guarantee the struct satisfies its interface before the test runs. (`var _ ledger.EntryInput = (*AnyEntryInput)(nil)`)
**Flat value structs with exported fields** — Test input structs store all values as exported fields with a *Value suffix (AmountValue, BookedAtValue, etc.) and expose them through the interface methods — no constructor functions needed. (`type AnyEntryInput struct { Address ledger.PostingAddress; AmountValue alpacadecimal.Decimal }`)
**lo.Map for slice interface coercion** — Concrete slice fields (e.g. []*AnyEntryInput) are converted to []ledger.EntryInput / []ledger.TransactionInput via samber/lo.Map inside the interface method, keeping field types concrete. (`return lo.Map(a.EntryInputsValues, func(e *AnyEntryInput, _ int) ledger.EntryInput { return e })`)
**AsGroupInput helper for single-transaction groups** — AnyTransactionInput exposes AsGroupInput(namespace, annotations) to wrap itself in an AnyTransactionGroupInput — allows tests to build a full TransactionGroupInput from a single transaction without boilerplate. (`func (a *AnyTransactionInput) AsGroupInput(ns string, ann models.Annotations) ledger.TransactionGroupInput { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `anytransaction.go` | Defines AnyEntryInput, AnyTransactionInput, and AnyTransactionGroupInput — the complete set of test doubles for the ledger transaction posting interfaces. | All three types must keep satisfying their respective ledger interfaces; any change to ledger.EntryInput, ledger.TransactionInput, or ledger.TransactionGroupInput signatures must be mirrored here. |

## Anti-Patterns

- Do not add production logic or database access here — this package must remain a pure test helper with no Ent/DB imports.
- Do not skip the `var _ Interface = (*Struct)(nil)` compile-time assertions when adding new test doubles.
- Do not import app/common or any Wire provider sets — test helpers must stay independent of the DI wiring layer.
- Do not use context.Background() in test helpers; callers should supply t.Context() when a testing.T is available.

## Decisions

- **Separate testutils package instead of embedding test doubles in the production package** — Keeps openmeter/ledger/transactions free of test-only dependencies and prevents accidental use of test doubles in production code; mirrors the project-wide convention of <domain>/testutils sub-packages.
- **Exported *Value field naming convention for all stored values** — Allows test code to set fields directly without constructor indirection, making test fixtures concise and self-documenting while still satisfying the interface contract through accessor methods.

## Example: Build a single-transaction group to post to the ledger in a test

```
import (
    "time"
    "github.com/alpacahq/alpacadecimal"
    "github.com/openmeterio/openmeter/openmeter/ledger"
    tutestutils "github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
    "github.com/openmeterio/openmeter/pkg/models"
)

entry := &tutestutils.AnyEntryInput{
    Address:     somePostingAddress,
    AmountValue: alpacadecimal.NewFromInt(100),
}
tx := &tutestutils.AnyTransactionInput{
    BookedAtValue:     time.Now(),
    EntryInputsValues: []*tutestutils.AnyEntryInput{entry},
// ...
```

<!-- archie:ai-end -->
