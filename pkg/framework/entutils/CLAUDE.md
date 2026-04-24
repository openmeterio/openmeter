# entutils

<!-- archie:ai-start -->

> Core Ent ORM infrastructure layer providing transaction management (TransactingRepo/TxDriver), mixin definitions (IDMixin, NamespaceMixin, TimeMixin, etc.), code-generation extensions (cursor, paginate, expose, mixinaccessor, setorclear), and utility types (ULID, JSONB helpers, InIDOrder). All domain adapters depend on this package; it must not import domain packages.

## Patterns

**TransactingRepo for all adapter DB access** — Every adapter method must call entutils.TransactingRepo(ctx, repo, cb) or TransactingRepoWithNoValue. It reads *TxDriver from ctx via GetDriverFromContext; if found, calls repo.WithTx(ctx, tx); if not, calls repo.Self(). Never use raw *entdb.Client directly. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo AdapterType) (*Domain, error) { ... })`)
**TxUser[T] + TxCreator dual interface on every adapter** — Each adapter must implement both TxCreator (Tx(ctx) returning context+Driver) using db.HijackTx, and TxUser[T] (WithTx returning a new adapter bound to the tx, and Self returning itself). See db1Adapter in transaction_test.go for the canonical pattern. (`func (d *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) { txCtx, cfg, drv, err := d.db.HijackTx(ctx, &sql.TxOptions{}); return txCtx, entutils.NewTxDriver(drv, cfg), err }`)
**Mixin composition via entutils mixins** — All entity schemas embed standard mixins: IDMixin (ULID char(26)), NamespaceMixin (immutable namespace field + index), TimeMixin (created_at/updated_at/deleted_at), MetadataMixin (jsonb). ResourceMixin composes all four plus name/description. Use RecursiveMixin[T] when a mixin itself has sub-mixins. (`func (MyEntity) Mixin() []ent.Mixin { return []ent.Mixin{entutils.ResourceMixin{}} }`)
**InIDOrder for deterministic multi-entity lookup** — When fetching a batch of entities by IDs and returning them in caller-specified order, use entutils.InIDOrder[T](namespace, targetOrderIDs, results). It validates namespace isolation, detects duplicates, and returns GenericNotFoundError for missing IDs. (`ordered, err := entutils.InIDOrder(namespace, ids, dbRows)`)
**JSONStringValueScanner for custom JSON fields** — Use entutils.JSONStringValueScanner[T]() as the ValueScanner for any Ent field that stores a Go type as JSON string in Postgres. Handles nil/invalid NullString gracefully. (`field.Other("my_field", MyType{}).SchemaType(...).ValueScanner(entutils.JSONStringValueScanner[MyType]())`)
**GetOrdering for sortx.Order to sql.OrderTermOption** — Translate pkg/sortx.Order values to []sql.OrderTermOption using entutils.GetOrdering(order). Never use raw sql.OrderAsc()/OrderDesc() directly with sortx values. (`q.Order(entutils.GetOrdering(params.Order)...)`)
**ULID wrapper for Postgres text storage** — Use entutils.ULID (wraps oklog/ulid.ULID) when storing ULIDs as text in Postgres. Use entutils.Ptr(*ulid.ULID) and entutils.Wrap(ulid.ULID) for nil-safe conversions. Raw ulid.ULID stored as binary would be interpreted as UTF-8. (`entutils.Ptr(someULIDPointer)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `transaction.go` | Defines TxDriver (savepoint-aware transaction wrapper), TransactingRepo/TransactingRepoWithNoValue helpers, TxCreator/TxUser interfaces, and GetDriverFromContext. Central to all adapter DB writes. | TxDriver.once.Do skips savepoint on first call — nested transactions use savepoints, outermost uses real commit/rollback. Do not call HijackTx on a client that already has a txDriver (returns error). |
| `mixins.go` | Standard Ent mixin definitions: IDMixin (ULID, char(26)), NamespaceMixin, TimeMixin, MetadataMixin, AnnotationsMixin, KeyMixin, ResourceMixin, UniqueResourceMixin, CadencedMixin, CustomerAddressMixin. | truncatedNow() truncates to microsecond — required for CI/macOS test comparisons. UniqueResourceMixin uses partial unique index workaround (namespace, key, deleted_at) — do not add a simpler (namespace, key) unique index on soft-deletable entities. |
| `idorder.go` | InIDOrder[T InIDOrderAccessor] reorders and validates a result slice against a target ID order. Enforces namespace isolation by only matching (namespace, id) pairs. | Cross-namespace results are tolerated in the input slice but only same-namespace matches are returned — this is intentional for multi-namespace queries. |
| `pgjsonb.go` | JSONBIn (IN filter on jsonb field key) and JSONBKeyExistsInObject (? operator on nested object). PostgreSQL-only; do not use with ClickHouse or non-jsonb columns. | JSONBIn with empty values generates 'false' predicate, not an empty IN — correct but surprising in test assertions. |
| `pgulid.go` | ULID valuer/scanner for string-format Postgres storage. Ptr() and Wrap() for nil-safe conversion between *ulid.ULID and *entutils.ULID. | Storing raw ulid.ULID without this wrapper causes binary-vs-text mismatch in Postgres. |
| `mapping.go` | MapPaged and MapPagedWithErr — type-safe transform of pagination.Result[I] to pagination.Result[O] preserving TotalCount and Page. | MapPagedWithErr stops on first mapper error; partial results are discarded. |
| `mixinhelper.go` | RecursiveMixin[T] flattens Fields/Indexes/Edges/Hooks/Interceptors/Annotations from a mixin that itself embeds sub-mixins. Policy() only delegates to the base, not sub-mixins — known limitation. | Policy() TODO — sub-mixin policies are silently ignored. |
| `sort.go` | GetOrdering translates pkg/sortx.Order to []sql.OrderTermOption. Fallback on unknown values returns empty slice (no ordering), not an error. | Unrecognized sortx.Order values silently produce no ordering — validate Order values before calling. |

