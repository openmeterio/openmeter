# noop

<!-- archie:ai-start -->

> No-op implementations of the ledger surface (Ledger, BalanceQuerier, AccountResolver, account.Service, namespace.Handler) used when credits.enabled is false, so callers can be wired without a real ledger backend. Every operation returns zero balances/empty results or trivially-constructed account stubs.

## Patterns

**Compile-time interface assertions for every type** — Each noop struct declares var _ Interface = T{} to guarantee it satisfies the corresponding ledger/account/namespace interface. (`var ( _ ledger.Ledger = Ledger{}; _ ledger.BalanceQuerier = Ledger{} )`)
**Zero-value returns** — Balance methods return balance{} whose Settled()/Pending() are alpacadecimal.Zero; list/get methods return empty slices/structs and nil errors. (`func (Ledger) ListTransactions(...) (ledger.ListTransactionsResult, error) { return ledger.ListTransactionsResult{}, nil }`)
**Construct real account types from data, not bespoke fakes** — newAccount/newSubAccount build via ledgeraccount.NewAccountFromData / NewSubAccountFromData with AccountLiveServices{SubAccountService: AccountService{}} so returned accounts are real ledger account types backed by noop services. (`account, err := ledgeraccount.NewAccountFromData(ledgeraccount.AccountData{...}, ledgeraccount.AccountLiveServices{SubAccountService: AccountService{}})`)
**Stable account-type defaults** — Resolvers return CustomerAccounts/BusinessAccounts populated with the canonical AccountTypeCustomerFBO/Receivable/Accrued/Wash/Earnings/Brokerage/Breakage stubs; normalizeID/normalizeNamespace substitute 'noop' fallbacks for empty IDs. (`return ledger.CustomerAccounts{FBOAccount: customerFBOAccount{...AccountTypeCustomerFBO}, ReceivableAccount: ..., AccruedAccount: ...}, nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `noop.go` | Single file implementing Ledger, AccountResolver, AccountService, NamespaceHandler, and the account/subAccount/postingAddress/balance/businessAccount stubs. | accountTypeForRoute infers type from route fields (TransactionAuthorizationStatus -> Receivable, CreditPriority -> FBO, else Accrued); noopCurrency defaults to USD; error branches in newAccount/newSubAccount fall back to zero-value stubs rather than panicking. |

## Anti-Patterns

- Adding behavior or side effects to a noop method — it must stay a pure zero/empty implementation usable when credits are disabled.
- Returning non-nil errors from noop methods, which would break callers that expect the disabled path to succeed silently.
- Hand-rolling fake account structs instead of going through ledgeraccount.NewAccountFromData/NewSubAccountFromData, diverging from real account semantics.
- Dropping a compile-time interface assertion when adding a new noop type, allowing it to silently fall out of sync with the ledger interface.

## Decisions

- **Noop is the wired implementation when credits.enabled is false** — app/common injects these so the rest of the system compiles and runs unchanged; real backfill must construct concrete adapters directly rather than relying on these.
- **Build real account types via NewAccountFromData rather than minimal fakes** — Keeps returned accounts behaving like real ledger accounts (routing keys, sub-account services) so callers don't special-case the noop path.

## Example: Returning canonical zero customer accounts

```
func (AccountResolver) GetCustomerAccounts(context.Context, customer.CustomerID) (ledger.CustomerAccounts, error) {
  return ledger.CustomerAccounts{
    FBOAccount:        customerFBOAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerFBO}},
    ReceivableAccount: customerReceivableAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerReceivable}},
    AccruedAccount:    customerAccruedAccount{customerAccount: customerAccount{accountType: ledger.AccountTypeCustomerAccrued}},
  }, nil
}
```

<!-- archie:ai-end -->
