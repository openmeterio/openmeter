# transactions

<!-- archie:ai-start -->

> The transaction-template intent layer: declarative templates (one per business movement, e.g. IssueCustomerReceivable, TransferCustomerFBOToAccrued, PlanCustomerFBOBreakage) that resolve into ledger.TransactionInput by looking up the right sub-accounts, and reverse into correction transactions. This package never commits — it produces inputs that the historical ledger commits.

## Patterns

**Template interface trio** — Every template implements TransactionTemplate (typeGuard()/code()/Validate()) plus either CustomerTransactionTemplate or OrgTransactionTemplate (resolve + correct). The private typeGuard() guard prevents foreign types satisfying the interface. (`var _ CustomerTransactionTemplate = (TransferCustomerFBOToAccruedTemplate{})`)
**resolve builds EntryInputs via routed sub-accounts** — resolve() fetches customer/business accounts from ResolverDependencies, calls GetSubAccountForRoute with a typed RouteParams, and returns a *TransactionInput of balanced *EntryInput pairs (neg/pos). (`fbo.Address(), amount: t.Amount.Neg() paired with accrued.Address(), amount: t.Amount`)
**Validate-before-resolve in the dispatcher** — ResolveTransactions validates scope, then per template calls template.Validate(), type-switches Customer vs Org, validates the matching scope half, resolves, and annotates with template code + direction. (`switch typ := any(template).(type) { case CustomerTransactionTemplate: ... case OrgTransactionTemplate: ... default: ErrResolutionTemplateUnknown }`)
**Code-based template registry** — Each template has a stable TransactionTemplateCode (codes.go) and is registered in transactionTemplatesByCode and transactionTemplatesByLegacyName so persisted annotations can be reversed into a template for correction. (`TemplateCodeTransferCustomerFBOToAccrued TransactionTemplateCode = "customer.fbo.collect"`)
**Correction reverses persisted templates** — CorrectTransaction reads the template code/direction from the original transaction's annotations, rejects correcting a correction, then calls template.correct(scope) and re-annotates outputs as direction=correction. (`direction, _ := ledger.TransactionDirectionFromAnnotations(scope.OriginalTransaction.Annotations())`)
**Private input implementations** — EntryInput/TransactionInput/TransactionGroupInput are unexported-field structs implementing the ledger.*Input interfaces; helpers GroupInputs/WithAnnotations/AsGroupInput compose them. (`var _ ledger.TransactionInput = (*TransactionInput)(nil)`)
**Deterministic correction ordering** — Correction legs are allocated against original entries sorted by a stable key (collection source order, then FBO credit priority, then sub-account id, then identity key) so corrections unwind in reverse collection order. (`slices.SortStableFunc(negativeFBOEntries, compareFBOAccrualCorrectionSourceEntries)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `template.go` | TransactionTemplate / CustomerTransactionTemplate / OrgTransactionTemplate interfaces with the private guard. | typeGuard() must return a value of the unexported guard type — that is what stops external packages from implementing the interface. |
| `resolve.go` | ResolverDependencies, ResolutionScope (+ its Validate variants), and the ResolveTransactions dispatcher. | ResolutionScope validates namespace match; customer templates require CustomerID.Namespace+ID, org templates require Namespace. resolve() returning a nil TransactionInput is dropped (appendResolvedTemplateTransaction). |
| `codes.go` | TransactionTemplateCode constants + by-code and by-legacy-name registries + annotateTemplateTransaction. | Adding a template means adding its code AND both registry entries; TemplateCode() panics on an empty code, so always set code(). |
| `correction.go` | CorrectionInput/CorrectionScope + CorrectTransaction: annotation-driven template lookup and reversal. | Reads template from new code annotation or legacy name annotation; correcting a correction is rejected; templates without correct support return templateCorrectionNotImplemented. |
| `input.go` | EntryInput/TransactionInput/TransactionGroupInput impls and GroupInputs/WithAnnotations/AsGroupInput helpers. | Fields are unexported; construct only within this package. WithAnnotations wraps in annotatedTransactionInput, merging maps (template annotations win). |
| `accrual.go` | FBO->Accrued / Receivable->Accrued / advance->Accrued transfer templates and their cost-basis-preserving accrual + correction logic. | Accrual preserves source cost basis via routePairingKey(currency, costBasis); accrued sub-account gets tax dimensions, FBO sources do not carry tax. |
| `legacy.go` | Archived templates (legacyFund/legacySettle...) kept only to interpret old persisted annotations. | Their code() returns "" so they are by-legacy-name only; do not use them in new flows — use the current Authorize/Settle templates. |
| `priority.go` | resolveCustomerFBOCreditPriority falls back to ledger.DefaultCustomerFBOPriority. | A nil CreditPriority means default priority, not 'no priority' — this drives FBO collection order. |

## Anti-Patterns

- Committing ledger transactions from inside a template — templates only resolve into TransactionInputs; the historical ledger commits.
- Adding a template without registering its code in both transactionTemplatesByCode and (if it has a legacy name) transactionTemplatesByLegacyName.
- Implementing a template interface from another package by faking typeGuard()/guard — the guard type is intentionally unexported.
- Constructing EntryInput/TransactionInput from outside the package, bypassing the balanced neg/pos entry convention.
- Importing transactions/testutils (Any*Input fixtures) from production code instead of using the real templates.

## Decisions

- **Business intent is expressed as one template per movement that returns inputs, separate from the ledger that commits them.** — Templates own routing/cost-basis/tax decisions and reversibility, while the historical ledger owns atomic posting and locking — keeping intent and persistence decoupled.
- **Templates are identified by a stable TransactionTemplateCode persisted in annotations, with a legacy-name fallback registry.** — Corrections must reconstruct the original template from a committed transaction; codes give a stable contract while legacy names keep old data correctable.
- **Correction allocates legs in a fixed deterministic order matching FBO collection order.** — Reversing in reverse-collection order keeps breakage/cost-basis attribution consistent without storing grant lineage.

## Example: A customer transaction template resolving into a balanced TransactionInput

```
func (t TransferCustomerReceivableToAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil { return nil, err }
	receivable, _ := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{Currency: t.Currency, CostBasis: t.CostBasis, TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen})
	accrued, _ := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{Currency: t.Currency, TaxCode: t.TaxCode, TaxBehavior: t.TaxBehavior, CostBasis: t.CostBasis})
	return &TransactionInput{bookedAt: t.At, entryInputs: []*EntryInput{
		{address: receivable.Address(), amount: t.Amount.Neg()},
		{address: accrued.Address(), amount: t.Amount},
	}}, nil
}
```

<!-- archie:ai-end -->
