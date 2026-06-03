# entutils

<!-- archie:ai-start -->

> Leaf-level Ent ORM infrastructure shared by every domain adapter: ctx-propagated transaction reuse (TransactingRepo / TxDriver with savepoint nesting), the standard schema mixins (IDMixin ULID char(26), NamespaceMixin, TimeMixin, ResourceMixin), Postgres value types, and a suite of Ent code-generation extensions. Its primary constraint: it must remain a domain-free leaf and never import openmeter/* packages, or it creates import cycles across all adapters.

## Patterns

**TransactingRepo on every adapter method body** — Adapter methods wrap their Ent access in entutils.TransactingRepo(ctx, repo, cb) or TransactingRepoWithNoValue. The helper calls GetDriverFromContext; on a *transaction.DriverNotFoundError it falls through to repo.Self(), otherwise it rebinds via repo.WithTx(ctx, tx). Never touch a.db directly outside this wrapper. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { return toDomain(tx.db.Entity.Create().Save(ctx)) })`)
**TxCreator + TxUser[T] triad on every adapter** — Each adapter implements Tx(ctx) via db.HijackTx + entutils.NewTxDriver, WithTx(ctx, tx) via db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client(), and Self() returning itself. All three are required for TransactingRepo to resolve; a missing one means TransactingRepo cannot rebind or fall back. The transaction_test.go db1Adapter/db2Adapter pair is the reference implementation. (`func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) { txCtx, cfg, drv, err := a.db.HijackTx(ctx, &sql.TxOptions{}); return txCtx, entutils.NewTxDriver(drv, cfg), err }`)
**Standard mixin composition on entity schemas** — Every Ent entity schema composes IDMixin + NamespaceMixin + TimeMixin (or ResourceMixin which bundles those four plus MetadataMixin, name and description). UniqueResourceMixin adds KeyMixin and the (namespace, key, deleted_at) partial-unique index. RecursiveMixin[T] flattens a mixin that itself embeds sub-mixins. (`func (MyEntity) Mixin() []ent.Mixin { return []ent.Mixin{entutils.ResourceMixin{}} }`)
**Generated cross-client tx + accessor surface, never hand-written** — The transaction mechanism and field accessors are produced by the codegen extensions (entexpose: GetConfig/HijackTx/NewTxClientFromRawConfig; entmixinaccessor: GetNamespace/GetID getters; entcursor/entpaginate: .Cursor()/.Paginate(); entsetorclear: SetOrClear<Field>). They are registered in openmeter/ent/entc.go and only take effect after make generate; output in openmeter/ent/db is never edited by hand. (`txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())`)
**InIDOrder for deterministic, namespace-isolated batch lookup** — When fetching by IDs and returning in caller order, use entutils.InIDOrder[T](namespace, targetOrderIDs, results). It enforces namespace isolation (cross-namespace rows tolerated in input but filtered out of output), rejects duplicate results with ErrDuplicateID, and returns models.NewGenericNotFoundError for missing target IDs. (`ordered, err := entutils.InIDOrder(namespace, ids, dbRows)`)
**JSON / Postgres value helpers** — Store Go types as JSONB via entutils.JSONStringValueScanner[T]() (nil/invalid NullString yields the zero value). Filter JSONB columns with JSONBIn / JSONBKeyExistsInObject (Postgres-only). Store ULIDs as char(26) text using entutils.ULID / Ptr / Wrap rather than raw ulid bytes. (`field.Other("cfg", Cfg{}).ValueScanner(entutils.JSONStringValueScanner[Cfg]())`)
**GetOrdering / MapPaged translation helpers** — Translate pkg/sortx.Order to []sql.OrderTermOption with entutils.GetOrdering(order) (an unrecognized order silently yields no ordering, not an error). Map pagination.Result[I] to pagination.Result[O] preserving TotalCount/Page via MapPaged / MapPagedWithErr (the latter aborts on the first mapper error). (`q.Order(entutils.GetOrdering(params.Order)...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `transaction.go` | Core: TxDriver (savepoint-aware tx wrapper), TransactingRepo / TransactingRepoWithNoValue, TxCreator/TxUser[T] interfaces, GetDriverFromContext. | TxDriver.once.Do skips the savepoint on the FIRST SavePoint() call so the outermost scope does a real Commit/Rollback while nested scopes use savepoints s1, s2, ...; Ent commit/rollback hooks are dropped because RawEntConfig omits hooks/inters; HijackTx on a client already in a tx errors. |
| `mixins.go` | All standard schema mixins: IDMixin (ULID char(26)), NamespaceMixin, TimeMixin, Metadata/Annotations/Key/Cadenced/CustomerAddress, plus composite ResourceMixin and UniqueResourceMixin. | truncatedNow() truncates timestamps to microsecond for Postgres/CI parity; UniqueResourceMixin uses the (namespace, key, deleted_at) partial-unique workaround — do not replace with a plain (namespace, key) unique index on soft-deletable entities. |
| `idorder.go` | InIDOrder[T InIDOrderAccessor] reorders/validates results against a target ID order with namespace isolation. | Cross-namespace results are silently filtered out (intentional for multi-namespace queries); duplicate target IDs yield duplicate output entries; missing IDs return a GenericNotFoundError wrapping ErrNotFound. |
| `mixinhelper.go` | RecursiveMixin[T] flattens Fields/Indexes/Edges/Hooks/Interceptors/Annotations from a mixin that embeds sub-mixins. | Policy() only delegates to the base T — sub-mixin policies are silently ignored (known TODO). |
| `pgjsonb.go` | JSONBIn (->> IN filter) and JSONBKeyExistsInObject (-> '?') Postgres-only JSONB query helpers. | JSONBIn with empty values emits a literal 'false' predicate rather than an invalid empty IN (); ->> coerces all values to string, so only string comparisons work; may misbehave with joins — unit-test it. |
| `pgulid.go` | ULID valuer/scanner storing ULIDs as text; Ptr()/Wrap() for nil-safe *ulid.ULID conversion. | Storing raw ulid.ULID without this wrapper causes binary-vs-text mismatch in Postgres. |
| `sort.go / mapping.go / valuescanner.go` | GetOrdering (sortx.Order -> OrderTermOption), MapPaged/MapPagedWithErr (paginated result transform), JSONStringValueScanner (JSON-string field codec). | GetOrdering returns an empty slice (no ordering) for unrecognized orders; MapPagedWithErr discards partial results on the first error; JSONStringValueScanner returns the zero value for nil/invalid NullString. |

## Anti-Patterns

- Calling a.db.Foo() or raw SQL directly in an adapter method body without wrapping in TransactingRepo — bypasses the ctx-bound transaction and causes partial writes under concurrency.
- Implementing WithTx by building an adapter from an arbitrary *entdb.Client instead of entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()) — breaks cross-client transaction sharing.
- Importing any openmeter/* domain package from entutils — entutils must stay a domain-free leaf; a domain import creates cycles across every adapter that depends on it.
- Hand-editing generated output in openmeter/ent/db or the codegen extension subfolders' templates without re-running make generate — generated methods go stale or duplicate.
- Adding ORDER BY before .Cursor() or applying .Limit()/.Offset() before .Paginate() — the entcursor/entpaginate templates append their own ordering and reset limit/offset, producing undefined or discarded results.

## Decisions

- **TxDriver uses Postgres savepoints for nested transactions rather than no-op inner transactions.** — Lets an inner scope roll back to its savepoint (partial rollback) without aborting the outer transaction, enabling error-recovery patterns in charge advancement and billing workflows while preserving outer-scope writes — verified by the 'rollback of child scope while keeping parent' test.
- **TransactingRepo reads the transaction from ctx rather than accepting *entdb.Tx as a parameter.** — Keeps adapter signatures clean and prevents tx plumbing leaking into every call site; ctx propagation is idiomatic Ent and enables one Postgres tx to be shared across independent adapters/clients (e.g. db1 + db2 in transaction_test.go).
- **Transaction internals and field accessors are exposed via code-generation extensions (entexpose, entmixinaccessor, entcursor, entpaginate, entsetorclear) rather than a hand-written public Ent API.** — Ent has no public surface for cross-client tx sharing or mixin-field getters; generating them onto every db package keeps the foundation uniform across all entities and confines the mechanism to a regen step instead of per-entity boilerplate.

## Example: Transaction-aware adapter implementing the full TxCreator + TxUser triad with TransactingRepo on the method body.

```
import (
    "context"
    "database/sql"
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "github.com/openmeterio/openmeter/pkg/framework/entutils"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type adapter struct{ db *entdb.Client }

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
    txCtx, cfg, drv, err := a.db.HijackTx(ctx, &sql.TxOptions{})
    if err != nil { return nil, nil, err }
    return txCtx, entutils.NewTxDriver(drv, cfg), nil
}
// ...
```

<!-- archie:ai-end -->
