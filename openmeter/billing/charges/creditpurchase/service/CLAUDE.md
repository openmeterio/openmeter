# service

<!-- archie:ai-start -->

> Service layer that drives the credit-purchase charge lifecycle: creates the charge, runs settlement-type-specific state machines (promotional/invoice/external), grants ledger credit, backfills lineage, and records payment settlements. Implements creditpurchase.Service.

## Patterns

**Constructor validates four collaborators** — New(Config) returns creditpurchase.Service; Config.Validate() collects errors for nil Adapter, Handler, Lineage, MetaAdapter via errors.Join. The service struct holds exactly these. (`func (c Config) Validate() error { var errs []error; if c.Adapter == nil { errs = append(errs, errors.New("adapter cannot be null")) } ...; return errors.Join(errs...) }`)
**Public methods wrap work in transaction.Run** — Create/List/GetByIDs/ListFundedCreditActivities and the External/Invoice handlers run inside transaction.Run(ctx, s.adapter, ...) or RunWithNoValue; the adapter is the transaction driver. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeWithGatheringLine, error) { ... })`)
**Settlement-type switch on Create dispatches lifecycle** — Create normalizes/validates input, persists via adapter.CreateCharge, then switches charge.Intent.Settlement.Type(): Promotional runs the state machine, Invoice is a noop (driven later by invoice hooks) and returns a GatheringLineToCreate, External calls onExternalCreditPurchase. (`switch charge.Intent.Settlement.Type() { case creditpurchase.SettlementTypePromotional: stateMachine, _ := NewPromotionalCreditPurchaseStateMachine(...); advancedCharge, _ := stateMachine.AdvanceUntilStateStable(ctx) ... }`)
**Credit grant always pairs with lineage backfill** — After OnCreditPurchase* yields a ledger TransactionGroupID, code calls adapter.CreateCreditGrant then, if the group id is non-empty, lineage.BackfillAdvanceLineageSegments with namespace/customer/currency/amount/featureFilters. (`if ledgerTransactionGroupReference.TransactionGroupID != "" { s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{...}) }`)
**State machines composed from chargestatemachine.Machine** — newStateMachineBase wires chargestatemachine.New with Persistence{UpdateBase->adapter.UpdateCharge, Refetch->adapter.GetByID with ExpandRealizations}. Concrete machines (e.g. PromotionalCreditpurchaseStateMachine) call configureStates to Permit(meta.TriggerNext, ...) and OnEntry(statelessx.EntryFunc(...)). (`s.Configure(creditpurchase.StatusFinal).OnEntry(statelessx.EntryFunc(s.GrantPromotionalCredit))`)
**Payment realization through ledger handler then adapter** — External/Invoice payment transitions call s.handler.OnCreditPurchasePaymentAuthorized/Settled to obtain a ledger group reference, then persist via adapter.Create/UpdateExternalPayment or Create/UpdateInvoicedPayment, with idempotency guards on existing settlements. (`ledgerRef, _ := s.handler.OnCreditPurchasePaymentAuthorized(ctx, creditpurchase.PaymentEventInput{Charge: charge, EventAt: eventAt}); s.adapter.CreateExternalPayment(ctx, charge.GetChargeID(), newPaymentSettlement)`)
**Invoice-settled side effects are gathering-line + post-hooks** — Invoice settlement does not act during Create; buildInvoiceCreditPurchaseGatheringLine emits a non-taxable flat in-advance GatheringLine (annotated AnnotationKeyTaxable=false, AnnotationKeyReason=CreditPurchase), and PostInvoiceDraftCreated/PaymentAuthorized/PaymentSettled run from billing PostUpdate hooks. (`annotations[billing.AnnotationKeyTaxable] = lo.ToPtr("false"); annotations[billing.AnnotationKeyReason] = lo.ToPtr(billing.AnnotationValueReasonCreditPurchase)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config/Validate, New, and the service struct (adapter, metaAdapter, handler, lineage) | all four deps are mandatory; struct field set must mirror Config |
| `create.go` | Create entrypoint with the settlement-type switch and buildInvoiceCreditPurchaseGatheringLine | Invoice case is intentionally a noop at create time and returns ChargeWithGatheringLine.GatheringLineToCreate; total cost = CreditAmount * CostBasis rounded by currency calculator |
| `external.go` | onExternalCreditPurchase plus HandleExternalPaymentAuthorized/Settled - drives external settlement to Active/Final | InitialStatus determines whether authorized/settled transitions fire; guards ErrPaymentAlreadyAuthorized / ErrCannotSettleNotAuthorizedPayment / ErrPaymentAlreadySettled |
| `invoice.go` | PostInvoiceDraftCreated/PostInvoicePaymentAuthorized/PostInvoicePaymentSettled invoked from billing PostUpdate hooks | these run already inside a billing transaction; they grant credit on draft, then create/settle the invoiced payment and flip status to Final |
| `promotional.go` | grantPromotionalCredit and the PromotionalCreditpurchaseStateMachine (configureStates, GrantPromotionalCredit) | rejects an already-realized credit grant; Final state OnEntry grants credit; both Created and Active permit TriggerNext->Final |
| `statemachine.go` | stateMachine base wrapping chargestatemachine.Machine with Persistence (UpdateBase, Refetch) | Refetch must expand meta.ExpandRealizations or downstream mapping fails; StateMachineConfig.Validate requires Charge.ID and Adapter |
| `get.go / funded_credit_activity.go` | Read-side List/GetByIDs and ListFundedCreditActivities, each transaction.Run + adapter delegation | thin wrappers; only add validation, do not bypass transaction.Run |
| `promotional_test.go` | Mock-driven unit tests for the promotional state machine using fake adapter/handler/lineage | lineage mock is matched on namespace/customer/currency/amount; tests assert single CreateCreditGrant call and Final status |

