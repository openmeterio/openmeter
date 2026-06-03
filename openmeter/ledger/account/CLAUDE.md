# account

<!-- archie:ai-start -->

> Domain-object core of the ledger: defines Account, SubAccount, Address, and Balance types that implement the ledger.* interfaces, plus the Repo (persistence) and Service (orchestration) contracts. All higher ledger packages (resolvers, historical, transactions, breakage) build on the types defined here.

## Patterns

**Data/Live split with construction-time injection** — *Data structs are plain DTOs; Account/SubAccount embed the data and receive AccountLiveServices (SubAccountService) only via NewAccountFromData / NewSubAccountFromData. Never set live services by field assignment after construction. (`NewAccountFromData(data, AccountLiveServices{SubAccountService: svc})`)
**Type-dispatching constructor** — NewAccountFromData switches on AccountType and returns typed wrappers (CustomerFBOAccount, CustomerReceivableAccount, CustomerAccruedAccount, BusinessAccount). Callers type-assert on the returned ledger.Account. (`fboAcc, ok := acc.(ledger.CustomerFBOAccount)`)
**EnsureSubAccount upsert via canonical route** — Sub-accounts are created idempotently through services.SubAccountService.EnsureSubAccount keyed on the canonical routing key; never call a raw create. (`svc.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{Namespace: ns, AccountID: id, Route: params.Route()})`)
**Validate before repo/service calls** — Every input and RouteParams type exposes Validate(); call it (GetSubAccountForRoute calls params.Validate() first) before any persistence call. (`if err := params.Validate(); err != nil { return nil, err }`)
**Compile-time interface assertions** — Each concrete type carries a var _ ledger.X = (*Type)(nil) line — the only compile-time proof of interface compliance. Do not remove. (`var _ ledger.CustomerFBOAccount = (*CustomerFBOAccount)(nil)`)
**SubAccountService self-wiring in service.New()** — account/service.New() wires SubAccountService as a forward reference to itself; do not inject it externally — the self-wiring is load-bearing. (`svc := &service{}; svc.live.SubAccountService = svc; return svc`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | Account domain object + AccountData DTO + NewAccountFromData type-dispatcher and AccountLiveServices container. | nil SubAccountService panics on GetSubAccountForRoute; populate AccountLiveServices fully at construction. |
| `account_customer.go` | CustomerFBO/Receivable/Accrued wrappers; each routes via EnsureSubAccount after params.Validate(). | RouteParams missing Currency error on Validate(); accrued is routed by currency only. |
| `account_business.go` | BusinessAccount wrapper for wash/earnings/brokerage/breakage; newBusinessAccount only valid for those types. | newBusinessAccount is only safe inside NewAccountFromData's business-type case. |
| `subaccount.go` | SubAccount domain object; NewSubAccountFromData builds the Address from RouteMeta. | Empty SubAccountID or RouteID errors during address construction. |
| `address.go` | Address wraps AddressData and prebuilds SubAccountRoute; implements ledger.PostingAddress with field-by-field Equal(). | RouteID and SubAccountID both required; Equal() is value comparison, not pointer. |
| `repo.go` | Repo interface embedding entutils.TxCreator with CreateAccount/EnsureSubAccount/Get/List methods. | Implementations must satisfy the full TxCreator triad for TransactingRepo to work. |
| `service.go` | Service interface aliasing ledger.AccountCatalog + ledger.AccountLocker; input types re-exported from ledger. | Service returns ledger.Account/ledger.SubAccount domain objects, not *Data DTOs. |

## Anti-Patterns

- Returning *AccountData / *SubAccountData from Service methods — callers expect live domain objects with services attached.
- Setting live.SubAccountService externally instead of relying on service.New() self-wiring.
- Manually building AddressData.RoutingKey strings instead of ledger.BuildRoutingKey — breaks canonical uniqueness.
- Removing var _ ledger.X = (*T)(nil) compile-time assertions.
- Using context.Background()/context.TODO() in any method — propagate the caller ctx.

## Decisions

- **AccountLiveServices is a value type injected at construction; SubAccountService is a forward reference so the service self-registers.** — Avoids a global service registry and resolves the chicken-and-egg dependency of the service needing itself as the SubAccountService.
- **Sub-accounts are keyed by a canonical routing key derived from the Route value.** — Guarantees idempotent upsert: two callers asking for the same route converge on one row without races.

## Example: Provision a sub-account for a customer FBO account

```
return svc.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{
  Namespace: ns,
  AccountID: accountID,
  Route:     ledger.Route{Currency: currencyx.Code("USD")},
})
```

<!-- archie:ai-end -->
