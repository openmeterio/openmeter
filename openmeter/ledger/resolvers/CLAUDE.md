# resolvers

<!-- archie:ai-start -->

> Provisioning + resolution layer that maps customers and namespaces to their concrete ledger accounts. AccountResolver creates/fetches the per-customer FBO/Receivable/Accrued accounts and shared per-namespace business accounts (Wash/Earnings/Brokerage/Breakage), backed by the ledger_customer_account linking table. Wiring is feature-gated behind credits.enabled.

## Patterns

**Provision-inside-transaction with advisory lock** — CreateCustomerAccounts/EnsureBusinessAccounts run inside transaction.Run and acquire a bounded lockr lock (provisioningLockTimeout = 5s) before creating missing accounts. (`transaction.Run(ctx, s.Repo, func(ctx){ s.lockCustomerProvisioning(ctx, customerID); ... })`)
**Idempotent create-or-reuse** — Provisioning skips account types already present, and on a duplicate-mapping error uses AsCustomerAccountAlreadyExistsError to reuse the existing AccountID instead of failing. (`if existingErr, ok := AsCustomerAccountAlreadyExistsError(err); ok { accountIDs[accountType] = existingErr.AccountID; continue }`)
**Typed account-type assertion on read** — Get* fetch each account, assert it to the expected ledger.*Account interface, and error with type detail on mismatch; missing types return ledger.ErrCustomerAccountMissing / ErrBusinessAccountMissing with attrs. (`fboAccount, ok := fboAcc.(ledger.CustomerFBOAccount); if !ok { return ..., fmt.Errorf("...expected %s", ledger.AccountTypeCustomerFBO) }`)
**Lifecycle hooks drive provisioning** — customerLedgerHook.PostCreate provisions customer accounts on customer creation; namespaceHandler.CreateNamespace ensures business accounts on namespace creation. (`func (h *customerLedgerHook) PostCreate(ctx, cust) error { _, err := h.config.Service.CreateCustomerAccounts(ctx, customer.CustomerID{...}); return err }`)
**Nil-locker tolerance** — lockCustomerProvisioning/lockBusinessProvisioning return nil when s.Locker == nil, so the resolver works in single-process tests without a real lock. (`if s.Locker == nil { return nil }`)
**Conflict error carries ValidationIssues** — CustomerAccountAlreadyExistsError implements ValidationErrors() returning ErrCustomerAccountConflict (HTTP 409) with namespace/customer/account-type attrs. (`ErrCustomerAccountConflict.WithAttrs(models.Attributes{"customer_id": e.CustomerID.ID, ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | AccountResolver: CreateCustomerAccounts, GetCustomerAccounts, EnsureBusinessAccounts, GetBusinessAccounts, and the lockr provisioning helpers. | Provisioning locks are a TODO guardrail with a 5s timeout (lockr.ErrLockTimeout on DeadlineExceeded); do not assume they replace upsert convergence. CustomerAccountTypes/BusinessAccountTypes drive which accounts get created. |
| `customeraccount.go` | CustomerAccountRepo interface (linking table) + CreateCustomerAccountInput. | The repo only manages the customer->account-id mapping by AccountType; account creation itself goes through AccountService. |
| `errors.go` | CustomerAccountAlreadyExistsError + AsCustomerAccountAlreadyExistsError + ErrCustomerAccountConflict validation issue. | Idempotency depends on the adapter wrapping its constraint error as this typed error; matching is via errors.As. |
| `hooks.go` | CustomerLedgerHook (ServiceHook[customer.Customer]) provisioning accounts on PostCreate, with OTel span. | These hooks must stay disabled when credits.enabled is false (see AGENTS.md); the hook only wires the create path, not updates. |
| `namespace.go` | namespaceHandler implementing namespace.Handler; CreateNamespace ensures business accounts, DeleteNamespace is a no-op. | Must be registered before initNamespace if business accounts are needed at startup. |
| `account_test.go` | Concurrency tests proving CreateCustomerAccounts/EnsureBusinessAccounts are idempotent under 6 concurrent callers. | Tests assert exactly 3 mapping rows and 3 accounts; provisioning regressions surface as count mismatches. |

## Anti-Patterns

- Failing on a duplicate-mapping error instead of reusing the existing AccountID via AsCustomerAccountAlreadyExistsError.
- Provisioning accounts outside transaction.Run or without the provisioning lock, allowing duplicate accounts under concurrency.
- Returning raw account values from Get* without asserting them to the expected ledger.*Account interface.
- Wiring the customer ledger hook / namespace handler when credits.enabled is false.
- Blocking indefinitely on the provisioning lock instead of honoring provisioningLockTimeout.

## Decisions

- **Provisioning is guarded by a bounded advisory lock plus idempotent create-or-reuse rather than a single upsert.** — Simultaneous multi-service startups must converge to one set of accounts; the lock fails fast (5s) and the duplicate-error reuse path keeps creation safe even if the lock is skipped.
- **Account creation is delegated to AccountService while this package owns only the customer->account linking table.** — Keeps the resolver focused on mapping/resolution and lets account construction (types, validation) live in the account service.

## Example: Idempotent per-customer account provisioning

```
return transaction.Run(ctx, s.Repo, func(ctx context.Context) (ledger.CustomerAccounts, error) {
	if err := s.lockCustomerProvisioning(ctx, customerID); err != nil {
		return ledger.CustomerAccounts{}, err
	}
	accountIDs, _ := s.Repo.GetCustomerAccountIDs(ctx, customerID)
	for _, accountType := range ledger.CustomerAccountTypes {
		if _, ok := accountIDs[accountType]; ok { continue }
		acc, _ := s.AccountService.CreateAccount(ctx, ledgeraccount.CreateAccountInput{Namespace: customerID.Namespace, Type: accountType})
		if err := s.Repo.CreateCustomerAccount(ctx, CreateCustomerAccountInput{CustomerID: customerID, AccountType: accountType, AccountID: acc.ID().ID}); err != nil {
			if existingErr, ok := AsCustomerAccountAlreadyExistsError(err); ok { accountIDs[accountType] = existingErr.AccountID; continue }
			return ledger.CustomerAccounts{}, err
		}
	}
	return s.GetCustomerAccounts(ctx, customerID)
})
```

<!-- archie:ai-end -->
