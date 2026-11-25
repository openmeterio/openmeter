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

## Incremental credit calculations

The credit checkpoint stores the booked FBO and advance balances, and each spendable
source in collection order. Every calculation still reads current charges, pending grants
and usage. In particular, tiered charges are rated on their current cumulative quantity.

- If the customer’s FBO and receivable accounts have the same number of entries and no
  known booking has become effective, the ledger projection can be reused as a whole.
- When the journal changes, RTE checks how many entries are booked at or before each
  saved boundary. Ledger postings are append-only, including corrections. A matching
  count proves the opening is intact. Starting at the newest such boundary, RTE adds
  booked FBO and advance movements through the calculation time.
- Spendability depends on grants and future reservations. RTE rereads the usual per-grant
  minima for every FBO subaccount with a posting after the opening, including future
  postings. It can carry the other sources forward in collection order.
- A backdated correction can rule out a recent opening while leaving an older one usable.
  If none qualify, RTE calculates from the full journal. If the journal changes during a
  forward calculation, RTE retries using the full calculation.

Checkpoint verification reads the source journal, so delayed or missing notifications do
not make an invalid opening reusable. A change of input still needs to trigger a calculation.
