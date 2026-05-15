# transactions

<!-- archie:ai-start -->

> Implements the transaction template layer for the double-entry ledger: defines reusable named templates (CustomerTransactionTemplate, OrgTransactionTemplate) that resolve abstract intents into concrete ledger.TransactionInput values, and provides a correction path for reversing posted templates. This is the only layer that constructs EntryInput/TransactionInput — callers never build raw ledger entries by hand.

## Patterns

**Private typeGuard() seals the template interface** — Every template struct must implement typeGuard() guard (unexported return type) so only this package can define templates. Templates added outside this package cannot satisfy CustomerTransactionTemplate or OrgTransactionTemplate. (`func (t IssueCustomerReceivableTemplate) typeGuard() guard { return true }`)
**Compile-time interface assertion on every template** — Each template must have a var _ CustomerTransactionTemplate = (TemplateType{}) line. Missing this means a template silently fails to dispatch. (`var _ CustomerTransactionTemplate = (IssueCustomerReceivableTemplate{})`)
**Validate() called by ResolveTransactions before resolve()** — ResolveTransactions calls template.Validate() for every template before calling resolve(). Templates must validate all required fields using ledger.ValidateTransactionAmount, ledger.ValidateCurrency, ledger.ValidateCostBasis, ledger.ValidateCreditPriority helpers. (`if err := ledger.ValidateTransactionAmount(t.Amount); err != nil { return fmt.Errorf("amount: %w", err) }`)
**nil from resolve() signals no-op (skip silently)** — If a template determines there is nothing to post (e.g. no FBO balance), resolve() must return (nil, nil). ResolveTransactions skips nil inputs and does NOT append them to the output slice. (`if len(collections) == 0 { return nil, nil }`)
**annotateTemplateTransaction wraps every produced TransactionInput** — ResolveTransactions wraps every non-nil input with annotateTemplateTransaction(..., ledger.TransactionDirectionForward). CorrectTransaction wraps corrections with TransactionDirectionCorrection. (`annotateTemplateTransaction(tx, template, ledger.TransactionDirectionForward)`)
**Register new templates in transactionTemplateByName in correction.go** — CorrectTransaction dispatches via annotation-stored template name string. Every new template must add a case to transactionTemplateByName(). If correction is unsupported, return templateCorrectionNotImplemented(templateName(t)). (`case templateName(IssueCustomerReceivableTemplate{}): return IssueCustomerReceivableTemplate{}, nil`)
**collectFromPrioritizedCustomerFBO for FBO balance collection** — Templates that drain FBO balances must use collectFromPrioritizedCustomerFBO which sorts sub-accounts by CreditPriority ascending, then SubAccountID for determinism. Never build custom FBO iteration logic inline. (`collections, err := collectFromPrioritizedCustomerFBO(ctx, customerID, t.Currency, t.Amount, resolvers)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `template.go` | Defines the three template interfaces (TransactionTemplate, CustomerTransactionTemplate, OrgTransactionTemplate) and the private guard type. | typeGuard() uses a private return type — do not try to implement TransactionTemplate outside this package. |
| `resolve.go` | ResolveTransactions is the main entry point — validates scope, calls Validate(), dispatches to resolve(), and auto-annotates outputs. | ResolverDependencies requires both AccountService (ledger.AccountResolver) and SubAccountService (ledgeraccount.Service); nil SubAccountService causes runtime errors. |
| `correction.go` | CorrectTransaction reconstructs the original template from annotations and calls correct(). transactionTemplateByName is the dispatch table — update when adding new templates. | Correcting a correction (TransactionDirectionCorrection) is rejected. Missing case in transactionTemplateByName produces an error at correction time only. |
| `input.go` | EntryInput, TransactionInput, TransactionGroupInput: concrete structs satisfying ledger input interfaces. GroupInputs() wraps multiple transactions into a group. | lo.Map is required to coerce []*EntryInput to []ledger.EntryInput — EntryInput fields are unexported. |
| `collection.go` | collectFromPrioritizedCustomerFBO / collectFromAttributableCustomerAccrued: balance-collection helpers with deterministic sub-account sorting. | collectFromAttributableCustomerAccrued skips sub-accounts with nil CostBasis — intentional for RecognizeEarnings which only handles attributed buckets. |
| `accrual.go` | Four accrual templates. TransferCustomerFBOToAccruedTemplate drains FBO in priority order and preserves cost-basis routing into accrued. | resolveAccruedSubAccByCostBasis aggregates by costBasisKey() — the key is the decimal string or 'null'. New accrued sub-accounts need GetSubAccountForRoute. |
| `customer.go` | Eight customer-scoped templates covering the full FBO/receivable/coverage lifecycle. | FundCustomerReceivableTemplate and SettleCustomerReceivablePaymentTemplate return templateCorrectionNotImplemented — not all templates are correctable. |

## Anti-Patterns

- Constructing EntryInput/TransactionInput outside this package — all ledger posting construction belongs here; callers pass templates to ResolveTransactions.
- Adding a new template without registering it in transactionTemplateByName in correction.go — the template will be unresolvable at correction time.
- Using context.Background() in resolve() or correct() — always propagate the ctx passed from ResolveTransactions/CorrectTransaction.
- Bypassing collectFromPrioritizedCustomerFBO for custom FBO iteration — priority ordering must remain deterministic and centralized.
- Implementing TransactionTemplate outside this package — the private guard type seals the interface.

## Decisions

- **Templates are value structs with unexported resolve()/correct() methods rather than functions.** — Keeps all template logic colocated, allows dispatch on interface type, and prevents callers from calling resolve() directly without going through the Validate() + annotation pipeline.
- **Correction dispatches via annotation-stored template name string, not a type registry.** — Transactions persisted to DB carry annotations; the correction path must reconstruct the correct template from a stored string, making a string-keyed switch the only viable dispatch mechanism.
- **nil return from resolve() means skip (no-op), not an error.** — Some templates are conditional (e.g. no FBO balance available). Returning nil lets ResolveTransactions accumulate only real postings without forcing callers to pre-check balances.

## Example: Resolve and commit templates for a customer receivable issuance

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func issueReceivable(ctx context.Context, deps transactions.ResolverDependencies, customerID ledger.CustomerID, amount alpacadecimal.Decimal) error {
	txInputs, err := transactions.ResolveTransactions(ctx, deps, customerID,
		transactions.IssueCustomerReceivableTemplate{At: time.Now(), Currency: currencyx.Code("USD"), Amount: amount},
	)
	if err != nil { return err }
	_, err = historicalLedger.CommitGroup(ctx, transactions.GroupInputs(customerID.Namespace, txInputs...))
	return err
}
```

<!-- archie:ai-end -->
