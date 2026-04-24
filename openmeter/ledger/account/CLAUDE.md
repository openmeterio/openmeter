# account

<!-- archie:ai-start -->

> Domain object layer for ledger accounts and sub-accounts: defines Account, SubAccount, Address, Balance types that implement ledger interfaces, and exposes the Repo and Service interfaces for persistence and orchestration. It is the structural core of the ledger domain — all higher-level packages (resolvers, historical) import from here.

## Patterns

**Data/Live split** — AccountData and SubAccountData are plain DTOs; Account and SubAccount embed the data and receive AccountLiveServices (Querier, Locker, SubAccountService) at construction time via NewAccountFromData / NewSubAccountFromData. (`NewAccountFromData(data, AccountLiveServices{Querier: ledger, Locker: locker, SubAccountService: svc})`)
**Type-asserting account specialisation** — Account.AsCustomerFBOAccount(), AsCustomerReceivableAccount(), AsCustomerAccruedAccount(), AsBusinessAccount() guard on AccountType and return typed wrappers — callers must call these before using type-specific methods. (`fboAcc, err := acc.AsCustomerFBOAccount()`)
**EnsureSubAccount upsert via route** — Sub-accounts are created idempotently: EnsureSubAccount in Service/Repo takes a CreateSubAccountInput with a Route and either finds an existing row matching the canonical routing key or creates a new one. (`svc.EnsureSubAccount(ctx, CreateSubAccountInput{Namespace: ns, AccountID: id, Route: params.Route()})`)
**Validate before repo calls** — All input types expose a Validate() method (e.g. CreateAccountInput.Validate, CreateSubAccountInput.Validate); call it before passing to repo. (`if err := input.Validate(); err != nil { return nil, err }`)
**Interface assertion at compile time** — Every concrete type asserts its interface: var _ ledger.Account = (*Account)(nil), var _ ledger.SubAccount = (*SubAccount)(nil), var _ ledger.PostingAddress = (*Address)(nil). (`var _ ledger.Account = (*Account)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | Account domain object — embeds AccountData + AccountLiveServices; implements ledger.Account (GetBalance, Type, ID); hosts AsCustomerAccount/AsFBOAccount/AsBusinessAccount type-asserting methods. | GetBalance returns error if Querier is nil; always inject AccountLiveServices fully. |
| `account_customer.go` | CustomerAccount, CustomerFBOAccount, CustomerReceivableAccount, CustomerAccruedAccount wrappers — each embed *Account or *CustomerAccount and implement type-specific GetSubAccountForRoute. | Lock() calls lockr.LockForTX — requires an active Postgres transaction in ctx. |
| `account_business.go` | BusinessAccount wrapper for wash/earnings/brokerage accounts; implements ledger.BusinessAccount. | AsBusinessAccount panics if AccountType is not one of the three business types. |
| `subaccount.go` | SubAccount domain object — holds SubAccountData + parent *Account + *Address; implements ledger.SubAccount (Address, Route, AccountID, GetBalance). | GetBalance delegates to parent account.GetBalance; parent account must not be nil. |
| `address.go` | Address wraps AddressData and pre-builds a SubAccountRoute via NewAddressFromData; implements ledger.PostingAddress. | Requires non-empty SubAccountID and RouteID; Equal() comparison is field-by-field, not pointer-equal. |
| `repo.go` | Repo interface — embeds entutils.TxCreator; declares CreateAccount, GetAccountByID, EnsureSubAccount, GetSubAccountByID, ListSubAccounts, ListAccounts. | All implementations must also implement TxCreator (WithTx, Self, Tx). |
| `service.go` | Service interface and all input/output types (CreateAccountInput, CreateSubAccountInput, ListAccountsInput, ListSubAccountsInput). | Service returns domain objects (*Account, *SubAccount), not DTOs — adapters must call New*FromData converters. |

## Anti-Patterns

- Calling repo methods directly from Account or SubAccount methods without going through AccountLiveServices.SubAccountService — bypasses service-layer transaction management.
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

func provisionSubAccount(ctx context.Context, svc ledgeraccount.Service, ns, accountID string) (*ledgeraccount.SubAccount, error) {
	return svc.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
		Namespace: ns,
		AccountID: accountID,
		Route: ledger.Route{Currency: currencyx.Code("USD")},
	})
}
```

<!-- archie:ai-end -->
