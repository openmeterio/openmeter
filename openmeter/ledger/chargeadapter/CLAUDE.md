# chargeadapter

<!-- archie:ai-start -->

> Bridges charge lifecycle events (credit purchase, flat fee, usage-based) to double-entry ledger postings by translating domain charge state transitions into typed transaction templates committed via ledger.Ledger.CommitGroup. Each charge type has its own private handler struct implementing the corresponding charge domain Handler interface.

## Patterns

**Handler-per-charge-type with compile-time assertion** — Each charge type (creditPurchase, flatFee, usageBased) has a private handler struct and a public New*Handler constructor returning the domain Handler interface. A var _ <interface> = (*handler)(nil) assertion ensures compile-time compliance. (`var _ chargecreditpurchase.Handler = (*creditPurchaseHandler)(nil)
func NewCreditPurchaseHandler(ledger ledger.Ledger, ...) chargecreditpurchase.Handler { return &creditPurchaseHandler{...} }`)
**Resolve-then-annotate-then-commit pipeline** — Every ledger write follows: transactions.ResolveTransactions → annotate each resolved input with transactions.WithAnnotations → h.ledger.CommitGroup(ctx, transactions.GroupInputs(ns, annotations, inputs...)). Never call CommitGroup directly with un-annotated or un-resolved inputs. (`inputs, _ := transactions.ResolveTransactions(ctx, h.deps, scope, template)
for i, in := range inputs { inputs[i] = transactions.WithAnnotations(in, annotations) }
h.ledger.CommitGroup(ctx, transactions.GroupInputs(ns, annotations, inputs...))`)
**ChargeAnnotations on every CommitGroup call** — Every transaction group receives charge-scoped annotations via chargeAnnotationsForXxxCharge → chargeTransactionAnnotations → ledger.ChargeTransactionAnnotations. These attach chargeID, namespace, subscriptionID, phaseID, itemID, featureID. Omitting them breaks per-charge traceability. (`annotations := chargeAnnotationsForFlatFeeCharge(charge)
inputs[i] = transactions.WithAnnotations(inputs[i], annotations)`)
**Settlement mode guard before any ledger work** — Every handler method that deals with invoice-side accrual calls validateSettlementMode with an explicit allowlist of permitted modes before doing any ledger work. Incompatible modes return an error, not a no-op. (`if err := validateSettlementMode(input.Charge.Intent.SettlementMode, productcatalog.InvoiceOnlySettlementMode, productcatalog.CreditThenInvoiceSettlementMode); err != nil { return ..., fmt.Errorf("invoice usage accrued: %w", err) }`)
**Return empty GroupReference on zero-amount** — All handler methods return an empty ledgertransaction.GroupReference{} (not an error) when the amount is zero rather than issuing a zero-value ledger group. (`if amount.IsZero() { return ledgertransaction.GroupReference{}, nil }`)
**Delegate FBO collection to collector.Service** — flatFeeHandler and usageBasedHandler delegate FBO→accrued collection to collector.Service.CollectToAccrued and CorrectCollectedAccrued instead of calling ledger directly. Only payment authorization/settlement calls ledger directly. (`realizations, err := h.collector.CollectToAccrued(ctx, collector.CollectToAccruedInput{Namespace: ..., ChargeID: ..., Amount: ...})`)
**clock.Now() for payment timestamps, not Intent.InvoiceAt** — Payment-authorized and payment-settled handler methods use clock.Now() for the booking timestamp to reflect wall-clock event time. InvoiceAt is only used for accrual booking (OnInvoiceUsageAccrued, OnAssignedToInvoice). Tests freeze clock via pkg/clock to verify this. (`transactions.AuthorizeCustomerReceivablePaymentTemplate{ At: clock.Now(), ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `annotations.go` | Shared helper constructing models.Annotations for each charge type via ledger.ChargeTransactionAnnotations. All three handler files import this. | If a new field is added to charge.Intent.Subscription (meta.SubscriptionReference), thread it through chargeTransactionAnnotations here or it will be silently omitted from all ledger annotations. |
| `creditpurchase.go` | Handles promotional, external, and invoice-settled credit purchase lifecycles. issueCreditPurchase orchestrates advance-attribution, accrued-translation, and receivable-issuance templates in one CommitGroup call. | outstandingAdvanceBalance and unattributedAccruedBalance reads happen before ResolveTransactions and are not inside the ledger transaction — ordering matters for correctness. |
| `flatfee.go` | Handles flat-fee invoice-assignment, invoice-usage-accrual, credits-only accrual, payment authorization, and settlement. invoiceCostBasis package-level constant (1.0) used for all invoice-backed transactions. | OnAssignedToInvoice and OnCreditsOnlyUsageAccrued return creditrealization.CreateAllocationInputs (via collector); OnInvoiceUsageAccrued and payment handlers return ledgertransaction.GroupReference. These are different return types. |
| `usagebased.go` | Mirrors flatfee.go but for usage-based charges; payment timestamps use clock.Now() not charge.Intent.InvoiceAt. | clock.Now() is used for authorization and settlement timestamps; tests freeze clock to assert this — do not change to InvoiceAt. |
| `helpers.go` | settledBalanceForSubAccount — wraps SubAccount.GetBalance and returns the Settled() value only. | Returns only the settled portion; pending balance is excluded. Do not call Pending() here. |

## Anti-Patterns

- Calling h.ledger.CommitGroup without first resolving through transactions.ResolveTransactions (bypasses sub-account routing)
- Omitting chargeAnnotations from a CommitGroup call (breaks per-charge traceability in ledger)
- Skipping validateSettlementMode in any new lifecycle event handler method
- Using charge.Intent.InvoiceAt as the booking timestamp in payment-authorized or payment-settled events (must use clock.Now())
- Writing tests without ledgertestutils.IntegrationEnv and instead mocking the ledger (breaks coverage of sub-account routing logic)

## Decisions

- **Handler interfaces are defined in the charge sub-packages (chargecreditpurchase.Handler etc.), not in chargeadapter itself.** — Keeps charge domain packages free of ledger imports; chargeadapter is the bridge that knows both sides without creating circular imports.
- **invoiceCostBasis is a package-level constant (*alpacadecimal = 1) used for all invoice-backed accrual and payment transactions.** — Invoice-backed receivables have a known 1:1 cost basis; encoding it as a constant prevents per-call divergence.

## Example: Adding a new flat-fee lifecycle event handler

```
// flatfee.go
func (h *flatFeeHandler) OnNewEvent(ctx context.Context, input flatfee.OnNewEventInput) (ledgertransaction.GroupReference, error) {
    if err := input.Validate(); err != nil {
        return ledgertransaction.GroupReference{}, err
    }
    if input.Amount.IsZero() {
        return ledgertransaction.GroupReference{}, nil
    }
    if err := validateSettlementMode(input.Charge.Intent.SettlementMode, productcatalog.CreditThenInvoiceSettlementMode); err != nil {
        return ledgertransaction.GroupReference{}, fmt.Errorf("new event: %w", err)
    }
    customerID := customer.CustomerID{Namespace: input.Charge.Namespace, ID: input.Charge.Intent.CustomerID}
    annotations := chargeAnnotationsForFlatFeeCharge(input.Charge)
    inputs, err := transactions.ResolveTransactions(ctx, h.deps, transactions.ResolutionScope{CustomerID: customerID, Namespace: input.Charge.Namespace}, transactions.SomeTemplate{At: clock.Now(), Amount: input.Amount, Currency: input.Charge.Intent.Currency, CostBasis: invoiceCostBasis})
    if err != nil { return ledgertransaction.GroupReference{}, fmt.Errorf("resolve transactions: %w", err) }
// ...
```

<!-- archie:ai-end -->
