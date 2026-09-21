# Advances

The service plans advance creation, backfill, and correction. It reads ledger
balances and original entries; it does not maintain a separate advance balance.
Callers hold customer posting locks and commit the plan and its bookkeeping in
one database transaction.

`PlanIssue` issues unknown-cost receivable and transfers the uncovered spend into
accrued. Both transactions share a fresh collection origin and spend charge.
The collector decides when a shortfall requires an advance.

## Backfill

Backfill attributes outstanding advance receivable and accrued value to purchased
credit. Planning returns attribution templates, consumed credit amounts, and
exact legacy segment selections.

Credit-purchase backfill orders collection origins by their original recording
time and ID, together with legacy advance roots in collection order. Capacity
comes from matching receivable and accrued routes. Candidate order does not
depend on query order. Legacy original transaction groups are loaded only until
the purchase amount is allocated. A partial purchase exhausts an older eligible
occurrence before funding a newer one;
tax treatment, feature eligibility, currency identity, and purchase cost basis
remain attached to the booked amounts. Corrections select the original spend
and posting route, including when a purchase or recognition group spans several
charges. This forward policy does not reconcile historical misallocations.

After accrued backfill, the remaining purchase attributes eligible outstanding
advance receivable even when matching accrued is absent. This receivable-only
attribution preserves spend and feature routes without creating accrued or
marking legacy lineage as backfilled. Only the excess is issued as available
FBO credit.

Backfill requires a known purchase cost basis. Unknown-cost custom promotional
credit is issued without attributing outstanding advances. Attribution is booked
at the earlier of purchase issuance time and the current time, so future-effective
purchases reserve their advance backing immediately. FIFO follows recording order,
not that effective timestamp.

The caller holds customer posting locks before planning and commits the templates
in the same database transaction as purchase issuance and legacy persistence.
It also coordinates breakage releases for the consumed credit. Planning does not
commit transactions or update legacy lineage.

## Correction

The collector selects amounts and funding sources and unwinds recognized earnings.
It passes original entry pairs and remaining reversible amounts to `PlanCorrection`.
Advance correction reverses the selected backfills, reopens their breakage releases,
restores the purchased credit to its original cost-basis and feature route, then
reverses the original advance collection and receivable issue. Restored credit is
not immediately used to backfill other advances. Cost basis and feature restrictions
come from the original attributed receivable route; credit priority comes from
the purchase group's `ledger.backfill.credit_priority` annotation, including when
the purchase was fully consumed by backfill and issued no ordinary FBO credit.

`PlanLegacyCorrection` uses original transaction groups for legacy histories.
It returns deferred template corrections so the collector can merge selections
against the same transaction before resolving them. Both paths return breakage
records for the caller to persist with the committed group; neither writes lineage.
