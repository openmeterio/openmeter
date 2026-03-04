package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *adapter) GetChargeByID(ctx context.Context, input charges.GetChargeByIDInput) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	result, err := a.GetChargesByIDs(ctx, charges.GetChargesByIDsInput{
		Namespace: input.ChargeID.Namespace,
		ChargeIDs: []string{input.ChargeID.ID},
		Expands:   input.Expands,
	})
	if err != nil {
		return charges.Charge{}, err
	}

	if len(result) == 0 {
		return charges.Charge{}, charges.NewChargeNotFoundError(input.ChargeID.Namespace, input.ChargeID.ID)
	}

	return result[0], nil
}

func (a *adapter) GetChargesByIDs(ctx context.Context, input charges.GetChargesByIDsInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if len(input.ChargeIDs) == 0 {
		return nil, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.Charges, error) {
		query := tx.db.Charge.Query().
			Where(dbcharge.Namespace(input.Namespace)).
			Where(dbcharge.IDIn(input.ChargeIDs...))

		if input.Expands.Has(charges.ExpandRealizations) {
			query = query.WithFlatFee(func(q *db.ChargeFlatFeeQuery) {
				q.WithChargeStandardInvoiceAccruedUsage().
					WithChargeStandardInvoicePaymentSettlement().
					WithChargeCreditRealizations()
			}).
				WithUsageBased().
				WithCreditPurchase(func(q *db.ChargeCreditPurchaseQuery) {
					q.WithChargeExternalPaymentSettlement()
				})
		} else {
			query = query.WithFlatFee().
				WithUsageBased().
				WithCreditPurchase()
		}

		entities, err := query.All(ctx)
		if err != nil {
			return nil, fmt.Errorf("querying charges: %w", err)
		}

		return slicesx.MapWithErr(entities, func(entity *db.Charge) (charges.Charge, error) {
			return MapChargeFromDB(entity, input.Expands)
		})
	})
}
