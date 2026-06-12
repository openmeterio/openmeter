# entutils

<!-- archie:ai-start -->

> Shared Ent ORM glue: schema mixins (ResourceMixin/IDMixin/NamespaceMixin/TimeMixin/AnnotationsMixin/CadencedMixin/CustomerAddressMixin), the cross-client savepoint-based transaction layer (TxDriver, TransactingRepo, TxUser/TxCreator), and Postgres-specific value scanners plus raw-SQL JSONB predicate builders. One of the codebase's biggest dependency magnets (74 in-edges); nearly every domain adapter depends on its mixins and transaction helpers.

## Patterns

**Transaction-aware repo bodies via TransactingRepo** — Adapter methods wrap their body in TransactingRepo/TransactingRepoWithNoValue so they reuse any tx already in ctx (via GetDriverFromContext) and otherwise run on repo.Self(). The adapter must implement TxUser[T] (WithTx/Self) + TxCreator (Tx). (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, rep Repo) error { return rep.doWork(ctx) })`)
**Savepoint-based nested transactions** — TxDriver models nesting with Postgres savepoints (txSavepoint.Next/Prev/String -> 's1','s2'). The first SavePoint() is skipped via once.Do; Commit at savepoint level issues Release, at top level issues Commit. Inner rollback only undoes the child scope (transaction_test 'rollback of child scope while keeping contents of parent'). (`t.driver.SavePoint(next.String()); t.currentSavepoint = next`)
**Mixin composition over per-schema fields** — Schemas embed ResourceMixin (= IDMixin+NamespaceMixin+MetadataMixin+TimeMixin + name/description) rather than redeclaring id/namespace/timestamps. UniqueResourceMixin adds KeyMixin plus the (namespace,key,deleted_at) unique index for soft-delete-safe keys. (`func (ResourceMixin) Fields() []ent.Field { fields = append(fields, IDMixin{}.Fields()...); ... }`)
**Getter/Creator/Setter interfaces per mixin** — Each mixin ships paired interfaces (IDMixinGetter/Creator, NamespaceMixinGetter/Creator, TimeMixinGetter/Creator/Updater, AnnotationsMixinGetter/Setter) so generic helpers constrain on capability not concrete entity. InIDOrder and MapTimeMixinFromDB are generic over these. (`func InIDOrder[T InIDOrderAccessor](...)  // InIDOrderAccessor = IDMixinGetter + NamespaceMixinGetter`)
**Namespace-isolated ordering with InIDOrder** — InIDOrder reorders query results to match a target id slice, keyed by NamespacedID{namespace,id}; cross-namespace ids never match (returns NewGenericNotFoundError), duplicate results error (ErrDuplicateID), extra results are tolerated. (`out, err := entutils.InIDOrder(namespace, idsInOrder, results)`)
**Microsecond-truncated timestamps** — TimeMixin defaults created_at/updated_at to truncatedNow() = clock.Now().Truncate(time.Microsecond) because Postgres has microsecond precision; never bypass clock.Now() or timestamp-comparison tests fail on CI. (`field.Time("created_at").Default(truncatedNow).Immutable()`)
**Raw-SQL JSONB predicates with empty-set guard** — JSONBIn / JSONBKeyExistsInObject build sql.P selectors by hand (->> , -> '?'); JSONBIn short-circuits to WHERE false when values is empty so it never emits invalid `IN ()`. Postgres-only, JSONB-only, string values only. (`JSONBIn("metadata", "tier", []string{"gold"}) // metadata->>'tier' IN ($1)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `transaction.go` | TxDriver (savepoint state machine), NewTxDriver, TransactingRepo/TransactingRepoWithNoValue, TxUser/TxCreator interfaces, GetDriverFromContext. | Commit/Rollback lock mu and short-circuit on prior t.err; once t.err is set the tx is dead. WithTx must rebuild the adapter via db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()). |
| `mixins.go` | All shared schema mixins + getter/creator/setter interfaces, truncatedNow, MapTimeMixinFromDB, CustomerAddressMixin (prefixed PII address fields for invoice snapshots). | id is char(26) ULID; namespace/key are NotEmpty+Immutable; AnnotationsMixin adds a GIN index. Changing field shapes here forces an Atlas migration across many tables. |
| `idorder.go` | InIDOrder generic reorder/validate helper with ErrNamespaceRequired/ErrIDRequired/ErrDuplicateID/ErrNotFound. | Groups by NamespacedID, so namespace mismatch silently becomes not-found (security boundary); missing ids aggregate into a NewGenericNotFoundError. |
| `mixinhelper.go` | RecursiveMixin[T] for mixins that embed sub-mixins (MixinWithAdditionalMixins.Mixin()); concatenates Fields/Indexes/Edges/Hooks/Interceptors/Annotations. | Policy() only evaluates the base mixin (TODO), not nested mixins. |
| `valuescanner.go` | JSONStringValueScanner[T] — generic field.ValueScannerFunc storing T as JSON text in a NullString column. | Returns zero T on null/invalid; unmarshal errors propagate, so malformed stored JSON fails reads. |
| `pgjsonb.go` | JSONBIn and JSONBKeyExistsInObject raw-SQL selector builders. | Postgres+JSONB only; ->> coerces to string so non-string equality is unsupported; comment warns these may break with joins — unit-test them. |
| `pgulid.go` | ULID Valuer/Scanner storing string ULIDs (not binary) so Postgres treats them as UTF-8; Ptr/Wrap/ULIDPointer helpers. | Use this wrapper, not raw ulid.ULID, when a column should hold the textual ULID. |
| `mapping.go / sort.go` | MapPaged/MapPagedWithErr translate pagination.Result[I]->[O]; GetOrdering maps sortx.Order to []sql.OrderTermOption. | GetOrdering falls back to empty (no ordering) for unrecognized order strings rather than erroring. |

## Anti-Patterns

- Hand-rolling SetID/SetNamespace/created_at/timestamps on a schema instead of embedding ResourceMixin/IDMixin/NamespaceMixin/TimeMixin.
- Accepting a raw *entdb.Client in an adapter helper and querying it directly instead of going through TransactingRepo, bypassing the in-context transaction.
- Relying on Ent's native client.Tx + onCommit/onRollback hooks for shared transactions — this layer uses HijackTx/savepoints and ignores those hooks.
- Building JSONB predicates with fmt.Sprintf-interpolated values instead of JSONBIn's parameterized b.Args, or emitting `IN ()` for empty value sets.
- Using time.Now() in schema defaults or test setup instead of clock.Now()/truncatedNow, breaking microsecond-precision timestamp comparisons on CI.

## Decisions

- **Cross-client transactions via a hand-written TxDriver + RawEntConfig + HijackTx rather than Ent's per-client Tx.** — Multiple generated Ent clients (db1/db2) must share one underlying transaction; HijackTx exposes the raw config so NewTxClientFromRawConfig can rehydrate any client onto the same tx, with savepoints providing nestable rollback scopes.
- **Store ULIDs as char(26) text and truncate timestamps to microseconds.** — Postgres interprets binary ULIDs as UTF-8 and has microsecond time precision; truncating in Go keeps in-memory results consistent with what Postgres round-trips so tests pass on both macOS and CI.
- **Soft-delete uniqueness approximated by a (namespace,key,deleted_at) unique index in UniqueResourceMixin.** — Ent cannot emit partial `WHERE deleted_at IS NULL` indexes without manual migrations, so deleted_at is folded into the unique key; documented caveat: two same-key deletes in the same microsecond collide.

## Example: A transaction-aware adapter participating in shared transactions

```
func (a *adapter) Self() Repo { return a }

func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) Repo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return &adapter{db: txClient.Client()}
}

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

// ...
```

<!-- archie:ai-end -->
