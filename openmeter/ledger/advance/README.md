# Advance backfill

Backfill attributes outstanding advance receivable and accrued value to purchased
credit. The package reads ledger balances and original legacy transactions and
returns attribution templates, consumed credit amounts, and exact legacy
segment selections. It does not maintain a separate advance balance.

Credit-purchase backfill orders collection origins by their original recording
time and ID, together with legacy advance roots in collection order. Capacity
comes from matching receivable and accrued routes. The ledger
sorts its inputs by original collection time and ID independently of query order,
and stops reading journals when the purchase amount is allocated. A partial
purchase exhausts an older eligible occurrence before funding a newer one;
tax treatment, feature eligibility, currency identity, and purchase cost basis
remain attached to the booked amounts. Corrections select the original spend
and posting route, including when a purchase or recognition group spans several
charges. This forward policy does not reconcile historical misallocations.

After accrued backfill, the remaining purchase attributes eligible outstanding
advance receivable even when matching accrued is absent. This receivable-only
attribution preserves spend and feature routes without creating accrued or
marking lineage as backfilled. Only the excess then becomes new credit.

Backfill requires a known purchase cost basis. Unknown-cost custom promotional
credit is issued without attributing outstanding advances.

The caller holds customer posting locks before planning and commits the templates
in the same database transaction as purchase issuance and legacy persistence.
It also coordinates breakage releases for the consumed credit. Planning does not
commit transactions or update legacy lineage.
