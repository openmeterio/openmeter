# collector

<!-- archie:ai-start -->

> Turns customer FBO credit (and advance) into accrued value, and unwinds it on correction, while preserving the exact order in which credit slices were collected so later breakage release/correction undo the same economic slices. The hard invariant is collection-order fidelity, not the FBO->accrued posting itself.

## Patterns

**Service splits into collector + corrector** — Service is implemented by struct{collector *accrualCollector; corrector *accrualCorrector}; CollectToAccrued routes to collector.collect, CorrectCollectedAccrued to corrector.correct. Build via NewService(Config) after Config.Validate(). (`return &service{collector: &accrualCollector{...}, corrector: &accrualCorrector{...}}, nil`)
**One DB transaction per flow** — collect and correct wrap their whole body in transaction.Run(ctx, c.transactionManager, run); source selection, ledger CommitGroup, breakage PersistCommittedRecords, and realization creation must share the transaction. (`return transaction.Run(ctx, c.transactionManager, run)`)
**Deterministic FBO collection order** — Sources sort by fboCollectionSource.Compare: creditPriority asc, featureRestricted first, expires_at asc (nil last via compareOptionalTime), then stable cursor asc. This ordering is a frozen contract. (`slices.SortStableFunc(sources, cmpx.Compare[fboCollectionSource])`)
**Lock FBO account before listing sources** — listCustomerFBOSources calls c.accountLocker.LockAccountsForPosting([]ledger.Account{customerAccounts.FBOAccount}) before ListSubAccounts to serialize concurrent collection. (`if err := c.accountLocker.LockAccountsForPosting(ctx, []ledger.Account{customerAccounts.FBOAccount}); err != nil { ... }`)
**Release breakage for consumed expiring sources** — For each selected source with a breakagePlan, call breakage.ReleasePlan with a NewCollectionSourceIdentityKey(idx) so release order matches collection order; append the release input and PendingRecord. (`releaseInput, releaseRecord, _ := c.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{Plan: *selection.source.breakagePlan, Amount: selection.amount, SourceKind: breakage.SourceKindUsage, SourceEntryIdentityKey: transactions.NewCollectionSourceIdentityKey(idx)})`)
**Credit-only shortfall becomes advance** — When SettlementMode==CreditOnly and FBO did not cover the amount, resolveAdvanceInputs issues a receivable then transfers FBO advance to accrued (IssueCustomerReceivableTemplate + TransferCustomerFBOAdvanceToAccruedTemplate). (`if shortfall := input.Amount.Sub(collectedInputs(inputs).collectedFBOAmount()); c.shouldAdvanceShortfall(input, shortfall) { ... }`)
**Coarse billing realizations from fine ledger entries** — toCreditRealizations buckets negative FBO entry amounts by SubAccountID (preserving order) into creditrealization.CreateAllocationInput; entry-level identity splits must not leak as separate realizations. (`amountsBySubAccountID[subAccountID] = amountsBySubAccountID[subAccountID].Add(entry.Amount().Abs())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `README.md` | Authoritative spec of collection order, source-entry identity, credit-only advance, advance backfill, correction unwind order, and the transaction-boundary invariant. | Read this before changing any ordering; correction uses reverse original collection order and must net with breakage plan/release/reopen exactly. |
| `collect.go` | accrualCollector.collect: resolves collected inputs, optionally issues advance, commits the group, persists breakage, returns CreateAllocationInputs. | Group annotations default to ledger.ChargeAnnotations when input.Annotations is nil; breakage persistence is skipped only when c.breakage is nil. |
| `collection_fbo.go` | FBO source listing, sorting (fboCollectionSource.Compare), and selection into fboCollectionSelection. | The TODO notes the Compare contract must be versioned before changing — existing entries/corrections/breakage assume it. compareOptionalTime treats nil expiry as sorting last. |
| `correct.go` | accrualCorrector.correct: plan-then-execute correction merging overlapping corrections, reissuing advance, reopening breakage. | plannedDirectInputs skip the merge/CorrectTransaction path; plannedTransactionCorrection.mergeKey dedupes by transaction namespaced ID. |
| `service.go` | Service interface, Config + Validate, input structs (CollectToAccruedInput, CorrectCollectedAccruedInput), constructor. | CollectToAccrued requires non-zero BookedAt and SourceBalanceAsOf — these are deliberately separate timestamps (booked time vs source-visibility time). |

## Anti-Patterns

- Changing fboCollectionSource.Compare ordering without versioning the contract — corrupts correction and breakage-release alignment for existing ledger data.
- Committing the ledger group, breakage records, or billing allocations outside the single transaction.Run boundary, leaving observable impossible intermediate states.
- Listing FBO sources without first LockAccountsForPosting, allowing concurrent double-collection of the same credit.
- Emitting per-entry credit realizations instead of bucketing by FBO sub-account, leaking ledger internals into billing.
- Selecting expiring sources without issuing the matching breakage ReleasePlan keyed by NewCollectionSourceIdentityKey(idx).

## Decisions

- **BookedAt and SourceBalanceAsOf are separate inputs** — A transaction booked at T1 may need to see credit/expiry state visible as of a later T5; conflating them would misselect sources.
- **Source entry identity records order, not amounts** — Billing allocations are coarser than ledger collection; amounts come from ledger entries, identity only preserves selection order so corrections/breakage can unwind the same slices.
- **Collection order is a frozen, versioned contract** — Corrections unwind in reverse and breakage releases attach to concrete source entries; any reorder would desync persisted ledger/breakage rows.

## Example: Collecting FBO into accrued and releasing breakage for consumed expiring sources

```
selections, _ := c.collectCustomerFBOSelections(ctx, c.customerID(input), input.Currency, input.FeatureKey, amount, input.SourceBalanceAsOf)
inputs, _ := transactions.ResolveTransactions(ctx, c.deps, c.resolutionScope(input), transactions.TransferCustomerFBOToAccruedTemplate{At: input.BookedAt, Currency: input.Currency, Sources: fboCollectionSelections(selections).postingAmounts()})
for idx, selection := range selections {
  if selection.source.breakagePlan == nil { continue }
  releaseInput, releaseRecord, _ := c.breakage.ReleasePlan(ctx, breakage.ReleasePlanInput{Plan: *selection.source.breakagePlan, Amount: selection.amount, SourceKind: breakage.SourceKindUsage, SourceEntryIdentityKey: transactions.NewCollectionSourceIdentityKey(idx)})
  inputs = append(inputs, releaseInput); pending = append(pending, releaseRecord)
}
```

<!-- archie:ai-end -->
