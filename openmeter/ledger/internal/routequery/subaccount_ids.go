package routequery

import (
	"fmt"

	"entgo.io/ent/dialect"
	sql "entgo.io/ent/dialect/sql"
	"github.com/lib/pq"

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

	for _, predicate := range q.selectorPredicates(routes.C, routeTableAlias) {
		selector.Where(predicate)
	}

	return selector
}

func (q SubAccountIDsByRoute) selectorPredicates(routeColumn func(string) string, routeTableAlias string) []*sql.Predicate {
	predicates := make([]*sql.Predicate, 0, 3)

	if q.serializedCurrency != nil {
		if q.currencyPrefix {
			predicates = append(predicates, sql.Like(routeColumn(ledgersubaccountroutedb.FieldCurrency), *q.serializedCurrency+"%"))
		} else {
			predicates = append(predicates, sql.EQ(routeColumn(ledgersubaccountroutedb.FieldCurrency), *q.serializedCurrency))
		}
	}

	if q.route.Features.IsPresent() {
		features, _ := q.route.Features.Get()
		features = ledger.SortedFeatures(features)
		if len(features) == 0 {
			predicates = append(predicates, sql.IsNull(routeColumn(ledgersubaccountroutedb.FieldFeatures)))
		} else {
			predicates = append(predicates, postgresArrayRouteExpression{
				column: postgresQualifiedColumn{
					tableAlias: routeTableAlias,
					field:      ledgersubaccountroutedb.FieldFeatures,
				},
				operator: postgresArrayRouteOperatorEqual,
				value:    pq.StringArray(features),
			}.predicate())
		}
	}

	if q.route.MatchFeature != "" {
		predicates = append(predicates, sql.Or(
			sql.IsNull(routeColumn(ledgersubaccountroutedb.FieldFeatures)),
			postgresArrayRouteExpression{
				column: postgresQualifiedColumn{
					tableAlias: routeTableAlias,
					field:      ledgersubaccountroutedb.FieldFeatures,
				},
				operator: postgresArrayRouteOperatorContains,
				value:    pq.StringArray{q.route.MatchFeature},
			}.predicate(),
		))
	}

	return predicates
}

type postgresArrayRouteOperator string

const (
	postgresArrayRouteOperatorEqual    postgresArrayRouteOperator = "="
	postgresArrayRouteOperatorContains postgresArrayRouteOperator = "@>"
)

type postgresQualifiedColumn struct {
	tableAlias string
	field      string
}

func (c postgresQualifiedColumn) appendSQL(b *sql.Builder) {
	b.Ident(c.tableAlias).WriteString(".").Ident(c.field)
}

type postgresArrayRouteExpression struct {
	column   postgresQualifiedColumn
	operator postgresArrayRouteOperator
	value    pq.StringArray
}

func (e postgresArrayRouteExpression) predicate() *sql.Predicate {
	return sql.P(func(b *sql.Builder) {
		e.column.appendSQL(b)
		b.WriteString(" ").WriteString(string(e.operator)).WriteString(" ").Arg(e.value)
	})
}
