# account

<!-- archie:ai-start -->

> Domain object layer for ledger accounts and sub-accounts: defines Account, SubAccount, Address, Balance types that implement ledger interfaces, and exposes the Repo and Service interfaces for persistence and orchestration. It is the structural core of the ledger domain — all higher-level packages (resolvers, historical, transactions) import from here.

## Patterns

**Data/Live split** — AccountData and SubAccountData are plain DTOs; Account and SubAccount embed the data and receive AccountLiveServices (SubAccountService) at construction time via NewAccountFromData / NewSubAccountFromData. Never pass live services via struct field assignment after construction. (`NewAccountFromData(data, AccountLiveServices{SubAccountService: svc})`)
**Type-asserting account specialisation** — NewAccountFromData dispatches on AccountType and returns typed wrappers (CustomerFBOAccount, CustomerReceivableAccount, BusinessAccount). Callers must type-assert via the returned ledger.Account interface before using type-specific methods. (`fboAcc, ok := acc.(ledger.CustomerFBOAccount)`)
**EnsureSubAccount upsert via route** — Sub-accounts are created idempotently: EnsureSubAccount takes a CreateSubAccountInput with a Route and either finds the existing row matching the canonical routing key or creates a new one. Never call CreateSubAccount directly. (`svc.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{Namespace: ns, AccountID: id, Route: params.Route()})`)
**Validate before repo calls** — All input types expose a Validate() method; call it before passing to repo. Service and adapter methods must not accept unvalidated inputs. (`if err := input.Validate(); err != nil { return nil, err }`)
**Compile-time interface assertions** — Every concrete type has a var _ ledger.X = (*ConcreteType)(nil) line at the package level. Do not remove these — they are the only compile-time proof of interface compliance. (`var _ ledger.Account = (*Account)(nil)`)
**SubAccountService self-wiring in account/service.New()** — The service package wires SubAccountService as a forward reference inside New() rather than requiring it as an external dependency. Do not inject SubAccountService externally — the self-wiring is intentional and load-bearing. (`svc := &service{}; svc.live.SubAccountService = svc; return svc`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | Account domain object — embeds AccountData + AccountLiveServices; implements ledger.Account; hosts NewAccountFromData type-dispatching constructor. | AccountLiveServices must be fully populated at construction; nil SubAccountService causes runtime panics on GetSubAccountForRoute calls. |
| `account_customer.go` | CustomerFBOAccount, CustomerReceivableAccount, CustomerAccruedAccount wrappers — each calls EnsureSubAccount via services.SubAccountService. | GetSubAccountForRoute calls Validate() on RouteParams before EnsureSubAccount; params missing Currency will error. |
| `account_business.go` | BusinessAccount wrapper for wash/earnings/brokerage accounts; implements ledger.BusinessAccount. | newBusinessAccount panics if AccountType is not one of the three business types — only called inside NewAccountFromData. |
| `subaccount.go` | SubAccount domain object — holds SubAccountData + *Address; implements ledger.SubAccount (Address, Route, AccountID). | NewSubAccountFromData requires non-empty SubAccountID and RouteID in SubAccountData or address construction errors. |
| `address.go` | Address wraps AddressData and pre-builds SubAccountRoute via NewAddressFromData; implements ledger.PostingAddress. | Equal() is field-by-field, not pointer-equal. RouteID and SubAccountID both required or constructor errors. |
| `repo.go` | Repo interface — embeds entutils.TxCreator; declares CreateAccount, GetAccountByID, EnsureSubAccount, GetSubAccountByID, ListSubAccounts, ListAccounts. | All Repo implementations must also implement TxCreator (Tx, WithTx, Self) for TransactingRepo to work. |
| `service.go` | Service interface type alias pointing to ledger.AccountCatalog + ledger.AccountLocker. Input types are re-exported from ledger package. | Service returns domain objects (ledger.Account, ledger.SubAccount), not DTOs — adapters must call NewAccountFromData/NewSubAccountFromData. |

## Anti-Patterns

- Calling repo methods directly from Account or SubAccount methods instead of going through AccountLiveServices.SubAccountService — bypasses service-layer transaction management.
- Using context.Background() or context.TODO() in any method — always propagate caller ctx.
- Manually constructing AddressData.RoutingKey as a string instead of calling ledger.BuildRoutingKey — breaks canonical uniqueness.
- Returning *AccountData or *SubAccountData from Service methods — callers expect live domain objects with services attached.
- Importing app/common in tests — causes import cycles; use adapter.NewRepo and account/service.New directly.

## Decisions

- **AccountLiveServices is a value type injected at construction (not stored globally), and SubAccountService is a forward reference so the service can self-register.** — Avoids a global service registry and allows the service to wire itself as the SubAccountService inside New(), preventing a chicken-and-egg dependency.
- **Sub-accounts are keyed by a canonical routing key (ledger.BuildRoutingKey) derived from the Route value.** — Ensures idempotent upsert semantics: two callers asking for the same route get the same sub-account row without race conditions.

## Example: Provision a sub-account for a customer FBO account

```
import (
	"context"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func provisionSubAccount(ctx context.Context, svc ledgeraccount.Service, ns, accountID string) (ledger.SubAccount, error) {
	return svc.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{
		Namespace: ns,
		AccountID: accountID,
		Route:     ledger.Route{Currency: currencyx.Code("USD")},
	})
}
```

<!-- archie:ai-end -->
