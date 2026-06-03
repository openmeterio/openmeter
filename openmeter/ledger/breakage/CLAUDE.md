# breakage

<!-- archie:ai-start -->

> Credit-expiration breakage sub-domain: keeps future credit-expiration ledger entries aligned with actual customer credit usage via plan/release/reopen breakage records. It is an allocation/index layer over the ledger (the ledger remains the accounting source of truth); the breakage/adapter child persists LedgerBreakageRecord rows.

## Patterns

**Service produces TransactionInputs + PendingRecords** — PlanIssuance/ReleasePlan/ReopenRelease return ledger.TransactionInput(s) plus PendingRecord metadata; the caller commits the ledger group, then PersistCommittedRecords durably writes the records. (`txs, pending, err := svc.PlanIssuance(ctx, input); ...; svc.PersistCommittedRecords(ctx, pending, group)`)
**Config.Validate() before NewService** — NewService validates that Adapter, Dependencies.AccountService and Dependencies.AccountCatalog are non-nil before constructing the service. (`if err := config.Validate(); err != nil { return nil, err }`)
**Validate() on every input** — All *Input types Validate() using ledger.ValidateTransactionAmount/ValidateCurrency/ValidateCostBasis/ValidateCreditPriority; service methods validate before doing work. (`if err := input.Validate(); err != nil { return ... }`)
**Shared FBO-consumption == breakage-release ordering** — Plans, releases and reopens follow credit_priority asc, expires_at asc, stable cursor asc — the same order the FBO collector consumes credit. Diverging silently corrupts which expiry is released. (`ListPlans returns plans in the order the FBO collector must consume expiring credit`)
**Breakage-impact netting** — ListExpiredBreakageImpacts groups expired records by (expiresAt, currency) and computes impact = -(plans - releases + reopens); zero-impact groups are hidden and a negative internal total is an error. (`group.amount.Neg() with cursorID set to the smallest plan record ID`)
**NoopService for disabled breakage** — NoopService implements Service with all-nil/empty returns for deployments/tests wiring credit purchases without expiration support. (`var _ Service = NoopService{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface, Config + Validate, service struct holding adapter + transactions.ResolverDependencies, and the PlanIssuance/ReleasePlan/ReopenRelease inputs. | Dependencies.AccountService and AccountCatalog are mandatory; ImmediateReleaseAmount must be 0..Amount. |
| `breakage_impacts.go` | ListExpiredBreakageImpacts: nets plan/release/reopen records into customer-visible BreakageImpact rows with stable cursor windowing. | A negative netted amount or a group with no plan record is an error, not a silent skip; sorts by descending cursor. |
| `types.go` | Record, Plan, Release, PendingRecord, BreakageImpact and list input/result types. | All breakage record amounts are positive — the sign lives in the ledger entries. |
| `noop.go` | NoopService no-op implementation and NewNoopService constructor. | Must stay in sync with the Service interface so credits-disabled wiring compiles. |
| `README.md` | Authoritative spec of plan/release/reopen/advance-backfill semantics and the core ordering invariant. | Read before changing any breakage math — the correctness argument depends on the shared ordering. |

## Anti-Patterns

- Building ledger TransactionInputs by hand instead of going through transactions.* / the breakage service.
- Diverging breakage release order from the FBO collection order (credit_priority, expires_at, stable cursor).
- Treating breakage-generated FBO entries as normal credit issuance/usage instead of marking ledger.breakage.kind.
- Persisting records before the corresponding ledger group has committed (must use PersistCommittedRecords post-commit).
- Putting Ent/persistence access in this package instead of breakage/adapter — this layer is service logic only.

## Decisions

- **Breakage records are a positive-amount allocation/index layer separate from the ledger entries.** — The ledger stays the source of truth; records only let later flows find open plans, reopen releases, and project customer-visible expired credit.
- **Correctness relies on FBO consumption order matching breakage release order rather than grant-level lineage.** — Avoids per-grant lineage; the shared ordering makes the open planned amount at an expiry exactly the remaining unused expiring credit.

## Example: Plan breakage for newly issued expiring credit and persist after commit

```
txs, pending, err := svc.PlanIssuance(ctx, breakage.PlanIssuanceInput{
  CustomerID: cid, Amount: amount, Currency: currencyx.Code("USD"), ExpiresAt: e,
})
if err != nil { return err }
group, err := ledger.CommitGroup(ctx, transactions.GroupInputs(cid.Namespace, txs...))
if err != nil { return err }
return svc.PersistCommittedRecords(ctx, pending, group)
```

<!-- archie:ai-end -->
