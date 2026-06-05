# historical

<!-- archie:ai-start -->

> The append-only double-entry ledger implementation: Ledger commits transaction groups (validate -> lock accounts -> create group -> book transactions) and answers balance/list queries. Domain Transaction/TransactionGroup/Entry/Balance values are always reconstructed from *Data DTOs, and the adapter is the only place that touches the ledger_transaction(_group)/ledger_entry tables.

## Patterns

**CommitGroup is validate-lock-write** — CommitGroup validates each TransactionInput, then inside transaction.Run locks all affected accounts preemptively, validates balances, creates the group, and books each transaction. (`transaction.Run(ctx, l.repo, func(ctx){...}) wrapping lockAccountsForTransactionInputs + repo.CreateTransactionGroup + repo.BookTransaction`)
**Preemptive account locking** — lockAccountsForTransactionInputs collects all sub-account IDs across entries, resolves their parent accounts, and calls accountLocker.LockAccountsForPosting once — not per sub-transaction. (`return l.accountLocker.LockAccountsForPosting(ctx, affectedAccounts)`)
**Data-DTO reconstruction** — Transaction/TransactionGroup/Entry are built only via NewTransactionFromData / NewTransactionGroupFromData / newEntryFromData; Entry rebuilds its PostingAddress from EntryData using account.NewAddressFromData. (`account.NewAddressFromData(account.AddressData{SubAccountID, AccountType, Route, RouteID, RoutingKey})`)
**Interface satisfaction asserted** — Ledger asserts ledger.Ledger, ledger.BalanceQuerier; Balance asserts ledger.Balance; Transaction asserts ledger.Transaction etc. (`var _ ledger.BalanceQuerier = (*Ledger)(nil)`)
**Balance via sumEntries** — GetAccountBalance builds a ledger.Query (Namespace + Filters{After, AsOf, AccountID, Route}) and delegates to repo.SumEntries; settled and pending currently both equal the same total. (`res, _ := l.sumEntries(ctx, ledger.Query{Namespace: account.ID().Namespace, Filters: ledger.Filters{AccountID: lo.ToPtr(account.ID().ID), Route: route}})`)
**Cursor pagination on transactions** — ListTransactions delegates to repo with ListTransactionsInput and returns Items + NextCursor; the adapter does keyset pagination over (BookedAt, CreatedAt, ID). (`return ledger.ListTransactionsResult{Items: res.Items, NextCursor: res.NextCursor}, nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Ledger struct + NewLedger; implements CommitGroup, ListTransactions, GetTransactionGroup and the locking helper. | validateAccountBalancesForTransaction is a TODO no-op; do not assume over-draw is rejected here yet. CommitGroup rejects empty groups with ledger.ErrTransactionGroupEmpty. |
| `transaction.go` | Transaction/TransactionGroup value objects and their NewXFromData factories; Cursor() exposes (BookedAt, CreatedAt, ID). | Entries()/Transactions() widen []*T to []ledger.X via lo.Map — keep slices of pointers internally. |
| `entry.go` | Entry value object; newEntryFromData rebuilds PostingAddress from route key version/value. | EntryData must carry RouteKeyVer + RouteKey or NewRoutingKey fails; this is hydrated by the adapter from joined route rows. |
| `balance.go` | Balance value + GetAccountBalance/GetSubAccountBalance; sub-account balance resolves its parent account then routes through GetAccountBalance. | Settled and Pending are currently identical (no settled/pending separation yet). |
| `repo.go` | Repo interface (embeds entutils.TxCreator) for group create/get, BookTransaction, SumEntries, ListTransactions, plus parameter/DTO types. | BookTransaction takes the group's NamespacedID; CreateTransactionGroupInput carries Annotations while CreateTransactionInput does not. |

## Anti-Patterns

- Bypassing transaction.Run / preemptive locking and writing transactions directly, reintroducing posting races.
- Constructing Transaction/TransactionGroup/Entry by hand instead of via NewTransactionFromData / NewTransactionGroupFromData / newEntryFromData.
- Assuming GetAccountBalance enforces non-negative balances — validateAccountBalancesForTransaction is still a TODO.
- Reading raw ent rows in this package instead of going through the Repo's *Data DTOs.
- Committing a transaction group without first validating every TransactionInput via ledger.ValidateTransactionInputWith(routingValidator).

## Decisions

- **Accounts are locked preemptively for the whole group, not per sub-transaction.** — Multi-transaction groups must be atomic; locking every affected account up front avoids deadlocks and partial posting within the transaction.Run scope.
- **Domain aggregates are reconstructed from a separate *Data DTO layer.** — Decouples ent row shape from rich ledger value objects and centralizes routing-key/address hydration in the From*Data constructors.

## Example: Committing a validated transaction group atomically

```
for idx, txInput := range txInputs {
	if err := ledger.ValidateTransactionInputWith(ctx, txInput, l.routingValidator); err != nil {
		return nil, fmt.Errorf("validate tx %d: %w", idx, err)
	}
}
return transaction.Run(ctx, l.repo, func(ctx context.Context) (*TransactionGroup, error) {
	if err := l.lockAccountsForTransactionInputs(ctx, group.Namespace(), txInputs); err != nil {
		return nil, err
	}
	txG, _ := l.repo.CreateTransactionGroup(ctx, CreateTransactionGroupInput{Namespace: group.Namespace(), Annotations: group.Annotations()})
	txGroup := &TransactionGroup{data: txG}
	for _, txInput := range txInputs {
		tx, _ := l.repo.BookTransaction(ctx, models.NamespacedID{Namespace: group.Namespace(), ID: txG.ID}, txInput)
		txGroup.transactions = append(txGroup.transactions, tx)
	}
// ...
```

<!-- archie:ai-end -->
