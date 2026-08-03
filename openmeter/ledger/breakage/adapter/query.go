package adapter

import (
	"fmt"

	"entgo.io/ent/dialect"
	sql "entgo.io/ent/dialect/sql"
	"github.com/lib/pq"

	dbledgerbreakagerecord "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerbreakagerecord"
	ledgersubaccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func expiredRecordRoutePredicate(route ledger.RouteFilter) (predicate.LedgerBreakageRecord, error) {
	if route.Currency.Code == "" && route.Features.IsAbsent() && route.MatchFeature == "" {
		return nil, nil
	}

	query, err := newExpiredRecordRouteQuery(route)
	if err != nil {
		return nil, err
	}

	return func(s *sql.Selector) {
		s.Where(query.predicate(s.C(dbledgerbreakagerecord.FieldFboSubAccountID)))
	}, nil
}

type expiredRecordRouteQuery struct {
	Route              ledger.RouteFilter
	serializedCurrency *string
	currencyPrefix     bool
}

func newExpiredRecordRouteQuery(route ledger.RouteFilter) (expiredRecordRouteQuery, error) {
	query := expiredRecordRouteQuery{Route: route}
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
		return expiredRecordRouteQuery{}, fmt.Errorf("serialize route currency filter: %w", err)
	}
	value := string(serialized)
	query.serializedCurrency = &value

	return query, nil
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

	if q.serializedCurrency != nil {
		if q.currencyPrefix {
			predicates = append(predicates, sql.Like(routeColumn(ledgersubaccountroutedb.FieldCurrency), *q.serializedCurrency+"%"))
		} else {
			predicates = append(predicates, sql.EQ(routeColumn(ledgersubaccountroutedb.FieldCurrency), *q.serializedCurrency))
		}
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
