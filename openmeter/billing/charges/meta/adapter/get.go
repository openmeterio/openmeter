package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbcharge "github.com/openmeterio/openmeter/openmeter/ent/db/charge"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) GetByIDs(ctx context.Context, input meta.GetByIDsInput) (meta.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if len(input.ChargeIDs) == 0 {
		return nil, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (meta.Charges, error) {
		dbEntities, err := tx.db.Charge.Query().
			Where(dbcharge.Namespace(input.Namespace)).
			Where(dbcharge.IDIn(input.ChargeIDs...)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("querying charges: %w", err)
		}

		return lo.Map(dbEntities, func(entity *entdb.Charge, idx int) meta.Charge {
			return MapChargeFromDB(entity)
		}), nil
	})
}

func (a *adapter) ListByCustomer(ctx context.Context, input meta.ListByCustomerInput) (meta.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (meta.Charges, error) {
		dbEntities, err := tx.db.Charge.Query().
			Where(dbcharge.Namespace(input.Customer.Namespace)).
			Where(dbcharge.CustomerID(input.Customer.ID)).
			Where(dbcharge.StatusNEQ(meta.ChargeStatusFinal)).
			Order(
				dbcharge.ByCreatedAt(),
				dbcharge.ByID(),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("querying charges by customer: %w", err)
		}

		return lo.Map(dbEntities, func(entity *entdb.Charge, idx int) meta.Charge {
			return MapChargeFromDB(entity)
		}), nil
	})
}
