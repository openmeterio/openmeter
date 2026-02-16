package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/charges"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) DeleteChargesByUniqueReferenceID(ctx context.Context, input charges.DeleteChargesByUniqueReferenceIDInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if len(input.UniqueReferenceIDs) == 0 {
		return nil
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.Charge.Update().
			Where(dbcharge.Namespace(input.Customer.Namespace)).
			Where(dbcharge.CustomerID(input.Customer.ID)).
			Where(dbcharge.UniqueReferenceIDIn(input.UniqueReferenceIDs...)).
			Where(dbcharge.UniqueReferenceIDNotNil()).
			Where(dbcharge.DeletedAtIsNil()).
			SetDeletedAt(clock.Now().UTC()).
			Exec(ctx)
	})
}
