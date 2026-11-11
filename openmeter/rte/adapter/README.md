# PostgreSQL storage

RTE keeps the latest successful result and the interest in future results separately. This allows
`Get` to return the last calculation after a caller deactivates a key or its pull interest
expires.

| Record | Identity | Stored information |
| --- | --- | --- |
| Credit balance | Namespace, customer ID, currency identity, feature scope | Calculation start and four decimal amounts |
| Metered entitlement balance | Namespace and entitlement ID | Calculation start and the complete domain balance as JSON, including per-grant amounts |
| Demand | Namespace, kind and hash of the semantic key | Decodable key, owner, active interest, dirty/completed versions, schedule and retry state |
| Invalidation receipt | CloudEvents source and ID | A fact already applied to matching demand |

For either balance store, an atomic upsert accepts a calculation whose start is strictly later
than the existing row. Even an equal timestamp retains the original result. The store reads back
the winning row in the transaction. Callers must use both the returned state and the `Updated`
flag to decide what to publish. Comparing clocks in different replicas requires synchronized
clocks.

The demand key hash comes from the decoded key dimensions. JSON field order or formatting cannot
create a second demand row for the same key. The decoded key, stored alongside the hash, lets a
worker dispatch work after a restart. Source selectors use the owner to narrow updates and the
hash to select an exact balance.

The repository keeps a demand row locked through the callback, including external usage reads.
Background workers skip locked keys. An explicit pull for the same target waits for the owner.
Always propagate the callback context into the calculation so the result, update outbox record and
completion share the transaction. See [demand processing](../demand/README.md) for timeout,
rollback and receipt retention.
