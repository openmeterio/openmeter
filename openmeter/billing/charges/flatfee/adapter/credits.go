package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfeerun "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerun"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ flatfee.ChargeCreditAllocationAdapter = (*adapter)(nil)

func (a *adapter) CreateCreditAllocations(ctx context.Context, runID flatfee.RealizationRunID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	if err := runID.Validate(); err != nil {
		return creditrealization.Realizations{}, err
	}

	if err := creditAllocations.Validate(); err != nil {
		return creditrealization.Realizations{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		if _, err := tx.db.ChargeFlatFeeRun.Query().
			Where(
				dbchargeflatfeerun.NamespaceEQ(runID.Namespace),
				dbchargeflatfeerun.IDEQ(runID.ID),
			).
			Only(ctx); err != nil {
			return creditrealization.Realizations{}, fmt.Errorf("querying flat fee run [run_id=%s]: %w", runID.ID, err)
		}

		dbEntities, err := tx.db.ChargeFlatFeeRunCreditAllocations.CreateBulk(
			lo.Map(creditAllocations, func(creditAllocation creditrealization.CreateInput, idx int) *db.ChargeFlatFeeRunCreditAllocationsCreate {
				create := tx.db.ChargeFlatFeeRunCreditAllocations.Create().
					SetRunID(runID.ID)

				create = creditrealization.Create(create, runID.Namespace, idx, creditAllocation)

				return create
			})...,
		).Save(ctx)
		if err != nil {
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
