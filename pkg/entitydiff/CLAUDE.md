# entitydiff

<!-- archie:ai-start -->

> Generic diffing library for entity collections that need create/update/delete reconciliation between expected state and persisted DB state. Used by billing subscription sync to reconcile invoice lines and subscription phases against their DB representations.

## Patterns

**Entity interface constraint** — All diffable types must implement Entity (GetID() string, IsDeleted() bool). ID-less items (GetID()=="") that are not deleted are always treated as creates. (`type InvoiceLine struct{...}
func (l InvoiceLine) GetID() string { return l.ID }
func (l InvoiceLine) IsDeleted() bool { return l.DeletedAt != nil }`)
**DiffByIDEqualer for leaf entities** — Use DiffByIDEqualer when the entity has no child entities and implements equal.Equaler[T]. It skips updates where PersistedState.Equal(ExpectedState) is true. (`diff := entitydiff.DiffByIDEqualer(expectedLines, dbLines)`)
**DiffByID with callbacks for parent entities** — Use DiffByID with HandleCreate/HandleUpdate/HandleDelete callbacks when the entity has child entities that require their own nested diffing inside the update handler. (`entitydiff.DiffByID(entitydiff.DiffByIDInput[Phase]{DBState: db, ExpectedState: exp, HandleUpdate: func(u DiffUpdate[Phase]) error { return diffChildren(u) }})`)
**NestedEntity wrapper for parent context propagation** — Wrap child entities in NestedEntity[T, P] or EqualerNestedEntity[T, P] when the diff handler needs access to the parent entity alongside the child during create/update/delete. (`wrapped := entitydiff.NewEqualersWithParent(phase.Lines, phase)`)
**Diff.Append and Union for combining diffs** — Use Diff.Append or Union[T] to merge diffs from multiple sub-trees before applying; never apply partial diffs and then compute a second diff on already-mutated state. (`combined := entitydiff.Union(diff1, diff2, diff3)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Core diff logic: diffByID (internal), DiffByID (callback-based), DiffByIDEqualer (equality-based), Diff[T] struct with NeedsUpdate/NeedsCreate/NeedsDelete/Append/IsEmpty/Union. | diffByID treats items missing from expectedState as deletes — pass the full expected slice, not a filtered subset, or items will be spuriously deleted. |
| `parent.go` | NestedEntity and EqualerNestedEntity wrappers for carrying parent context into diff callbacks; NewEqualersWithParent helper. | EqualerNestedEntity.Equal delegates only to Entity.Equal — parent fields are NOT compared for equality, only used for context in callbacks. |

## Anti-Patterns

- Implementing Entity.GetID() to return a non-stable ID (e.g. index-based) — IDs must be stable DB identifiers or empty string for new items.
- Using DiffByID for leaf entities without children — use DiffByIDEqualer to skip no-op updates.
- Mutating DB state inside a diff callback and then running another diff on the same collection — diffs are computed once against the original DB snapshot.
- Passing a filtered expectedState slice to diffByID — items absent from expectedState are unconditionally deleted.

## Decisions

- **Split diffByID (returns UpdateCandidates) from DiffByIDEqualer (filters candidates via Equal) into two separate code paths.** — Entities with children cannot use equality comparison because child state is not embedded; callers with children need all update candidates to recurse into child diffs.
- **IsDeleted() on the expected state drives deletion, not absence from the expected list alone.** — Subscription sync needs to represent soft-delete intent explicitly (e.g. managedBy change co-located with deleted_at) so the deleted expected state carries the full target row to persist before deleting.

## Example: Diff invoice lines (leaf entities with equality) and apply changes

```
import "github.com/openmeterio/openmeter/pkg/entitydiff"

diff := entitydiff.DiffByIDEqualer(expectedLines, dbLines)
for _, c := range diff.Create { _ = adapter.CreateLine(ctx, c) }
for _, u := range diff.Update { _ = adapter.UpdateLine(ctx, u.ExpectedState) }
for _, d := range diff.Delete { _ = adapter.DeleteLine(ctx, d) }
```

<!-- archie:ai-end -->
