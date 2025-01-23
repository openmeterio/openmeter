package billingadapter

import (
	"context"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type upsertInput[T any, CreateBulkType any] struct {
	Create      func(*entdb.Client, T) (CreateBulkType, error)
	UpsertItems func(context.Context, *entdb.Client, []CreateBulkType) error
	MarkDeleted func(context.Context, T) (T, error)
}

type upsertOption[T any, CreateBulkType any] func(upsertInput[T, CreateBulkType]) upsertInput[T, CreateBulkType]

func upsertWithOptions[T any, CreateBulkType any](ctx context.Context, db *entdb.Client, itemDiff diff[T], baseSettings upsertInput[T, CreateBulkType], options ...upsertOption[T, CreateBulkType]) error {
	opts := baseSettings
	for _, option := range options {
		opts = option(opts)
	}

	upsertItems := make([]CreateBulkType, 0, len(itemDiff.ToCreate)+len(itemDiff.ToUpdate)+len(itemDiff.ToDelete))

	// Delete must be first, as we might have a constraint that prevents us from creating the item if not deleted before.
	if len(itemDiff.ToDelete) > 0 && opts.MarkDeleted != nil {
		// We formulate delete as a soft delete update, so that any changes happening alongside the deletion are persisted
		// to the database.

		toDelete, err := slicesx.MapWithErr(itemDiff.ToDelete, func(item T) (T, error) {
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

	if len(itemDiff.ToCreate) > 0 {
		toCreate, err := slicesx.MapWithErr(itemDiff.ToCreate, func(item T) (CreateBulkType, error) {
			return opts.Create(db, item)
		})
		if err != nil {
			return err
		}

		upsertItems = append(upsertItems, toCreate...)
	}

	if len(itemDiff.ToUpdate) > 0 {
		toUpdate, err := slicesx.MapWithErr(itemDiff.ToUpdate, func(item T) (CreateBulkType, error) {
			return opts.Create(db, item)
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
