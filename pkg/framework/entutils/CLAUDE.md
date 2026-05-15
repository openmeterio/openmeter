# entutils

<!-- archie:ai-start -->

> Core Ent ORM infrastructure: ctx-propagated transaction management (TransactingRepo/TxDriver with savepoint nesting), standard schema mixins (IDMixin ULID, NamespaceMixin, TimeMixin), code-generation extensions (cursor, paginate, expose, mixinaccessor, setorclear), and Postgres utility types. All domain adapters depend on this package; it must never import domain packages.

## Patterns

**TransactingRepo on every adapter method body** — Every adapter method must call entutils.TransactingRepo(ctx, repo, cb) or TransactingRepoWithNoValue. It reads *TxDriver from ctx via GetDriverFromContext; if found, calls repo.WithTx(ctx, tx); otherwise calls repo.Self(). Never call tx.db.Foo() directly. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { row, err := tx.db.Entity.Create().SetNamespace(in.Namespace).Save(ctx); return toDomain(row), err })`)
**TxCreator + TxUser[T] triad on every adapter** — Each adapter must implement: Tx(ctx) using db.HijackTx + entutils.NewTxDriver (starts a transaction), WithTx(ctx, tx) using db.NewTxClientFromRawConfig (rebinds to existing tx), and Self() returning itself. All three are required for TransactingRepo to function correctly. (`func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) { txCtx, cfg, drv, err := a.db.HijackTx(ctx, &sql.TxOptions{}); return txCtx, entutils.NewTxDriver(drv, cfg), err }
func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { return &adapter{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()} }
func (a *adapter) Self() *adapter { return a }`)
**Standard mixin composition on all entity schemas** — Every new Ent entity schema must include IDMixin (ULID char(26)), NamespaceMixin (multi-tenancy), and TimeMixin (created_at/updated_at/deleted_at). Use ResourceMixin to compose all four plus name/description. Use RecursiveMixin[T] when a mixin itself has sub-mixins. (`func (MyEntity) Mixin() []ent.Mixin { return []ent.Mixin{entutils.ResourceMixin{}} }`)
**TxDriver savepoints for nested transactions** — TxDriver.once.Do skips creating a savepoint on the first call; subsequent nested calls create savepoints named s1, s2, ... Commit at savepoint level releases it (not real commit); Rollback at savepoint level rolls back to that savepoint. Never call HijackTx on a client that already has a txDriver — it returns an error. (`// Nested transaction.Run calls inside an outer transaction.Run use savepoints automatically via TxDriver`)
**InIDOrder for deterministic batch lookup** — When fetching entities by IDs and returning in caller-specified order, use entutils.InIDOrder[T](namespace, targetOrderIDs, results). It validates namespace isolation (cross-namespace results are tolerated in input but filtered in output), detects duplicates, and returns GenericNotFoundError for missing IDs. (`ordered, err := entutils.InIDOrder(namespace, ids, dbRows)`)
**JSONStringValueScanner for JSON-stored fields** — Use entutils.JSONStringValueScanner[T]() as the ValueScanner for any Ent field that stores a Go type as JSON string in Postgres. Handles nil/invalid NullString gracefully by returning zero value. (`field.Other("my_field", MyType{}).SchemaType(map[string]string{dialect.Postgres: "jsonb"}).ValueScanner(entutils.JSONStringValueScanner[MyType]())`)
**GetOrdering for sortx.Order translation** — Translate pkg/sortx.Order to []sql.OrderTermOption using entutils.GetOrdering(order). Unrecognized sortx.Order values silently produce an empty slice (no ordering), not an error — validate Order values before calling. (`q.Order(entutils.GetOrdering(params.Order)...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `transaction.go` | Defines TxDriver (savepoint-aware transaction wrapper), TransactingRepo/TransactingRepoWithNoValue helpers, TxCreator/TxUser interfaces, and GetDriverFromContext. Central to all adapter DB writes. | TxDriver.once.Do skips savepoint on the first call — nested transactions use savepoints, outermost uses real commit/rollback. Do not call HijackTx on a client that already has a txDriver (returns error). Ent commit/rollback hooks are dropped in NewTxClientFromRawConfig paths. |
| `mixins.go` | Standard Ent mixin definitions: IDMixin (ULID char(26)), NamespaceMixin, TimeMixin, MetadataMixin, AnnotationsMixin, KeyMixin, ResourceMixin, UniqueResourceMixin, CadencedMixin, CustomerAddressMixin. | truncatedNow() truncates to microsecond — required for CI/macOS test comparisons. UniqueResourceMixin uses partial unique index workaround (namespace, key, deleted_at) — do not add a simpler (namespace, key) unique index on soft-deletable entities. RecursiveMixin.Policy() only delegates to the base mixin, not sub-mixins — known limitation. |
| `idorder.go` | InIDOrder[T InIDOrderAccessor] reorders and validates a result slice against a target ID order, enforcing namespace isolation and returning GenericNotFoundError for missing IDs. | Cross-namespace results are tolerated in the input slice but only same-namespace matches are returned — intentional for multi-namespace queries. Duplicate target IDs in targetOrderIDs produce duplicate output entries. |
| `pgjsonb.go` | JSONBIn (IN filter on jsonb field key) and JSONBKeyExistsInObject (? operator on nested object). PostgreSQL-only; do not use with ClickHouse or non-jsonb columns. | JSONBIn with empty values generates a 'false' predicate, not an empty IN — correct but surprising in test assertions. |
| `pgulid.go` | ULID valuer/scanner for string-format Postgres storage. Ptr() and Wrap() for nil-safe conversion between *ulid.ULID and *entutils.ULID. | Storing raw ulid.ULID without this wrapper causes binary-vs-text mismatch in Postgres. |
| `mapping.go` | MapPaged and MapPagedWithErr — type-safe transform of pagination.Result[I] to pagination.Result[O] preserving TotalCount and Page. | MapPagedWithErr stops on first mapper error; partial results are discarded. |
| `sort.go` | GetOrdering translates pkg/sortx.Order to []sql.OrderTermOption for use in Ent query builders. | Unrecognized sortx.Order values silently return an empty slice (no ordering applied), not an error — validate Order values before calling. |
| `mixinhelper.go` | RecursiveMixin[T] flattens Fields/Indexes/Edges/Hooks/Interceptors/Annotations from a mixin that itself embeds sub-mixins. | Policy() only delegates to the base T, not sub-mixins — sub-mixin policies are silently ignored (known TODO). |

## Anti-Patterns

- Calling a.db.Foo() or raw SQL directly in adapter method bodies without wrapping in TransactingRepo — bypasses the ctx-bound Ent transaction and produces partial writes under concurrency.
- Implementing WithTx by constructing a new adapter with a different *entdb.Client instead of entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()) — breaks cross-client transaction sharing.
- Importing openmeter/* domain packages from entutils — creates import cycles; entutils must remain a leaf dependency with no domain imports.
- Adding ORDER BY on an Ent query before calling .Cursor() — the entcursor template appends (created_at ASC, id ASC) and the combined ordering becomes undefined.
- Hand-editing files in testutils/ent1/db/ or testutils/ent2/db/ — they are generated fixtures; run go generate in testutils/ent1 or testutils/ent2 instead.

## Decisions

- **TxDriver uses savepoints for nested transactions rather than no-op inner transactions** — Allows inner transaction rollbacks to be partial (roll back to savepoint) without aborting the outer transaction, enabling error recovery patterns in charge advancement and billing workflows without losing outer-scope writes.
- **TransactingRepo reads transaction from ctx rather than accepting *entdb.Tx as a parameter** — Keeps adapter method signatures clean and prevents tx plumbing leaking into every call site; ctx-based propagation is idiomatic Ent and enables cross-client transaction sharing (e.g., billing + customer adapters sharing one Postgres tx).
- **IDMixin uses ULID stored as char(26) text rather than UUID or integer** — ULIDs are sortable by creation time (monotonic), URL-safe, and globally unique without coordination; char(26) avoids binary encoding issues in Postgres compared to raw ulid.ULID bytes.

## Example: Implementing a transaction-aware adapter with full TxCreator + TxUser triad and TransactingRepo on every method body

```
import (
    "context"
    "database/sql"
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "github.com/openmeterio/openmeter/pkg/framework/entutils"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type myAdapter struct{ db *entdb.Client }

func (a *myAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
    txCtx, cfg, drv, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
    if err != nil { return nil, nil, fmt.Errorf("hijack tx: %w", err) }
    return txCtx, entutils.NewTxDriver(drv, cfg), nil
}
// ...
```

<!-- archie:ai-end -->
