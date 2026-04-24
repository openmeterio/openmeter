# chargeadapter

<!-- archie:ai-start -->

> Bridges charge lifecycle events (credit purchase, flat fee, usage-based) to ledger transaction templates, translating domain-level charge state transitions into double-entry ledger postings via ledger.Ledger.CommitGroup. Each charge type gets its own handler struct implementing the corresponding charges Handler interface.

## Patterns

**Handler-per-charge-type** — Each charge type has a private handler struct (creditPurchaseHandler, flatFeeHandler, usageBasedHandler) with a public constructor (NewCreditPurchaseHandler, NewFlatFeeHandler, NewUsageBasedHandler) returning the domain Handler interface. Verified by var _ interface = (*handler)(nil) compile-time assertions. (`var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil)`)
**Resolve-then-annotate-then-commit** — Every ledger write follows: transactions.ResolveTransactions → annotate each input with transactions.WithAnnotations → h.ledger.CommitGroup(ctx, transactions.GroupInputs(namespace, annotations, inputs...)). Never call CommitGroup directly without resolving through templates first. (`inputs, _ := transactions.ResolveTransactions(ctx, h.deps, scope, template); for i, in := range inputs { inputs[i] = transactions.WithAnnotations(in, annotations) }; h.ledger.CommitGroup(ctx, transactions.GroupInputs(ns, annotations, inputs...))`)
**ChargeAnnotations on every group** — Every transaction group receives charge-scoped annotations produced by chargeAnnotationsForXxxCharge → chargeTransactionAnnotations → ledger.ChargeTransactionAnnotations. These attach chargeID, namespace, subscriptionID, phaseID, itemID, featureID to the group and to each resolved input. (`annotations := chargeAnnotationsForFlatFeeCharge(charge); inputs[i] = transactions.WithAnnotations(inputs[i], annotations)`)
**Settlement mode guard** — Every handler method that deals with invoice-side accrual calls validateSettlementMode before doing any ledger work. Each event has an explicit allowlist; using an incompatible mode returns an error, not a no-op. (`if err := validateSettlementMode(input.Charge.Intent.SettlementMode, productcatalog.InvoiceOnlySettlementMode, productcatalog.CreditThenInvoiceSettlementMode); err != nil { return ..., fmt.Errorf("invoice usage accrued: %w", err) }`)
**Return empty reference on zero-amount** — All handler methods return an empty ledgertransaction.GroupReference{} (not an error) when the amount is zero, rather than issuing a zero-value ledger group. (`if amount.IsZero() { return ledgertransaction.GroupReference{}, nil }`)
**collector.Service for FBO collection** — flatFeeHandler and usageBasedHandler delegate FBO→accrued collection to collector.Service.CollectToAccrued and CorrectCollectedAccrued instead of calling ledger directly. Only payment authorization/settlement calls ledger directly. (`realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{...})`)
**Integration tests use ledgertestutils.IntegrationEnv** — All _test.go files embed *ledgertestutils.IntegrationEnv and call ledgertestutils.NewIntegrationEnv(t, prefix) to get a live Postgres-backed ledger. Tests verify exact sub-account balances via env.sumBalance(t, subAccount). (`base := ledgertestutils.NewIntegrationEnv(t, "chargeadapter-creditpurchase"); handler := chargeadapter.NewCreditPurchaseHandler(base.Deps.HistoricalLedger, base.Deps.ResolversService, base.Deps.AccountService)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `annotations.go` | Shared helper that constructs models.Annotations for each charge type by calling ledger.ChargeTransactionAnnotations. All three handler files import this. | If a new field is added to a charge's subscription reference it must be threaded through chargeTransactionAnnotations here. |
| `creditpurchase.go` | Handles promotional, external, and invoice-settled credit purchase lifecycles. issueCreditPurchase orchestrates advance-attribution, accrued-translation, and receivable-issuance templates in a single CommitGroup call. | The advance and accrued query logic (outstandingAdvanceBalance, unattributedAccruedBalance) runs before template resolution; these reads are not part of the ledger transaction so ordering matters. |
| `flatfee.go` | Handles flat-fee invoice-assignment, invoice-usage-accrual, credits-only accrual, payment authorization, and settlement. invoiceCostBasis constant (1.0) is the cost basis for invoice-backed transactions. | OnAssignedToInvoice and OnCreditsOnlyUsageAccrued return creditrealization.CreateAllocationInputs (not a GroupReference); OnInvoiceUsageAccrued and payment handlers return GroupReference. |
| `usagebased.go` | Mirrors flat-fee but for usage-based charges; payment timestamps use clock.Now() not charge.Intent.InvoiceAt. | clock.Now() is used for authorization and settlement timestamps to reflect wall-clock event time, not the invoice period end. Tests freeze clock to assert this. |
| `helpers.go` | settledBalanceForSubAccount — wraps SubAccount.GetBalance and returns Settled() value. | Only returns the settled portion; pending balance is not included. |

## Anti-Patterns

- Calling h.ledger.CommitGroup without first resolving through transactions.ResolveTransactions (bypasses sub-account routing logic)
- Omitting chargeAnnotations from a CommitGroup call (breaks traceability by charge ID)
- Skipping validateSettlementMode in a new lifecycle event handler
- Using charge.Intent.InvoiceAt as the booking timestamp in payment-authorized/settled events (should be clock.Now())
- Writing tests without ledgertestutils.IntegrationEnv and instead mocking the ledger (breaks coverage of sub-account routing)

## Decisions

- **Handler interfaces are defined in the charge sub-packages (chargecreditpurchase.Handler etc.), not in chargeadapter itself.** — Keeps charge domain logic free of ledger imports; chargeadapter is the bridge that knows both sides.
- **invoiceCostBasis is a package-level constant (1.0) used for all invoice-backed accrual and payment transactions.** — Invoice-backed receivables have a known 1:1 cost basis; encoding it as a constant prevents per-call divergence.

## Example: Adding a new charge lifecycle event handler for a flat-fee charge

```
// In flatfee.go
func (h *flatFeeHandler) OnNewEvent(ctx context.Context, charge flatfee.Charge) (ledgertransaction.GroupReference, error) {
    if err := charge.Validate(); err != nil {
        return ledgertransaction.GroupReference{}, err
    }
    if err := validateSettlementMode(charge.Intent.SettlementMode, productcatalog.CreditThenInvoiceSettlementMode); err != nil {
        return ledgertransaction.GroupReference{}, fmt.Errorf("new event: %w", err)
    }
    customerID := customer.CustomerID{Namespace: charge.Namespace, ID: charge.Intent.CustomerID}
    annotations := chargeAnnotationsForFlatFeeCharge(charge)
    inputs, err := transactions.ResolveTransactions(ctx, h.deps,
        transactions.ResolutionScope{CustomerID: customerID, Namespace: charge.Namespace},
        transactions.SomeTemplate{At: charge.Intent.InvoiceAt, Amount: someAmount, Currency: charge.Intent.Currency, CostBasis: invoiceCostBasis},
    )
    if err != nil { return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err) }
// ...
```

<!-- archie:ai-end -->
