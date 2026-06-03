# service

<!-- archie:ai-start -->

> Business-logic service layer for ledger accounts and sub-accounts. Delegates all persistence to account.Repo and wraps raw AccountData/SubAccountData in domain objects enriched with live runtime dependencies (Locker, SubAccountService) via account.AccountLiveServices.

## Patterns

**Self-wiring of SubAccountService in New()** — New() builds the service, sets live.SubAccountService = svc before storing live, so Account domain objects receive a real SubAccountService. Never set it externally. (`svc := &service{repo: repo, locker: locker}; svc.live = account.AccountLiveServices{SubAccountService: svc}; return svc`)
**Validate input before repo calls** — CreateAccount and EnsureSubAccount call input.Validate() and return immediately on error before any repo interaction. (`if err := input.Validate(); err != nil { return nil, err }`)
**transaction.Run for multi-step mutations** — Wrap a write-then-read sequence in transaction.Run(ctx, s.repo, ...) for atomicity (e.g. EnsureSubAccount). (`return transaction.Run(ctx, s.repo, func(ctx context.Context) (ledger.SubAccount, error) { d, _ := s.repo.EnsureSubAccount(ctx, input); return account.NewSubAccountFromData(*d) })`)
**Domain object construction via NewAccountFromData / NewSubAccountFromData** — Never return raw *AccountData/*SubAccountData; always wrap via account.NewAccountFromData(data, s.live) to attach live services. (`return account.NewAccountFromData(*accData, s.live)`)
**Sorted deterministic lock ordering in LockAccountsForPosting** — LockAccountsForPosting dedups by NamespacedID, sorts lexicographically (namespace then ID), and acquires advisory locks in order to prevent deadlocks across concurrent callers. (`sort.Slice(ids, func(i, j int) bool { if ids[i].Namespace == ids[j].Namespace { return ids[i].ID < ids[j].ID }; return ids[i].Namespace < ids[j].Namespace })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Entire service implementation; exposes New() and satisfies account.Service via var _ account.Service = (*service)(nil). | The self-wiring of live.SubAccountService is load-bearing — a nil SubAccountService panics on Account.SubAccount(). LockAccountsForPosting only locks FBO and Receivable types; new lockable types need a switch update. |

## Anti-Patterns

- Returning *AccountData or *SubAccountData directly — callers expect domain objects with live services attached.
- Calling s.repo.EnsureSubAccount outside transaction.Run when the method also reads — creates a TOCTOU window.
- Constructing the service and setting live.SubAccountService externally — self-wiring is intentional and load-bearing.
- Introducing context.Background()/context.TODO() — always propagate the caller's ctx.

## Decisions

- **AccountLiveServices.SubAccountService is self-wired inside New()** — Account domain objects need SubAccountService to load sub-accounts lazily; the same service implements both operations, resolving the circular dependency without an external intermediary.
- **LockAccountsForPosting acquires advisory locks in sorted NamespacedID order** — Consistent lock-acquisition order across concurrent callers prevents deadlocks when goroutines lock overlapping customer account sets.

## Example: Add a mutating service method with validation and domain wrapping

```
func (s *service) ArchiveAccount(ctx context.Context, id models.NamespacedID) (ledger.Account, error) {
    if id.ID == "" || id.Namespace == "" { return nil, models.NewGenericValidationError("account id and namespace are required") }
    return transaction.Run(ctx, s.repo, func(ctx context.Context) (ledger.Account, error) {
        accData, err := s.repo.ArchiveAccount(ctx, id)
        if err != nil { return nil, fmt.Errorf("archive account: %w", err) }
        return account.NewAccountFromData(*accData, s.live)
    })
}
```

<!-- archie:ai-end -->
