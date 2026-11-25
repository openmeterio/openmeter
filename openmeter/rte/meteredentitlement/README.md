# Metered entitlement balance keys

The namespaced entitlement ID selects one metered entitlement. A customer can hold multiple
entitlements, so a customer ID or feature key alone cannot select the state to read. Usage periods
are part of the result. The same key continues to identify the current balance after a reset.

RTE uses the complete [domain balance](../../entitlement/metered/balance.go). `Balance` and
`Overage` describe the point in time, `UsageInPeriod` and `StartOfPeriod` describe its usage
period, and `GrantBalances` preserves the per-grant breakdown. For example, consuming 25 units
against a 100-unit grant can yield a balance of 75 and usage of 25. Persisting only the remaining
75 would lose the usage and grant information consumers need.

A stored result has its own RTE calculation timestamp. The domain engine rounds the supplied
instant when evaluating usage. A newer stored timestamp can therefore describe the same usage
minute as the previous result.

RTE can read the engine’s existing snapshots and suppresses snapshot writes during its own
calculations. Late usage that predates a snapshot follows the same rules as a direct balance
read. Recalculating RTE state does not rebuild the entitlement’s snapshot history.

## Reusing empty grant phases

A grant becoming effective, expiring or recurring can split one usage period into several
meter queries. RTE saves a range when at least two consecutive phases had a zero count
or returned no SUM rows. The range ends at a completed minute. On a later refresh, one source query can establish
that the whole range still contains no matching events. The grant engine then applies the
usual transitions with zero usage for each covered phase and reads any remaining interval.
Customer attribution, meter filters and unit conversion come from the current entitlement.

For `COUNT`, the meter itself answers whether events exist. For `SUM`, matching positive and
negative usage can cancel, so only a source that can check raw events can certify the range.
A matching event discards the hint. A transient source error also sends the calculation
through its ordinary phase reads. RTE makes at most one such check per calculation. Single-phase reads and other aggregations
use their usual queries. [Balance and query-count comparison](../../../test/rte/metered_checkpoint_test.go)
