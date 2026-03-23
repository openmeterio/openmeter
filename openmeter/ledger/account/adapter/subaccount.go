package adapter

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbledgersubaccount "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	dbledgersubaccountroute "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (r *repo) EnsureSubAccount(ctx context.Context, input ledgeraccount.CreateSubAccountInput) (*ledgeraccount.SubAccountData, error) {
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

// We can use this upsert pattern for routes as they're a hidden internal & in practice routes would be shared between subaccounts.
// Not making Routes a dependent of SubAccounts makes sense as the Routes table gives us meaningful information on what "type of currencies" we hold without the structural details of how subaccounts are grouped, e.g. its easy to see from the routes table if we hold EUR or USD...
func (r *repo) resolveOrCreateRoute(ctx context.Context, input ledgeraccount.CreateSubAccountInput) (*db.LedgerSubAccountRoute, error) {
	normalizedRoute, err := input.Route.Normalize()
	if err != nil {
		return nil, fmt.Errorf("failed to normalize route: %w", err)
	}

	routeKey, err := ledger.BuildRoutingKey(ledger.RoutingKeyVersionV1, normalizedRoute)
	if err != nil {
		return nil, fmt.Errorf("failed to build routing key: %w", err)
	}

	create := r.db.LedgerSubAccountRoute.Create().
		SetNamespace(input.Namespace).
		SetAccountID(input.AccountID).
		SetRoutingKeyVersion(routeKey.Version()).
		SetRoutingKey(routeKey.Value()).
		SetCurrency(string(normalizedRoute.Currency)).
		SetNillableTaxCode(normalizedRoute.TaxCode).
		SetFeatures(normalizedRoute.Features).
		SetNillableCostBasis(normalizedRoute.CostBasis).
		SetNillableCreditPriority(normalizedRoute.CreditPriority)

	routeEntity, err := create.Save(ctx)
	if err == nil {
		return routeEntity, nil
	}

	if !db.IsConstraintError(err) {
		return nil, fmt.Errorf("failed to create sub-account route: %w", err)
	}

	routeEntity, err = r.db.LedgerSubAccountRoute.Query().
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

	return routeEntity, nil
}

func (r *repo) GetSubAccountByID(ctx context.Context, id models.NamespacedID) (*ledgeraccount.SubAccountData, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgeraccount.SubAccountData, error) {
		entity, err := r.db.LedgerSubAccount.Query().
			Where(dbledgersubaccount.ID(id.ID)).
			Where(dbledgersubaccount.Namespace(id.Namespace)).
			WithRoute().
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

		normalizedRoute, err := input.Route.Normalize()
		if err != nil {
			return nil, fmt.Errorf("failed to normalize route filter: %w", err)
		}

		routePredicates := make([]predicate.LedgerSubAccountRoute, 0, 5)
		if normalizedRoute.Currency != "" {
			routePredicates = append(routePredicates, dbledgersubaccountroute.Currency(string(normalizedRoute.Currency)))
		}
		if normalizedRoute.CreditPriority != nil {
			routePredicates = append(routePredicates,
				dbledgersubaccountroute.CreditPriority(*normalizedRoute.CreditPriority),
			)
		}
		// DEFERRED: tax/feature route filters are not active yet but plumbing is in place.
		if normalizedRoute.TaxCode != nil {
			routePredicates = append(routePredicates, dbledgersubaccountroute.TaxCode(*normalizedRoute.TaxCode))
		}
		if len(normalizedRoute.Features) > 0 {
			// DB stores features as a sorted jsonb array; filter value is also sorted for canonical comparison.
			routePredicates = append(routePredicates, func(s *sql.Selector) {
				s.Where(sqljson.ValueEQ(dbledgersubaccountroute.FieldFeatures, normalizedRoute.Features))
			})
		}
		if normalizedRoute.CostBasis != nil {
			routePredicates = append(routePredicates, dbledgersubaccountroute.CostBasis(*normalizedRoute.CostBasis))
		}
		if len(routePredicates) > 0 {
			predicates = append(predicates, dbledgersubaccount.HasRouteWith(routePredicates...))
		}

		entities, err := r.db.LedgerSubAccount.Query().
			Where(predicates...).
			WithRoute().
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

	dbRoute := entity.Edges.Route

	return ledgeraccount.SubAccountData{
		ID:          entity.ID,
		Namespace:   entity.Namespace,
		Annotations: entity.Annotations,
		CreatedAt:   entity.CreatedAt,
		AccountID:   entity.AccountID,
		AccountType: entity.Edges.Account.AccountType,
		Route: ledger.Route{
			Currency:       currencyx.Code(dbRoute.Currency),
			TaxCode:        dbRoute.TaxCode,
			Features:       dbRoute.Features,
			CostBasis:      dbRoute.CostBasis,
			CreditPriority: dbRoute.CreditPriority,
		},
		RouteMeta: ledgeraccount.SubAccountRouteData{
			ID:                dbRoute.ID,
			RoutingKeyVersion: dbRoute.RoutingKeyVersion,
			RoutingKey:        dbRoute.RoutingKey,
		},
	}, nil
}
