# Replacing credit-realization lineage with ledger provenance

## Decision

Use the journal as the source of both amounts and unwind history. New credit
collections carry an immutable `origin_id` on their ledger entries. Backfill,
recognition, and corrections preserve it. No new lineage roots or segment
snapshots are needed for these collections.

Keep the existing lineage reader/writer only for pre-cutover collections. Do
not guess missing attribution or rewrite historical monetary entries. This is
an additive rollout with a legacy drain, not a flag that permits old and new
writers to process the same origin interchangeably.

The implementation is an experiment with an additive storage migration. The
verification and cutover requirements below apply before deployment.

## Why an additional identity is necessary

Source charge identifies the purchase; spend charge identifies the consumer.
Neither distinguishes two collections by the same charge from the same
purchase. A usage-based charge can have several independently correctable runs.
The extra identity therefore starts at collection, per selected source slice,
and at advance creation for the paired receivable and accrued postings.

`origin_id` is required throughout that chain. Unrelated payment, issuance, and
legacy entries can have no collection origin. Making up an ID independently
for every missing field would conceal a broken propagation path.

An origin is not a route dimension, charge ID, idempotency key, or snapshot.
It does not identify an account. Source and spend provenance remain separate
because advance backfill adds a purchase identity after consumption occurred.
Reusing returned FBO credit starts a new collection origin.

| Fact | Authoritative representation |
| --- | --- |
| Original collection and selection order | Original negative FBO entries and collection order |
| Remaining accrued or recognized value | Sum of entries by origin, source, spend, and full route |
| Later purchase attribution | Cost-basis translation entries with the same origin |
| Recognition | Accrued-to-earnings entries with the same origin and source |
| Already reversed portion | Correction entries referencing original source entries |
| Expiry release and reopening | Existing breakage records and their journal postings |
| Charge tax, currency, and settlement identity | Immutable charge attribution and original routes |

Currency identity includes the managed custom-currency identity, not merely
its display code. Cost basis, its fiat currency, tax treatment, feature routes,
priority, and authorization stage must survive the applicable translations.

## Forward flows

1. Collection generates each origin before resolving entries. Both sides carry
   it, even when several source slices share an accrued subaccount. Billing
   realization granularity stays unchanged.
2. An advance's receivable issue and FBO-to-accrued transfer share one origin.
3. A purchase selects source-less receivable and accrued balances grouped by
   spend **and origin**. Partial backfill changes the source/cost-basis bucket
   for only the selected amount. No numeric side table is updated.
4. Recognition selects actual positive, known-cost accrued buckets with distinct
   source and spend charges. It preserves the origin and complete route on both
   legs. Invoice-backed and source-less amounts remain deferred, as today.
5. Each operation locks the customer's posting accounts before reading balances
   and holds those locks through journal and breakage persistence.

## Correction

Begin with the allocation's persisted transaction group and original source
order. For each selected origin, load its journal history and outstanding
amounts. Correct only entries belonging to that origin; a transaction group is
an atomicity boundary, not sufficient identity for choosing reversal legs.

Unwind recognized value before its backing accrued value. For backfilled
advance, unwind recognition, cost-basis translation, and receivable attribution,
then the original advance collection and issue. Reopen the matching purchase's
breakage release and return released purchased credit to its original FBO
route. Do not immediately redirect it to another advance.

Partial and repeated corrections subtract prior reversals before selecting
entries. A correction amount still comes from the owning billing operation; an
origin is not a retry key. Several origins in one batch cannot independently claim the same
remaining entry. All reads, postings, breakage changes, and billing realizations
share a transaction; a failed invariant aborts the whole operation.

Example: an advance of 10 is backfilled by purchase A for 4 and B for 3. With
7 recognized and 3 still uncovered, correcting 5 reverses exactly 5 of those
recognized/backfilled slices, returns the corresponding purchased credit, and
leaves 2 recognized plus 3 uncovered. It cannot reverse another run's earnings
even if that run has the same spend, currency, and cost basis.

## Core validation and storage

Add an immutable nullable `origin_id` column and a namespace/origin index.
Extend canonical entry identity serialization to v3; retain v1/v2 decoding.
The serialized identity and explicit column must agree. Reject empty or invalid
origin IDs and conflicting source/spend attribution within an origin-bearing
posting pair. Origin presence must be preserved across the relevant legs,
including corrections; balancing a transaction in aggregate is insufficient.

Expose origin filtering and grouping through ledger queries. Queries that omit
it preserve existing customer balance behavior. Journal traversal uses indexed
origin predicates and pagination rather than customer-wide history scans. It
hydrates only that origin's entries, so correcting several origins from one
large recognition batch does not repeatedly load the whole batch.

## Beta compatibility and migration

1. Apply the additive schema migration before deploying the writer. It adds no
   monetary entries and modifies no existing amounts, routes, or identities.
   Building the namespace/origin index scans the journal and blocks writes for
   its duration; schedule this step with the writer cutover.
2. Stop old application writers before enabling the new writer. Old binaries
   cannot preserve an identity they do not understand; mixed-version posting
   is not supported during this cutover.
3. New collections use provenance. Old collections continue through the legacy
   path. Purchases and recognition must keep those pools separate and update
   legacy state only for amounts actually posted against legacy origins.
   Nil-spend backfill is eligible only for roots whose original allocation
   entries actually lack spend provenance; an exhausted charge bucket cannot
   claim another advance's nil-spend backing.
4. Retain legacy tables during beta. Inspect active
   `credit_realization_lineage_segments` (`closed_at IS NULL`, positive amount),
   grouped through their roots by namespace, managed currency identity, and
   state. Inspect unmatched recognition/backfill histories before removal.
   A zero active count alone does not erase audit or future correction needs.
5. A later backfill may tag histories only where the mapping is uniquely
   provable from original entries, correction references, charge provenance,
   and lineage facts. Reconcile by complete route and require exact equality.
   Ambiguous historic batch attribution stays legacy and requires a separately
   reviewed repair. Assigning an arbitrary origin does not recover lost facts.

Application rollback after origin-bearing writes requires a version that can
read and preserve v3 identities. Schema rollback is unsafe after such writes;
retain the additive column and fix forward. Do not drop lineage tables until
no active legacy history depends on them and its audit data is retained.

## Evidence from the lifecycle experiment

The two-charge lifecycle regression exposes why lineage amounts cannot be the
new source of truth. A purchase of 25 is posted proportionally across two
20-credit advances: 12.5 per charge. The previous lineage expectations assign
20 to the older charge and 5 to the newer one. Correcting the newer charge must
follow its actual journal backing. The regression now pins both source/spend
balances before correction and preserves the other charge's exact backing.

## Tradeoffs and acceptance

This removes duplicate mutable accounting amounts and segment transitions at
the cost of more precise ledger queries. An origin bounds traversal to one
collection's lifecycle, and balance queries retain database aggregation.
Pathologically long histories may later justify a rebuildable projection;
they do not justify a second authoritative accounting state now.

Required verification includes source splitting on identical routes, several
runs of the same charge, multiple spends/cost bases recognized together,
partial/repeated correction, partial/multiple advance backfills, expiry
release/reopen, custom currencies sharing a code, historical time boundaries,
mixed legacy/new balances, retries, rollback, and concurrent correction versus
recognition/backfill. Existing ledger/charge suites and generated-schema and
migration checks must pass before the experiment is considered deployable.

The public custom-currency balance/history and purchase API changes remain
independent. This change preserves their transaction templates and accounting
visibility contracts. Invoice-backed recognition policy is a separate feature.
