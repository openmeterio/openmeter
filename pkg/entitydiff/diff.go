package entitydiff

import (
	"errors"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/equal"
)

type Entity interface {
	GetID() string
	IsDeleted() bool
}

type DiffUpdate[T Entity] struct {
	// PersistedState is the persisted state of the entity (e.g. in database or any storage), if empty there is no persisted state
	PersistedState T
	// ExpectedState is the expected state of the entity (e.g. after the changes done by the backend)
	ExpectedState T
}

type speculativeDiff[T Entity] struct {
	UpdateCandidates []DiffUpdate[T]
	Create           []T
	Delete           []T
}

type Diff[T Entity] struct {
	Update []DiffUpdate[T]
	Create []T
	Delete []T
}

func (d *Diff[T]) NeedsUpdate(item ...DiffUpdate[T]) {
	d.Update = append(d.Update, item...)
}

func (d *Diff[T]) NeedsCreate(item ...T) {
	d.Create = append(d.Create, item...)
}

func (d *Diff[T]) NeedsDelete(item ...T) {
	d.Delete = append(d.Delete, item...)
}

func (d *Diff[T]) Append(a Diff[T]) Diff[T] {
	out := Diff[T]{
		Delete: make([]T, 0, len(a.Delete)+len(d.Delete)),
		Update: make([]DiffUpdate[T], 0, len(a.Update)+len(d.Update)),
		Create: make([]T, 0, len(a.Create)+len(d.Create)),
	}

	out.Delete = append(out.Delete, a.Delete...)
	out.Delete = append(out.Delete, d.Delete...)

	out.Update = append(out.Update, a.Update...)
	out.Update = append(out.Update, d.Update...)

	out.Create = append(out.Create, a.Create...)
	out.Create = append(out.Create, d.Create...)

	return out
}

func (d *Diff[T]) IsEmpty() bool {
	return len(d.Delete) == 0 && len(d.Update) == 0 && len(d.Create) == 0
}

func Union[T Entity](diffs ...Diff[T]) Diff[T] {
	out := Diff[T]{
		Create: []T{},
		Delete: []T{},
		Update: []DiffUpdate[T]{},
	}

	for _, diff := range diffs {
		out.Create = append(out.Create, diff.Create...)
		out.Delete = append(out.Delete, diff.Delete...)
		out.Update = append(out.Update, diff.Update...)
	}

	return out
}

// diffByID compares the expected state with the db state by ID.
// It returns the UpdateCandidates as it does not compare entities. This call should be used for entities with child entities.
//
// If the entity does not have a child, please use DiffByIDEqualer instead.
func diffByID[T Entity](expectedState, dbState []T) speculativeDiff[T] {
	diff := speculativeDiff[T]{}

	itemsWithID := lo.Filter(dbState, func(item T, _ int) bool {
		return item.GetID() != ""
	})

	dbStateByID := lo.SliceToMap(itemsWithID, func(item T) (string, T) {
		return item.GetID(), item
	})

	for _, expected := range expectedState {
		if expected.GetID() == "" {
			// If the expected state has no id, we cannot correlate it with the db state
			if expected.IsDeleted() {
				// If the expected state is deleted, we can skip it
				continue
			}

			// If the expected state is not deleted, we need to create it
			diff.Create = append(diff.Create, expected)
			continue
		}

		dbState, ok := dbStateByID[expected.GetID()]
		if !ok {
			if expected.IsDeleted() {
				// If the expected state is deleted, but we don't have it in the db, we can skip it
				continue
			}

			// If the expected state is not deleted, but we don't have it in the db, we need to create it
			diff.Create = append(diff.Create, expected)
			continue
		}

		if expected.IsDeleted() {
			if !dbState.IsDeleted() {
				// If the expected state is deleted, but we have it in the db, we need to delete it, based on the target state
				// so that if somebody not just set the deleted_at field, but also some other fields that should be persisted
				// to the database.
				//
				// For example if you delete a line that is subscription synced, the edit will cause managedBy to become manual
				// in the same change as the deleted_at change.

				diff.Delete = append(diff.Delete, expected)
				continue
			}

			// If the expected state is deleted, and we have it in the db, but it's deleted, we can skip it
			continue
		}

		diff.UpdateCandidates = append(diff.UpdateCandidates, DiffUpdate[T]{
			PersistedState: dbState,
			ExpectedState:  expected,
		})
	}

	// Let's collect expected items with an ID so that we can check if they would require deletion
	expectedItemsByID := lo.SliceToMap(
		lo.Filter(expectedState, func(item T, _ int) bool {
			return item.GetID() != ""
		}),
		func(item T) (string, T) {
			return item.GetID(), item
		},
	)

	for dbID, dbItemState := range dbStateByID {
		// If the expected state does not contain the item we need to delete it
		if _, ok := expectedItemsByID[dbID]; !ok {
			diff.Delete = append(diff.Delete, dbItemState)
		}
	}

	return diff
}

type DiffByIDInput[T Entity] struct {
	DBState       []T
	ExpectedState []T

	HandleDelete func(item T) error
	HandleCreate func(item T) error
	HandleUpdate func(item DiffUpdate[T]) error
}

// DiffByID compares the expected state with the db state by ID.
//
// The callback functions are used to allow for custom handling of the diff entries.
//
// The call does not compare the entities themselves, it only compares the IDs. Can be used for
// entities that have child entities.
func DiffByID[T Entity](input DiffByIDInput[T]) error {
	diff := diffByID(input.ExpectedState, input.DBState)

	var errs []error
	for _, delete := range diff.Delete {
		if input.HandleDelete != nil {
			if err := input.HandleDelete(delete); err != nil {
				errs = append(errs, err)
			}
		}
	}

	for _, expected := range diff.Create {
		if input.HandleCreate != nil {
			if err := input.HandleCreate(expected); err != nil {
				errs = append(errs, err)
			}
		}
	}

	for _, update := range diff.UpdateCandidates {
		if input.HandleUpdate != nil {
			if err := input.HandleUpdate(update); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

type EqualerEntity[T any] interface {
	Entity
	equal.Equaler[T]
}

// DiffByIDEqualer compares the expected state with the db state by ID.
//
// This call uses the Equal method of the entity to compare the expected state with the db state in
// case of updates.
//
// Should be used for entities that do not have any child entities.
func DiffByIDEqualer[T EqualerEntity[T]](expectedState, dbState []T) Diff[T] {
	diff := diffByID(expectedState, dbState)

	out := Diff[T]{
		Create: diff.Create,
		Delete: diff.Delete,
		Update: make([]DiffUpdate[T], 0, len(diff.UpdateCandidates)),
	}

	for _, update := range diff.UpdateCandidates {
		if !update.PersistedState.Equal(update.ExpectedState) {
			out.Update = append(out.Update, update)
		}
	}

	return out
}
