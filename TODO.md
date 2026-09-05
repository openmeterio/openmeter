# Legacy billing dependency recovery: implementation handoff

## Objective

Complete the customer journey for legacy invoice lines whose feature or meter
becomes unavailable: create an invoice with visible validation errors, retry
after repairing dependencies, or delete the invoice while dependencies remain
broken. Repair must re-snapshot usage and allow correct issuance.

This file is the implementation checklist. The detailed evidence and coverage
limits are in [retryability.md](openmeter/billing/docs/retryability.md).

## Starting state and working constraints

- Worktree: `/Users/turip/.codex/worktrees/02d4/openmeter`.
- Branch: `feat/legacy-billing-dependency-recovery-tests`.
- Base: [PR #5082](https://github.com/openmeterio/openmeter/pull/5082), commit
  `83bd2c50799b5571c25c99fdd55edd91548c94ac`. The original comparison was
  [PR #5045](https://github.com/openmeterio/openmeter/pull/5045).
- The regression inventory and documentation are committed on this branch.
  Start from the draft PR branch so the tests and evidence stay together.
- Production code has not been changed. The tests intentionally assert the
  desired recovery behavior and currently include failures.
- Read [AGENTS.md](AGENTS.md) and applicable skills before implementation.
  Preserve user-owned changes. Prefix every Git CLI invocation with
  `GIT_OPTIONAL_LOCKS=0`.
- Keep follow-up changes separate from this inventory unless they are needed to
  fix an item in this checklist.
- Retain explicit customer actions in test bodies and separate retry/delete
  journeys. Do not skip or weaken failing tests to hide implementation gaps.
- Item 3 needs an explicit contract decision; do not silently choose semantics.

## Verified baseline

The 2026-09-05 run produced **19 passing and 12 failing leaf subtests, no skips**.
The passing count includes seven steps of the existing missing-meter test.
Formatting and the full configured linter passed for both touched packages.

| Root cause | Currently failing leaf cases | Classification |
| --- | ---: | --- |
| Missing-feature issue omitted from calculation guard | 8 | Implementation bug |
| Flat-price snapshotting bypasses feature validation | 2 | Implementation bug |
| API failure flag disagrees with test expectation | 2 | Contract decision |

Counts classify the first observed failure. Fixing creation can expose later
failures; eight removed panics do not necessarily mean eight passing journeys.

## 1. Missing-feature calculation guard

- [ ] Fix calculation blocking for incomplete missing-feature snapshots.

**Cause:** `HasLineSnapshotValidationIssueForComponent` in
[stdinvoice.go](openmeter/billing/stdinvoice.go) recognizes missing-meter and
snapshot-failure issues but omits `ErrInvoiceLineFeatureNotFound`.
[Invoice calculation](openmeter/billing/service/invoicecalc/details.go) then
allows [rating](openmeter/billing/rating/service/detailedline.go) to dereference
nil quantities. Repairing a meter first on an invoice with two broken
dependencies also exposes this omission.

**Implementation direction:** recognize the missing-feature snapshot issue in
the calculation guard. Preserve protection when retry downgrades old critical
issues to warnings; quantities remain incomplete until snapshotting succeeds.
Keep the line-scoped validation issue instead of converting the failure into
a generic request error or treating missing usage as zero.

**Acceptance criteria:**

- Collection persists `draft.invalid_created` with critical line issues and
  retry/delete actions instead of panicking.
- Unrepaired retry stays usable and does not accumulate duplicate issues.
- Either partial-repair order removes only resolved issues.
- Complete repair preserves invoice/line identities, refreshes quantities,
  and allows correct issuance. Deletion remains available without repair.

**Eight affected cases:**

| Test method | Variant |
| --- | --- |
| `TestMissingFeatureRecoveryThroughIssuance` | Entire journey |
| `TestMultipleDependenciesPartialRepair` | `meter_first` |
| `TestSharedMissingDependencyKeepsLineIssues` | `feature` |
| `TestDeleteWithBrokenDependency` | `feature` |
| `TestNotDueBrokenLinesDoNotBlockHealthyCollection` | `feature`, once the broken line becomes due |
| `TestBrokenDependencyFallbackWaitsForInvoiceAt` | `feature`, at `InvoiceAt` |
| `TestCustomerRetryWithBrokenDependency` | `feature` |
| `TestCustomerDeleteWithBrokenDependency` | `feature` |

## 2. Flat-price feature validation

- [ ] Validate supplied feature references on flat-price lines.

**Cause:** [gathering collection](openmeter/billing/service/gatheringinvoicependinglines.go)
logs billability validation issues and continues with fallback eligibility.
The early flat-price branch in
[quantitysnapshot.go](openmeter/billing/lineengine/quantitysnapshot.go) bypasses
the feature lookup, so standard-invoice validation never reports the issue
again. The invoice reaches `draft.manual_approval_needed` with no critical issue.

**Implementation direction:** validate a supplied feature reference even when
quantity is fixed. A flat-price line does not require a meter; preserve valid
meterless features and lines without a feature reference.

**Acceptance criteria:**

- A missing referenced feature persists a critical line issue and blocks approval.
- Repair and retry remove the issue and allow issuance for the original amount.
- Deletion works while the feature remains unavailable.
- The meterless-feature and no-feature controls remain passing.

**Two affected cases:** `TestFlatPriceDependencyRecovery/missing_feature` and
`TestDeleteWithBrokenDependency/flat_feature`.

## 3. API failure-flag contract

- [ ] Decide whether validation-invalid invoices count as `statusDetails.failed`.
- [ ] Align implementation, tests, and documentation with that decision.

**Cause:** `failedStatuses` in [stdinvoice.go](openmeter/billing/stdinvoice.go)
excludes `draft.invalid_created`. The HTTP tests expect `failed = true`, while
responses expose `false` alongside critical issues and retry/delete actions.

**Decision required:** if validation-invalid invoices should count as failed,
align status classification. Otherwise document how clients distinguish
invalidity from operational failure and update only the disputed expectation.
Do not remove validation-issue or available-action assertions.

**Acceptance criteria:** collection, list, detail, and unrepaired retry responses
consistently express the chosen contract. Recovery and deletion still complete.

**Two affected cases:** `TestCustomerRetryWithBrokenDependency/meter` and
`TestCustomerDeleteWithBrokenDependency/meter`. Both journeys already complete;
only the failure-flag assertions fail. Missing-feature HTTP cases may expose
the same mismatch after item 1 removes their creation blocker.

## 4. Completion verification

- [ ] Re-run the inventory after items 1 and 2 and record newly reachable failures.
- [ ] Verify missing-feature repair/retry/issuance and deletion through services
  and HTTP, including partial repair and flat-price recovery.
- [ ] Keep existing missing-meter recovery, repeated external deletion,
  dependency loss after preliminary snapshotting, and progressive retry/delete
  flows passing.
- [ ] Run formatting and the configured linter for every touched package.
- [ ] Update the assessment with results and remaining gaps.

There is currently no independent evidence for a separate deletion-engine or
progressive-billing fix. Feature deletion tests fail while creating the invalid
invoice, before reaching the delete action.

## Test locations and commands

- [Service journeys](test/billing/dependency_recovery_test.go):
  `TestLegacyDependencyRecovery` suite.
- [Collection timing](test/billing/collection_test.go): additional methods on
  the same recovery suite.
- [Existing missing-meter journey](test/billing/invoice_test.go):
  `TestInvoicing/TestSnapshotQuantityInvalidDatabaseState`.
- [HTTP journeys](openmeter/billing/httpdriver/invoice_dependency_recovery_test.go):
  `TestLegacyInvoiceDependencyHTTP` suite.

Run from the repository root with the configured Go toolchain. PostgreSQL tests
require host/unsandboxed database access and `POSTGRES_HOST=127.0.0.1`; without
that variable, suites can silently skip.

```sh
POSTGRES_HOST=127.0.0.1 go test -json -tags=dynamic \
  ./test/billing ./openmeter/billing/httpdriver \
  -run '^(TestLegacyDependencyRecovery|TestLegacyInvoiceDependencyHTTP)$' -count=1

POSTGRES_HOST=127.0.0.1 go test -json -tags=dynamic ./test/billing \
  -run '^TestInvoicing$/^TestSnapshotQuantityInvalidDatabaseState$' -count=1

golangci-lint run -v ./test/billing/... ./openmeter/billing/httpdriver/...
```

Save complete output outside the repository and record exit status. Use the
repository formatter for touched files. Prefer `direnv exec .` when it can
initialize safely. In this worktree, shell startup attempted to modify hooks
in `/Users/turip/src/openmeter` and automatic approval review rejected it.
Validation instead used cached tool binaries without running startup hooks.
The temporary helper `/tmp/legacy_cached_tools.py` performed that setup; inspect
it before reuse and do not assume it exists in another environment.

Local baseline logs, also temporary:
`/tmp/legacy-readability-inventory.jsonl`,
`/tmp/legacy-readability-existing-meter.jsonl`, and
`/tmp/legacy-readability-lint.log`.

## Evidence boundaries

These tests use real billing services and PostgreSQL, test meters/streaming,
and a sandbox invoicing app. HTTP tests invoke handlers in process, without
the full router, authentication, workers, or UI. Lines explicitly use the
legacy invoice engine and have no charge association.

Feature archival preserves historical resolution, so fixtures physically remove
features and repair by recreating the same key. Meter repair restores the
original identity through the test adapter. This does not establish a customer
self-service dependency-repair API. Progressive deletion verifies surviving
period billing and exclusion of deleted amounts, not automatic rebilling of
the deleted period. Do not silently expand scope to those untested behaviors.
