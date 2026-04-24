# service

<!-- archie:ai-start -->

> Business-logic service layer for ledger accounts and sub-accounts, delegating all persistence to account.Repo and enriching raw AccountData/SubAccountData with live runtime dependencies (Locker, Querier) via account.AccountLiveServices.

## Patterns

**Service wires itself as SubAccountService** — New() injects the constructed service back into live.SubAccountService before storing live, so Account domain objects receive a real SubAccountService reference. Never construct service without this self-wiring step. (`func New(repo account.Repo, live account.AccountLiveServices) account.Service { svc := &service{repo: repo}; live.SubAccountService = svc; svc.live = live; return svc }`)
**Validate input before repo calls** — CreateAccount and EnsureSubAccount call input.Validate() and return immediately on error before touching the repo. All mutating methods must validate first. (`if err := input.Validate(); err != nil { return nil, err }`)
**transaction.Run for multi-step mutations** — EnsureSubAccount wraps the repo call and subsequent GetAccountByID in a single transaction.Run(ctx, s.repo, ...) to ensure both complete atomically. (`return transaction.Run(ctx, s.repo, func(ctx context.Context) (*account.SubAccount, error) { subAccountData, _ := s.repo.EnsureSubAccount(ctx, input); acc, _ := s.GetAccountByID(ctx, ...); return account.NewSubAccountFromData(...) })`)
**Domain object construction via NewAccountFromData / NewSubAccountFromData** — Service never returns raw *AccountData or *SubAccountData — it always wraps them in account.NewAccountFromData(data, s.live) or account.NewSubAccountFromData(data, acc). This attaches live runtime services to the domain object. (`return account.NewAccountFromData(*accData, s.live)`)
**N+1 mitigation with local accountsByID cache in ListSubAccounts** — ListSubAccounts deduplicates GetAccountByID calls using an in-memory accountsByID map keyed by accountID. New batch-list methods should follow this pattern until the Ent query is fixed. (`accountsByID := make(map[string]*account.Account, len(subAccountDatas)); for _, d := range subAccountDatas { acc, ok := accountsByID[d.AccountID]; if !ok { acc, _ = s.GetAccountByID(...); accountsByID[d.AccountID] = acc } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single file containing the entire service implementation. Exposes New() constructor and implements account.Service via compile-time assertion var _ account.Service = (*service)(nil). | The self-wiring of live.SubAccountService in New() is load-bearing — if a caller passes a pre-built AccountLiveServices without going through New(), SubAccountService will be nil and Account.SubAccount() calls will panic. |

## Anti-Patterns

- Returning *AccountData or *SubAccountData directly from service methods — callers expect domain objects with live services attached
- Calling s.repo.EnsureSubAccount outside a transaction.Run when the method also needs to call s.GetAccountByID — creates a TOCTOU window
- Constructing service and setting live.SubAccountService externally instead of via New() — SubAccountService self-wiring is intentional
- Introducing context.Background() or context.TODO() — propagate the caller's ctx throughout

## Decisions

- **AccountLiveServices is injected as a value type and the service self-registers as SubAccountService inside New().** — Account domain objects need a SubAccountService reference to load their sub-accounts lazily; circular DI is resolved by having the single service implement both Account and SubAccount operations and wiring itself into the live struct at construction time.
- **ListSubAccounts uses an in-memory accountsByID cache rather than a joined Ent query.** — The FIXME comment notes this is an intentional N+1 deferral — the cache makes it O(distinct accounts) rather than O(sub-accounts) while the proper Ent join is deferred.

<!-- archie:ai-end -->
