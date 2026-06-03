# historical

<!-- archie:ai-start -->

> The double-entry ledger engine: the Ledger struct implements ledger.Ledger (CommitGroup, GetTransactionGroup, ListTransactions) and ledger.BalanceQuerier (account/sub-account balances) over the Repo (historical/adapter) which books LedgerTransaction + LedgerEntry rows. Owns transaction/entry/balance domain reconstruction.

## Patterns

**transaction.Run wraps the full CommitGroup sequence** — CommitGroup validates routing, then inside one transaction.Run block locks accounts, validates balances, creates the group, and books each transaction so all phases roll back atomically. Never split phases or call BookTransaction outside the block. (`return transaction.Run(ctx, l.repo, func(ctx context.Context) (*TransactionGroup, error) { ... })`)
**Pre-lock all accounts before writes** — lockAccountsForTransactionInputs collects every sub-account's parent account and calls accountLocker.LockAccountsForPosting (sorted NamespacedID ordering) before any balance check or insert. (`if err := l.lockAccountsForTransactionInputs(ctx, group.Namespace(), txInputs); err != nil { ... }`)
**Validate inputs before repo delegation** — ListTransactions, SumEntries and CommitGroup call Validate()/params.Validate() and return early before touching the repo. (`if err := params.Validate(); err != nil { return ..., fmt.Errorf("...: %w", err) }`)
**Domain reconstruction via constructors** — Transaction and Entry are always built via NewTransactionFromData / newEntryFromData (which calls account.NewAddressFromData) — never instantiate the structs directly. (`tx, err := NewTransactionFromData(txData, entryDataSlice)`)
**Tri-tuple cursor via Transaction.Cursor()** — TransactionCursor carries (BookedAt, CreatedAt, ID); always produce cursors through Transaction.Cursor(), never build TransactionCursor manually. (`func (t *Transaction) Cursor() ledger.TransactionCursor { return ledger.TransactionCursor{...} }`)
**Routing validation before booking** — CommitGroup runs ledger.ValidateTransactionInputWith(ctx, txInput, l.routingValidator) per transaction before the transaction.Run block. (`if err := ledger.ValidateTransactionInputWith(ctx, txInput, l.routingValidator); err != nil { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Ledger struct + NewLedger constructor; implements CommitGroup, ListTransactions, GetTransactionGroup and the account-locking flow. | validateAccountBalancesForTransaction is a TODO no-op today; do not assume balance enforcement at commit. |
| `repo.go` | Repo interface (TxCreator + CreateTransactionGroup/GetTransactionGroup/BookTransaction/SumEntries/ListTransactions) and DTOs. | Repo implementations must eagerly load SubAccount→Account→Route edges or hydration fails. |
| `transaction.go` | Transaction / TransactionGroup domain objects; NewTransactionFromData builds entries. | Cursor() is the only correct way to produce a TransactionCursor. |
| `entry.go` | Entry domain object; newEntryFromData builds the PostingAddress via account.NewAddressFromData. | EntryData needs SubAccountID, AccountType, RouteID, RouteKey, RouteKeyVer or address construction errors. |
| `balance.go` | Balance struct + GetAccountBalance/GetSubAccountBalance delegating to sumEntries. | SumEntries returns the same value for Settled and Pending — no settled/pending separation yet. |

## Anti-Patterns

- Calling repo.BookTransaction outside the transaction.Run block — loses lock and validation atomicity.
- Skipping ledger.ValidateTransactionInputWith before booking — invalid account-type combinations persist silently.
- Mapping db rows to domain structs inline instead of via newEntryFromData / NewTransactionFromData.
- Constructing TransactionCursor manually instead of via Transaction.Cursor().
- Using context.Background() instead of the caller ctx — drops the Ent transaction and OTel spans.

## Decisions

- **SumEntries returns identical SettledSum and PendingSum.** — The historical ledger has no pending/settled split yet; a TODO in ledger.go flags the future divergence.
- **Tri-tuple cursor (bookedAt, createdAt, ID) instead of a single auto-increment offset.** — Stable across concurrent inserts sharing a booked_at; ID is a tie-breaker requiring no sequence column.

## Example: Commit a debit/credit transaction group

```
// CommitGroup validates routing, locks customer accounts, then books each
// transaction inside one transaction.Run.
return l.CommitGroup(ctx, group)
```

<!-- archie:ai-end -->
