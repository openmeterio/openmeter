package adapter

import (
	"entgo.io/ent/dialect"
	sql "entgo.io/ent/dialect/sql"
	"github.com/lib/pq"

	dbledgerbreakagerecord "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerbreakagerecord"
	ledgersubaccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func expiredRecordRoutePredicate(route ledger.RouteFilter) predicate.LedgerBreakageRecord {
	if route.Currency == "" && route.Features.IsAbsent() && route.MatchFeature == "" {
		return nil
	}

	return func(s *sql.Selector) {
		s.Where(expiredRecordRouteQuery{Route: route}.predicate(s.C(dbledgerbreakagerecord.FieldFboSubAccountID)))
	}
}

type expiredRecordRouteQuery struct {
	Route ledger.RouteFilter
}

func (q expiredRecordRouteQuery) predicate(fboSubAccountIDColumn string) *sql.Predicate {
	return sql.In(fboSubAccountIDColumn, q.selector())
}

func (q expiredRecordRouteQuery) SQL() (string, []any) {
	selector := q.selector()
	selector.SetDialect(dialect.Postgres)

	return selector.Query()
}

func (q expiredRecordRouteQuery) selector() *sql.Selector {
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

func (q expiredRecordRouteQuery) selectorPredicates(routeColumn func(string) string, routeTableAlias string) []*sql.Predicate {
	predicates := make([]*sql.Predicate, 0, 3)

	if q.Route.Currency != "" {
		predicates = append(predicates, sql.EQ(routeColumn(ledgersubaccountroutedb.FieldCurrency), string(q.Route.Currency)))
	}

	if q.Route.Features.IsPresent() {
		features, _ := q.Route.Features.Get()
		features = ledger.SortedFeatures(features)
		if len(features) == 0 {
			predicates = append(predicates, sql.IsNull(routeColumn(ledgersubaccountroutedb.FieldFeatures)))
		} else {
			predicates = append(predicates, postgresArrayRouteExpression{
				Column: postgresQualifiedColumn{
					TableAlias: routeTableAlias,
					Field:      ledgersubaccountroutedb.FieldFeatures,
				},
				Operator: postgresArrayRouteOperatorEqual,
				Value:    pq.StringArray(features),
			}.predicate())
		}
	}

	if q.Route.MatchFeature != "" {
		predicates = append(predicates, sql.Or(
			sql.IsNull(routeColumn(ledgersubaccountroutedb.FieldFeatures)),
			postgresArrayRouteExpression{
				Column: postgresQualifiedColumn{
					TableAlias: routeTableAlias,
					Field:      ledgersubaccountroutedb.FieldFeatures,
				},
				Operator: postgresArrayRouteOperatorContains,
				Value:    pq.StringArray{q.Route.MatchFeature},
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
	TableAlias string
	Field      string
}

func (c postgresQualifiedColumn) ident(b *sql.Builder) {
	b.Ident(c.TableAlias).WriteString(".").Ident(c.Field)
}

type postgresArrayRouteExpression struct {
	Column   postgresQualifiedColumn
	Operator postgresArrayRouteOperator
	Value    pq.StringArray
}

func (e postgresArrayRouteExpression) predicate() *sql.Predicate {
	return sql.P(func(b *sql.Builder) {
		e.appendSQL(b)
	})
}

func (e postgresArrayRouteExpression) appendSQL(b *sql.Builder) {
	e.Column.ident(b)
	b.WriteString(" ").WriteString(string(e.Operator)).WriteString(" ").Arg(e.Value)
}
