# customerbalance

<!-- archie:ai-start -->

> Customer-facing read model over ledger + billing activity: exposes credit balance at a point in time and a merged credit-transaction listing (funded/consumed/expired). It is a temporary bridge until the Real-Time-Engine (RTE) lands; it must keep API semantics stable while the implementation may stop querying the ledger directly.

## Patterns

**Facade validates and delegates to Service** — Facade wraps a Service; every method validates a typed input (GetBalancesInput/GetBalanceInput/ListCreditTransactionsInput Validate via models.NewNillableGenericValidationError) then calls the service. Build with NewFacade(service) (non-nil required). (`func (f *Facade) GetBalance(ctx, input GetBalanceInput) (alpacadecimal.Decimal, error) { if err := input.Validate(); err != nil { return ..., err }; ... return balance.Settled(), nil }`)
**Service depends on narrow local interfaces** — service depends on small interfaces (chargesService, creditPurchaseActivityService, subAccountLister, usageBasedTotalsService) plus ledger.AccountResolver/Ledger/BalanceQuerier/breakage.Service, all validated in Config.Validate and assembled in New. (`type chargesService interface { GetByIDs(...); ListCharges(...) }`)
**Default to current as-of when unfiltered** — GetBalance applies currentBalanceQuery, which sets AsOf=clock.Now() unless After or AsOf is already provided; balance = booked FBO settled + advance receivable settled, minus pending charge impacts. (`query = currentBalanceQuery(query) // sets AsOf = clock.Now() when neither After nor AsOf set`)
**Pluggable per-type transaction loaders** — creditTransactionLoaderFactories maps funded/consumed/expired to loader factories; creditTransactionLoaders returns all (in creditTransactionLoaderOrder) or one when a type filter is set. Each loader implements Load(ctx, creditTransactionLoaderInput). (`factory, ok := creditTransactionLoaderFactories[*txType]`)
**K-way merge by ledger cursor** — Per-type loader results are merged via mergeSortedLists with a max-heap keyed by ledger.TransactionCursor{BookedAt, CreatedAt, ID}; descending order is enforced by Less returning cmp(...) > 0. (`merged, hasMore := mergeSortedLists(lists, limit, compareCreditTransactionsByCursor)`)
**Expired rows come from breakage net impacts** — expiredCreditTransactionLoader reads service.Breakage.ListExpiredBreakageImpacts (plans - releases + reopens), not raw breakage transactions; zero-net groups are hidden; cursor is the newest contributing breakage transaction. (`result, _ := l.service.Breakage.ListExpiredBreakageImpacts(ctx, breakage.ListExpiredBreakageImpactsInput{...})`)
**NoopService when credits disabled** — NoopService implements Service returning zero balances/empty listings; New supplies a breakage Noop when Config.Breakage is nil. (`var _ Service = NoopService{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `README.md` | Spec of balance/asOf semantics, transaction listing types, expired projection (plan/release/reopen netting), cursor semantics, and the presentation boundary. | This package must NOT decide how plan/release/reopen net, FBO collection order, or correction unwind order — those belong to breakage and collector. |
| `service.go` | Service interface, dependency interfaces, Config/Validate, struct, GetBalance/GetFBOCurrencies; explicitly documented as not the RTE. | Pending balance is a current projection from open charges via balanceCalculator even when AsOf/cursor filters the booked side; chargeListPageSize=100. |
| `calculation.go` | Impact wrapper over charges.Charge and chargePendingBalanceCalculator; bounded (credit_then_invoice) vs unbounded (credit_only) amounts. | RealizedCredits skips IsVoidedBillingHistory runs; applyBoundedAmount clamps at zero (can't go negative) while unbounded credit_only can drive balance negative. |
| `facade.go` | Public entry: GetBalances/GetBalance/ListCreditTransactions with input validation and currency dedup. | GetBalanceInput rejects After+AsOf together; GetBalances falls back to GetFBOCurrencies when no currency filter is given. |
| `loaders.go` | creditTransactionLoader interface, ordering, and factory map per CreditTransactionType. | consumed reuses newLedgerCreditTransactionLoader with ListTransactionsCreditMovementNegative; adding a type requires updating both order slice and factory map. |
| `merge.go` | container/heap-based descending k-way merge and cursor extraction. | Less inverts cmp (>0) to produce descending order; limit<=0 returns empty with HasMore=false. |
| `transactions.go` | CreditTransactionType enum, ListCreditTransactions input/result/validation, CreditTransaction view model. | AsOf boundary must be applied before projection so future breakage entries don't leak; After/Before are mutually exclusive. |

## Anti-Patterns

- Computing expired amounts from raw breakage ledger transactions instead of ListExpiredBreakageImpacts net (plan - release + reopen) — duplicates or miscounts expiry.
- Letting RealizedCredits count IsVoidedBillingHistory runs, double-counting reversed billing against the customer balance.
- Leaking future-dated breakage/FBO entries past AsOf into balance or listing.
- Encoding FBO collection order or correction unwind order here — that logic belongs to collector/breakage.
- Constructing the service without validating its dependency interfaces (Config.Validate) or bypassing the NoopService when credits are disabled.

## Decisions

- **Implemented as a temporary balance bridge, not the RTE** — Balance currently queries the ledger per view; the package is structured so API semantics stay stable when the RTE replaces the implementation.
- **Listing is a merged read model with per-type loaders + heap merge** — Funded/consumed/expired come from different sources (charges, ledger movements, breakage impacts) but must page as one cursor-ordered stream.
- **Expired rows project net breakage impact with the newest contributing cursor** — Gives one stable customer-facing expiry row per bucket while preserving cursor pagination and hiding zero-net (immediately-backfilled) groups.

## Example: Computing settled balance from booked FBO + advance receivable minus pending charge impacts

```
bookedBalance, _ := s.BalanceQuerier.GetAccountBalance(ctx, customerAccounts.FBOAccount, ledger.RouteFilter{Currency: currency}, query)
advanceBalance, _ := s.BalanceQuerier.GetAccountBalance(ctx, customerAccounts.ReceivableAccount, ledger.RouteFilter{Currency: currency, CostBasis: mo.Some[*alpacadecimal.Decimal](nil)}, query)
impacts, _ := s.getChargePendingBalanceImpacts(ctx, customerID, currency)
settled := bookedBalance.Settled().Add(advanceBalance.Settled())
return balance{settled: settled, pending: s.balanceCalculator.CalculatePendingBalance(settled, impacts)}, nil
```

<!-- archie:ai-end -->
