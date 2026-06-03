# service

<!-- archie:ai-start -->

> Orchestration layer for the charges domain: implements charges.Service by routing to flatfee, usagebased, and creditpurchase sub-services, owns the Create/AdvanceCharges/ApplyPatches lifecycle with a two-phase transaction design (creation TX then auto-advance TX), and manages gathering-line creation for invoiceable charges.

## Patterns

**transaction.Run wraps all multi-write operations** — Create, AdvanceCharges, and HandleCreditPurchaseExternalPaymentStateTransition wrap DB-touching logic in transaction.Run(ctx, s.adapter, func(ctx) ...) for atomicity. The adapter carries the TxCreator contract. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) { ... })`)
**validateNamespaceLockdown before every write** — Create and AdvanceCharges call s.validateNamespaceLockdown(namespace) before any DB work; mandatory for all state-changing service methods. (`if err := s.validateNamespaceLockdown(input.Namespace); err != nil { return nil, err }`)
**Two separate transactions for Create** — After the create TX commits, autoAdvanceCreatedCharges identifies customers with newly created credit-only charges and calls AdvanceCharges in a separate TX so creation state persists even if advancement fails. (`result, err := transaction.Run(ctx, s.adapter, func(ctx) ...)
return s.autoAdvanceCreatedCharges(ctx, result.charges)`)
**Fan-out to sub-services with index tracking** — Create fans out intents to flatFeeService.Create, usageBasedService.Create, creditPurchaseService.Create separately, collecting results with charges.WithIndex[T] to map back to original intent ordering. (`charges.WithIndex[charges.Charge]{Index: intent.Index, Value: charges.NewCharge(result.Charge)}`)
**chargesByType classifier for dispatch** — chargesByType in helpers.go classifies a charges.Charges slice into flatFees, usageBased, creditPurchase sub-slices for dispatch; panics on unknown charge types — new types must be added here. (`chargesByType, err := chargesByType(inScopeCharges.Items)
for _, charge := range chargesByType.flatFees { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Config, New constructor, dependency validation; all sub-service and adapter deps injected here, Config.Validate() enforces non-nil for required fields. | All deps (FlatFeeService, UsageBasedService, CreditPurchaseService, BillingService, FeatureService, MetaAdapter, Adapter) must be non-nil. |
| `create.go` | Implements charges.Service.Create — fans out to sub-services, collects gathering lines, post-commit triggers auto-advance and optional invoice-now for credit purchases. | The two-phase design (TX for creation + separate TX for auto-advance) is intentional; do not merge or crash-recovery loses created state. |
| `advance.go` | Implements AdvanceCharges — wraps in transaction.Run, lists non-final charges, dispatches to flatFeeService.AdvanceCharge or usageBasedService.AdvanceCharge by type. | Only CreditOnlySettlementMode flat fees are advanced here; non-credit-only flat fees are deliberately skipped. |
| `helpers.go` | chargesByType classifier and shared helpers used across create.go, advance.go, patch.go. | chargesByType panics on unknown charge types — add new charge types here before adding a new sub-service. |
| `base_test.go` | Shared BaseSuite wiring the full charges stack atop billingtest.BaseSuite; all service test suites embed it. | Line engines must be registered on s.BillingService via RegisterLineEngine before tests run — omitting causes silent no-ops during invoice processing. |

## Anti-Patterns

- Calling sub-service Create/Advance outside a transaction.Run — breaks atomicity across charge types
- Merging the create transaction and the auto-advance transaction — creation state lost if advancement panics mid-flight
- Adding billing-logic (gathering-line creation) directly to adapter methods — orchestration belongs in service
- Calling s.billingService methods without a prior validateNamespaceLockdown — bypasses namespace write protection
- Skipping RegisterLineEngine for a new charge type's engine in base_test.go — billing service produces silent no-op lines

## Decisions

- **Two separate transactions for Create: creation then auto-advance** — The balance-worker can retry advancement independently; losing the creation write to an advance panic would be unrecoverable. Separation makes both steps individually durable.
- **Service delegates per-type work to flatfee/usagebased/creditpurchase sub-services** — Each charge type has distinct state machines, realization logic, and handler callbacks; one inline service would be unmanageable and harder to test.

## Example: Implementing a new top-level charges.Service method spanning multiple charge types

```
func (s *service) NewOperation(ctx context.Context, input charges.NewOperationInput) (charges.Charges, error) {
    if err := input.Validate(); err != nil { return nil, err }
    if err := s.validateNamespaceLockdown(input.Namespace); err != nil { return nil, err }
    return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
        inScopeCharges, err := s.ListCharges(ctx, charges.ListChargesInput{Namespace: input.Namespace, CustomerIDs: []string{input.CustomerID}})
        if err != nil { return nil, fmt.Errorf("list charges: %w", err) }
        // ...
        return charges.Charges{}, nil
    })
}
```

<!-- archie:ai-end -->
