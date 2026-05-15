# service

<!-- archie:ai-start -->

> Business-logic service layer for ledger accounts and sub-accounts. Delegates all persistence to account.Repo and wraps raw AccountData/SubAccountData in domain objects enriched with live runtime dependencies (Locker, SubAccountService) via account.AccountLiveServices.

## Patterns

**Self-wiring of SubAccountService in New()** — New() constructs the service, sets live.SubAccountService = svc before storing live in the struct, so Account domain objects receive a real SubAccountService reference. Never construct the service and set SubAccountService externally. (`svc := &service{repo: repo, locker: locker}; svc.live = account.AccountLiveServices{SubAccountService: svc}; return svc`)
**Validate input before repo calls** — CreateAccount and EnsureSubAccount call input.Validate() and return immediately on error before any repo interaction. All mutating methods must validate first. (`if err := input.Validate(); err != nil { return nil, err }`)
**transaction.Run for multi-step mutations** — EnsureSubAccount wraps the repo call in transaction.Run(ctx, s.repo, ...) to ensure atomicity. Use transaction.Run whenever a method calls both a write and a subsequent read on the repo. (`return transaction.Run(ctx, s.repo, func(ctx context.Context) (ledger.SubAccount, error) { subAccountData, _ := s.repo.EnsureSubAccount(ctx, input); return account.NewSubAccountFromData(*subAccountData) })`)
**Domain object construction via NewAccountFromData / NewSubAccountFromData** — Service methods must never return raw *AccountData or *SubAccountData. Always wrap via account.NewAccountFromData(data, s.live) or account.NewSubAccountFromData(data) to attach live runtime services. (`return account.NewAccountFromData(*accData, s.live)`)
**Sorted deterministic lock ordering in LockAccountsForPosting** — LockAccountsForPosting deduplicates by NamespacedID, sorts IDs lexicographically (namespace then ID), and acquires advisory locks in order to prevent deadlocks between concurrent callers locking overlapping account sets. (`sort.Slice(ids, func(i, j int) bool { if ids[i].Namespace == ids[j].Namespace { return ids[i].ID < ids[j].ID }; return ids[i].Namespace < ids[j].Namespace })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Entire service implementation. Exposes New() and satisfies account.Service via var _ account.Service = (*service)(nil). | The self-wiring of live.SubAccountService in New() is load-bearing — if SubAccountService is nil, Account.SubAccount() calls will panic at runtime. LockAccountsForPosting only locks FBO and Receivable account types; adding new account types that need locking requires updating the switch case. |

## Anti-Patterns

- Returning *AccountData or *SubAccountData directly from service methods — callers expect domain objects with live services attached
- Calling s.repo.EnsureSubAccount outside a transaction.Run when the method also calls s.GetAccountByID — creates a TOCTOU window
- Constructing service and setting live.SubAccountService externally instead of via New() — self-wiring is intentional and load-bearing
- Introducing context.Background() or context.TODO() — always propagate the caller's ctx

## Decisions

- **AccountLiveServices.SubAccountService is self-wired inside New() rather than injected as a separate dependency.** — Account domain objects need SubAccountService to load sub-accounts lazily; the single service implements both operations, so self-wiring resolves the circular dependency without an external intermediary.
- **LockAccountsForPosting acquires advisory locks in sorted NamespacedID order.** — Consistent lock acquisition order across concurrent callers prevents deadlocks when two goroutines lock overlapping sets of customer accounts for ledger postings.

## Example: Add a new mutating service method with validation and domain object wrapping

```
func (s *service) ArchiveAccount(ctx context.Context, id models.NamespacedID) (ledger.Account, error) {
	// validate inputs before touching the repo
	if id.ID == "" || id.Namespace == "" {
		return nil, models.NewGenericValidationError("account id and namespace are required")
	}
	return transaction.Run(ctx, s.repo, func(ctx context.Context) (ledger.Account, error) {
		accData, err := s.repo.ArchiveAccount(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("archive account: %w", err)
		}
		// always wrap raw data in domain object
		return account.NewAccountFromData(*accData, s.live)
	})
}
```

<!-- archie:ai-end -->
