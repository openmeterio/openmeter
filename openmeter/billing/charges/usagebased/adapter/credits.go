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

func (a *adapter) CreateChargeCurrencyCreditRealizations(ctx context.Context, input usagebased.CreateCreditRealizationsInput) (creditrealization.Realizations, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		creates := lo.Map(input.CreditRealizations, func(realization creditrealization.CreateInput, idx int) *entdb.ChargeUsageBasedRunCreditAllocationsCreate {
			create := tx.db.ChargeUsageBasedRunCreditAllocations.Create().
				SetRunID(input.RunID.ID).
				SetNamespace(input.RunID.Namespace)

			create = creditrealization.Create(create, input.RunID.Namespace, idx, realization)

			return create
		})

		dbEntities, err := tx.db.ChargeUsageBasedRunCreditAllocations.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		return creditrealization.FromDBRealizations(dbEntities), nil
	})
}

func (a *adapter) CreateFiatOverageCreditRealizations(ctx context.Context, input usagebased.CreateCreditRealizationsInput) (creditrealization.Realizations, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (creditrealization.Realizations, error) {
		creates := lo.Map(input.CreditRealizations, func(realization creditrealization.CreateInput, idx int) *entdb.ChargeUsageBasedRunOverageCreditAllocationsCreate {
			create := tx.db.ChargeUsageBasedRunOverageCreditAllocations.Create().
				SetRunID(input.RunID.ID).
				SetNamespace(input.RunID.Namespace)

			create = creditrealization.Create(create, input.RunID.Namespace, idx, realization)

			return create
		})

		dbEntities, err := tx.db.ChargeUsageBasedRunOverageCreditAllocations.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		return creditrealization.FromDBRealizations(dbEntities), nil
	})
}
