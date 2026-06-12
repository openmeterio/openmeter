# service

<!-- archie:ai-start -->

> Service layer implementing ledger.account.Service over a ledgeraccount.Repo: validates inputs, maps Data structs into rich domain Account/SubAccount values, and provides advisory locking for posting. Single file service.go.

## Patterns

**Validate-then-delegate** — Mutating methods call input.Validate() first and return early on error before touching the repo. (`func (s *service) CreateAccount(ctx, input) { if err := input.Validate(); err != nil { return nil, err }; accData, err := s.repo.CreateAccount(ctx, input)... }`)
**Data-to-domain mapping** — Repo returns *Data; service wraps each into a domain value via account.NewAccountFromData(data, s.live) or account.NewSubAccountFromData(data). (`return account.NewAccountFromData(*accData, s.live)`)
**Self-wired live services** — New() sets svc.live = account.AccountLiveServices{SubAccountService: svc} so accounts can resolve sub-account posting addresses through the same service. (`svc.live = account.AccountLiveServices{SubAccountService: svc}; return svc`)
**transaction.Run for multi-step writes** — EnsureSubAccount wraps repo call in transaction.Run(ctx, s.repo, ...) so the create-and-map sequence shares one tx. (`return transaction.Run(ctx, s.repo, func(ctx) (ledger.SubAccount, error) { ... })`)
**Deterministic lock ordering** — LockAccountsForPosting dedupes by ID, sorts by (namespace, id), then LockForTX each, to avoid deadlocks; only customer FBO/Receivable/Accrued account types are locked. (`sort.Slice(ids, ...); key, _ := lockr.NewKey("namespace", id.Namespace, "account", id.ID); s.locker.LockForTX(ctx, key)`)
**Nil-locker tolerance** — LockAccountsForPosting returns nil immediately when s.locker is nil, so locking is optional dependency. (`if s.locker == nil { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Whole service: struct {repo, locker, live}, New constructor, account/sub-account CRUD, and LockAccountsForPosting. | Always pass s.live into NewAccountFromData; locking only covers customer FBO/Receivable/Accrued types and must keep the sorted-by-namespace-then-id order to stay deadlock-free. |

## Anti-Patterns

- Skipping input.Validate() before delegating mutations to the repo.
- Returning repo *Data structs directly instead of NewAccountFromData / NewSubAccountFromData domain values.
- Forgetting to pass s.live to NewAccountFromData, breaking sub-account address resolution.
- Locking accounts in arbitrary order, reintroducing deadlock risk.
- Using slog.Default() or context.Background() instead of injected dependencies / propagated ctx.

## Decisions

- **SubAccountService is self-wired into AccountLiveServices inside New rather than injected.** — Account-specific route helpers need a concrete sub-account service to create posting addresses; the service is its own provider so no external wiring is required.
- **Posting locks are scoped to customer account types and ordered deterministically.** — Only customer FBO/Receivable/Accrued accounts participate in concurrent posting; sorted lock acquisition prevents deadlocks across concurrent transactions.

## Example: Constructor self-wiring live services and validate-then-delegate create

```
func New(repo account.Repo, locker *lockr.Locker) account.Service {
	svc := &service{repo: repo, locker: locker}
	svc.live = account.AccountLiveServices{SubAccountService: svc}
	return svc
}

func (s *service) CreateAccount(ctx context.Context, input ledger.CreateAccountInput) (ledger.Account, error) {
	if err := input.Validate(); err != nil { return nil, err }
	accData, err := s.repo.CreateAccount(ctx, input)
	if err != nil { return nil, fmt.Errorf("failed to create account: %w", err) }
	return account.NewAccountFromData(*accData, s.live)
}
```

<!-- archie:ai-end -->
