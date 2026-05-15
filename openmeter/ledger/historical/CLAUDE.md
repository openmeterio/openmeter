# historical

<!-- archie:ai-start -->

> Implements the double-entry ledger engine: books multi-entry transaction groups, sums entries for balance queries, and lists transactions with tri-tuple cursor pagination. It owns the Ledger struct satisfying both ledger.Ledger and ledger.BalanceQuerier, and the Repo interface that the Ent adapter must implement.

## Patterns

**transaction.Run for all CommitGroup phases** — CommitGroup wraps the entire lock → validate → create-group → book sequence in a single transaction.Run block so all steps roll back atomically. Never call BookTransaction outside this block. (`return transaction.Run(ctx, l.repo, func(ctx context.Context) (*TransactionGroup, error) { ... })`)
**Pre-lock all accounts before balance validation** — lockAccountsForTransactionInputs collects all sub-account parent accounts and calls accountLocker.LockAccountsForPosting before any balance check or write. Lock ordering uses sorted NamespacedID to prevent deadlocks. (`if err := l.lockAccountsForTransactionInputs(ctx, group.Namespace(), txInputs); err != nil { ... }`)
**Validate inputs before repo delegation** — ListTransactions and SumEntries both call params.Validate() and return early before touching the repo. Never skip validation even in internal helpers. (`if err := params.Validate(); err != nil { return ..., fmt.Errorf("...: %w", err) }`)
**NewTransactionFromData / newEntryFromData for domain reconstruction** — Domain objects are always built from data DTOs via these constructors. Never directly instantiate Transaction or Entry structs. EntryData requires full routing key and route fields. (`entry, err := newEntryFromData(data); tx, err := NewTransactionFromData(txData, entryDataSlice)`)
**Tri-tuple cursor via Transaction.Cursor()** — TransactionCursor carries (BookedAt, CreatedAt, ID). Always produce cursors via Transaction.Cursor() — never construct TransactionCursor manually. (`func (t *Transaction) Cursor() ledger.TransactionCursor { return ledger.TransactionCursor{BookedAt: ..., CreatedAt: ..., ID: ...} }`)
**Eager edge loading required on all Repo queries** — The adapter must eagerly load SubAccount→Account→Route edges on every query returning hydrated transactions or hydrateHistoricalTransaction will error on nil edge pointers. (`db.LedgerTransaction.Query().WithSubAccount(func(q *db.LedgerSubAccountQuery) { q.WithAccount().WithRoute() })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Ledger struct — implements ledger.Ledger (CommitGroup, GetTransactionGroup, ListTransactions) and ledger.BalanceQuerier (GetAccountBalance, GetSubAccountBalance); constructor is NewLedger(repo, accountCatalog, accountLocker, routingValidator). | CommitGroup performs routing validation via ledger.ValidateTransactionInputWith, then locks accounts, then writes — all inside one transaction.Run. Do not split these phases. |
| `repo.go` | Repo interface (TxCreator + CreateTransactionGroup, GetTransactionGroup, BookTransaction, SumEntries, ListTransactions) and all input/output DTOs. | Repo implementations must eagerly load SubAccount→Account→Route edges or hydration will fail. |
| `transaction.go` | Transaction and TransactionGroup domain objects implementing ledger.Transaction and ledger.TransactionGroup; NewTransactionFromData builds entries via newEntryFromData. | Transaction.Cursor() is the only correct way to produce a TransactionCursor — never construct manually. |
| `entry.go` | Entry domain object — implements ledger.Entry; newEntryFromData calls account.NewAddressFromData to build the PostingAddress. | EntryData must have non-empty SubAccountID, AccountType, RouteID, RouteKey, RouteKeyVer or address construction will error. |
| `balance.go` | Balance struct implementing ledger.Balance (Settled, Pending); GetAccountBalance and GetSubAccountBalance delegate to sumEntries. | SumEntries currently returns the same value for both Settled and Pending — the historical ledger has no pending/settled separation yet. |

## Anti-Patterns

- Calling repo.BookTransaction outside a transaction.Run block — loses account lock and validation atomicity.
- Omitting routing validation (ledger.ValidateTransactionInputWith) before calling BookTransaction — invalid account-type combinations persist silently.
- Adding DB-to-domain mapping logic outside newEntryFromData / NewTransactionFromData — all hydration must route through these constructors.
- Using context.Background() instead of propagating the caller's ctx — breaks OTel tracing and lock inheritance.
- Omitting eager edge loading (WithSubAccount/WithAccount/WithRoute) on adapter queries — hydrateHistoricalTransaction returns errors on nil edges.

## Decisions

- **SumEntries returns the same value for both SettledSum and PendingSum because the historical ledger has no pending/settled separation yet.** — Simplifies the initial implementation; the TODO comment in ledger.go flags this for future split.
- **Tri-tuple cursor (bookedAt, createdAt, ID) instead of a single auto-increment offset.** — Stable across concurrent inserts with the same booked_at timestamp; ID provides a tie-breaker without requiring a sequence column.

## Example: Commit a two-entry debit/credit transaction group

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/historical"
)

func commit(ctx context.Context, l *historical.Ledger, group ledger.TransactionGroupInput) (ledger.TransactionGroup, error) {
	// CommitGroup: validates entries via routingValidator, locks customer accounts,
	// creates transaction group row, books each transaction inside one transaction.Run
	return l.CommitGroup(ctx, group)
}
```

<!-- archie:ai-end -->
