# service

<!-- archie:ai-start -->

> Business-logic service implementing creditpurchase.Service — orchestrates charge creation, settlement-path dispatch (promotional/external/invoice), and invoice lifecycle callbacks. All multi-step flows run inside transaction.Run for atomicity; ledger interactions are always mediated through creditpurchase.Handler.

## Patterns

**transaction.Run wraps every service method** — Every public method wraps adapter calls in transaction.Run(ctx, s.adapter, func(ctx context.Context) ...) or transaction.RunWithNoValue — never calls adapter methods outside a transaction boundary. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeWithGatheringLine, error) { charge, err := s.adapter.CreateCharge(ctx, ...); ... })`)
**Settlement-type dispatch in Create** — Create inspects charge.Intent.Settlement.Type() and dispatches to onPromotionalCreditPurchase, onExternalCreditPurchase, or the invoice path — each path lives in its own file (promotional.go, external.go, invoice.go). (`switch charge.Intent.Settlement.Type() {
case creditpurchase.SettlementTypePromotional: charge, err = s.onPromotionalCreditPurchase(ctx, charge)
case creditpurchase.SettlementTypeInvoice: // noop until invoice created
case creditpurchase.SettlementTypeExternal: charge, err = s.onExternalCreditPurchase(ctx, charge)
}`)
**Handler mediation for all ledger interactions** — Ledger interactions (credit grants, payment authorization, settlement) are always routed through creditpurchase.Handler methods (OnCreditPurchaseInitiated, OnCreditPurchasePaymentAuthorized, OnCreditPurchasePaymentSettled, OnPromotionalCreditPurchase) — never called directly against the ledger. (`ledgerTransactionGroupReference, err := s.handler.OnCreditPurchaseInitiated(ctx, charge)`)
**lineage.BackfillAdvanceLineageSegments after non-empty TransactionGroupID** — After creating a credit grant where ledgerTransactionGroupReference.TransactionGroupID is non-empty, call s.lineage.BackfillAdvanceLineageSegments — required for both external and invoice settlement paths. (`if ledgerTransactionGroupReference.TransactionGroupID != "" { if err := s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{...}); err != nil { return ... } }`)
**input.Normalize() before Validate() in Create** — Create calls input.Intent = input.Intent.Normalized() before input.Validate() to normalize denormalized intents before they reach the adapter. (`func (s *service) Create(ctx context.Context, input creditpurchase.CreateInput) (...) { input.Intent = input.Intent.Normalized(); if err := input.Validate(); err != nil { ... } ... }`)
**Config.Validate() collects all nil-check errors with errors.Join** — Config.Validate() appends all missing-field errors into a slice and returns errors.Join(errs...) — never returns after the first nil field. (`func (c Config) Validate() error { var errs []error; if c.Adapter == nil { errs = append(errs, errors.New("adapter cannot be null")) }; ...; return errors.Join(errs...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, Validate(), New() constructor, service struct declaration with four required fields. | All four fields (Adapter, Handler, Lineage, MetaAdapter) are required — nil Handler means ledger calls will panic at runtime. |
| `create.go` | Create entry-point: normalizes intent, validates, creates DB row, dispatches to settlement path, optionally builds GatheringLine for invoice settlement. | buildInvoiceCreditPurchaseGatheringLine sets AnnotationKeyTaxable=false and AnnotationKeyReason=CreditPurchase — these annotations must stay for downstream billing correctness. |
| `external.go` | External settlement path: onExternalCreditPurchase, HandleExternalPaymentAuthorized, HandleExternalPaymentSettled. | onExternalCreditPurchase may chain through two state transitions (authorize then settle) based on targetStatus — ordering must be: initiate → authorize → settle. |
| `invoice.go` | Invoice settlement lifecycle hooks: PostInvoiceDraftCreated, PostInvoicePaymentAuthorized, PostInvoicePaymentSettled. | PostInvoicePaymentAuthorized and PostInvoicePaymentSettled are called inside billing's PostUpdate hook (already in a transaction) — do not wrap in transaction.Run again. |
| `promotional.go` | Promotional credit purchase path: grants credits immediately and moves charge to StatusFinal in one step. | Checks activePromotionalCreditPurchaseStatuses before processing to prevent double-grant on re-invocation — do not remove this guard. |
| `funded_credit_activity.go` | Thin delegation to adapter.ListFundedCreditActivities inside transaction.Run. | No business logic here — if logic is needed, add it to the adapter's package-level function, not this file. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run — breaks atomicity across multi-step charge flows.
- Directly calling ledger APIs instead of routing through creditpurchase.Handler — bypasses the credits.enabled=false noop guard.
- Skipping lineage.BackfillAdvanceLineageSegments after a non-empty TransactionGroupID — leaves lineage records incomplete.
- Adding settlement-type logic directly in Create instead of dispatching to a dedicated file (promotional.go, external.go, invoice.go).
- Omitting input.Intent.Normalized() before Validate() in Create — allows denormalized intents to reach the adapter.

## Decisions

- **Settlement paths are split across three files (promotional.go, external.go, invoice.go) rather than inlined in create.go.** — Each settlement type has distinct ledger interactions and state machine steps; separate files avoid a large switch-case blob and make each path independently testable.
- **Ledger interactions are always mediated through creditpurchase.Handler rather than called directly.** — Enables noop handler injection when credits are disabled (credits.enabled=false) without changing service code.
- **transaction.Run is used in the service layer even though the adapter also uses entutils.TransactingRepo.** — The service needs to atomically combine multiple adapter calls (create charge + create grant + update status); TransactingRepo alone only scopes individual adapter calls.

## Example: Adding a new service method that writes to adapter and calls handler

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) SomeOperation(ctx context.Context, input creditpurchase.SomeInput) (creditpurchase.SomeResult, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.SomeResult{}, err
	}
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.SomeResult, error) {
		ledgerRef, err := s.handler.OnSomeEvent(ctx, input.Charge)
		if err != nil {
			return creditpurchase.SomeResult{}, err
		}
// ...
```

<!-- archie:ai-end -->
