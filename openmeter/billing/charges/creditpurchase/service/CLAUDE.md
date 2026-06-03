# service

<!-- archie:ai-start -->

> Business-logic service implementing creditpurchase.Service — orchestrates charge creation, settlement-path dispatch (promotional/external/invoice), and invoice lifecycle callbacks. Multi-step flows run inside transaction.Run; all ledger interactions go through creditpurchase.Handler.

## Patterns

**transaction.Run wraps every service method** — Public methods wrap adapter calls in transaction.Run / RunWithNoValue — never call adapter methods outside a transaction boundary. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeWithGatheringLine, error) { charge, err := s.adapter.CreateCharge(ctx, ...); ... })`)
**Settlement-type dispatch in Create** — Create inspects charge.Intent.Settlement.Type() and dispatches to onPromotionalCreditPurchase, onExternalCreditPurchase, or the invoice path — each in its own file. (`switch charge.Intent.Settlement.Type() { case creditpurchase.SettlementTypePromotional: charge, err = s.onPromotionalCreditPurchase(ctx, charge); case creditpurchase.SettlementTypeExternal: charge, err = s.onExternalCreditPurchase(ctx, charge) }`)
**Handler mediation for ledger interactions** — Ledger interactions are always routed through creditpurchase.Handler methods (OnCreditPurchaseInitiated, OnCreditPurchasePaymentAuthorized/Settled, OnPromotionalCreditPurchase) — never called directly. (`ledgerTransactionGroupReference, err := s.handler.OnCreditPurchaseInitiated(ctx, charge)`)
**BackfillAdvanceLineageSegments after non-empty TransactionGroupID** — After a credit grant where TransactionGroupID is non-empty, call s.lineage.BackfillAdvanceLineageSegments — required for external and invoice paths. (`if ledgerTransactionGroupReference.TransactionGroupID != "" { s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{...}) }`)
**input.Normalize() before Validate() in Create** — Create calls input.Intent = input.Intent.Normalized() before input.Validate() so denormalized intents never reach the adapter. (`input.Intent = input.Intent.Normalized(); if err := input.Validate(); err != nil { ... }`)
**Config.Validate() collects nil-checks with errors.Join** — Config.Validate() appends all missing-field errors and returns errors.Join(errs...) — never returns after the first nil field. (`var errs []error; if c.Adapter == nil { errs = append(errs, errors.New("adapter cannot be null")) }; return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config + Validate(), New(), service struct with four required fields. | All four fields (Adapter, Handler, Lineage, MetaAdapter) are required; nil Handler panics on ledger calls. |
| `create.go` | Create: normalizes intent, validates, creates DB row, dispatches to settlement path, optionally builds GatheringLine for invoice settlement. | buildInvoiceCreditPurchaseGatheringLine sets AnnotationKeyTaxable=false and AnnotationKeyReason=CreditPurchase — keep these for downstream billing correctness. |
| `external.go` | External settlement: onExternalCreditPurchase, HandleExternalPaymentAuthorized/Settled. | onExternalCreditPurchase may chain two state transitions; ordering must be initiate -> authorize -> settle. |
| `invoice.go` | Invoice settlement hooks: PostInvoiceDraftCreated, PostInvoicePaymentAuthorized/Settled. | PostInvoicePayment* run inside billing's PostUpdate hook (already in a transaction) — do not wrap in transaction.Run again. |
| `promotional.go` | Promotional path: grants credits immediately and moves charge to StatusFinal in one step. | Checks activePromotionalCreditPurchaseStatuses to prevent double-grant on re-invocation — do not remove this guard. |
| `funded_credit_activity.go` | Thin delegation to adapter.ListFundedCreditActivities inside transaction.Run. | No business logic here — add logic to the adapter package-level function instead. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run — breaks atomicity across multi-step charge flows.
- Directly calling ledger APIs instead of routing through creditpurchase.Handler — bypasses the credits.enabled=false noop guard.
- Skipping lineage.BackfillAdvanceLineageSegments after a non-empty TransactionGroupID — leaves lineage records incomplete.
- Adding settlement-type logic directly in Create instead of a dedicated file (promotional/external/invoice).
- Omitting input.Intent.Normalized() before Validate() in Create — denormalized intents reach the adapter.

## Decisions

- **Settlement paths split across promotional.go/external.go/invoice.go.** — Each settlement type has distinct ledger interactions and state-machine steps; separate files avoid a switch blob and are independently testable.
- **Ledger interactions are always mediated through creditpurchase.Handler.** — Enables noop handler injection when credits are disabled without changing service code.
- **transaction.Run is used in the service even though the adapter uses TransactingRepo.** — The service atomically combines multiple adapter calls (create charge + grant + update status); TransactingRepo alone scopes only individual adapter calls.

## Example: Service method writing to adapter and calling the handler

```
func (s *service) SomeOperation(ctx context.Context, input creditpurchase.SomeInput) (creditpurchase.SomeResult, error) {
	if err := input.Validate(); err != nil { return creditpurchase.SomeResult{}, err }
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.SomeResult, error) {
		ledgerRef, err := s.handler.OnSomeEvent(ctx, input.Charge)
		if err != nil { return creditpurchase.SomeResult{}, err }
		return s.adapter.SomeUpdate(ctx, input)
	})
}
```

<!-- archie:ai-end -->
