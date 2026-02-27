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

func (a *adapter) GetChargeByID(ctx context.Context, input charges.ChargeID) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	result, err := a.GetChargesByIDs(ctx, input.Namespace, []string{input.ID})
	if err != nil {
		return charges.Charge{}, err
	}

	if len(result) == 0 {
		return charges.Charge{}, charges.NewChargeNotFoundError(input.Namespace, input.ID)
	}

	return result[0], nil
}

func (a *adapter) GetChargesByIDs(ctx context.Context, namespace string, ids []string) (charges.Charges, error) {
	if namespace == "" {
		return nil, charges.ErrChargeNamespaceEmpty
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.Charges, error) {
		entities, err := tx.db.Charge.Query().
			Where(dbcharge.Namespace(namespace)).
			Where(dbcharge.IDIn(ids...)).
			WithFlatFee().
			WithUsageBased().
			WithCreditPurchase().
			WithCreditRealizations().
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("querying charges: %w", err)
		}

		return slicesx.MapWithErr(entities, func(entity *db.Charge) (charges.Charge, error) {
			return MapChargeFromDB(entity)
		})
	})
}
