# customerbalance

<!-- archie:ai-start -->

> Computes customer credit balances by combining settled ledger account balances with pending charge impacts, exposing a Service interface and Facade for balance queries used by billing and the v3 credits API.

## Patterns

**Bounded vs unbounded impact split in pending balance** — chargePendingBalanceCalculator.CalculatePendingBalance separates CreditThenInvoice impact (bounded — capped at existing positive balance) from CreditOnly impact (unbounded — can drive balance negative as advance exposure). (`pendingBalance := applyBoundedAmount(bookedBalance, boundedAmount); return pendingBalance.Sub(unboundedAmount)`)
**Facade wrapping Service for validated access** — Facade is a thin wrapper exposing GetBalances / GetBalance with validated inputs; NewFacade requires a non-nil Service. API handlers must use Facade, not Service directly. (`func NewFacade(service Service) (*Facade, error) { if service == nil { return nil, errors.New("service is required") }; return &Facade{service: service}, nil }`)
**Noop implementation for credits-disabled deployments** — noop.go provides a no-op Service returning empty balances without touching the ledger; app/common wires it when credits.enabled=false. Must stay in sync with Service interface. (`func (noopService) GetBalances(ctx context.Context, in GetBalancesInput) ([]BalanceByCurrency, error) { return nil, nil }`)
**RealizedCredits type-switches on ChargeType** — Impact.RealizedCredits() switches on meta.ChargeType (FlatFee vs UsageBased). CreditPurchase charges return zero realized credits because purchased credits live in FBO balance, not in realizations. (`switch i.Type() { case meta.ChargeTypeFlatFee: charge, _ := i.AsFlatFeeCharge(); return charge.Realizations.CurrentRun.CreditRealizations.Sum() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `calculation.go` | Impact type and chargePendingBalanceCalculator; Impact derives RealizedCredits, OutstandingAmount, BoundedAmount, UnboundedAmount from a charge plus amount. | RealizedCredits switches on ChargeType — adding a new billing/charges charge type without a case here silently returns zero realized credits. |
| `facade.go` | GetBalancesInput, GetBalanceInput, BalanceByCurrency, and the Facade struct; entry point for v3 API handlers. | GetBalanceInput.After is an optional TransactionCursor for pagination; omitting it returns the full balance. |
| `service.go` | Service interface and NewService, composing funded_loader, ledger_loader and chargePendingBalanceCalculator. | Service is the wiring boundary; app/common provides real impl or noop.go depending on credits.enabled. |
| `noop.go` | No-op Service for credits-disabled deployments. | Must be updated when the Service interface gains methods; the compile-time var _ assertion catches misses. |

## Anti-Patterns

- Instantiating the real Service when credits.enabled=false (must use noop)
- Calling Service methods directly without going through Facade when input validation is required
- Adding pending-balance calculation logic outside calculation.go
- Treating CreditPurchase charges as having realized credits in RealizedCredits() — they return zero

## Decisions

- **Bounded vs unbounded impact split in CalculatePendingBalance.** — CreditThenInvoice consumption is bounded by existing positive balance (cannot create credit debt); CreditOnly can drive balance negative as advance exposure. Mixing them into a single deduction would misrepresent available credit.

## Example: Computing pending balance impact for a charge

```
impact, err := customerbalance.NewImpact(charge, chargeAmount)
if err != nil { return err }
// BoundedAmount is capped at existing balance (CreditThenInvoice)
boundedDelta := impact.BoundedAmount()
// UnboundedAmount can exceed balance (CreditOnly advance)
unboundedDelta := impact.UnboundedAmount()
```

<!-- archie:ai-end -->
