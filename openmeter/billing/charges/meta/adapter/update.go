package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) UpdateStatus(ctx context.Context, in meta.UpdateStatusInput) (meta.Charge, error) {
	if err := in.Validate(); err != nil {
		return meta.Charge{}, fmt.Errorf("invalid input: %w", err)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (meta.Charge, error) {
		dbEntity, err := tx.db.Charge.UpdateOneID(in.ChargeID.ID).
			Where(dbcharge.NamespaceEQ(in.ChargeID.Namespace)).
			SetStatus(in.Status).
			SetOrClearAdvanceAfter(convert.SafeToUTC(in.AdvanceAfter)).
			Save(ctx)
		if err != nil {
			return meta.Charge{}, fmt.Errorf("failed to update charge: %w", err)
		}

		return MapChargeFromDB(dbEntity), err
	})
}
