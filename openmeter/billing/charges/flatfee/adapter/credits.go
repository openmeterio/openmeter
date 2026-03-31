package adapter

import (
	"context"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) CreateCreditAllocations(ctx context.Context, chargeID meta.ChargeID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	if err := creditAllocations.Validate(); err != nil {
		return creditrealization.Realizations{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		dbEntities, err := tx.db.ChargeFlatFeeCreditAllocations.CreateBulk(
			lo.Map(creditAllocations, func(creditAllocation creditrealization.CreateInput, idx int) *db.ChargeFlatFeeCreditAllocationsCreate {
				create := tx.db.ChargeFlatFeeCreditAllocations.Create().
					SetChargeID(chargeID.ID)

				create = creditrealization.Create(create, chargeID.Namespace, idx, creditAllocation)

				return create
			})...,
		).Save(ctx)
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		return slicesx.MapWithErr(dbEntities, func(entity *db.ChargeFlatFeeCreditAllocations) (creditrealization.Realization, error) {
			return creditrealization.MapFromDB(entity), nil
		})
	})
}
