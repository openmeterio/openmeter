package adapter

import (
	"context"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ flatfee.ChargeCreditAllocationAdapter = (*adapter)(nil)

func (a *adapter) CreateChargeCurrencyCreditRealizations(ctx context.Context, input flatfee.CreateCreditRealizationsInput) (creditrealization.Realizations, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		dbEntities, err := tx.db.ChargeFlatFeeRunCreditAllocations.CreateBulk(
			lo.Map(input.CreditRealizations, func(realization creditrealization.CreateInput, idx int) *db.ChargeFlatFeeRunCreditAllocationsCreate {
				create := tx.db.ChargeFlatFeeRunCreditAllocations.Create().
					SetRunID(input.RunID.ID)
				create = creditrealization.Create(create, input.RunID.Namespace, idx, realization)

				return create
			})...,
		).Save(ctx)
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		return creditrealization.FromDBRealizations(dbEntities), nil
	})
}

func (a *adapter) CreateFiatOverageCreditRealizations(ctx context.Context, input flatfee.CreateCreditRealizationsInput) (creditrealization.Realizations, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		dbEntities, err := tx.db.ChargeFlatFeeRunOverageCreditAllocations.CreateBulk(
			lo.Map(input.CreditRealizations, func(realization creditrealization.CreateInput, idx int) *db.ChargeFlatFeeRunOverageCreditAllocationsCreate {
				create := tx.db.ChargeFlatFeeRunOverageCreditAllocations.Create().
					SetRunID(input.RunID.ID)
				create = creditrealization.Create(create, input.RunID.Namespace, idx, realization)

				return create
			})...,
		).Save(ctx)
		if err != nil {
			return creditrealization.Realizations{}, err
		}

		return creditrealization.FromDBRealizations(dbEntities), nil
	})
}
