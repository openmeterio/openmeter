# transactions

<!-- archie:ai-start -->

> The transaction-template layer of the double-entry ledger: named value-struct templates (CustomerTransactionTemplate, OrgTransactionTemplate) resolve abstract intents into concrete ledger.TransactionInput values, with a correction path for reversing posted templates. This is the ONLY layer that constructs EntryInput/TransactionInput — callers always pass templates to ResolveTransactions.

## Patterns

**Private typeGuard() seals the template interface** — Every template implements typeGuard() guard returning an unexported type so templates cannot be defined outside this package. (`func (t IssueCustomerReceivableTemplate) typeGuard() guard { return true }`)
**Compile-time interface assertion per template** — Each template carries a var _ CustomerTransactionTemplate = (T{}) line; missing it means the template silently fails to dispatch. (`var _ CustomerTransactionTemplate = (TransferCustomerFBOToAccruedTemplate{})`)
**Validate() before resolve()** — ResolveTransactions calls template.Validate() (using ledger.ValidateTransactionAmount/ValidateCurrency/ValidateCostBasis/ValidateCreditPriority) for every template before resolve(). (`if err := ledger.ValidateTransactionAmount(t.Amount); err != nil { return fmt.Errorf("amount: %w", err) }`)
**nil from resolve() means skip (no-op)** — When a template has nothing to post, resolve() returns (nil, nil); ResolveTransactions skips nil inputs rather than appending or erroring. (`if len(t.Sources) == 0 { return nil, nil }`)
**Auto-annotate produced inputs with direction** — ResolveTransactions wraps each non-nil input via annotateTemplateTransaction(..., TransactionDirectionForward); CorrectTransaction uses TransactionDirectionCorrection. (`annotateTemplateTransaction(tx, template, ledger.TransactionDirectionForward)`)
**Register templates in transactionTemplateByName** — CorrectTransaction reconstructs templates from the annotation-stored template name; every new template adds a case in correction.go (or returns templateCorrectionNotImplemented). (`case templateName(IssueCustomerReceivableTemplate{}): return IssueCustomerReceivableTemplate{}, nil`)
**Deterministic FBO collection helper** — Templates draining FBO balances use collectFromPrioritizedCustomerFBO (sorts by CreditPriority asc, then SubAccountID) — never inline custom FBO iteration. (`collections, err := collectFromPrioritizedCustomerFBO(ctx, customerID, t.Currency, t.Amount, resolvers)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `template.go` | TransactionTemplate / CustomerTransactionTemplate / OrgTransactionTemplate interfaces and the private guard type. | typeGuard() returns a private type — templates cannot be implemented outside this package. |
| `resolve.go` | ResolveTransactions entry point: validates scope, calls Validate(), dispatches resolve(), auto-annotates outputs. | ResolverDependencies needs both AccountService (ledger.AccountResolver) and SubAccountService (ledgeraccount.Service). |
| `correction.go` | CorrectTransaction reconstructs the original template from annotations; transactionTemplateByName is the dispatch table. | Correcting a correction is rejected; a missing dispatch case only surfaces at correction time. |
| `input.go` | EntryInput/TransactionInput/TransactionGroupInput concrete structs + GroupInputs() wrapper. | EntryInput fields are unexported; use lo.Map to coerce []*EntryInput to []ledger.EntryInput. |
| `collection.go` | collectFromPrioritizedCustomerFBO / collectFromAttributableCustomerAccrued balance-collection helpers. | collectFromAttributableCustomerAccrued skips nil-CostBasis sub-accounts intentionally. |
| `accrual.go` | Accrual templates (e.g. TransferCustomerFBOToAccruedTemplate) preserving cost-basis routing into accrued. | resolveAccruedSubAccByRoutePairingKey aggregates by costBasisKey() (decimal string or 'null'). |
| `customer.go` | Customer-scoped templates for the FBO/receivable/coverage lifecycle. | Some templates return templateCorrectionNotImplemented — not every template is correctable. |

## Anti-Patterns

- Constructing EntryInput/TransactionInput outside this package — all posting construction belongs here.
- Adding a template without a transactionTemplateByName case — it becomes unresolvable at correction time.
- Using context.Background() in resolve()/correct() — propagate the ctx from ResolveTransactions/CorrectTransaction.
- Bypassing collectFromPrioritizedCustomerFBO with custom FBO iteration — priority ordering must stay deterministic.
- Implementing TransactionTemplate outside this package — the private guard seals the interface.

## Decisions

- **Templates are value structs with unexported resolve()/correct() methods, not functions.** — Colocates template logic, enables interface-type dispatch, and forces callers through the Validate()+annotation pipeline.
- **Correction dispatches via the annotation-stored template name string, not a type registry.** — Persisted transactions carry annotations; the correction path must rebuild the template from a stored string.
- **nil from resolve() means skip, not error.** — Conditional templates (e.g. no FBO balance) let ResolveTransactions accumulate only real postings without pre-checks.

## Example: Resolve and commit a customer receivable issuance

```
txInputs, err := transactions.ResolveTransactions(ctx, deps, customerID,
  transactions.IssueCustomerReceivableTemplate{At: time.Now(), Currency: currencyx.Code("USD"), Amount: amount},
)
if err != nil { return err }
_, err = historicalLedger.CommitGroup(ctx, transactions.GroupInputs(customerID.Namespace, txInputs...))
return err
```

<!-- archie:ai-end -->
