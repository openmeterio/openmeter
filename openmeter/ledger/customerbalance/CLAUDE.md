# customerbalance

<!-- archie:ai-start -->

> Computes customer credit balances by combining settled ledger account balances with pending charge impacts, exposing a Service interface and Facade for balance queries used by billing and the v3 credits API.

## Patterns

**Impact pending balance calculation** — chargePendingBalanceCalculator.CalculatePendingBalance takes a booked balance and []Impact and applies bounded (CreditThenInvoice, capped at balance) and unbounded (CreditOnly, can go negative) amounts to produce a pending balance. (`pendingBalance := applyBoundedAmount(bookedBalance, boundedAmount); return pendingBalance.Sub(unboundedAmount)`)
**Facade wrapping Service** — Facade is a thin wrapper around Service that exposes GetBalances and GetBalance with validated inputs. NewFacade requires a non-nil Service. (`func NewFacade(service Service) (*Facade, error) { if service == nil { return nil, errors.New("service is required") }; return &Facade{service: service}, nil }`)
**Noop implementation** — noop.go provides a no-op Service used when credits.enabled=false; it returns empty balances without touching the ledger. Wired in app/common when credits are disabled. (`// noop.go implements Service returning zero balances`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `calculation.go` | Impact type and chargePendingBalanceCalculator. Impact wraps a charges.Charge with an amount and computes RealizedCredits, OutstandingAmount, BoundedAmount, UnboundedAmount. | RealizedCredits switches on ChargeType (FlatFee vs UsageBased); CreditPurchase charges return zero realized credits. |
| `facade.go` | GetBalancesInput, GetBalanceInput, BalanceByCurrency types and Facade struct. Entry point for external callers (API handlers). | GetBalanceInput.After is an optional TransactionCursor for pagination. |
| `service.go` | Service interface definition and constructor. Composes loaders (funded_loader, ledger_loader) and calculator. | Service is the boundary for wiring; app/common provides either a real or noop implementation. |
| `noop.go` | No-op implementation used when credits are disabled. | Must stay in sync with Service interface signature. |

## Anti-Patterns

- Instantiating a real Service when credits.enabled=false (must use noop from noop.go)
- Calling service methods directly without going through Facade when input validation is required
- Adding balance calculation logic outside calculation.go (keep Impact and pending-balance math co-located)

## Decisions

- **Bounded vs unbounded impact split in CalculatePendingBalance.** — CreditThenInvoice credit consumption is bounded by the existing positive balance (cannot create credit debt); CreditOnly can drive balance negative representing advance exposure.

<!-- archie:ai-end -->
