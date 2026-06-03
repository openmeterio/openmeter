# resolvers

<!-- archie:ai-start -->

> Bridges the ledger account layer to customer and namespace lifecycle: AccountResolver provisions customer FBO/Receivable/Accrued and business Wash/Earnings/Brokerage accounts on demand, CustomerLedgerHook wires account creation into customer.PostCreate, and namespaceHandler provisions business accounts on namespace creation. Only wired when credits.enabled=true.

## Patterns

**transaction.Run + provisioning lock** — CreateCustomerAccounts and EnsureBusinessAccounts open transaction.Run, acquire a provisioning advisory lock (lockr.LockForTX with a 5s bounded wait), then create missing accounts. Always run inside transaction.Run even when Locker is nil so ctx propagates. (`return transaction.Run(ctx, s.Repo, func(ctx context.Context) (ledger.CustomerAccounts, error) { s.lockCustomerProvisioning(ctx, customerID); ... })`)
**Idempotent create via typed conflict error** — CreateCustomerAccounts catches CustomerAccountAlreadyExistsError (via AsCustomerAccountAlreadyExistsError) from concurrent inserts and reuses the existing AccountID; the adapter must return this typed error on unique-constraint violations. (`if existingErr, ok := AsCustomerAccountAlreadyExistsError(err); ok { accountIDs[accountType] = existingErr.AccountID; continue }`)
**Hook embeds NoopCustomerLedgerHook** — customerLedgerHook overrides only PostCreate; all other ServiceHook methods come from the embedded NoopCustomerLedgerHook. (`type customerLedgerHook struct { NoopCustomerLedgerHook; config CustomerLedgerHookConfig }`)
**namespaceHandler accepts a businessAccountProvisioner interface** — NewNamespaceHandler takes an interface, not *AccountResolver, so wiring does not depend on the concrete type. (`func NewNamespaceHandler(provisioner businessAccountProvisioner) namespace.Handler`)
**Typed-account assertion on read** — GetCustomerAccounts type-asserts each fetched account to its ledger.CustomerXAccount interface and errors on mismatch; missing mappings return ledger.ErrCustomerAccountMissing. (`fboAccount, ok := fboAcc.(ledger.CustomerFBOAccount)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | AccountResolver — implements ledger.AccountResolver; provisioning locks and idempotent customer/business account creation. | Locker is optional (nil-check before LockForTX) but always wrap in transaction.Run for ctx propagation. |
| `customeraccount.go` | CustomerAccountRepo interface (TxCreator + CreateCustomerAccount + GetCustomerAccountIDs) and CreateCustomerAccountInput. | Adapter must return CustomerAccountAlreadyExistsError on unique-constraint violations, not a raw DB error. |
| `errors.go` | CustomerAccountAlreadyExistsError, ErrCustomerAccountConflict (HTTP 409 ValidationIssue), AsCustomerAccountAlreadyExistsError unwrap helper. | Use the unwrap helper in CreateCustomerAccounts to handle concurrent inserts. |
| `hooks.go` | customerLedgerHook with PostCreate calling CreateCustomerAccounts; NewCustomerLedgerHook constructor. | Register via customerService.RegisterHooks only when credits.enabled=true in app/common. |
| `namespace.go` | namespaceHandler: CreateNamespace calls EnsureBusinessAccounts, DeleteNamespace is a no-op. | Must be registered before initNamespace() in cmd/server/main.go or the default namespace lacks business accounts. |

## Anti-Patterns

- Registering CustomerLedgerHook or namespaceHandler when credits.enabled=false — use ledgernoop.* variants instead.
- Calling GetCustomerAccounts before CreateCustomerAccounts for a new customer — returns ErrCustomerAccountMissing.
- Constructing AccountResolver without a Locker in production — concurrent provisioning creates duplicate accounts.
- Adding balance/route business logic to resolvers/adapter — it belongs in the AccountResolver service layer.
- Using context.Background() in hooks or the namespace handler — severs OTel traces and transaction context.

## Decisions

- **Provisioning lock uses a 5s bounded wait converted to lockr.ErrLockTimeout on deadline.** — Multi-service startup races fail fast instead of blocking indefinitely.
- **CustomerAccountRepo is a minimal two-method interface (+ TxCreator), not a full Ent client.** — Lets AccountResolver be unit-tested with a mock repo without a Postgres instance.

## Example: Wire CustomerLedgerHook and namespace handler when credits are enabled

```
hook, err := resolvers.NewCustomerLedgerHook(resolvers.CustomerLedgerHookConfig{Service: accountResolver, Tracer: tracer})
if err != nil { return err }
customerService.RegisterHooks(hook)
ns.RegisterHandler(resolvers.NewNamespaceHandler(accountResolver))
```

<!-- archie:ai-end -->
