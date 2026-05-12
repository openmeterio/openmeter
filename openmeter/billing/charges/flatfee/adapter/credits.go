package adapter

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) CreateCreditAllocations(ctx context.Context, chargeID meta.ChargeID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	if err := creditAllocations.Validate(); err != nil {
		return creditrealization.Realizations{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		run, err := tx.currentRunByChargeID(ctx, chargeID)
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		dbEntities, err := tx.db.ChargeFlatFeeRunCreditAllocations.CreateBulk(
			lo.Map(creditAllocations, func(creditAllocation creditrealization.CreateInput, idx int) *db.ChargeFlatFeeRunCreditAllocationsCreate {
				create := tx.db.ChargeFlatFeeRunCreditAllocations.Create().
					SetRunID(run.ID)

				create = creditrealization.Create(create, chargeID.Namespace, idx, creditAllocation)

				return create
			})...,
		).Save(ctx)
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		if err := tx.updateCreditOnlyRunTotals(ctx, chargeID, run, creditAllocations); err != nil {
			return creditrealization.Realizations{}, err
		}

		realizations, err := slicesx.MapWithErr(dbEntities, func(entity *db.ChargeFlatFeeRunCreditAllocations) (creditrealization.Realization, error) {
			return creditrealization.MapFromDB(entity), nil
		})
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		return realizations, nil
	})
}

func (a *adapter) updateCreditOnlyRunTotals(
	ctx context.Context,
	chargeID meta.ChargeID,
	run *db.ChargeFlatFeeRun,
	creditAllocations creditrealization.CreateInputs,
) error {
	if len(creditAllocations) == 0 {
		return nil
	}

	for _, creditAllocation := range creditAllocations {
		if creditAllocation.Type != creditrealization.TypeAllocation {
			return nil
		}
	}

	charge, err := a.db.ChargeFlatFee.Query().
		Where(
			dbchargeflatfee.NamespaceEQ(chargeID.Namespace),
			dbchargeflatfee.IDEQ(chargeID.ID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	if charge.SettlementMode != productcatalog.CreditOnlySettlementMode {
		return nil
	}

	allocatedAmount := creditAllocations.Sum()
	_, err = a.updateCurrentRunTotals(ctx, run, totals.Totals{
		Amount:       allocatedAmount,
		ChargesTotal: allocatedAmount,
		CreditsTotal: allocatedAmount,
		Total:        alpacadecimal.Zero,
	})

	return err
}
