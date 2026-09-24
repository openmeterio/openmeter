package routequery

import (
	"fmt"

	"entgo.io/ent/dialect"
	sql "entgo.io/ent/dialect/sql"

	ledgersubaccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// SubAccountIDsByRoute selects ledger subaccount IDs whose persisted route matches a filter.
type SubAccountIDsByRoute struct {
	route              ledger.RouteFilter
	serializedCurrency *string
	currencyPrefix     bool
}

// NewSubAccountIDsByRoute prepares a route query, including its persisted currency representation.
func NewSubAccountIDsByRoute(route ledger.RouteFilter) (SubAccountIDsByRoute, error) {
	query := SubAccountIDsByRoute{route: route}
	if route.Currency.Code == "" {
		return query, nil
	}

	var serialized []byte
	var err error
	if route.Currency.IsCustom() && !route.Currency.IsResolved() {
		serialized, err = route.Currency.MarshalTextPrefix()
		query.currencyPrefix = true
	} else {
		serialized, err = route.Currency.MarshalText()
	}
	if err != nil {
		return SubAccountIDsByRoute{}, fmt.Errorf("serialize route currency filter: %w", err)
	}

	value := string(serialized)
	query.serializedCurrency = &value

	return query, nil
}

// Predicate matches a subaccount ID column against the selected IDs.
func (q SubAccountIDsByRoute) Predicate(subAccountIDColumn string) *sql.Predicate {
	return sql.In(subAccountIDColumn, q.selector())
}

func (q SubAccountIDsByRoute) sql() (string, []any) {
	selector := q.selector()
	selector.SetDialect(dialect.Postgres)

	return selector.Query()
}

func (q SubAccountIDsByRoute) selector() *sql.Selector {
	const (
		subAccountTableAlias = "lsa"
		routeTableAlias      = "lsar"
	)

	subAccounts := sql.Table(ledgersubaccountdb.Table).As(subAccountTableAlias)
	routes := sql.Table(ledgersubaccountroutedb.Table).As(routeTableAlias)

	selector := sql.Select(subAccounts.C(ledgersubaccountdb.FieldID)).
		From(subAccounts).
		Join(routes).
		On(subAccounts.C(ledgersubaccountdb.FieldRouteID), routes.C(ledgersubaccountroutedb.FieldID))

	for _, predicate := range q.selectorPredicates(routes.C) {
		selector.Where(predicate)
	}

	return selector
}

func (q SubAccountIDsByRoute) selectorPredicates(routeColumn func(string) string) []*sql.Predicate {
	predicates := make([]*sql.Predicate, 0, 3)

	if q.serializedCurrency != nil {
		if q.currencyPrefix {
			predicates = append(predicates, sql.Like(routeColumn(ledgersubaccountroutedb.FieldCurrency), *q.serializedCurrency+"%"))
		} else {
			predicates = append(predicates, sql.EQ(routeColumn(ledgersubaccountroutedb.FieldCurrency), *q.serializedCurrency))
		}
	}

	if exact, ok := q.route.CreditFilters.Get(); ok {
		predicates = append(predicates, ExactFiltersPredicate(routeColumn, exact))
	}
	if plan, ok := q.route.MatchPlan.Get(); ok {
		predicates = append(predicates, MatchPlanPredicate(routeColumn, plan))
	}
	if features, ok := q.route.Features.Get(); ok {
		predicates = append(predicates, ExactFeaturesPredicate(routeColumn, features))
	}
	if q.route.MatchFeature != "" {
		predicates = append(predicates, MatchFeaturePredicate(routeColumn, q.route.MatchFeature))
	}
	return predicates
}
