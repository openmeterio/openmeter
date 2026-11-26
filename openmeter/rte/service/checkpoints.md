# Incremental balance calculations

A usage event can change a customer's live credit without changing a single ledger posting.
A new posting can in turn change the booked balance without rewriting the journal before it.
Both cases call for a fresh answer, but part of the previous calculation can provide the
starting point. Each calculator defines what to save and how to prove that it still applies.

## Which calls can reuse work

The worker enables checkpoints with `rte.incrementalCalculations`. The default is off.
When demand requires a calculation, the service offers the domain calculator the most recent
checkpoints for the same key, newest first, up to the calculation start time. The calculator
may use an older candidate, start from scratch, or compute some inputs again. It always returns
the complete current balance. A clean `EnsureFresh` and a plain `Get` still just reuse stored
state. An explicit `Calculate` always reads the sources. It leaves existing checkpoints
available to the next demand refresh, including when full calculations are replayed for comparison.

The client APIs and `rte.Calculator` contract stay the same. A calculator opts in by
implementing `rte.IncrementalCalculator`. RTE owns candidate selection, retention and the
atomic write. The domain owns the payload, its version and the decision to reuse it.

Invalidations keep their existing job of deciding which keys need work. They cannot establish
which historical inputs are unchanged. A late posting might already be in PostgreSQL while its
notification is still in Kafka, and temporary tracking can miss an event altogether. Verification
therefore uses the same domain sources the full calculation would use. It takes place during the
refresh. A caller reading a previously stored result gets the existing freshness guarantees.

## Two examples

**Credits.** Suppose RTE saved booked balances at 12:00 and 12:10. A posting entered at
12:12 with a booking time of 12:05 changes the history covered by the 12:10 checkpoint.
The calculator can still start from 12:00 if the count of immutable postings through that
boundary matches. It adds the journal movements since 12:00, recalculates spendability
for affected sources, and reads current charges and usage. Bookings exactly at 12:00 belong to the opening
balance. A new one at that boundary disqualifies the 12:00 checkpoint too.

**Metered entitlements.** Suppose four grant phases have different eligibility rules, but
the first three contained no events on the last run. One source query can verify that whole
interval is still empty. The grant engine still applies each phase's transitions and reads
the last phase's usage. If a late event has appeared in the earlier interval, the individual
queries run again. The calculator still starts from the entitlement engine's own valid grant
snapshot. [Metered checkpoint eligibility](../meteredentitlement/README.md)

**Dense metered usage** still follows the ordinary grant burn. To resume that burn in the
middle of a nonempty history, the opening would need the remaining amount per grant,
overage and period usage at a natural grant-engine boundary. It would also need evidence
that the starting grant snapshot, grant/reset history, conversion rules and usage in each
earlier phase still match. Moving usage from before to after a grant expires can change
which grant paid for it while leaving total usage unchanged.

## What is persisted

Each balance row has an optional JSONB timeline. A checkpoint contains a domain time boundary
and a versioned, calculator-specific payload. Existing rows start cold. RTE keeps up to four
distinct boundaries per key and skips payloads larger than 64 KiB. A newly calculated payload
replaces an earlier one for the same boundary. Calculators can discard a boundary when the
source disproves it, so future refreshes can try another. Ordinary reads skip this column.

An accepted balance write can replace or prune its timeline in the same transaction. Stable
hits leave the checkpoint column untouched. If two attempts finish out of order, the earlier
start cannot overwrite the later result or its history. The outbox and client carry the public
balance state. Apply the additive migration before starting checkpoint-capable workers.
Disabling the flag resumes ordinary calculations immediately. Keep the columns while any
checkpoint-capable binaries are still deployed.

## How to evaluate it

Compare `rte.calculation.duration` by kind and method. `full` runs the ordinary calculator.
`checkpoint_hit` means the domain reused a verified input. `checkpoint_advanced` identifies
credit calculations that applied new journal movements to a verified opening. `checkpoint_miss` includes the
cost of checking an unusable candidate or building the first checkpoint. Include misses when
assessing overall savings. A clean demand cache hit avoids calculation entirely and appears in
the [demand metrics](../metrics/README.md#instruments).

### Local measurements

Credit figures are median times for a dirty `EnsureFresh` followed by `Get`, including the
PostgreSQL state and outbox writes. Each variant ran 40 times against equivalent funded
customers in one currency, with distinct grant priorities and no usage-based charges.
The [credit timing test](../../../test/rte/README.md#credit-checkpoint-timings) reproduces the comparison.

| Credit workload | Full | Incremental |
| --- | ---: | ---: |
| 1 grant, unchanged journal | 14.6 ms | 12.5 ms |
| 32 grants, unchanged journal | 176.1 ms | 22.4 ms |
| 1 grant, credit funded before each refresh | 22.2 ms | 26.7 ms |
| 32 grants, credit funded before each refresh | 223.3 ms | 167.7 ms |

Journal verification covers the customer's accounts across currencies and feature scopes.
Activity in another currency can therefore send a clean projection through the more expensive
verification path. The funding workload changes one source each time. Repeated runs
showed the same direction of change, with substantial timing variation for 32 grants.

For metered entitlements, a separate 30-run test measured the ClickHouse reads across 12
grant phases. When usage appeared only in the last phase, the median fell from 25.8 ms to
4.4 ms (12 reads to 2). With usage in every phase it was 27.2 ms and 27.5 ms (12 reads
each). A late event that disproves an empty interval costs one additional probe on that
refresh. These figures cover meter reads only. Use worker timings to evaluate total latency.
