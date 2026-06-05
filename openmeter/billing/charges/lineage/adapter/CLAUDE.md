# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter for credit-realization lineages and their segments — the audit trail that tracks how each charge's credit realizations are covered/corrected/backfilled over time. Implements the lineage.Adapter interface (defined in ../lineage/service.go).

## Patterns

**Implements lineage.Adapter + entutils.TxCreator** — New(Config) returns lineage.Adapter. The adapter struct must satisfy Tx/WithTx/Self (TxCreator) plus all CRUD methods declared in lineage.Adapter; missing a method breaks the interface assertion at the New return. (`func New(config Config) (lineage.Adapter, error) { ...; return &adapter{db: config.Client}, nil }`)
**Every method body wrapped in entutils.TransactingRepo / TransactingRepoWithNoValue** — All query/mutation methods rebind to the tx carried in ctx via entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter){...}). Use TransactingRepoWithNoValue for write-only methods (CreateLineages, CloseSegment, CreateSegment). This keeps a raw *entdb.Client honest inside an outer transaction. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { ... })`)
**Package-level helper rebuilds adapter from raw *entdb.Client** — LoadActiveSegmentsByRealizationID exists both as a standalone func taking *entdb.Client (constructing repo := &adapter{db: db} then TransactingRepo) and as a method delegating to it. Standalone helpers handed a raw client must still wrap with TransactingRepo so they join the ctx transaction. (`func LoadActiveSegmentsByRealizationID(ctx, db *entdb.Client, ...) { repo := &adapter{db: db}; return entutils.TransactingRepo(ctx, repo, ...) }`)
**Lock methods assert in-transaction then use ForUpdate()** — LockCorrectionLineages and LockAdvanceLineagesForBackfill first call entutils.GetDriverFromContext(ctx) and error if not in a tx, then chain .ForUpdate() on the query. Row-locking methods must keep both guards. (`if _, err := entutils.GetDriverFromContext(ctx); err != nil { return nil, fmt.Errorf("must be called in a transaction: %w", err) }; ...Query()....ForUpdate().All(ctx)`)
**Active-segment queries filter ClosedAtIsNil + order ByCreatedAt** — Segments are append-only; 'active' means closed_at IS NULL. Every WithSegments / segment query uses creditrealizationlineagesegment.ClosedAtIsNil() and orders ByCreatedAt to keep FIFO consumption stable. (`WithSegments(func(q *entdb.CreditRealizationLineageSegmentQuery){ q.Where(creditrealizationlineagesegment.ClosedAtIsNil()).Order(creditrealizationlineagesegment.ByCreatedAt()) })`)
**Ent rows mapped to domain via mapLineage / mapSegment** — DB rows (*entdb.CreditRealizationLineage / Segment) are translated to lineage.Lineage / lineage.Segment through mapLineage and mapSegment using lo.Map/lo.SliceToMap. AdvanceFeatures is stored as pq.StringArray and read back as []string(entry.AdvanceFeatures). (`return lo.Map(lineages, mapLineage), nil`)
**Bulk creates for initial lineages** — CreateLineages builds parallel slices of CreditRealizationLineageCreate and ...SegmentCreate and saves them with CreateBulk(...).Save(ctx); each segment SetState(spec.InitialState). (`tx.db.CreditRealizationLineage.CreateBulk(rootCreates...).Save(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config{Client *entdb.Client} + Validate, New constructor, adapter struct, and the entutils.TxCreator trio (Tx via HijackTx, WithTx via NewTxClientFromRawConfig, Self). | Tx uses HijackTx with ReadOnly:false; do not flip to read-only — lock/correction flows mutate. WithTx must rebuild via entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()). |
| `lineage.go` | All lineage.Adapter method implementations (Load/Create/Lock/List/Close/CreateSegment) plus mapLineage/mapSegment converters. | AdvanceFeatures must be written as pq.StringArray(spec.AdvanceFeatures) and CreateSegment uses ulid.Make().String() for the segment ID; CreateLineages does NOT set segment IDs (relies on schema default). CreateSegment calls input.Validate() before persisting; other writers do not. |

## Anti-Patterns

- Accepting a raw *entdb.Client in a helper without wrapping the body in entutils.TransactingRepo — it will bypass the ctx transaction.
- Calling a Lock* method outside a transaction — the GetDriverFromContext guard returns an error by design; do not remove it.
- Querying segments without ClosedAtIsNil(), which would resurrect closed (consumed) segments.
- Hard-deleting or UPDATEing segment amounts in place instead of closing + creating a remainder segment (the append-only model lives in the service layer).
- Putting business orchestration (FIFO consumption, correction math) here — the adapter is persistence-only; that logic belongs in lineage/service.

## Decisions

- **Adapter exposes both a package-level LoadActiveSegmentsByRealizationID func and a method that delegates to it.** — Lets callers that already hold a raw *entdb.Client (e.g. other charge adapters) reuse the same tx-aware query without instantiating the full adapter.
- **Segments are append-only with closed_at instead of mutable rows.** — Preserves a complete audit trail of credit coverage state transitions for corrections and advance backfill reconciliation.

## Example: A tx-aware adapter read method with active-segment eager loading

```
func (a *adapter) LoadLineagesByCustomer(ctx context.Context, input lineage.LoadLineagesByCustomerInput) ([]lineage.Lineage, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]lineage.Lineage, error) {
		lineages, err := tx.db.CreditRealizationLineage.Query().
			Where(creditrealizationlineage.Namespace(input.Namespace), creditrealizationlineage.CustomerIDEQ(input.CustomerID), creditrealizationlineage.CurrencyEQ(input.Currency)).
			WithSegments(func(q *entdb.CreditRealizationLineageSegmentQuery) {
				q.Where(creditrealizationlineagesegment.ClosedAtIsNil()).Order(creditrealizationlineagesegment.ByCreatedAt())
			}).
			Order(creditrealizationlineage.ByCreatedAt()).All(ctx)
		if err != nil { return nil, err }
		return lo.Map(lineages, mapLineage), nil
	})
}
```

<!-- archie:ai-end -->
