package adapter

import (
	"context"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ usagebased.RealizationRunCreditAllocationAdapter = (*adapter)(nil)

func (a *adapter) CreateRunCreditAllocations(ctx context.Context, runID usagebased.RealizationRunID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	if err := runID.Validate(); err != nil {
		return nil, err
	}

	if err := creditAllocations.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		creates := lo.Map(creditAllocations, func(creditAllocation creditrealization.CreateInput, idx int) *entdb.ChargeUsageBasedRunCreditAllocationsCreate {
			create := tx.db.ChargeUsageBasedRunCreditAllocations.Create().
				SetRunID(runID.ID).
				SetNamespace(runID.Namespace)

			create = creditrealization.Create(create, runID.Namespace, idx, creditAllocation)

			return create
		})

		dbEntities, err := tx.db.ChargeUsageBasedRunCreditAllocations.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		return lo.Map(dbEntities, func(entity *entdb.ChargeUsageBasedRunCreditAllocations, _ int) creditrealization.Realization {
			return creditrealization.MapFromDB(entity)
		}), nil
	})
}
