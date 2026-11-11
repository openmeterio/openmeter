# Routing source changes

The resolver translates a committed source fact into selectors for active calculation keys. For
push demand this can schedule a refresh. For pull-only demand it marks the key dirty for the next
pull. Resolving a fact does not activate keys.

| Source fact | Candidates |
| --- | --- |
| Ingest batch | Metered entitlements and credit balances related to the batch subject, meters and event times |
| Committed ledger group | Credit balances whose customer accounts and currency were touched by entries in the group, including other feature scopes in that currency |
| Charge inputs changed | Credit balances for the affected customer, across currencies and scopes |
| Entitlement created/deleted, grant created/voided, entitlement reset | The named metered entitlement |
| Customer or subscription lifecycle | That customer’s credit balances and metered entitlements |
| Feature or meter lifecycle | Metered entitlements using the feature and credit balances for matching usage-based charges |

Ingest routing uses the customer key or an active subject mapping and follows the meter to its
features. A usage timestamp must fall on or after the entitlement’s measurement start and before
its end/deletion, or within an eligible charge’s effective service period. Credit candidates
require an open usage charge settled by credit, the matching currency, and a compatible portfolio
or feature scope. The batch carries a set of meter slugs and raw event times, so selectors can
conservatively include extra work. A selector establishes that the result *may* have changed.

Use occurrence time for period matching. A batch stored today can carry yesterday’s usage.
Discarding it because an event with a later occurrence time was already seen can leave a balance
stale. [The worker](../worker/README.md) uses the batch storage time separately to ensure credit
rating can see the rows.

When adding a source, choose a fact emitted after its effects are durable, or enqueued in the same
transaction as those effects. A command may be consumed before the mutation it requests has
happened. Return no selectors when there are no candidates. A selector with just a namespace and
kind marks **every active key** of that kind. Keep emission coverage explicit. For example, the
billing [charge input event](../../billing/charges/README.md) covers root creates and patches.
Time-dependent changes need a refresh policy even when no source fact arrives.
