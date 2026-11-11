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
