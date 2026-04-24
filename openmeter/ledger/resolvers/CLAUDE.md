# resolvers

<!-- archie:ai-start -->

> Bridges the ledger account layer to customer and namespace lifecycle: AccountResolver provisions customer FBO/Receivable/Accrued accounts and business Wash/Earnings/Brokerage accounts on demand, CustomerLedgerHook wires account creation into customer.PostCreate, and namespaceHandler calls EnsureBusinessAccounts on namespace creation.

## Patterns

**transaction.Run for provisioning with lock** — CreateCustomerAccounts and EnsureBusinessAccounts both open a transaction.Run block, acquire a provisioning lock (lockr.LockForTX with 5s timeout), then create accounts if missing. (`return transaction.Run(ctx, s.Repo, func(ctx context.Context) (ledger.CustomerAccounts, error) {
	s.lockCustomerProvisioning(ctx, customerID)
	...
})`)
**Idempotent create with AsCustomerAccountAlreadyExistsError** — CreateCustomerAccount can fail with CustomerAccountAlreadyExistsError from a concurrent insert; the resolver catches it and uses the existing AccountID rather than returning an error. (`if existingErr, ok := AsCustomerAccountAlreadyExistsError(err); ok { accountIDs[accountType] = existingErr.AccountID; continue }`)
**CustomerLedgerHook embeds NoopCustomerLedgerHook** — customerLedgerHook only overrides PostCreate; all other ServiceHook methods are satisfied by the embedded NoopCustomerLedgerHook. (`type customerLedgerHook struct {
	NoopCustomerLedgerHook
	config CustomerLedgerHookConfig
}`)
**namespaceHandler delegates to businessAccountProvisioner interface** — NewNamespaceHandler accepts a businessAccountProvisioner interface (not *AccountResolver directly) so the namespace.Handler can be wired without depending on the concrete type. (`func NewNamespaceHandler(provisioner businessAccountProvisioner) namespace.Handler`)
**ValidationIssue error for conflict** — CustomerAccountAlreadyExistsError implements validationErrors interface and maps to ErrCustomerAccountConflict (HTTP 409) via commonhttp.WithHTTPStatusCodeAttribute. (`var ErrCustomerAccountConflict = models.NewValidationIssue(ErrCodeCustomerAccountConflict, "...", commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `account.go` | AccountResolver — implements ledger.AccountResolver (CreateCustomerAccounts, GetCustomerAccounts, EnsureBusinessAccounts, GetBusinessAccounts); manages provisioning locks and idempotent account creation. | Locker is optional (nil check before LockForTX); always use transaction.Run even when Locker is nil so the tx context propagates. |
| `customeraccount.go` | CustomerAccountRepo interface definition (TxCreator + CreateCustomerAccount + GetCustomerAccountIDs) and CreateCustomerAccountInput type. | Adapter must return CustomerAccountAlreadyExistsError on unique-constraint violations, not a raw DB error. |
| `errors.go` | CustomerAccountAlreadyExistsError type, ErrCustomerAccountConflict validation issue, AsCustomerAccountAlreadyExistsError unwrap helper. | AsCustomerAccountAlreadyExistsError must be used in CreateCustomerAccounts to handle concurrent inserts gracefully. |
| `hooks.go` | CustomerLedgerHook type alias, customerLedgerHook implementation with PostCreate calling CreateCustomerAccounts, NewCustomerLedgerHook constructor. | Hook must be registered via customerService.RegisterHooks only when credits.enabled=true. |
| `namespace.go` | namespaceHandler — implements namespace.Handler; CreateNamespace calls EnsureBusinessAccounts, DeleteNamespace is a no-op. | Register this handler before initNamespace in cmd/server/main.go or the default namespace will not have business accounts. |

## Anti-Patterns

- Registering CustomerLedgerHook or namespaceHandler when credits.enabled=false — these are real write paths; noop variants must be used instead.
- Calling GetCustomerAccounts without a preceding CreateCustomerAccounts or EnsureSubAccount — returns ErrCustomerAccountMissing for un-provisioned customers.
- Constructing AccountResolver without a Locker in production — concurrent provisioning will create duplicate accounts without the advisory lock.
- Adding business logic (balance calculation, route resolution) to the adapter (resolvers/adapter) — belongs in AccountResolver service layer.
- Using context.Background() inside hooks or namespace handler — OTel traces and transaction context will be severed.

## Decisions

- **Provisioning lock uses a 5-second context.WithTimeout to bound the wait, converted to lockr.ErrLockTimeout on deadline exceeded.** — Prevents multi-service startup races from blocking indefinitely; fail-fast is preferable to indefinite serialization.
- **CustomerAccountRepo is a minimal interface (just two methods + TxCreator) rather than a full Ent client, keeping the adapter testable in isolation.** — Allows testing AccountResolver with a mock repo without standing up a Postgres database.

## Example: Wire the CustomerLedgerHook and namespace handler when credits are enabled

```
import (
	"github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	"github.com/openmeterio/openmeter/openmeter/customer"
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
