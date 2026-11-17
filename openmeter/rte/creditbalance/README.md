# Credit balance keys

A credit balance belongs to a customer in a namespace, in one currency, with one eligibility
filter. Fiat uses the currency code. For custom currencies supply the currency reference with its
ID. Two custom currencies that reuse a display code have separate balances.

The table shows how the JSON field `featureFilter` selects a balance for a customer with
20 unrestricted credits, 10 credits restricted to `search`, and no usage.

| `featureFilter` | Eligible credit | Amount in the example |
| --- | --- | --- |
| omitted or `null` | All grants, regardless of restrictions | 30 |
| `[]` | Unrestricted grants only | 20 |
| `["search"]` | Unrestricted grants plus grants eligible for `search` | 30 |

The two 30s identify independent views. A grant restricted to another feature changes only the
portfolio in this example. A live usage charge tied to `search` can affect the portfolio and
`search` views. Build keys with the existing `customerbalance` filter constructors. Preserve the
distinction between an absent and a present empty filter when encoding a request or comparing
identities.

The result contains the domain calculator’s four decimal amounts, `Booked`, `Spendable`, `Live`
and `Pending`. The calculation time belongs to the enclosing RTE state. See
[customer credit balance semantics](../../ledger/customerbalance/README.md) for allocation,
advance debt and the difference between amounts.
