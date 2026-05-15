# ledger

<!-- archie:ai-start -->

> Double-entry ledger for customer financial balances (FBO, Receivable, Accrued) and business accounts (Wash, Earnings, Brokerage). Transaction inputs are constructed exclusively via transactions.ResolveTransactions with typed template structs; noop/ provides zero-value implementations when credits.enabled=false.

## Patterns

**TransactionInput constructed only via transaction templates** — All ledger postings are created by calling transactions.ResolveTransactions with typed template structs. Raw EntryInput/TransactionInput construction outside the transactions/ package bypasses sub-account routing and routing rule validation. (`transactions/resolve.go: ResolveTransactions iterates templates, calls resolve(), then calls ledger.CommitGroup`)
**credits.enabled=false routes to noop/ implementations** — app/common wires ledger.Ledger, ledger.AccountResolver, and ledgeraccount.Service to noop/ structs when credits.enabled=false. Every new ledger write path must have a corresponding noop path — enforce via compile-time interface assertions in noop/noop.go. (`noop/noop.go: var _ ledger.Ledger = Ledger{}; var _ ledger.AccountResolver = AccountResolver{}`)
**Route → canonical routing key via BuildRoutingKeyV1** — SubAccount keys are derived from Route via ledger.BuildRoutingKeyV1(Route{...}). Never manually construct routing key strings — the canonical format normalizes decimals and nil fields. (`routing.go: BuildRoutingKeyV1 produces 'currency:USD|tax_code:null|features:null|cost_basis:0.7|...'`)
**BookTransaction always inside transaction.Run with ValidateTransactionInputWith** — historical/ledger.go calls ValidateTransactionInputWith(ctx, tx, routingValidator) then repo.BookTransaction inside transaction.Run. Calling BookTransaction without validation or outside transaction.Run loses the account lock and routing invariants. (`historical/ledger.go: transaction.Run(..., func(...) { ValidateTransactionInputWith; repo.BookTransaction })`)
**ChargeAnnotations on every CommitGroup call** — chargeadapter handlers annotate every ledger.CommitGroup call with ChargeAnnotations (charge ID, type, direction) via helpers. Omitting annotations breaks traceability and correction logic. (`chargeadapter/annotations.go: ChargeAnnotations struct; helpers.go sets them on every CommitGroup input`)
**Provisioning lock before EnsureCustomerAccounts** — resolvers/account.go acquires a lockr.Locker advisory lock before EnsureCustomerAccounts to prevent duplicate account creation under concurrency. Lock uses a 5-second timeout. (`resolvers/account.go: locker.LockForTX(ctx, lockKey) before CreateCustomerAccounts`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/ledger/account.go` | AccountType constants (FBO, Receivable, Accrued, Wash, Earnings, Brokerage), CustomerAccounts/BusinessAccounts structs, AccountResolver interface. | AccountResolver.GetCustomerAccounts panics for unprovisioned customers — always call EnsureCustomerAccounts first or check via resolvers. |
| `openmeter/ledger/accounts.go` | Typed account interfaces per account type (CustomerFBOAccount, CustomerReceivableAccount, CustomerAccruedAccount, BusinessAccount) with GetSubAccountForRoute signatures. | CustomerFBORouteParams requires CreditPriority (non-pointer int). CustomerReceivableRouteParams requires TransactionAuthorizationStatus. Missing required fields fail Validate(). |
| `openmeter/ledger/routing.go` | Route struct, BuildRoutingKeyV1, RoutingKey — the canonical hash key for sub-account routing. | CostBasis decimal is canonicalized by trimming trailing zeros. Always construct Route via typed field assignment, never string parsing. |
| `openmeter/ledger/validations.go` | ValidateInvariance (debit-credit balance check), ValidateEntryInput, ValidateTransactionInputWith — called before every BookTransaction. | ValidateInvariance checks that all entry amounts sum to zero. Use alpacadecimal, not float64. |
| `openmeter/ledger/noop/noop.go` | Zero-value noop implementations of all ledger interfaces used when credits.enabled=false. | Every new ledger interface method must be added to the noop with a zero-value return. Missing methods break compile when credits.enabled=false. |

## Anti-Patterns

- Constructing EntryInput/TransactionInput outside the transactions/ package — bypasses sub-account routing and routing rule validation.
- Registering CustomerLedgerHook or namespace ledger handler when credits.enabled=false — these are real DB write paths; noop variants must be used instead.
- Calling repo.BookTransaction outside a transaction.Run block — loses the account lock and validation atomicity.
- Adding a new transaction template without registering it in transactionTemplateByName in correction.go — the template becomes uncorrectable.
- Using context.Background() in any ledger method — breaks OTel tracing and transaction context propagation.

## Decisions

- **Transaction templates (transactions/) are the only permitted construction site for EntryInput/TransactionInput.** — Centralizing construction enforces routing rule compliance and makes corrections (reversals) traceable via annotation-stored template names.
- **noop/ package provides compile-time-checked no-ops for all ledger interfaces.** — Allows billing and customer domain to remain structurally identical regardless of credits.enabled flag — no nil checks needed in business logic.
- **Tri-tuple cursor (bookedAt, createdAt, ID) for transaction pagination instead of auto-increment offset.** — Ledger transactions are append-only but may be inserted out of clock order; a composite cursor with ID tie-breaking provides stable pagination.

## Example: Posting a double-entry transaction via templates (the only permitted pattern)

```
import (
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// Inside a charge handler:
template := transactions.CustomerTransactionTemplate{
	CustomerAccounts: customerAccounts,
	BusinessAccounts: bizAccounts,
	Currency:         currencyx.Code("USD"),
	Amount:           amount,
}

txInputs, err := transactions.ResolveTransactions(ctx, []transactions.TransactionTemplate{template})
if err != nil { return err }
// ...
```

<!-- archie:ai-end -->
