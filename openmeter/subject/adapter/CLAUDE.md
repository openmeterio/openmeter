# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for the subject domain. Implements subject.Adapter (CRUD + List) against the entdb Subject table, mapping DB entities to subject.Subject domain models and translating Ent errors into models.Generic* errors.

## Patterns

**TxUser adapter struct** — adapter holds *entdb.Client and satisfies subject.Adapter + entutils.TxUser[*adapter] via Tx/WithTx/Self. New(db) returns error if db is nil. (`var _ entutils.TxUser[*adapter] = (*adapter)(nil); func (a *adapter) Self() *adapter { return a }`)
**Every method wraps in TransactingRepo** — All read/write methods run inside entutils.TransactingRepo(ctx, a, func(ctx, tx){...}) (or TransactingRepoWithNoValue for Delete) so they rebind to a ctx-carried transaction. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) { ... })`)
**Validate input then wrap error** — Mutating methods call input.Validate() first and wrap failures with models.NewGenericValidationError before any DB call. (`if err := input.Validate(); err != nil { return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err)) }`)
**Ent error → domain error translation** — db.IsNotFound→NewGenericNotFoundError, db.IsConstraintError→NewGenericConflictError; raw errors are wrapped with %w and namespace/key context. (`if db.IsNotFound(err) { return subject.Subject{}, models.NewGenericNotFoundError(...) }`)
**Soft-delete via DeletedAt window** — Reads (GetByKey, GetByIdOrKey, List) include subjectdb.Or(DeletedAtIsNil(), DeletedAtGTE(now)); Delete sets DeletedAt=now rather than removing the row, and short-circuits if already deleted. (`subjectdb.Or(subjectdb.DeletedAtIsNil(), subjectdb.DeletedAtGTE(now))`)
**OptionalNullable explicit clear on Update** — Update inspects input.<Field>.IsSet and Value to decide SetX vs ClearX, working around Ent issue 2108 where nil does not null a column. (`if input.DisplayName.IsSet { if input.DisplayName.Value != nil { query.SetDisplayName(*...) } else { query.ClearDisplayName() } }`)
**mapEntity is the single DB→domain mapper** — All methods return mapEntity(e); it backfills empty Metadata to an empty map and copies nullable DisplayName/StripeCustomerID pointers. (`return mapEntity(subjectEntity), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor New(db) + TxUser plumbing (Tx via HijackTx, WithTx via NewTxClientFromRawConfig, Self). | Tx uses ReadOnly:false; do not bypass WithTx or the tx context will not rebind the client. |
| `subject.go` | All CRUD/List methods + mapEntity. GetByIdOrKey matches ID exactly OR key within the not-deleted window. | Update's OptionalNullable clear logic is flagged as a unique pattern to be refactored; List sort for display-name-desc uses sql.OrderNullsLast()/OrderDesc() builders, not db.Desc. |

## Anti-Patterns

- Using a passed *entdb.Client directly instead of tx.db inside a TransactingRepo closure (breaks transaction binding).
- Returning raw Ent errors without db.IsNotFound/IsConstraintError translation to models.Generic* errors.
- Hard-deleting subject rows or ignoring the DeletedAtGTE(now) window in reads.
- On Update, calling SetX unconditionally instead of honoring OptionalNullable.IsSet (would silently null or overwrite fields).

## Decisions

- **Soft delete with a DeletedAt timestamp window** — Subjects may be referenced by usage/entitlements; deletion is logical and time-gated rather than physical, and usage for the subject is intentionally not deleted.
- **Manual OptionalNullable clear handling on Update** — Ent does not null fields when nil is provided (ent/ent#2108), so set-vs-clear must be decided from IsSet/Value explicitly.

## Example: Adapter method with transaction, validation, and error mapping

```
func (a *adapter) Create(ctx context.Context, input subject.CreateInput) (subject.Subject, error) {
  if err := input.Validate(); err != nil {
    return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
  }
  return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (subject.Subject, error) {
    e, err := tx.db.Subject.Create().SetNamespace(input.Namespace).SetKey(input.Key).Save(ctx)
    if err != nil {
      if db.IsConstraintError(err) {
        return subject.Subject{}, models.NewGenericConflictError(fmt.Errorf("subject with key already exists: %s", input.Key))
      }
      return subject.Subject{}, fmt.Errorf("failed to create subject: %w", err)
    }
    return mapEntity(e), nil
  })
}
```

<!-- archie:ai-end -->
