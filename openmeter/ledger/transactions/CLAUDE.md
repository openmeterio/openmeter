# transactions

<!-- archie:ai-start -->

> Implements the transaction template layer for the double-entry ledger: defines reusable, named templates (CustomerTransactionTemplate, OrgTransactionTemplate) that resolve abstract intents (issue receivable, fund FBO, accrue, recognize earnings, convert currency) into concrete ledger.TransactionInput values, and provides a correction path for reversing posted templates. This is the only layer that constructs EntryInput/TransactionInput structs — callers never build raw entries by hand.

## Patterns

**Template interface seal via private typeGuard()** — Every template struct must implement `typeGuard() guard` (unexported return type) so only this package can define templates. The two dispatch interfaces (CustomerTransactionTemplate, OrgTransactionTemplate) embed TransactionTemplate. New templates must live in this package. (`func (t IssueCustomerReceivableTemplate) typeGuard() guard { return true }`)
**Compile-time interface assertion on every template** — Each template must have `var _ CustomerTransactionTemplate = (TemplateType{})` or `var _ OrgTransactionTemplate = ...` at the top level. Missing this means a template silently fails to dispatch. (`var _ CustomerTransactionTemplate = (IssueCustomerReceivableTemplate{})`)
**Validate() called by ResolveTransactions before resolve()** — ResolveTransactions calls template.Validate() for every template before calling resolve(). Templates must validate all required fields (At, Amount, Currency, CostBasis) using ledger.ValidateTransactionAmount, ledger.ValidateCurrency, ledger.ValidateCostBasis, ledger.ValidateCreditPriority helpers. (`if err := ledger.ValidateTransactionAmount(t.Amount); err != nil { return fmt.Errorf("amount: %w", err) }`)
**resolve() returns nil to signal no-op (skip silently)** — If a template determines there is nothing to post (e.g. no FBO balance), resolve() must return (nil, nil). ResolveTransactions skips nil inputs and does NOT append them to the output slice. (`if len(collections) == 0 { return nil, nil }`)
**Every produced TransactionInput is annotated with template name + direction** — ResolveTransactions wraps every non-nil input with annotateTemplateTransaction(..., ledger.TransactionDirectionForward). CorrectTransaction wraps corrections with TransactionDirectionCorrection. The annotation is stored as ledger.AnnotationTransactionTemplateName and AnnotationTransactionDirection. (`annotateTemplateTransaction(tx, template, ledger.TransactionDirectionForward)`)
**correct() must be registered in transactionTemplateByName switch** — CorrectTransaction looks up the original template by name string from annotations. When adding a new template, add a case to transactionTemplateByName() in correction.go. If correction is not supported, return templateCorrectionNotImplemented(templateName(t)). (`case templateName(IssueCustomerReceivableTemplate{}): return IssueCustomerReceivableTemplate{}, nil`)
**FBO collection uses priority ordering from collection.go helpers** — TransferCustomerFBOToAccruedTemplate and CoverCustomerReceivableTemplate must use collectFromPrioritizedCustomerFBO which sorts by CreditPriority ascending, then by SubAccountID for determinism. Never build custom FBO iteration logic inline. (`collections, err := collectFromPrioritizedCustomerFBO(ctx, customerID, t.Currency, t.Amount, resolvers)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `template.go` | Defines the three template interfaces (TransactionTemplate, CustomerTransactionTemplate, OrgTransactionTemplate) and the private guard type. Start here when understanding what a template must implement. | typeGuard() uses a private return type — do not try to implement TransactionTemplate outside this package. |
| `resolve.go` | ResolveTransactions is the main entry point for callers. It validates scope, calls template.Validate(), dispatches to resolve(), and auto-annotates outputs. ResolutionScope validates namespace consistency between CustomerID and explicit Namespace. | ResolverDependencies requires both AccountService (ledger.AccountResolver) and SubAccountService (ledgeraccount.Service); nil SubAccountService causes a runtime error in collectFromPrioritizedCustomerFBO. |
| `correction.go` | CorrectTransaction reconstructs the original template from annotations and calls its correct() method. transactionTemplateByName is the dispatch table — must be updated when adding new templates. | Correcting a correction (TransactionDirectionCorrection) is rejected. Missing case in transactionTemplateByName returns an error at correction time, not at template creation time. |
| `input.go` | EntryInput, TransactionInput, TransactionGroupInput: the concrete structs satisfying ledger input interfaces. GroupInputs() wraps multiple transactions into a group. WithAnnotations() merges annotation maps without mutation. | lo.Map is used to coerce []*EntryInput to []ledger.EntryInput — required because EntryInput fields are unexported. Do not add exported field accessors to EntryInput without updating all consumers. |
| `collection.go` | collectFromPrioritizedCustomerFBO / collectFromAttributableCustomerAccrued: the two balance-collection helpers. Both sort sub-accounts deterministically before draining. settledBalanceForSubAccount uses SubAccount.GetBalance().Settled() — only settled (non-pending) balances are used. | collectFromAttributableCustomerAccrued skips sub-accounts with nil CostBasis — that is intentional for RecognizeEarnings which only handles attributed buckets. |
| `accrual.go` | Four accrual templates. TransferCustomerFBOToAccruedTemplate drains FBO in priority order and preserves cost-basis routing into accrued. correct() reverses LIFO over the original FBO entries. | resolveAccruedSubAccByCostBasis aggregates by costBasisKey() — the key is the decimal string or 'null'. Any accrued sub-account with a new cost basis must be resolved via AccruedAccount.GetSubAccountForRoute. |
| `customer.go` | Eight customer-scoped templates covering the full FBO/receivable/coverage lifecycle. All resolve() implementations call resolvers.AccountService.GetCustomerAccounts then call GetSubAccountForRoute on the appropriate account object. | FundCustomerReceivableTemplate and SettleCustomerReceivablePaymentTemplate have correct() returning templateCorrectionNotImplemented — do not assume all templates are correctable. |
| `testenv_test.go` | Internal test environment wrapping ledgertestutils.IntegrationEnv. resolveAndCommit() is the primary test helper — it resolves templates then commits via HistoricalLedger.CommitGroup. Tests are in the same package (transactions) so they access unexported resolve() directly. | Uses t.Context() throughout — never context.Background(). fundPriority / fundPriorityWithCostBasis commit three templates as setup; do not replicate this inline in new tests. |

## Anti-Patterns

- Constructing EntryInput/TransactionInput outside this package — all ledger posting construction belongs here; callers pass templates to ResolveTransactions.
- Adding a new template without registering it in transactionTemplateByName in correction.go — the template will be unresolvable at correction time.
- Using context.Background() in resolve() or correct() — always propagate the ctx passed from ResolveTransactions/CorrectTransaction.
- Bypassing collectFromPrioritizedCustomerFBO for custom FBO iteration — priority ordering must remain deterministic and centralized.
- Implementing TransactionTemplate outside this package — the private guard type seals the interface; external templates must be added here.

## Decisions

- **Templates are value structs with unexported resolve()/correct() methods rather than functions** — Keeps all template logic colocated, allows dispatch on interface type in ResolveTransactions/CorrectTransaction, and prevents callers from calling resolve() directly without going through the Validate() + annotation pipeline.
- **Correction dispatches via annotation-stored template name string, not a type registry** — Transactions persisted to the DB carry annotations; the correction path must reconstruct the correct template from a stored string, making a string-keyed switch the only viable dispatch mechanism for corrections arriving long after the original posting.
- **nil return from resolve() means skip (no-op), not an error** — Some templates are conditional (e.g. no FBO balance available). Returning nil lets ResolveTransactions accumulate only real postings without forcing callers to pre-check balances, keeping the template API ergonomic.

## Example: Add a new customer-scoped template that moves balance from FBO to a new account type

```
package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type MyNewTemplate struct {
// ...
```

<!-- archie:ai-end -->
