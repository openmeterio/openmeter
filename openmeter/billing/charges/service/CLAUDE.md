# service

<!-- archie:ai-start -->

> Orchestration layer for the charges domain: implements charges.Service by routing to sub-services (flatfee, usagebased, creditpurchase). Owns the Create/AdvanceCharges/ApplyPatches lifecycle, auto-advancement of credit-only charges, and gathering-line creation for invoiceable charges.

## Patterns

**transaction.Run wraps all multi-write operations** — Create, AdvanceCharges, and HandleCreditPurchaseExternalPaymentStateTransition all wrap their DB-touching body in transaction.Run(ctx, s.adapter, func(ctx) ...) to ensure atomicity. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) { ... })`)
**Delegate per-type work to sub-services** — service.Create calls s.flatFeeService.Create, s.usageBasedService.Create, and s.creditPurchaseService.Create separately in a single transaction; results are mapped back to original intent indexes via charges.WithIndex[T]. (`flatFees, err := s.flatFeeService.Create(ctx, flatfee.CreateInput{...})
usageBasedCharges, err := s.usageBasedService.Create(ctx, usagebased.CreateInput{...})`)
**Auto-advance credit-only charges post-create** — After the create transaction commits, autoAdvanceCreatedCharges identifies customers with newly created credit-only charges and calls AdvanceCharges in a separate transaction so creation state is persisted even if advancement fails. (`return s.autoAdvanceCreatedCharges(ctx, result.charges)`)
**invokeInvoiceNowOnCreate for bypassed collection alignment** — Credit purchases that bypass collection alignment call s.billingService.InvoicePendingLines with billing.WithBypassCollectionAlignment() after the main TX to trigger immediate invoicing. (`s.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{..., IncludePendingLines: mo.Some(lines), AsOf: lo.ToPtr(clock.Now())}, billing.WithBypassCollectionAlignment())`)
**validateNamespaceLockdown before every write operation** — Create and AdvanceCharges both call s.validateNamespaceLockdown(namespace) before any DB work; this is mandatory for all state-changing service methods. (`if err := s.validateNamespaceLockdown(input.Namespace); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines the service struct, Config, New constructor, and dependency validation. Config holds all sub-service and adapter dependencies injected at construction. | All dependencies (FlatFeeService, UsageBasedService, CreditPurchaseService, BillingService, FeatureService, MetaAdapter, Adapter) must be non-nil — Config.Validate() enforces this. |
| `create.go` | Implements charges.Service.Create — fans out to sub-services, collects gathering lines for billing, and post-commit triggers auto-advance and optional invoice-now. | The two-phase design (TX for creation + separate TX for auto-advance) is intentional; do not merge them into one transaction or crash-recovery will lose the created state. |
| `advance.go` | Implements charges.Service.AdvanceCharges — wraps in transaction.Run, lists non-final charges, dispatches to flatFeeService.AdvanceCharge or usageBasedService.AdvanceCharge by type. | Only CreditOnlySettlementMode flat fees are advanced here; non-credit-only flat fees are skipped intentionally. |
| `creditpurchase.go` | Implements HandleCreditPurchaseExternalPaymentStateTransition — type-asserts the charge to creditpurchase.Charge and dispatches to HandleExternalPaymentAuthorized or HandleExternalPaymentSettled. | Must run inside transaction.Run so the state-machine transition and DB write are atomic. |
| `base_test.go` | Shared BaseSuite that wires the full charges stack (all sub-adapters, sub-services, line engines) atop billingtest.BaseSuite. All service tests embed this. | Line engines must be registered on s.BillingService before tests run — omitting RegisterLineEngine causes silent no-ops during invoice processing. |
| `helpers.go` | Contains chargesByType classifier and other shared helpers used across create.go, advance.go, and patch.go. | chargesByType panics on unknown charge types — new charge types must be added here. |

## Anti-Patterns

- Calling sub-service Create/Advance methods outside a transaction.Run — breaks atomicity across charge types
- Merging the create transaction and the auto-advance transaction — creation state would be lost if advancement panics mid-flight
- Adding billing-logic (e.g., gathering-line creation) directly to adapter methods — all orchestration belongs in service
- Calling s.billingService methods without a prior validateNamespaceLockdown — bypasses namespace-level write protection
- Skipping RegisterLineEngine for a new charge type's engine — the billing service will silently produce no-op lines

## Decisions

- **Two separate transactions for Create: one for creation, one for auto-advance** — The balance-worker can retry advancement independently; losing the creation write due to an advance panic would be unrecoverable. Separation makes both steps individually durable.
- **Service delegates per-type work to flatfee/usagebased/creditpurchase sub-services rather than handling inline** — Each charge type has distinct state machines, realization logic, and handler callbacks; a single service method would become unmanageable. Delegation keeps each type's logic cohesive.

## Example: Implementing a new top-level charges.Service method that spans multiple charge types

```
func (s *service) NewOperation(ctx context.Context, input charges.NewOperationInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if err := s.validateNamespaceLockdown(input.Namespace); err != nil {
		return nil, err
	}
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		inScopeCharges, err := s.ListCharges(ctx, charges.ListChargesInput{
			Namespace:   input.Namespace,
			CustomerIDs: []string{input.CustomerID},
		})
		if err != nil {
			return nil, fmt.Errorf("list charges: %w", err)
		}
// ...
```

<!-- archie:ai-end -->
