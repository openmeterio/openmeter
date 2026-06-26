# breakage

<!-- archie:ai-start -->

> Credit-expiration breakage sub-domain: keeps future credit-expiration ledger entries aligned with actual usage via plan/release/reopen records, and projects customer-visible expired-credit impacts. The ledger is the source of truth; breakage records are an allocation/index layer whose correctness rests on the invariant 'FBO consumption order == breakage release order' (credit_priority asc, expires_at asc, stable cursor asc).

## Patterns

**Service returns ledger inputs, caller commits** — PlanIssuance/ReleasePlan/ReopenRelease return []ledger.TransactionInput + PendingRecord rather than committing; the caller owns the surrounding ledger transaction group so credit and breakage movement stay atomic. (`func (s *service) PlanIssuance(...) ([]ledger.TransactionInput, []PendingRecord, error)`)
**Validate-first on every input** — Each Input type has a Validate() that joins errors with errors.Join and uses ledger.Validate* helpers; service methods call input.Validate() before any work. (`if err := input.Validate(); err != nil { return nil, nil, err }`)
**Pending then persisted records** — Operations emit PendingRecord; durable rows are written only after the ledger commits via PersistCommittedRecords(pending, group). (`planRecord := PendingRecord{Record: Record{ID: planID, Kind: ledger.BreakageKindPlan, ...}}`)
**Config-validated constructor** — NewService(Config) validates Adapter + Dependencies (AccountService, AccountCatalog) before returning the service. (`if c.Adapter == nil { errs = append(errs, errors.New("adapter is required")) }`)
**Explicit noop implementation** — NoopService implements every Service method as a no-op for deployments/tests without expiration support; wiring picks it when breakage is disabled. (`func NewNoopService() Service { return NoopService{} }`)
**Net impact by (expiresAt, currency)** — ListExpiredBreakageImpacts groups records, computes plans - releases + reopens, hides zero groups, rejects negative totals, and exposes -(net) as the customer-visible amount. (`impact = -(plans - releases + reopens); group.amount.Neg()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `README.md` | Authoritative spec of plan/release/reopen semantics, ledger notation (FBO/BR/ACCRUED), and the consumption==release ordering invariant. | Read this before changing any breakage math — the correctness argument depends on collection order matching release order. |
| `service.go` | Service interface, Config + NewService, all *Input types and their Validate() methods, and the concrete service. | ReleasePlanInput requires SourceKind in {Usage,UsageCorrection,CreditPurchaseCorrection,AdvanceBackfill}; ReopenReleaseInput only allows correction kinds. |
| `breakage_impacts.go` | ListExpiredBreakageImpacts: nets records into BreakageImpact items, applies cursor window, sorts by descending cursor. | A negative netted group amount is a hard error (releases exceeded plans); a group with no plan record (empty cursorID) is also an error. |
| `noop.go` | NoopService — disabled breakage that returns empty results for every method. | Keep it in lockstep with the Service interface; a new interface method needs a noop here or wiring breaks. |
| `types.go` | Declares Adapter interface, Record/PendingRecord/Plan/Release and BreakageImpact types (referenced by service.go and the adapter). | Record.ValidateForReference requires ID, CustomerID, Currency, ExpiresAt, and both FBO + breakage sub-account IDs. |

## Anti-Patterns

- Committing ledger entries inside a breakage service method instead of returning TransactionInputs for the caller's group.
- Diverging release/reopen ordering from FBO collection ordering (credit_priority asc, expires_at asc, stable cursor asc) — breaks the core invariant.
- Persisting record rows before the corresponding ledger transactions commit instead of going through PersistCommittedRecords.
- Putting metadata (kind, source links) into route dimensions, or putting collection-eligibility fields (credit_priority) into annotations.
- Returning a negative or plan-less netted impact group instead of erroring — it signals corrupted accounting.

## Decisions

- **Breakage records are an index layer, not the accounting source of truth.** — The ledger holds the signed entries; records only let later flows find open plans, reopen releases, and project expired credit without grant-level lineage.
- **Plan/release/reopen rely on consumption-order == release-order instead of explicit grant lineage.** — Sharing one deterministic ordering lets a release reduce exactly the right expiry without tracking which grant the usage came from.

## Example: Netting expired breakage records into a customer-visible impact

```
switch record.Kind {
case ledger.BreakageKindPlan, ledger.BreakageKindReopen:
	group.amount = group.amount.Add(record.Amount)
	if record.Kind == ledger.BreakageKindPlan && (group.cursorID.ID == "" || record.ID.ID < group.cursorID.ID) {
		group.cursorID = record.ID
	}
case ledger.BreakageKindRelease:
	group.amount = group.amount.Sub(record.Amount)
}
// item.Amount = group.amount.Neg()
```

<!-- archie:ai-end -->
