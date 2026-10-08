# FBO consumption

FBO owns eligible credit source selection, consumption postings, expiry releases,
and advance coordination. Billing decides the amount and settlement policy;
the collector maps consume plans to billing allocations and commits their group.

A planning scope belongs to one customer and one caller database transaction.
Each successful consume plan reserves its selected subaccount/source-charge slices
immediately. Later plans subtract these reservations from the same persisted
balance queries, even when their posting times differ. Breakage bookkeeping is
written in that transaction and names the preassigned consumption entries.

`GroupInput` finalizes the scope and includes all its plans plus caller-provided
companion postings. The caller commits it once before committing the database
transaction. A failed plan invalidates the scope; the caller must roll back.
All FBO consumption in that group must be planned through the scope so its
reservations stay complete.

`SourceBalanceAsOf` retains the existing eligibility cutoff; `BookedAt` retains
the posting time. Scope reservations prevent reuse inside the group, but do not
implement the future minimum-balance calculation. Pending issuance/restoration
are not available as new sources in this consumption-only scope. Advances are
issued and consumed together, so they create no available source balance.

Source order remains priority, restriction, expiry, then stable source cursor.
Receivable coverage never creates advance. Accrued consumption may ask the
existing [advance service](../advance/README.md) to cover its shortfall.
