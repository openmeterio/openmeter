# account

<!-- archie:ai-start -->

> Domain layer for ledger accounts and sub-accounts: defines the Account/SubAccount/Address value objects, account-type behaviors (FBO, Receivable, Accrued, business), the Repo and Service interfaces, and the posting-address model. The hard constraint is that all rich domain objects are constructed from plain *Data DTOs via NewAccountFromData / NewSubAccountFromData — never by hand.

## Patterns

**Data DTO to domain construction** — Every domain value is built from a flat *Data struct through a NewXFromData constructor that validates and resolves embedded sub-objects (e.g. routing key, address). (`NewAccountFromData(AccountData{...}, services) switches on AccountType to return newCustomerFBOAccount / newBusinessAccount etc.`)
**Account-type polymorphism via embedding** — Concrete account types embed *Account (and CustomerAccount embeds it transitively) and implement GetSubAccountForRoute with a type-specific RouteParams. (`type CustomerFBOAccount struct { *CustomerAccount }; var _ ledger.CustomerFBOAccount = (*CustomerFBOAccount)(nil)`)
**Compile-time interface assertions** — Each type asserts the ledger interface it satisfies with a var _ ledger.X = (*T)(nil) line so interface drift fails at build time. (`var _ ledger.SubAccount = (*SubAccount)(nil)`)
**Validate params before EnsureSubAccount** — GetSubAccountForRoute always calls params.Validate() before delegating to services.SubAccountService.EnsureSubAccount. (`if err := params.Validate(); err != nil { return nil, err }`)
**SubAccount creation is find-or-create** — Sub-accounts are obtained through EnsureSubAccount(CreateSubAccountInput{Namespace, AccountID, Route}); the route is the idempotency key. (`a.services.SubAccountService.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{Route: params.Route()})`)
**Address carries the SubAccountRoute** — Address is the ledger.PostingAddress impl built from AddressData; its Equal() compares SubAccountID, AccountType, route ID, routing-key version and value. (`NewAddressFromData(AddressData{SubAccountID, AccountType, Route, RouteID, RoutingKey})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | Account value object + NewAccountFromData factory that dispatches on AccountType to the concrete type. | Adding a new AccountType requires a new case here AND a concrete type file; the default branch only validates and returns the base Account. |
| `account_customer.go` | Customer FBO / Receivable / Accrued account types, each with its own RouteParams. | Accrued accounts route by currency (+tax/cost-basis) only, not credit priority — do not copy FBO routing into accrued. |
| `account_business.go` | BusinessAccount (Wash/Earnings/Brokerage/Breakage) GetSubAccountForRoute via BusinessRouteParams. | Business routes should carry only dimensions relevant to that account's accounting, not customer credit-priority. |
| `address.go` | Address (ledger.PostingAddress) built from AddressData; defines structural Equal(). | SubAccountID, AccountType and RouteID are all required; missing fields error in newSubAccountRouteFromAddressData. |
| `subaccount.go` | SubAccount + NewSubAccountFromData, hydrates Address from RouteMeta (routing key version/value). | RouteMeta.RoutingKeyVersion/RoutingKey must round-trip through ledger.NewRoutingKey or construction fails. |
| `repo.go` | Repo interface (embeds entutils.TxCreator) for account/sub-account create/get/list and EnsureSubAccount. | Repo only persists *Data structs; it never returns domain Account/SubAccount values. |
| `service.go` | Service interface = ledger.AccountCatalog + ledger.AccountLocker, plus Input type aliases re-exported from ledger. | Input types are aliases of ledger.* — do not redeclare them. |

## Anti-Patterns

- Constructing Account/SubAccount/Address structs directly instead of via NewAccountFromData / NewSubAccountFromData / NewAddressFromData.
- Returning repo *Data structs from a service method instead of mapping to domain values.
- Skipping params.Validate() before EnsureSubAccount in a GetSubAccountForRoute implementation.
- Adding an AccountType without a switch case in NewAccountFromData and a concrete type implementing GetSubAccountForRoute.
- Dropping a var _ ledger.X = (*T)(nil) assertion, letting interface drift go undetected.

## Decisions

- **Account behavior is split into concrete types embedding *Account rather than a single struct with a type switch.** — Each account type exposes a strongly-typed RouteParams (CustomerFBORouteParams vs BusinessRouteParams), so the type system enforces correct routing per account kind.
- **Domain objects are reconstructed from flat *Data DTOs.** — Keeps persistence (adapter) decoupled from rich domain construction and lets a single factory enforce validation and address resolution.

## Example: Account type resolving a routed sub-account

```
func (a *CustomerFBOAccount) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerFBORouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return a.services.SubAccountService.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}
```

<!-- archie:ai-end -->
