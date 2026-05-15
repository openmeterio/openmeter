# testutils

<!-- archie:ai-start -->

> Provides compile-time-checked test doubles for ledger.EntryInput, ledger.TransactionInput, and ledger.TransactionGroupInput interfaces so tests can construct arbitrary ledger postings without touching production adapters, Ent, or the database.

## Patterns

**Compile-time interface assertion** — Every test struct declares a blank-identifier assertion (var _ <Interface> = (*Struct)(nil)) so interface drift is caught at compile time, not at test runtime. (`var _ ledger.EntryInput = (*AnyEntryInput)(nil)`)
**Exported *Value field naming** — All stored values use an exported field with a *Value suffix (AmountValue, BookedAtValue, AnnotationsValue). Interface accessor methods return these fields directly — no constructors needed, test fixtures are set by direct field assignment. (`type AnyEntryInput struct { Address ledger.PostingAddress; AmountValue alpacadecimal.Decimal }`)
**lo.Map for slice interface coercion** — Concrete slice fields (e.g. []*AnyEntryInput) are projected to the interface slice type ([]ledger.EntryInput) inside the accessor method using samber/lo.Map — keeps struct field types concrete while satisfying the interface. (`return lo.Map(a.EntryInputsValues, func(e *AnyEntryInput, _ int) ledger.EntryInput { return e })`)
**AsGroupInput convenience wrapper** — AnyTransactionInput exposes AsGroupInput(namespace, annotations) to wrap itself in an AnyTransactionGroupInput — eliminates boilerplate when a test only needs a single transaction in a group. (`func (a *AnyTransactionInput) AsGroupInput(ns string, ann models.Annotations) ledger.TransactionGroupInput { return &AnyTransactionGroupInput{...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `anytransaction.go` | Defines AnyEntryInput, AnyTransactionInput, and AnyTransactionGroupInput — the complete set of test doubles for the three ledger transaction posting interfaces. Any change to ledger.EntryInput, ledger.TransactionInput, or ledger.TransactionGroupInput signatures must be mirrored here. | Missing or stale compile-time var _ assertions after interface changes; forgetting lo.Map when adding a new slice field; adding Ent/DB imports which would turn this into a production dependency. |

## Anti-Patterns

- Do not add production logic or database/Ent access — this package must stay a pure in-memory test helper.
- Do not skip the var _ Interface = (*Struct)(nil) compile-time assertions when adding new test doubles.
- Do not import app/common or any Wire provider sets — test helpers must be independent of the DI wiring layer.
- Do not use context.Background() inside helpers; callers supply t.Context() when *testing.T is available.
- Do not put this package's types into production code paths — the testutils sub-package convention isolates test-only types.

## Decisions

- **Separate testutils package instead of embedding test doubles in the production package** — Keeps openmeter/ledger/transactions free of test-only dependencies; prevents accidental use of test doubles in production; mirrors the project-wide <domain>/testutils convention used by billing, customer, streaming, etc.
- **Exported *Value fields instead of constructor functions** — Direct field assignment makes test fixtures concise and self-documenting; the interface contract is still satisfied through accessor methods; no constructor indirection means fewer lines of fixture setup per test.

## Example: Build a single-transaction group to post to the ledger in a test

```
import (
    "time"
    "github.com/alpacahq/alpacadecimal"
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
}
// ...
```

<!-- archie:ai-end -->
