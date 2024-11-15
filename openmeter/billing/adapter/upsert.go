package billingadapter

import (
	"context"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type upsertInput[T any, CreateBulkType any] struct {
	Create      func(T) (CreateBulkType, error)
	UpsertItems func(context.Context, *entdb.Client, []CreateBulkType) error
	Delete      func(context.Context, *entdb.Client, []T) error
}

type upsertOption[T any, CreateBulkType any] func(upsertInput[T, CreateBulkType]) upsertInput[T, CreateBulkType]

func upsertWithOptions[T any, CreateBulkType any](ctx context.Context, db *entdb.Client, itemDiff diff[T], baseSettings upsertInput[T, CreateBulkType], options ...upsertOption[T, CreateBulkType]) error {
	opts := baseSettings
	for _, option := range options {
		opts = option(opts)
	}

	// Delete must be first, as we might have a constraint that prevents us from creating the item if not deleted before.
	if len(itemDiff.ToDelete) > 0 && opts.Delete != nil {
		if err := opts.Delete(ctx, db, itemDiff.ToDelete); err != nil {
			return err
		}
	}

	upsertItems := make([]CreateBulkType, 0, len(itemDiff.ToCreate)+len(itemDiff.ToUpdate))

	if len(itemDiff.ToCreate) > 0 {
		toCreate, err := slicesx.MapWithErr(itemDiff.ToCreate, opts.Create)
		if err != nil {
			return err
		}

		upsertItems = append(upsertItems, toCreate...)
	}

	if len(itemDiff.ToUpdate) > 0 {
		toUpdate, err := slicesx.MapWithErr(itemDiff.ToUpdate, opts.Create)
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
