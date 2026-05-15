# customerbalance

<!-- archie:ai-start -->

> Computes customer credit balances by combining settled ledger account balances with pending charge impacts, exposing a Service interface and Facade for balance queries used by billing and the v3 credits API.

## Patterns

**Bounded vs unbounded impact split in pending balance** — chargePendingBalanceCalculator.CalculatePendingBalance separates CreditThenInvoice impact (bounded — capped at existing positive balance, cannot create credit debt) from CreditOnly impact (unbounded — can drive balance negative representing advance exposure). (`pendingBalance := applyBoundedAmount(bookedBalance, boundedAmount); return pendingBalance.Sub(unboundedAmount)`)
**Facade wrapping Service for validated external access** — Facade is a thin wrapper around Service that exposes GetBalances and GetBalance with validated inputs. NewFacade requires a non-nil Service. External callers (API handlers) must go through Facade, not Service directly. (`func NewFacade(service Service) (*Facade, error) { if service == nil { return nil, errors.New("service is required") }; return &Facade{service: service}, nil }`)
**Noop implementation for credits-disabled deployments** — noop.go provides a no-op Service implementation returning empty balances without touching the ledger. app/common wires this when credits.enabled=false. Must stay in sync with Service interface. (`// noop.go
type noopService struct{}
func (noopService) GetBalances(ctx context.Context, in GetBalancesInput) ([]BalanceByCurrency, error) { return nil, nil }`)
**RealizedCredits type-switches on ChargeType** — Impact.RealizedCredits() switches on meta.ChargeType (FlatFee vs UsageBased). CreditPurchase charges return zero realized credits because purchased credits are reflected in FBO balance, not in charge realizations. (`switch i.Type() { case meta.ChargeTypeFlatFee: charge, _ := i.AsFlatFeeCharge(); return charge.Realizations.CurrentRun.CreditRealizations.Sum() ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `calculation.go` | Impact type and chargePendingBalanceCalculator. Impact wraps charges.Charge with an amount and derives RealizedCredits, OutstandingAmount, BoundedAmount, UnboundedAmount. | RealizedCredits switches on ChargeType — if a new charge type is added to billing/charges, add a case here or it silently returns zero realized credits. |
| `facade.go` | GetBalancesInput, GetBalanceInput, BalanceByCurrency types, and Facade struct. Entry point for external callers including v3 API handlers. | GetBalanceInput.After is an optional TransactionCursor for pagination; callers that omit it get the full balance. |
| `service.go` | Service interface definition and NewService constructor. Composes loaders (funded_loader, ledger_loader) and chargePendingBalanceCalculator. | Service is the wiring boundary; app/common provides either the real implementation or noop from noop.go. |
| `noop.go` | No-op Service implementation for credits-disabled deployments. | Must be updated whenever the Service interface gains new methods; compile-time var _ assertion will catch this. |

## Anti-Patterns

- Instantiating the real Service when credits.enabled=false (must use noop)
- Calling Service methods directly without going through Facade when input validation is required
- Adding pending balance calculation logic outside calculation.go
- Treating CreditPurchase charges as having realized credits in RealizedCredits() — they return zero

## Decisions

- **Bounded vs unbounded impact split in CalculatePendingBalance.** — CreditThenInvoice credit consumption is bounded by the existing positive balance (cannot create credit debt); CreditOnly can drive balance negative representing advance exposure. Mixing them into a single deduction would misrepresent available credit.

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
