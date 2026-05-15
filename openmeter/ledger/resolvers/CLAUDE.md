# resolvers

<!-- archie:ai-start -->

> Bridges the ledger account layer to customer and namespace lifecycle: AccountResolver provisions customer FBO/Receivable/Accrued accounts and business Wash/Earnings/Brokerage accounts on demand, CustomerLedgerHook wires account creation into customer.PostCreate, and namespaceHandler calls EnsureBusinessAccounts on namespace creation. Must only be wired when credits.enabled=true.

## Patterns

**transaction.Run + provisioning lock** — CreateCustomerAccounts and EnsureBusinessAccounts both open a transaction.Run block, acquire a provisioning advisory lock (lockr.LockForTX with 5s timeout via context.WithTimeout), then create accounts if missing. Never skip the lock in production. (`return transaction.Run(ctx, s.Repo, func(ctx context.Context) (ledger.CustomerAccounts, error) { s.lockCustomerProvisioning(ctx, customerID); ... })`)
**Idempotent create via AsCustomerAccountAlreadyExistsError** — CreateCustomerAccounts catches CustomerAccountAlreadyExistsError from concurrent inserts and uses the existing AccountID rather than returning an error. Adapter must return this typed error on unique-constraint violations. (`if existingErr, ok := AsCustomerAccountAlreadyExistsError(err); ok { accountIDs[accountType] = existingErr.AccountID; continue }`)
**CustomerLedgerHook embeds NoopCustomerLedgerHook** — customerLedgerHook only overrides PostCreate; all other ServiceHook methods are satisfied by the embedded NoopCustomerLedgerHook. New hooks follow the same embed pattern. (`type customerLedgerHook struct { NoopCustomerLedgerHook; config CustomerLedgerHookConfig }`)
**namespaceHandler accepts businessAccountProvisioner interface** — NewNamespaceHandler accepts a businessAccountProvisioner interface (not *AccountResolver directly) so the handler can be wired without depending on the concrete type. (`func NewNamespaceHandler(provisioner businessAccountProvisioner) namespace.Handler { ... }`)
**ValidationIssue error for conflict with HTTP 409** — CustomerAccountAlreadyExistsError implements the validationErrors interface and maps to ErrCustomerAccountConflict (HTTP 409) via commonhttp.WithHTTPStatusCodeAttribute on the ValidationIssue. (`var ErrCustomerAccountConflict = models.NewValidationIssue(ErrCodeCustomerAccountConflict, "...", commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | AccountResolver — implements ledger.AccountResolver; manages provisioning locks and idempotent account creation for customers and business accounts. | Locker is optional (nil check before LockForTX); always use transaction.Run even when Locker is nil so the tx context propagates correctly. |
| `customeraccount.go` | CustomerAccountRepo interface (TxCreator + CreateCustomerAccount + GetCustomerAccountIDs) and CreateCustomerAccountInput type. | Adapter must return CustomerAccountAlreadyExistsError on unique-constraint violations — not a raw DB error. |
| `errors.go` | CustomerAccountAlreadyExistsError type, ErrCustomerAccountConflict validation issue, AsCustomerAccountAlreadyExistsError unwrap helper. | AsCustomerAccountAlreadyExistsError must be used in CreateCustomerAccounts to handle concurrent inserts gracefully. |
| `hooks.go` | customerLedgerHook with PostCreate calling CreateCustomerAccounts, NewCustomerLedgerHook constructor. | Hook must be registered via customerService.RegisterHooks only when credits.enabled=true in app/common. |
| `namespace.go` | namespaceHandler implementing namespace.Handler; CreateNamespace calls EnsureBusinessAccounts, DeleteNamespace is a no-op. | Must be registered before initNamespace() in cmd/server/main.go or the default namespace will not have business accounts. |

## Anti-Patterns

- Registering CustomerLedgerHook or namespaceHandler when credits.enabled=false — these are real write paths; use noop variants (ledgernoop.*) instead.
- Calling GetCustomerAccounts without a preceding CreateCustomerAccounts for new customers — returns ErrCustomerAccountMissing.
- Constructing AccountResolver without a Locker in production — concurrent provisioning will create duplicate accounts.
- Adding business logic (balance calculation, route resolution) to the adapter (resolvers/adapter) — belongs in AccountResolver service layer.
- Using context.Background() inside hooks or namespace handler — severs OTel traces and transaction context.

## Decisions

- **Provisioning lock uses a 5-second context.WithTimeout converted to lockr.ErrLockTimeout on deadline exceeded.** — Prevents multi-service startup races from blocking indefinitely; fail-fast is preferable to indefinite serialization.
- **CustomerAccountRepo is a minimal interface (just two methods + TxCreator) rather than a full Ent client.** — Allows testing AccountResolver with a mock repo without standing up a Postgres database.

## Example: Wire CustomerLedgerHook and namespace handler when credits are enabled

```
import (
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
)

// In app/common/customer.go (credits-enabled path only):
hook, err := resolvers.NewCustomerLedgerHook(resolvers.CustomerLedgerHookConfig{
	Service: accountResolver,
	Tracer:  tracer,
})
if err != nil { return err }
customerService.RegisterHooks(hook)

// In app/common/namespace.go:
ns.RegisterHandler(resolvers.NewNamespaceHandler(accountResolver))
```

<!-- archie:ai-end -->
