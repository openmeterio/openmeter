package billingadapter

import (
	"context"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/entitydiff"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type upsertInput[T any, CreateBulkType any] struct {
	Create      func(*entdb.Client, T) (CreateBulkType, error)
	UpsertItems func(context.Context, *entdb.Client, []CreateBulkType) error
	MarkDeleted func(context.Context, T) (T, error)
}

type upsertOption[T any, CreateBulkType any] func(upsertInput[T, CreateBulkType]) upsertInput[T, CreateBulkType]

func upsertWithOptions[T entitydiff.Entity, CreateBulkType any](ctx context.Context, db *entdb.Client, itemDiff entitydiff.Diff[T], baseSettings upsertInput[T, CreateBulkType], options ...upsertOption[T, CreateBulkType]) error {
	opts := baseSettings
	for _, option := range options {
		opts = option(opts)
	}

	upsertItems := make([]CreateBulkType, 0, len(itemDiff.Create)+len(itemDiff.Update)+len(itemDiff.Delete))

	// Delete must be first, as we might have a constraint that prevents us from creating the item if not deleted before.
	if len(itemDiff.Delete) > 0 && opts.MarkDeleted != nil {
		// We formulate delete as a soft delete update, so that any changes happening alongside the deletion are persisted
		// to the database.

		toDelete, err := slicesx.MapWithErr(itemDiff.Delete, func(item T) (T, error) {
			return opts.MarkDeleted(ctx, item)
		})
		if err != nil {
			return err
		}

		deleteCommands, err := slicesx.MapWithErr(toDelete, func(item T) (CreateBulkType, error) {
			return opts.Create(db, item)
		})
		if err != nil {
			return err
		}

		upsertItems = append(upsertItems, deleteCommands...)
	}

	if len(itemDiff.Create) > 0 {
		toCreate, err := slicesx.MapWithErr(itemDiff.Create, func(item T) (CreateBulkType, error) {
			return opts.Create(db, item)
		})
		if err != nil {
			return err
		}

		upsertItems = append(upsertItems, toCreate...)
	}

	if len(itemDiff.Update) > 0 {
		toUpdate, err := slicesx.MapWithErr(itemDiff.Update, func(item entitydiff.DiffUpdate[T]) (CreateBulkType, error) {
			return opts.Create(db, item.ExpectedState)
		})
		if err != nil {
			return err
		}

		upsertItems = append(upsertItems, toUpdate...)
	}

	if len(upsertItems) > 0 {
		if err := opts.UpsertItems(ctx, db, upsertItems); err != nil {
			return err
		}
	}

	return nil
}
