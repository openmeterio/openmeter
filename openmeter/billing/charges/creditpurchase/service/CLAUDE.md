# service

<!-- archie:ai-start -->

> Business-logic service implementing creditpurchase.Service — orchestrates charge creation, settlement path dispatch (promotional/external/invoice), and invoice lifecycle callbacks. All multi-step flows run inside transaction.Run to ensure atomicity.

## Patterns

**transaction.Run wraps every service method** — Every public method wraps its adapter calls in transaction.Run(ctx, s.adapter, func(ctx context.Context) ...) or transaction.RunWithNoValue for side-effect-only calls — never calls adapter methods outside a transaction boundary. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeWithGatheringLine, error) { charge, err := s.adapter.CreateCharge(ctx, ...); ... })`)
**Settlement-type dispatch in Create** — Create inspects charge.Intent.Settlement.Type() and dispatches to onPromotionalCreditPurchase, onExternalCreditPurchase, or the invoice path — each path has its own file (promotional.go, external.go, invoice.go). (`switch charge.Intent.Settlement.Type() {
case creditpurchase.SettlementTypePromotional: charge, err = s.onPromotionalCreditPurchase(ctx, charge)
case creditpurchase.SettlementTypeInvoice: // noop until invoice created
case creditpurchase.SettlementTypeExternal: charge, err = s.onExternalCreditPurchase(ctx, charge)
}`)
**handler.On* for ledger side-effects** — Ledger interactions (credit grants, payment authorization, payment settlement) are always routed through creditpurchase.Handler methods (OnCreditPurchaseInitiated, OnCreditPurchasePaymentAuthorized, OnCreditPurchasePaymentSettled, OnPromotionalCreditPurchase) — never called directly against the ledger. (`ledgerTransactionGroupReference, err := s.handler.OnCreditPurchaseInitiated(ctx, charge)`)
**lineage.BackfillAdvanceLineageSegments after grant** — After creating a credit grant with a non-empty TransactionGroupID, call s.lineage.BackfillAdvanceLineageSegments to record lineage — required for both external and invoice settlement paths. (`if ledgerTransactionGroupReference.TransactionGroupID != "" { if err := s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{...}); err != nil { return ... } }`)
**Validate input then delegate to adapter** — Every service method calls input.Validate() before entering the transaction. Delegation to adapter is always inside transaction.Run. (`func (s *service) ListFundedCreditActivities(ctx context.Context, input ...) (...) { if err := input.Validate(); err != nil { return ..., err }; return transaction.Run(ctx, s.adapter, func(...) { return s.adapter.ListFundedCreditActivities(ctx, input) }) }`)
**Config.Validate() + New() constructor pattern** — service.New(Config) calls config.Validate() collecting all nil-check errors with errors.Join before returning a service. (`func New(config Config) (creditpurchase.Service, error) { if err := config.Validate(); err != nil { return nil, err } return &service{...}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config, Validate, New constructor, service struct declaration. | All four fields (Adapter, Handler, Lineage, MetaAdapter) are required — nil Handler means ledger calls will panic. |
| `create.go` | Create entry-point: normalizes intent, validates, creates DB row, dispatches to settlement path, optionally builds GatheringLine for invoice settlement. | buildInvoiceCreditPurchaseGatheringLine sets AnnotationKeyTaxable=false and AnnotationKeyReason=CreditPurchase — these annotations must stay for downstream billing correctness. |
| `external.go` | External settlement path: onExternalCreditPurchase, HandleExternalPaymentAuthorized, HandleExternalPaymentSettled. | onExternalCreditPurchase may chain through two state transitions (authorized then settled) based on targetStatus — ordering is: initiate → authorize → settle. |
| `invoice.go` | Invoice settlement lifecycle hooks called by billing service: PostInvoiceDraftCreated, PostInvoicePaymentAuthorized, PostInvoicePaymentSettled. | PostInvoicePaymentAuthorized and PostInvoicePaymentSettled are called inside billing's PostUpdate hook (already in a transaction) — do not start a new outer transaction. |
| `promotional.go` | Promotional credit purchase path: grants credits immediately and moves charge to StatusFinal in one step. | Checks activePromotionalCreditPurchaseStatuses before processing to prevent double-grant on re-invocation. |
| `funded_credit_activity.go` | Thin delegation to adapter.ListFundedCreditActivities inside transaction.Run. | No business logic here — if logic is needed, add it to the adapter's package-level function, not this file. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run — breaks atomicity across multi-step charge flows.
- Directly calling ledger APIs instead of routing through creditpurchase.Handler — bypasses the handler abstraction and breaks credits.enabled=false guard.
- Skipping lineage.BackfillAdvanceLineageSegments after a non-empty TransactionGroupID — leaves lineage records incomplete.
- Adding settlement-type logic directly in Create instead of dispatching to a dedicated file (promotional.go, external.go, invoice.go).
- Omitting input.Normalize() (e.g., input.Intent.Normalized()) before Validate() in Create — allows denormalized intents to reach the adapter.

## Decisions

- **Settlement paths are split across three files (promotional.go, external.go, invoice.go) rather than inlined in create.go.** — Each settlement type has distinct ledger interactions and state machine steps; keeping them separate avoids a large switch-case blob and makes each path independently testable.
- **Ledger interactions are always mediated through creditpurchase.Handler rather than called directly.** — Enables noop handler injection when credits are disabled (credits.enabled=false) without changing service code.
- **transaction.Run is used in the service layer even though the adapter also uses entutils.TransactingRepo.** — The service needs to atomically combine multiple adapter calls (create charge + create grant + update status); TransactingRepo alone in the adapter only scopes individual calls.

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
