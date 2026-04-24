# historical

<!-- archie:ai-start -->

> Implements the double-entry ledger engine: books multi-entry transactions into groups, sums entries for balance queries, and lists transactions with cursor pagination. It owns the Ledger struct that satisfies both ledger.Ledger and ledger.Querier interfaces, and the Repo interface that its Ent adapter must implement.

## Patterns

**transaction.Run for multi-step mutations** — CommitGroup wraps the entire lock→validate→create-group→book sequence in a single transaction.Run(ctx, l.repo, ...) block so all steps roll back together on failure. (`return transaction.Run(ctx, l.repo, func(ctx context.Context) (*TransactionGroup, error) { ... })`)
**Pre-lock customer accounts before balance checks** — lockAccountsForTransactionInputs collects all sub-account IDs, resolves their parent accounts, and calls CustomerAccount.Lock(ctx) for FBO/Receivable types before any balance validation or writes. (`if err := l.lockAccountsForTransactionInputs(ctx, group.Namespace(), txInputs); err != nil { ... }`)
**Validate input before delegating to repo** — ListTransactions and SumEntries both call params.Validate() / query.Validate() and return early on error before touching the repo. (`if err := params.Validate(); err != nil { return ..., fmt.Errorf("...: %w", err) }`)
**NewTransactionFromData / newEntryFromData for domain reconstruction** — Domain objects are always built from data DTOs via these constructors — never directly instantiated. entryData requires full routing key and route fields to build the PostingAddress. (`entry, err := newEntryFromData(data); tx, err := NewTransactionFromData(txData, entryDataSlice)`)
**Tri-tuple cursor for transaction pagination** — TransactionCursor carries (BookedAt, CreatedAt, ID) — use Transaction.Cursor() to produce it; do not construct manually. (`func (t *Transaction) Cursor() ledger.TransactionCursor { return ledger.TransactionCursor{BookedAt: ..., CreatedAt: ..., ID: ...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Ledger struct — implements ledger.Ledger (CommitGroup, GetTransactionGroup, ListTransactions) and ledger.Querier (SumEntries); constructor is NewLedger(repo, accountService, locker, routingValidator). | CommitGroup performs routing validation (ledger.ValidateTransactionInputWith), then locks accounts, then writes — all inside one transaction.Run; do not split these phases. |
| `repo.go` | Repo interface (TxCreator + CreateTransactionGroup, GetTransactionGroup, BookTransaction, SumEntries, ListTransactions) and all input/output DTOs (TransactionData, TransactionGroupData, CreateTransactionInput, ListEntriesInput). | Repo implementations must eagerly load SubAccount→Account→Route edges or hydration will fail. |
| `transaction.go` | Transaction, TransactionGroup domain objects — implement ledger.Transaction and ledger.TransactionGroup; NewTransactionFromData builds entries via newEntryFromData. | Transaction.Cursor() is the only correct way to produce a TransactionCursor. |
| `entry.go` | Entry domain object — implements ledger.Entry; newEntryFromData calls account.NewAddressFromData to build the PostingAddress. | EntryData must have non-empty SubAccountID, AccountType, RouteID, RouteKey, RouteKeyVer or address construction will error. |

## Anti-Patterns

- Calling repo.BookTransaction outside a transaction.Run block — loses the account lock and validation atomicity.
- Omitting routing validation (ledger.ValidateTransactionInputWith) before calling BookTransaction — invalid account-type combinations will persist silently.
- Adding DB-to-domain mapping logic outside newEntryFromData / NewTransactionFromData — all hydration must route through these constructors.
- Using context.Background() instead of propagating the caller's ctx — breaks OTel tracing and lock inheritance.
- Inserting LedgerEntry rows one-by-one in the adapter instead of using Ent's CreateBulk — causes unnecessary round-trips.

## Decisions

- **SumEntries returns the same value for both SettledSum and PendingSum because the historical ledger has no pending/settled separation yet.** — Simplifies the initial implementation; the TODO comment in ledger.go flags this for future split.
- **Tri-tuple cursor (bookedAt, createdAt, ID) instead of a single auto-increment offset.** — Stable across concurrent inserts with the same booked_at timestamp; ID provides a tie-breaker that does not require a sequence column.

## Example: Commit a two-entry debit/credit transaction group

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
)

func commit(ctx context.Context, l *historical.Ledger, group ledger.TransactionGroupInput) (ledger.TransactionGroup, error) {
	return l.CommitGroup(ctx, group)
	// CommitGroup: validates entries via routingValidator, locks customer accounts,
	// creates transaction group row, books each transaction inside one transaction.Run
}
```

<!-- archie:ai-end -->
