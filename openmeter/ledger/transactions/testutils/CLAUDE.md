# testutils

<!-- archie:ai-start -->

> Compile-time-checked test doubles for ledger.EntryInput, ledger.TransactionInput, and ledger.TransactionGroupInput so tests can construct arbitrary ledger postings without touching production adapters, Ent, or the database.

## Patterns

**Compile-time interface assertion** — Every test struct declares a blank-identifier assertion so interface drift is caught at compile time, not test runtime. (`var _ ledger.EntryInput = (*AnyEntryInput)(nil)`)
**Exported *Value field naming** — Stored values use an exported field with a *Value suffix (AmountValue, BookedAtValue, AnnotationsValue); accessor methods return these directly so fixtures are set by direct field assignment, no constructors. (`type AnyEntryInput struct { Address ledger.PostingAddress; AmountValue alpacadecimal.Decimal }`)
**lo.Map for slice interface coercion** — Concrete slice fields ([]*AnyEntryInput) are projected to interface slices ([]ledger.EntryInput) in the accessor via samber/lo.Map, keeping field types concrete while satisfying the interface. (`return lo.Map(a.EntryInputsValues, func(e *AnyEntryInput, _ int) ledger.EntryInput { return e })`)
**AsGroupInput convenience wrapper** — AnyTransactionInput exposes AsGroupInput(namespace, annotations) to wrap itself in an AnyTransactionGroupInput, eliminating boilerplate for single-transaction groups. (`func (a *AnyTransactionInput) AsGroupInput(ns string, ann models.Annotations) ledger.TransactionGroupInput { return &AnyTransactionGroupInput{...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `anytransaction.go` | Defines AnyEntryInput, AnyTransactionInput, and AnyTransactionGroupInput — the complete set of test doubles for the three ledger posting interfaces. | Any change to ledger.EntryInput/TransactionInput/TransactionGroupInput signatures must be mirrored here; missing/stale var _ assertions or forgetting lo.Map on a new slice field; adding Ent/DB imports would turn this into a production dependency. |

## Anti-Patterns

- Adding production logic or database/Ent access — this package must stay a pure in-memory test helper.
- Skipping the var _ Interface = (*Struct)(nil) compile-time assertions when adding new test doubles.
- Importing app/common or Wire provider sets — test helpers must be independent of the DI layer.
- Using context.Background() inside helpers — callers supply t.Context() when *testing.T is available.
- Putting these types into production code paths — the testutils convention isolates test-only types.

## Decisions

- **Separate testutils package instead of embedding test doubles in the production package.** — Keeps openmeter/ledger/transactions free of test-only dependencies, prevents accidental production use, and mirrors the project-wide <domain>/testutils convention.
- **Exported *Value fields instead of constructor functions.** — Direct field assignment makes fixtures concise and self-documenting while accessor methods still satisfy the interface contract.

## Example: Building a single-transaction group to post in a test

```
import (
	"time"
	"github.com/alpacahq/alpacadecimal"
	tutestutils "github.com/openmeterio/openmeter/openmeter/ledger/transactions/testutils"
)

entry := &tutestutils.AnyEntryInput{
	Address:     somePostingAddress,
	AmountValue: alpacadecimal.NewFromInt(100),
}
tx := &tutestutils.AnyTransactionInput{
	BookedAtValue:     time.Now(),
	EntryInputsValues: []*tutestutils.AnyEntryInput{entry},
}
group := tx.AsGroupInput("my-namespace", nil)
```

<!-- archie:ai-end -->