## Anti-Patterns

- Calling d.db.QueryContext or raw SQL inside an adapter instead of using TransactingRepo — bypasses the ctx-bound transaction.
- Implementing WithTx by returning a new adapter with a different *entdb.Client instead of using db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()) — breaks cross-client transaction sharing.
- Adding ORDER BY on an Ent query before calling .Cursor() — the entcursor template appends its own (created_at ASC, id ASC) order, producing undefined combined ordering.
- Importing openmeter/* domain packages from entutils — creates import cycles; entutils must remain a leaf dependency.
- Hand-editing files in testutils/ent1/db/ or testutils/ent2/db/ — they are generated fixtures; run go generate in testutils/ent1 or testutils/ent2 instead.

## Decisions

- **TxDriver uses savepoints for nested transactions rather than no-op inner transactions** — Allows inner transaction rollbacks to be partial (roll back to savepoint) without aborting the outer transaction, enabling error recovery patterns in charge advancement and billing workflows.
- **TransactingRepo reads transaction from ctx rather than accepting *entdb.Tx as a parameter** — Keeps adapter method signatures clean and prevents tx plumbing leaking into every call site; the ctx-based approach is idiomatic Ent and consistent with how Go context propagation works.
- **IDMixin uses ULID stored as char(26) text rather than UUID or integer** — ULIDs are sortable by creation time (monotonic), URL-safe, and globally unique without coordination; char(26) avoids binary encoding issues in Postgres compared to raw ulid.ULID bytes.

## Example: Implementing a transaction-aware adapter method that respects ctx-propagated transactions

```
import (
    "context"
    "database/sql"
    "fmt"

    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "github.com/openmeterio/openmeter/pkg/framework/entutils"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type myAdapter struct {
    db *entdb.Client
}

// Tx implements TxCreator
// ...
```

<!-- archie:ai-end -->
