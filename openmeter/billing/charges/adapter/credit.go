package adapter

import (
	"context"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CreateCreditRealizations(ctx context.Context, chargeID charges.ChargeID, realizations []charges.CreditRealizationCreateInput) (charges.CreditRealizations, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.CreditRealizations, error) {
		creates := lo.Map(realizations, func(realization charges.CreditRealizationCreateInput, _ int) *db.ChargeCreditRealizationCreate {
			return tx.db.ChargeCreditRealization.Create().
				SetNamespace(chargeID.Namespace).
				SetChargeID(chargeID.ID).
				SetAnnotations(realization.Annotations).
				SetServicePeriodFrom(realization.ServicePeriod.From.In(time.UTC)).
				SetServicePeriodTo(realization.ServicePeriod.To.In(time.UTC)).
				SetAmount(realization.Amount)
		})

		entities, err := tx.db.ChargeCreditRealization.CreateBulk(creates...).Save(ctx)
		if err != nil {
			return nil, err
		}

		return lo.Map(entities, func(entity *db.ChargeCreditRealization, _ int) charges.CreditRealization {
			return mapCreditRealizationFromDB(entity)
		}), nil
	})
}