## Anti-Patterns

- Granting credit without immediately backfilling lineage via lineage.BackfillAdvanceLineageSegments when the ledger group id is non-empty
- Acting on invoice-settlement charges during Create instead of deferring to PostInvoice* hooks
- Bypassing the settlement-type state machine and mutating charge.Status directly outside the lifecycle methods
- Building a Refetch in a state machine without meta.ExpandRealizations (realizations come back unmapped)
- Re-authorizing/settling a payment without the existing-settlement idempotency guards

## Decisions

- **Settlement type (Promotional/Invoice/External) selects a distinct lifecycle path, with promotional/external driven synchronously and invoice driven by billing invoice hooks** — Promotional/external grants happen at charge time, but invoiced credit purchases must follow the invoice state machine, so their grant/payment is deferred to PostInvoice* callbacks
- **Lifecycle progression is expressed as a chargestatemachine.Machine with OnEntry side effects rather than imperative status writes** — Keeps grant/transition effects coupled to state entry and reuses the shared charges state-machine framework for persistence and refetch
- **Ledger interaction is mediated by a Handler interface (OnCreditPurchaseInitiated/PaymentAuthorized/PaymentSettled/OnPromotionalCreditPurchase) injected into the service** — Decouples charge lifecycle from concrete ledger wiring and lets tests substitute a fake handler

## Example: Create dispatches on settlement type and advances the promotional state machine

```
func (s *service) Create(ctx context.Context, input creditpurchase.CreateInput) (creditpurchase.ChargeWithGatheringLine, error) {
	input.Intent = input.Intent.Normalized()
	if err := input.Validate(); err != nil { return creditpurchase.ChargeWithGatheringLine{}, err }
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeWithGatheringLine, error) {
		charge, err := s.adapter.CreateCharge(ctx, creditpurchase.CreateChargeInput(input))
		if err != nil { return creditpurchase.ChargeWithGatheringLine{}, err }
		switch charge.Intent.Settlement.Type() {
		case creditpurchase.SettlementTypePromotional:
			sm, err := NewPromotionalCreditPurchaseStateMachine(StateMachineConfig{Charge: charge, Adapter: s.adapter, Service: s})
			if err != nil { return creditpurchase.ChargeWithGatheringLine{}, err }
			advanced, err := sm.AdvanceUntilStateStable(ctx)
			if err != nil { return creditpurchase.ChargeWithGatheringLine{}, err }
			if advanced != nil { charge = *advanced }
		case creditpurchase.SettlementTypeExternal:
			charge, err = s.onExternalCreditPurchase(ctx, charge)
// ...
```

<!-- archie:ai-end -->
