# entitydiff

<!-- archie:ai-start -->

> Generic diffing library for entity collections needing create/update/delete reconciliation between expected and persisted DB state. Used by billing subscription sync to reconcile invoice lines and subscription phases against their DB representation.

## Patterns

**Entity interface constraint** — Diffable types implement Entity (GetID() string, IsDeleted() bool). ID-less non-deleted items are always treated as creates. (`func (l InvoiceLine) GetID() string { return l.ID }
func (l InvoiceLine) IsDeleted() bool { return l.DeletedAt != nil }`)
**DiffByIDEqualer for leaf entities** — Use DiffByIDEqualer when the entity has no children and implements equal.Equaler[T]; it skips updates where PersistedState.Equal(ExpectedState) is true. (`diff := entitydiff.DiffByIDEqualer(expectedLines, dbLines)`)
**DiffByID with callbacks for parent entities** — Use DiffByID with HandleCreate/Update/Delete callbacks when children need their own nested diffing inside the update handler. (`entitydiff.DiffByID(entitydiff.DiffByIDInput[Phase]{DBState: db, ExpectedState: exp, HandleUpdate: func(u DiffUpdate[Phase]) error { return diffChildren(u) }})`)
**NestedEntity wrapper for parent context** — Wrap children in NestedEntity[T,P] / EqualerNestedEntity[T,P] when the diff handler needs the parent alongside the child. (`wrapped := entitydiff.NewEqualersWithParent(phase.Lines, phase)`)
**Diff.Append and Union for combining diffs** — Merge diffs from multiple sub-trees via Diff.Append or Union[T] before applying; never apply partial diffs then re-diff mutated state. (`combined := entitydiff.Union(diff1, diff2, diff3)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Core diff: diffByID (internal), DiffByID (callback), DiffByIDEqualer (equality), Diff[T] with NeedsUpdate/Create/Delete/Append/IsEmpty/Union. | Items absent from expectedState are treated as deletes — pass the full expected slice, not a filtered subset. |
| `parent.go` | NestedEntity / EqualerNestedEntity wrappers carrying parent context; NewEqualersWithParent helper. | EqualerNestedEntity.Equal delegates only to Entity.Equal — parent fields are not compared, only used for context. |

## Anti-Patterns

- Implementing GetID() to return a non-stable (e.g. index-based) ID — IDs must be stable DB identifiers or empty for new items.
- Using DiffByID for leaf entities without children — use DiffByIDEqualer to skip no-op updates.
- Mutating DB state inside a diff callback then re-diffing the same collection — diffs are computed once against the original snapshot.
- Passing a filtered expectedState to diffByID — absent items are unconditionally deleted.

## Decisions

- **Split diffByID (returns candidates) from DiffByIDEqualer (filters via Equal)** — Entities with children cannot use equality because child state is not embedded; callers with children need all update candidates to recurse.
- **IsDeleted() drives deletion, not absence from the expected list** — Subscription sync must represent soft-delete intent explicitly so the deleted expected state carries the full target row to persist before deleting.

## Example: Diff leaf invoice lines (with equality) and apply changes

```
import "github.com/openmeterio/openmeter/pkg/entitydiff"

diff := entitydiff.DiffByIDEqualer(expectedLines, dbLines)
for _, c := range diff.Create { _ = adapter.CreateLine(ctx, c) }
for _, u := range diff.Update { _ = adapter.UpdateLine(ctx, u.ExpectedState) }
for _, d := range diff.Delete { _ = adapter.DeleteLine(ctx, d) }
```

<!-- archie:ai-end -->
