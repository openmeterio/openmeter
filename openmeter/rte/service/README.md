# Running a calculation

The service constructor registers a typed calculator and store for each supported kind. Go callers
use `CreditBalances()` or `MeteredEntitlements()` to get a calculation whose key and result types
match. The worker decodes event keys at its boundary. Callers already holding a typed key can use
the corresponding calculation directly.

`Calculate` validates the key, records a start time, and runs the exact calculator. It then opens
or joins the store transaction. If the store accepts the result, the service enqueues a
`CalculationUpdatedEvent` containing the state read back from storage. An enqueue failure rolls
back the write. The outbox publishes after commit and can deliver the same event more than once.
See [outbox delivery](../../watermill/outbox/README.md).

When two callers race, an older attempt can finish last. It receives the retained newer state and
emits no update. Two executions that produce equal amounts at successively later starts can both
be accepted. An update means RTE stored a calculation, even if its numeric value stayed the same.

`Get` only reads the store. A missing result is a typed not-found error, and a recorded zero is a
successful read. Call [the demand service](../demand/README.md) to require freshness or coalesce
work. An explicit calculation does not register interest or complete pending demand by itself.

The execution counter and duration histogram distinguish `updated`, `superseded` and `failed`.
Duration includes domain work and storing/enqueuing the result. The separate read counter
distinguishes `hit`, `missing` and `failed`. These labels let operators distinguish expensive
attempts from accepted writes and plain reads.
