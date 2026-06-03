# chargeadapter

<!-- archie:ai-start -->

> Bridges charge lifecycle events (credit purchase, flat fee, usage-based) to double-entry ledger postings by translating domain charge state transitions into typed transaction templates committed via ledger.Ledger.CommitGroup. Each charge type has a private handler implementing the corresponding charge-domain Handler interface, keeping charge packages free of ledger imports.

## Patterns

**Handler-per-charge-type with compile-time assertion** — Each charge type (creditPurchase, flatFee, usageBased) has a private handler struct and a public New*Handler constructor returning the charge-domain Handler interface, guarded by a var _ <interface> = (*handler)(nil) assertion. (`var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil)`)
**Resolve-then-annotate-then-commit pipeline** — Every ledger write follows transactions.ResolveTransactions then per-input transactions.WithAnnotations then h.ledger.CommitGroup(ctx, transactions.GroupInputs(...)). Never call CommitGroup with un-resolved/un-annotated inputs. (`inputs, _ := transactions.ResolveTransactions(ctx, h.deps, scope, template); for i := range inputs { inputs[i] = transactions.WithAnnotations(inputs[i], annotations) }`)
**ChargeAnnotations on every CommitGroup** — Every group gets charge-scoped annotations via chargeAnnotationsForXxxCharge -> chargeTransactionAnnotations -> ledger.ChargeTransactionAnnotations (chargeID, namespace, subscription/phase/item IDs, featureID). (`annotations := chargeAnnotationsForFlatFeeCharge(charge)`)
**Settlement mode guard before any ledger work** — Invoice-side accrual methods call validateSettlementMode with an explicit allowlist of permitted modes; incompatible modes return an error, not a no-op. (`if err := validateSettlementMode(input.Charge.Intent.SettlementMode, productcatalog.InvoiceOnlySettlementMode); err != nil { return ..., err }`)
**Return empty GroupReference on zero-amount** — Handler methods return an empty ledgertransaction.GroupReference{} (not an error, not a zero-value ledger group) when the amount is zero. (`if amount.IsZero() { return ledgertransaction.GroupReference{}, nil }`)
**Delegate FBO collection to collector.Service** — flatFeeHandler and usageBasedHandler delegate FBO->accrued collection to collector.Service.CollectToAccrued / CorrectCollectedAccrued; only payment authorization/settlement call ledger directly. (`realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{...})`)
**clock.Now() for payment timestamps** — Payment-authorized and payment-settled methods book at clock.Now() (wall-clock event time); Intent.InvoiceAt is used only for accrual booking. Tests freeze clock to assert this. (`transactions.AuthorizeCustomerReceivablePaymentTemplate{ At: clock.Now(), ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `annotations.go` | Constructs models.Annotations for each charge type via ledger.ChargeTransactionAnnotations; imported by all three handler files. | Adding a field to charge.Intent.Subscription (meta.SubscriptionReference) requires threading it through chargeTransactionAnnotations or it is silently omitted from all ledger annotations. |
| `creditpurchase.go` | Handles promotional, external, and invoice-settled credit purchase lifecycles; issueCreditPurchase combines advance-attribution, accrued-translation, and receivable-issuance in one CommitGroup. | outstandingAdvanceBalance / unattributedAccruedBalance reads happen before ResolveTransactions and are outside the ledger transaction — ordering matters for correctness. |
| `flatfee.go` | Flat-fee invoice-assignment, invoice/credits-only usage accrual, payment authorization and settlement. invoiceCostBasis constant (1.0) for invoice-backed transactions. | OnAssignedToInvoice / OnCreditsOnlyUsageAccrued return creditrealization.CreateAllocationInputs (via collector); OnInvoiceUsageAccrued and payment handlers return ledgertransaction.GroupReference — different return types. |
| `usagebased.go` | Mirrors flatfee.go for usage-based charges. | Authorization/settlement timestamps use clock.Now(), not charge.Intent.InvoiceAt; tests freeze clock to assert this. |
| `helpers.go` | settledBalanceForSubAccount wraps SubAccount.GetBalance returning only Settled(). | Returns settled portion only; do not call Pending() here. |

## Anti-Patterns

- Calling h.ledger.CommitGroup without first resolving through transactions.ResolveTransactions (bypasses sub-account routing)
- Omitting chargeAnnotations from a CommitGroup call (breaks per-charge traceability)
- Skipping validateSettlementMode in any new lifecycle event handler method
- Using charge.Intent.InvoiceAt as the booking timestamp in payment-authorized/settled events (must use clock.Now())
- Writing tests by mocking the ledger instead of using ledgertestutils.IntegrationEnv (loses sub-account routing coverage)

## Decisions

- **Handler interfaces are defined in the charge sub-packages (chargecreditpurchase.Handler etc.), not in chargeadapter.** — Keeps charge domain packages free of ledger imports; chargeadapter is the bridge that knows both sides without creating circular imports.
- **invoiceCostBasis is a package-level constant (=1) used for all invoice-backed accrual and payment transactions.** — Invoice-backed receivables have a known 1:1 cost basis; encoding it once prevents per-call divergence.

## Example: Adding a new flat-fee lifecycle event handler

```
func (h *flatFeeHandler) OnNewEvent(ctx context.Context, input flatfee.OnNewEventInput) (ledgertransaction.GroupReference, error) {
    if err := input.Validate(); err != nil { return ledgertransaction.GroupReference{}, err }
    if input.Amount.IsZero() { return ledgertransaction.GroupReference{}, nil }
    if err := validateSettlementMode(input.Charge.Intent.SettlementMode, productcatalog.CreditThenInvoiceSettlementMode); err != nil {
        return ledgertransaction.GroupReference{}, fmt.Errorf("new event: %w", err)
    }
    annotations := chargeAnnotationsForFlatFeeCharge(input.Charge)
    inputs, err := transactions.ResolveTransactions(ctx, h.deps, transactions.ResolutionScope{Namespace: input.Charge.Namespace}, transactions.SomeTemplate{At: clock.Now(), Amount: input.Amount, Currency: input.Charge.Intent.Currency, CostBasis: invoiceCostBasis})
    if err != nil { return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err) }
    for i := range inputs { inputs[i] = transactions.WithAnnotations(inputs[i], annotations) }
    return h.ledger.CommitGroup(ctx, transactions.GroupInputs(input.Charge.Namespace, annotations, inputs...))
}
```

<!-- archie:ai-end -->
