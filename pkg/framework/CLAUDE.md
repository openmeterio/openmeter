# framework

<!-- archie:ai-start -->

> Structural namespace owning the codebase's foundational framework primitives: the transaction abstraction (transaction, entutils) every adapter builds on, the bottom-up HTTP operation/handler stack (operation, transport/httptransport, commonhttp), distributed locking (lockr), the Postgres driver (pgdriver), and observability wrappers (tracex, clickhouseotel). The constraint that binds all children: these are domain-agnostic foundations — nothing here may import openmeter/* domain packages.

## Patterns

**Bottom-up operation then handler (README)** — Business logic is an operation.Operation[Request,Response] func; transport is wired separately via DecodeRequest/EncodeResponse/EncodeError and httptransport.NewHandler. Children operation/, transport/httptransport/, and commonhttp/ each own one stage of this chain. (`httptransport.NewHandler(op, DecodeReq, EncodeResp, EncodeErr, WithErrorHandler(eh), WithOperationName("op"))`)
**Transaction foundation flows up from transaction -> entutils -> adapters** — transaction/ defines Driver + Run/RunWithNoValue and context-carried Driver reuse; entutils/ wraps it in TransactingRepo + TxDriver (savepoints, HijackTx). Domain adapters consume the entutils layer, never Ent's native client.Tx hooks. (`entutils.TransactingRepo(ctx, repo.db, func(ctx, tx) (T, error) { ... })`)
**Domain-agnostic, dependency-magnet foundations** — These packages are imported by nearly the entire tree (entutils 74 in-edges, transaction 70, httptransport 47) but import only stdlib, third-party, and sibling pkg/* utilities — never openmeter/* domains. (`lockr, pgdriver, tracex, operation all consume only pkg/* and external libs`)
**Constructor + Validate config across children** — Stateful primitives (lockr Locker/SessionLocker, pgdriver Driver, clickhouseotel tracer/metrics) are built via New* constructors that validate a Config struct and reject missing deps — never construct the concrete types directly. (`NewLocker(cfg) / NewPostgresDriver(opts...) / clickhouseotel.New(cfg)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `README.md` | Canonical recipe for adding an operation: define Request/Response + operation func, then Decode/Encode/EncodeError functions, then a NewXHandler constructor calling httptransport.NewHandler. | EncodeOperationError must return true only when it fully handled the error; returning false passes the error to the outer ErrorHandler. |

## Anti-Patterns

- Importing any openmeter/* domain package from this tree — the framework layer must stay domain-agnostic so every domain can depend on it.
- Bypassing the operation->handler split by putting business logic directly in an http.Handler instead of an operation.Operation wired through httptransport.NewHandler.
- Reaching past entutils to Ent's native client.Tx / onCommit hooks for shared transactions — the savepoint-based entutils/transaction layer is the only sanctioned path.
- Constructing lockr/pgdriver/clickhouseotel primitives directly instead of via their validating New* constructors, skipping dependency and Config validation.
- Using context.WithTimeout to bound PG lock or query waits where the children expose Postgres-side lock_timeout (lockr, pgdriver) instead.

## Decisions

- **framework is a structural folder with only README.md at top level; all real code lives in single-purpose child packages.** — Keeps each foundational concern (transaction, transport, locking, driver, tracing) independently importable and prevents a god-package that every domain would import wholesale.
- **The transport stack is split into operation (logic), httptransport (decode/operate/encode), and commonhttp (shared encoders/error mapping) rather than one HTTP package.** — Lets business operations stay transport-free and reusable while RFC-7807/ValidationIssue mapping lives in one shared place.

<!-- archie:ai-end -->
