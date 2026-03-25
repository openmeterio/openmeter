package adapter

import (
	"context"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargessearchv1 "github.com/openmeterio/openmeter/openmeter/ent/db/chargessearchv1"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ charges.ChargesSearchAdapter = (*adapter)(nil)

func (a *adapter) GetTypesByIDs(ctx context.Context, input charges.GetTypesByIDsInput) (charges.GetTypesByIDsResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.GetTypesByIDsResult, error) {
		dbCharges, err := tx.db.ChargesSearchV1.Query().
			Where(dbchargessearchv1.Namespace(input.Namespace)).
			Where(dbchargessearchv1.IDIn(input.IDs...)).
			All(ctx)
		if err != nil {
			return nil, err
		}

		// Apply namespace filtering/ID checks
		resultsInOrder, err := entutils.InIDOrder(input.Namespace, input.IDs, withIDAccessor(dbCharges))
		if err != nil {
			return nil, err
		}

		return lo.Map(resultsInOrder, func(result searchResultIDAccessor, _ int) charges.ChargeWithType {
			return mapChargeSearchToChargeWithType(result.ChargesSearchV1)
		}), nil
	})
}

func (a *adapter) ListCharges(ctx context.Context, input charges.ListChargesInput) (pagination.Result[charges.ChargeWithType], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[charges.ChargeWithType]{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.Result[charges.ChargeWithType], error) {
		query := tx.db.ChargesSearchV1.Query().
			Where(dbchargessearchv1.Namespace(input.Namespace))

		if len(input.CustomerIDs) > 0 {
			query = query.Where(dbchargessearchv1.CustomerIDIn(input.CustomerIDs...))
		}

		if len(input.StatusNotIn) > 0 {
			query = query.Where(dbchargessearchv1.StatusNotIn(input.StatusNotIn...))
		}

		dbEntities, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return pagination.Result[charges.ChargeWithType]{}, err
		}

		return pagination.Result[charges.ChargeWithType]{
			Page:       dbEntities.Page,
			TotalCount: dbEntities.TotalCount,
			Items: lo.Map(dbEntities.Items, func(item *db.ChargesSearchV1, _ int) charges.ChargeWithType {
				return mapChargeSearchToChargeWithType(item)
			}),
		}, nil
	})
}

func mapChargeSearchToChargeWithType(item *db.ChargesSearchV1) charges.ChargeWithType {
	return charges.ChargeWithType{
		ID:   item.ID,
		Type: item.Type,
	}
}

var _ entutils.InIDOrderAccessor = (*searchResultIDAccessor)(nil)

type searchResultIDAccessor struct {
	*db.ChargesSearchV1
}

func (s searchResultIDAccessor) GetID() string {
	return s.ID
}

func (s searchResultIDAccessor) GetNamespace() string {
	return s.Namespace
}

func (s searchResultIDAccessor) GetChargeID() meta.ChargeID {
	return meta.ChargeID{
		Namespace: s.Namespace,
		ID:        s.ID,
	}
}

func withIDAccessor(entity []*db.ChargesSearchV1) []searchResultIDAccessor {
	return lo.Map(entity, func(entity *db.ChargesSearchV1, _ int) searchResultIDAccessor {
		return searchResultIDAccessor{
			ChargesSearchV1: entity,
		}
	})
}
