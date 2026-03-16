package adapter

import (
	"context"
	"fmt"
	"strconv"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerdimensiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerdimension"
	dbledgersubaccount "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	dbledgersubaccountroute "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (r *repo) CreateSubAccount(ctx context.Context, input ledgeraccount.CreateSubAccountInput) (*ledgeraccount.SubAccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.SubAccountData, error) {
		route, err := r.resolveOrCreateRoute(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve route: %w", err)
		}

		entity, err := r.db.LedgerSubAccount.Create().
			SetNamespace(input.Namespace).
			SetAnnotations(input.Annotations).
			SetAccountID(input.AccountID).
			SetRouteID(route.ID).
			Save(ctx)
		if err != nil {
			if db.IsConstraintError(err) {
				entity, err = r.db.LedgerSubAccount.Query().
					Where(
						dbledgersubaccount.Namespace(input.Namespace),
						dbledgersubaccount.AccountID(input.AccountID),
						dbledgersubaccount.RouteID(route.ID),
					).
					Only(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve existing ledger sub-account after conflict: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to create ledger sub-account: %w", err)
			}
		}

		// We need to load the edges
		res, err := r.GetSubAccountByID(ctx, models.NamespacedID{
			Namespace: input.Namespace,
			ID:        entity.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get ledger sub-account: %w", err)
		}

		return res, nil
	})
}

func (r *repo) resolveOrCreateRoute(ctx context.Context, input ledgeraccount.CreateSubAccountInput) (*db.LedgerSubAccountRoute, error) {
	routeKey, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, ledger.SubAccountRouteInput{
		CurrencyDimensionID:       input.Dimensions.CurrencyDimensionID,
		TaxCodeDimensionID:        input.Dimensions.TaxCodeDimensionID,
		FeaturesDimensionID:       input.Dimensions.FeaturesDimensionID,
		CreditPriorityDimensionID: input.Dimensions.CreditPriorityDimensionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build routing key: %w", err)
	}

	create := r.db.LedgerSubAccountRoute.Create().
		SetNamespace(input.Namespace).
		SetAccountID(input.AccountID).
		SetRoutingKeyVersion(routeKey.Version()).
		SetCurrencyDimensionID(input.Dimensions.CurrencyDimensionID).
		SetNillableTaxCodeDimensionID(input.Dimensions.TaxCodeDimensionID).
		SetNillableFeaturesDimensionID(input.Dimensions.FeaturesDimensionID).
		SetNillableCreditPriorityDimensionID(input.Dimensions.CreditPriorityDimensionID)

	route, err := create.Save(ctx)
	if err == nil {
		return route, nil
	}

	if !db.IsConstraintError(err) {
		return nil, fmt.Errorf("failed to create sub-account route: %w", err)
	}

	route, err = r.db.LedgerSubAccountRoute.Query().
		Where(
			dbledgersubaccountroute.Namespace(input.Namespace),
			dbledgersubaccountroute.AccountID(input.AccountID),
			dbledgersubaccountroute.RoutingKeyVersion(routeKey.Version()),
			dbledgersubaccountroute.RoutingKey(routeKey.Value()),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve existing route after conflict: %w", err)
	}

	return route, nil
}

func (r *repo) GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*ledgeraccount.SubAccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.SubAccountData, error) {
		entity, err := r.db.LedgerSubAccount.Query().
			Where(dbledgersubaccount.ID(id.ID)).
			Where(dbledgersubaccount.Namespace(id.Namespace)).
			WithRoute(func(query *db.LedgerSubAccountRouteQuery) {
				query.WithCurrencyDimension()
				query.WithTaxCodeDimension()
				query.WithFeaturesDimension()
				query.WithCreditPriorityDimension()
			}).
			WithAccount().
			Only(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get ledger sub-account: %w", err)
		}

		subAccountData, err := MapSubAccountData(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map sub-account data: %w", err)
		}

		return &subAccountData, nil
	})
}

