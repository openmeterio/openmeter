# chargeadapter

<!-- archie:ai-start -->

> Adapter layer that implements the billing/charges Handler interfaces (creditpurchase.Handler, flatfee.Handler, usagebased.Handler) by mapping charge lifecycle events onto ledger transaction templates. It is the bridge between the billing charge state machines and the double-entry ledger; it acknowledges/books value but does NOT recognize revenue.

## Patterns

**Implement charge Handler interfaces** — Each file defines an unexported handler struct with a compile-time interface assertion and a NewXHandler constructor. Methods follow the OnX naming of the upstream Handler interface. (`var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil); func NewCreditPurchaseHandler(...) (chargecreditpurchase.Handler, error)`)
**Resolve templates then CommitGroup** — Build transactions.TransactionTemplate values, call transactions.ResolveTransactions(ctx, deps, ResolutionScope{...}, templates...) to get inputs, then ledger.CommitGroup(ctx, transactions.GroupInputs(namespace, annotations, inputs...)). Return ledgertransaction.GroupReference{TransactionGroupID: group.ID().ID}. (`inputs, _ := transactions.ResolveTransactions(ctx, h.deps, scope, transactions.TransferCustomerReceivableToAccruedTemplate{...}); group, _ := h.ledger.CommitGroup(ctx, transactions.GroupInputs(ns, annotations, inputs...))`)
**Stamp charge annotations on every input and the group** — Compute annotations via chargeAnnotationsForFlatFeeCharge / ...UsageBasedCharge / ...CreditPurchaseCharge (which delegate to ledger.ChargeTransactionAnnotations), then wrap each non-nil input with transactions.WithAnnotations and pass the same annotations to GroupInputs. (`for i, txInput := range inputs { if txInput != nil { inputs[i] = transactions.WithAnnotations(txInput, annotations) } }`)
**Validate input and short-circuit zero amounts** — Every On* method calls input.Validate() first and returns an empty GroupReference (no ledger write) when the relevant amount IsZero()/!IsPositive(). (`if err := input.Validate(); err != nil { return ledgertransaction.GroupReference{}, err }; if amount.IsZero() { return ledgertransaction.GroupReference{}, nil }`)
**Gate on settlement mode** — Credit-vs-invoice flows assert allowed productcatalog.SettlementMode via validateSettlementMode(actual, allowed...) before booking; wrong mode is an error, not a no-op. (`if err := validateSettlementMode(charge.Intent.SettlementMode, productcatalog.CreditThenInvoiceSettlementMode); err != nil { return ..., fmt.Errorf("invoice usage accrued: %w", err) }`)
**Delegate accrual collection to collector.Service** — OnAllocateCredits / OnCreditsOnlyUsageAccrued and their corrections delegate to h.collector.CollectToAccrued / CorrectCollectedAccrued; do not re-implement FBO source selection here. (`realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{...})`)
**Wrap multi-step flows in transaction.Run** — Credit purchase issuance spanning attribution + breakage planning + commit is wrapped in transaction.Run(ctx, h.transactionManager, ...) so ledger group + breakage records persist atomically. (`return transaction.Run(ctx, h.transactionManager, func(ctx context.Context) (ledgertransaction.GroupReference, error) { return h.issueCreditPurchaseGroup(ctx, charge) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `creditpurchase.go` | creditPurchaseHandler: promotional/initiated/payment-authorized/settled events; advance attribution + breakage planning via breakage.Service.PlanIssuance/PersistCommittedRecords. | Promotional grants emit Authorize+Settle wash templates; deferred (External/Invoice) settlement is left for later events; unsupported settlement Type() returns an error. Breakage records must be persisted in the same tx as the group. |
| `flatfee.go` | flatFeeHandler: FBO credit allocation, invoice-usage accrual (TransferCustomerReceivableToAccruedTemplate), correction, payment authorize/settle. | OnAllocateCredits/OnInvoiceUsageAccrued acknowledge usage, NOT revenue. invoiceCostBasis is fixed at 1. OnPaymentUncollectible is intentionally unimplemented and returns an error. |
| `usagebased.go` | usageBasedHandler: invoice-usage accrual, credits-only accrual + correction, payment authorize/settle; receivable replenishment derived from Run.InvoiceUsage.Totals. | OnCreditsOnlyUsageAccrued uses clock.Now() for SourceBalanceAsOf while flat-fee uses Intent.InvoiceAt — booked-at vs source-as-of are deliberately distinct. |
| `annotations.go` | chargeAnnotationsForX helpers feeding ledger.ChargeTransactionAnnotations with subscription/feature references from charge.Intent. | FeatureID extraction differs per charge type (flatfee uses State.FeatureID, usagebased uses lo.EmptyableToPtr(State.FeatureID)); keep subscription nil-handling intact. |
| `helpers.go` | settledBalanceForSubAccount and taxCodeIDFromIntent/taxBehaviorFromIntent mapping productcatalog.TaxCodeConfig to ledger fields. | taxBehaviorFromIntent returns nil unless both TaxCodeID and Behavior are set. |

## Anti-Patterns

- Booking ledger entries without stamping charge annotations on both each input and the group — breaks downstream charge/transaction correlation.
- Treating accrual/acknowledgement transactions as revenue recognition — recognition lives in openmeter/ledger/recognizer.
- Calling CommitGroup without first validating the input and short-circuiting zero amounts, producing empty/noise ledger groups.
- Re-implementing FBO source selection or breakage release ordering here instead of delegating to collector.Service / breakage.Service.
- Skipping validateSettlementMode so a credit-only charge books invoice-mode receivables (or vice versa).

## Decisions

- **Charge handlers are thin mappers from charge events to transaction templates** — Keeps double-entry correctness centralized in the transactions/ledger/collector packages; the adapter only owns the charge-event-to-template mapping and annotation stamping.
- **Acknowledgement (accrual) is separated from revenue recognition** — Booking usage against accrued must not prematurely recognize earnings; recognition is a distinct deferred flow keyed off lineage segment state.

## Example: Mapping an invoice-usage event to a ledger transfer with annotations

```
annotations := chargeAnnotationsForFlatFeeCharge(input.Charge)
inputs, err := transactions.ResolveTransactions(ctx, h.deps, transactions.ResolutionScope{CustomerID: customerID, Namespace: input.Charge.Namespace}, transactions.TransferCustomerReceivableToAccruedTemplate{At: input.BookedAt, Amount: amount, Currency: input.Charge.Intent.Currency, CostBasis: invoiceCostBasis})
for i, txInput := range inputs { if txInput != nil { inputs[i] = transactions.WithAnnotations(txInput, annotations) } }
group, err := h.ledger.CommitGroup(ctx, transactions.GroupInputs(input.Charge.Namespace, annotations, inputs...))
return ledgertransaction.GroupReference{TransactionGroupID: group.ID().ID}, nil
```

<!-- archie:ai-end -->
