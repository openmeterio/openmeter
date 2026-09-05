# Legacy invoice retryability

## Scope and evidence

Assessment recorded on 2026-09-05 against
[#5082](https://github.com/openmeterio/openmeter/pull/5082), commit
`83bd2c50799b5571c25c99fdd55edd91548c94ac`, with the local regression inventory
on `feat/legacy-billing-dependency-recovery-tests`. This is an observed gap
inventory, not a claim that all recovery paths are supported.

The tests exercise the legacy `LineEngineTypeInvoice` engine without charge
associations. They use real billing services and PostgreSQL, with test meters,
streaming data, and the sandbox invoicing app. HTTP cases exercise real handlers
in process; they do not cover the full server router, authentication, or a UI.

The inventory contains 13 scenario families. The recorded run produced 19
passing and 12 failing leaf subtests, with no skips; the passing count includes
seven steps of the existing missing-meter recovery test. The configured linter
passed for both touched packages. Production code was not changed.

## Recovery contract under test

When a due line cannot resolve its feature or required meter, collection should
persist an invoice with critical, line-scoped validation issues and preserve
its line identities. The customer must be able to inspect the invoice, retry
after repairing dependencies, or delete the invoice while they remain broken.
Unresolved dependencies must prevent external issuance.

Retry should re-evaluate dependencies and snapshot usage again before issuing.
Partial repair should remove resolved issues while keeping unresolved issues
visible. Deletion should complete without needing the unavailable metering
dependency, and later collection must respect deleted invoice history.

The API tests additionally assert `statusDetails.failed = true` for
`draft.invalid_created`. The current implementation disagrees with that
assertion; the intended meaning of this flag needs to be resolved explicitly.

## Findings

### Missing features can cause calculation to panic

If a feature is missing before initial collection, the legacy engine returns
the line with an `invoice_line_feature_not_found` issue and unset quantities.
The calculation guard in [StandardInvoice](../stdinvoice.go),
`HasLineSnapshotValidationIssueForComponent`, recognizes missing-meter and
snapshot-failure codes but omits missing-feature issues.
[Detailed-line rating](../rating/service/detailedline.go) can consequently
dereference nil metered quantities before the invalid invoice is returned.

The same gap appears during partial repair. An invoice containing both a
missing feature and a missing meter can initially be persisted because the
missing-meter issue suppresses calculation for the engine. Repairing the
meter first leaves only the missing-feature issue, and retry panics. Repairing
the feature first, followed by the meter, succeeds through issuance.

The missing-feature creation failures prevent the associated retry/delete
journeys from reaching those actions. They do not independently prove that
deletion of an already-existing invoice with a missing feature fails.

When a feature disappears after a preliminary quantity snapshot already
exists, the tested collection-failure, repair, retry, and issuance path passes.

### Flat-price lines can lose missing-feature validation

[Gathering collection](../service/gatheringinvoicependinglines.go) logs
billability validation issues and continues with fallback eligibility.
[Legacy quantity snapshotting](../lineengine/quantitysnapshot.go) checks the
flat-price branch before looking up its feature dependency, so it does not
report the missing feature again.

The resulting invoice reaches `draft.manual_approval_needed` without the
critical issue. The tests expect `draft.invalid_created` instead. Valid
meterless features and flat-price lines without a feature reference pass.

### Invalid invoices expose a contradictory failure flag under the tested contract

The HTTP responses contain `extendedStatus = draft.invalid_created`, critical
validation issues, and retry/delete actions, but `statusDetails.failed = false`.
The `failedStatuses` classification in [StandardInvoice](../stdinvoice.go)
excludes this status.

The missing-meter HTTP cases still execute their complete recovery actions:
retry before repair retains the invalid invoice, repair and retry permit
issuance, and deletion succeeds without repair. These cases fail only their
failure-flag assertions. Decide whether validation-invalid invoices should
set this flag or whether clients should distinguish invalidity from operational
failure; neither interpretation should be inferred solely from the test name.

## Implementation handoff

The actionable checklist is in [TODO.md](../../../TODO.md), including root-cause
ownership, affected tests, acceptance criteria, working constraints, and test
commands. The 12 failures group into eight calculation-guard failures, two lost
flat-price validation failures, and two API-contract mismatches. The checklist
also calls out recovery steps that remain blocked by earlier failures.

## Scenario inventory

The numbers match the agreed test plan. A failed multi-step test can contain
successful earlier steps; the result column identifies where execution fails.

| # | Scenario and test | Observed result |
| --- | --- | --- |
| 1 | [Recovery suite][recovery]: `TestMissingFeatureRecoveryThroughIssuance` | Collection panics before an invalid invoice is returned; later recovery steps are not reached. |
| 2 | [Existing invoice suite][invoice]: `TestSnapshotQuantityInvalidDatabaseState` | Passes, including the added approval and issuance step, original line identity, amount, and one finalization call. |
| 3 | [Recovery suite][recovery]: `TestMultipleDependenciesPartialRepair` | Feature-first repair passes through issuance. Meter-first repair panics on retry. |
| 4 | [Recovery suite][recovery]: `TestSharedMissingDependencyKeepsLineIssues` | Missing-meter variant preserves distinct line issues across retries. Missing-feature variant panics at collection. |
| 5 | [Recovery suite][recovery]: `TestFlatPriceDependencyRecovery` | Missing feature produces no issue and reaches approval. Meterless-feature and no-feature controls pass. |
| 6 | [Recovery suite][recovery]: `TestDeleteWithBrokenDependency` | Missing-meter deletion passes and consumed gathering lines are not recreated. Missing-feature and flat-price variants cannot establish the expected invalid invoice. |
| 7 | [Recovery suite][recovery]: `TestDeleteRetryWithBrokenMeter` | External deletion failure is recoverable by repeating delete, without meter repair or changing the original deletion timestamp. |
| 8 | [Recovery suite][recovery]: `TestDependencyLostBeforeCollectionCompletes` | Feature and meter variants pass: preliminary quantity 7 becomes 10 after repair, and issuance uses the refreshed amount. |
| 9 | [Recovery suite][recovery]: `TestProgressiveRetryPreservesFinalBilling` | Passes: partial and final invoices preserve their split group and discounted totals of 18 and 36. |
| 10 | [Recovery suite][recovery]: `TestProgressiveDeletePreservesFinalBilling` | Passes: final collection remains usable and deleted partial amounts are excluded from prior billed amounts. |
| 11 | [Collection tests][collection]: `TestNotDueBrokenLinesDoNotBlockHealthyCollection` | Healthy due collection succeeds for both variants. Later collection of the broken line produces an invalid invoice for a missing meter but panics for a missing feature. |
| 12 | [Collection tests][collection]: `TestBrokenDependencyFallbackWaitsForInvoiceAt` | Both variants remain unsplit before `InvoiceAt`. At that boundary, the missing-meter variant passes and the missing-feature variant panics. |
| 13 | [HTTP suite][http]: `TestCustomerRetryWithBrokenDependency` and `TestCustomerDeleteWithBrokenDependency` | Missing-meter retry and delete actions work, but the failure-flag assertions fail. Missing-feature collection panics before either recovery action is reached. |

[recovery]: ../../../test/billing/dependency_recovery_test.go
[invoice]: ../../../test/billing/invoice_test.go
[collection]: ../../../test/billing/collection_test.go
[http]: ../httpdriver/invoice_dependency_recovery_test.go

## Limits of the recovery evidence

- Archiving a feature or soft-deleting a meter is not the same as losing it:
  [historical resolution](../featuremeter/README.md#historical-resolution)
  intentionally includes these resources. Tests inject physical feature loss
  or remove meters from the test adapter after valid line creation.
- Feature repair recreates the same key through the feature service. Meter
  repair restores the original meter identity through the test adapter. These
  fixtures do not establish a customer self-service repair API or permission
  model, or apply to charge-backed references pinned to feature IDs.
- Invoice deletion does not itself recreate the abandoned service period. The
  progressive deletion case verifies surviving-period collection and exclusion
  of deleted prior amounts, not rebilling of the deleted period or subscription
  reconciliation.
- The progressive scenarios use a sum meter, unit pricing, and a percentage
  discount. They do not establish correctness for every price, commitment, or
  discount combination.
- Gathering live preview, full worker delivery/retry behavior, and charge
  realization conflicts are outside this executed inventory.

## Reproducing the inventory

Run from the repository root with local PostgreSQL available. These commands
are expected to fail until the regression gaps or the disputed API contract
are resolved. See [Testing](../../../docs/development/testing.md) for setup.

```sh
direnv exec . env POSTGRES_HOST=127.0.0.1 go test -tags=dynamic \
  ./test/billing ./openmeter/billing/httpdriver \
  -run '^(TestLegacyDependencyRecovery|TestLegacyInvoiceDependencyHTTP)$' -count=1

direnv exec . env POSTGRES_HOST=127.0.0.1 go test -tags=dynamic \
  ./test/billing \
  -run '^TestInvoicing$/^TestSnapshotQuantityInvalidDatabaseState$' -count=1
```
