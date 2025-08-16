package entitydiff

import (
	"time"

	"github.com/samber/lo"
)

type Entity interface {
	GetID() string
	GetDeletedAt() *time.Time
}

type DiffUpdate[T Entity] struct {
	DBState       T
	ExpectedState T
}

type Diff[T Entity] struct {
	UpdateCandidates []DiffUpdate[T]
	Create           []T
	Delete           []T
}

func DiffByID[T Entity](expectedState, dbState []T) Diff[T] {
	diff := Diff[T]{}

	dbStateByID := lo.SliceToMap(dbState, func(item T) (string, T) {
		return item.GetID(), item
	})

	for _, expected := range expectedState {
		if expected.GetID() == "" {
			// If the expected state has no id, we cannot correlate it with the db state
			if expected.GetDeletedAt() != nil {
				// If the expected state is deleted, we can skip it
				continue
			}

			// If the expected state is not deleted, we need to create it
			diff.Create = append(diff.Create, expected)
			continue
		}

		dbState, ok := dbStateByID[expected.GetID()]
		if !ok {
			if expected.GetDeletedAt() != nil {
				// If the expected state is deleted, but we don't have it in the db, we can skip it
				continue
			}

			// If the expected state is not deleted, but we don't have it in the db, we need to create it
			diff.Create = append(diff.Create, expected)
			continue
		}

		if expected.GetDeletedAt() != nil {
			if dbState.GetDeletedAt() == nil {
				// If the expected state is deleted, but we have it in the db, we need to delete it
				diff.Delete = append(diff.Delete, dbState)
				continue
			}

			// If the expected state is deleted, and we have it in the db, but it's deleted, we can skip it
			continue
		}

		diff.UpdateCandidates = append(diff.UpdateCandidates, DiffUpdate[T]{
			DBState:       dbState,
			ExpectedState: expected,
		})
	}

	return diff
}
