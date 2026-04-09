package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargessearchv1 "github.com/openmeterio/openmeter/openmeter/ent/db/chargessearchv1"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ charges.ChargesSearchAdapter = (*adapter)(nil)

func (a *adapter) GetByIDs(ctx context.Context, input charges.GetByIDsInput) (charges.ChargeSearchItems, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.ChargeSearchItems, error) {
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

		return lo.Map(resultsInOrder, func(result searchResultIDAccessor, _ int) charges.ChargeSearchItem {
			return mapChargeSearchToChargeWithType(result.ChargesSearchV1)
		}), nil
	})
}

func (a *adapter) ListCharges(ctx context.Context, input charges.ListChargesInput) (pagination.Result[charges.ChargeSearchItem], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[charges.ChargeSearchItem]{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.Result[charges.ChargeSearchItem], error) {
		query := tx.db.ChargesSearchV1.Query().
			Where(dbchargessearchv1.Namespace(input.Namespace))

		if !input.IncludeDeleted {
			query = query.Where(dbchargessearchv1.DeletedAtIsNil())
		}

		if len(input.CustomerIDs) > 0 {
			query = query.Where(dbchargessearchv1.CustomerIDIn(input.CustomerIDs...))
		}

		if len(input.SubscriptionIDs) > 0 {
			query = query.Where(dbchargessearchv1.SubscriptionIDIn(input.SubscriptionIDs...))
		}

		if len(input.ChargeTypes) > 0 {
			query = query.Where(dbchargessearchv1.TypeIn(input.ChargeTypes...))
		}

		if len(input.StatusIn) > 0 {
			query = query.Where(dbchargessearchv1.StatusIn(input.StatusIn...))
		} else if len(input.StatusNotIn) > 0 {
			query = query.Where(dbchargessearchv1.StatusNotIn(input.StatusNotIn...))
		}

		dbEntities, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return pagination.Result[charges.ChargeSearchItem]{}, err
		}

		return pagination.Result[charges.ChargeSearchItem]{
			Page:       dbEntities.Page,
			TotalCount: dbEntities.TotalCount,
			Items: lo.Map(dbEntities.Items, func(item *db.ChargesSearchV1, _ int) charges.ChargeSearchItem {
				return mapChargeSearchToChargeWithType(item)
			}),
		}, nil
	})
}

func (a *adapter) ListCustomersToAdvance(ctx context.Context, input charges.ListCustomersToAdvanceInput) (pagination.Result[customer.CustomerID], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[customer.CustomerID]{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.Result[customer.CustomerID], error) {
		query := tx.db.ChargesSearchV1.Query().
			Where(
				dbchargessearchv1.DeletedAtIsNil(),
				dbchargessearchv1.StatusNotIn(meta.ChargeStatusFinal, meta.ChargeStatusDeleted),
				dbchargessearchv1.AdvanceAfterLTE(input.AdvanceAfterLTE),
			)

		if len(input.Namespaces) > 0 {
			query = query.Where(dbchargessearchv1.NamespaceIn(input.Namespaces...))
		}

		var results []struct {
			Namespace  string `json:"namespace"`
			CustomerID string `json:"customer_id"`
		}

		err := query.
			Order(dbchargessearchv1.ByNamespace(), dbchargessearchv1.ByCustomerID()).
			GroupBy(dbchargessearchv1.FieldNamespace, dbchargessearchv1.FieldCustomerID).
			Scan(ctx, &results)
		if err != nil {
			return pagination.Result[customer.CustomerID]{}, fmt.Errorf("list customers to advance: %w", err)
		}

		// Apply pagination manually since GroupBy doesn't support Paginate directly
		totalCount := len(results)

		page := input.Page
		if page.IsZero() {
			page = pagination.Page{
				PageSize:   totalCount,
				PageNumber: 1,
			}
		}

		start := page.Offset()
		if start > totalCount {
			start = totalCount
		}
		end := start + page.Limit()
		if end > totalCount {
			end = totalCount
		}

		pageResults := results[start:end]
		customers := make([]customer.CustomerID, 0, len(pageResults))
		for _, r := range pageResults {
			customers = append(customers, customer.CustomerID{
				Namespace: r.Namespace,
				ID:        r.CustomerID,
			})
		}

		return pagination.Result[customer.CustomerID]{
			Page:       page,
			TotalCount: totalCount,
			Items:      customers,
		}, nil
	})
}

func mapChargeSearchToChargeWithType(item *db.ChargesSearchV1) charges.ChargeSearchItem {
	return charges.ChargeSearchItem{
		ID:         meta.ChargeID{Namespace: item.Namespace, ID: item.ID},
		Type:       item.Type,
		CustomerID: item.CustomerID,
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