func (r *repo) ListSubAccounts(ctx context.Context, input ledgeraccount.ListSubAccountsInput) ([]*ledgeraccount.SubAccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) ([]*ledgeraccount.SubAccountData, error) {
		predicates := []predicate.LedgerSubAccount{
			dbledgersubaccount.Namespace(input.Namespace),
			dbledgersubaccount.AccountID(input.AccountID),
		}

		routePredicates := make([]predicate.LedgerSubAccountRoute, 0, 3)
		if input.Dimensions.CurrencyID != "" {
			routePredicates = append(routePredicates, dbledgersubaccountroute.CurrencyDimensionID(input.Dimensions.CurrencyID))
		}
		if input.Dimensions.CreditPriority != nil {
			routePredicates = append(routePredicates,
				dbledgersubaccountroute.HasCreditPriorityDimensionWith(
					ledgerdimensiondb.DimensionKey(string(ledger.DimensionKeyCreditPriority)),
					ledgerdimensiondb.DimensionValue(strconv.Itoa(*input.Dimensions.CreditPriority)),
				),
			)
		}
		// DEFERRED: tax/feature filters are not active yet.
		if len(routePredicates) > 0 {
			predicates = append(predicates, dbledgersubaccount.HasRouteWith(routePredicates...))
		}

		entities, err := r.db.LedgerSubAccount.Query().
			Where(predicates...).
			WithRoute(func(query *db.LedgerSubAccountRouteQuery) {
				query.WithCurrencyDimension()
				query.WithTaxCodeDimension()
				query.WithFeaturesDimension()
				query.WithCreditPriorityDimension()
			}).
			WithAccount().
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list ledger sub-accounts: %w", err)
		}

		out := make([]*ledgeraccount.SubAccountData, 0, len(entities))
		for _, entity := range entities {
			subAccountData, err := MapSubAccountData(entity)
			if err != nil {
				return nil, fmt.Errorf("failed to map sub-account data: %w", err)
			}
			out = append(out, &subAccountData)
		}

		return out, nil
	})
}

func MapSubAccountData(entity *db.LedgerSubAccount) (ledgeraccount.SubAccountData, error) {
	if entity.Edges.Account == nil {
		return ledgeraccount.SubAccountData{}, fmt.Errorf("account edge is required")
	}
	if entity.Edges.Route == nil {
		return ledgeraccount.SubAccountData{}, fmt.Errorf("route edge is required")
	}

	route := entity.Edges.Route
	dimensions := ledgeraccount.SubAccountDimensions{}

	if route.Edges.CurrencyDimension == nil {
		return ledgeraccount.SubAccountData{}, fmt.Errorf("currency dimension edge is required")
	}

	currencyDimensionData, err := MapDimensionData(route.Edges.CurrencyDimension)
	if err != nil {
		return ledgeraccount.SubAccountData{}, fmt.Errorf("failed to map currency dimension data: %w", err)
	}

	cDim, err := currencyDimensionData.AsCurrencyDimension()
	if err != nil {
		return ledgeraccount.SubAccountData{}, fmt.Errorf("failed to map currency dimension: %w", err)
	}

	dimensions.Currency = cDim

	if route.Edges.CreditPriorityDimension != nil {
		creditPriorityDimensionData, err := MapDimensionData(route.Edges.CreditPriorityDimension)
		if err != nil {
			return ledgeraccount.SubAccountData{}, fmt.Errorf("failed to map credit priority dimension data: %w", err)
		}
		priorityDim, err := creditPriorityDimensionData.AsCreditPriorityDimension()
		if err != nil {
			return ledgeraccount.SubAccountData{}, fmt.Errorf("failed to map credit priority dimension: %w", err)
		}
		dimensions.CreditPriority = mo.Some[ledger.DimensionCreditPriority](priorityDim)
	}

	return ledgeraccount.SubAccountData{
		ID:          entity.ID,
		Namespace:   entity.Namespace,
		Annotations: entity.Annotations,
		CreatedAt:   entity.CreatedAt,
		AccountID:   entity.AccountID,
		AccountType: entity.Edges.Account.AccountType,
		Dimensions:  dimensions,
		Route: ledgeraccount.SubAccountRouteData{
			ID:                route.ID,
			RoutingKeyVersion: route.RoutingKeyVersion,
			RoutingKey:        route.RoutingKey,
		},
	}, nil
}
