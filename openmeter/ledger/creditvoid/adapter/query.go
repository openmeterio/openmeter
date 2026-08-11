package adapter

import (
	sql "entgo.io/ent/dialect/sql"

	dbledgercreditvoidrecord "github.com/openmeterio/openmeter/openmeter/ent/db/ledgercreditvoidrecord"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/internal/routequery"
)

func voidRecordRoutePredicate(route ledger.RouteFilter) (predicate.LedgerCreditVoidRecord, error) {
	if route.Currency.Code == "" && route.Features.IsAbsent() && route.MatchFeature == "" {
		return nil, nil
	}

	query, err := routequery.NewSubAccountIDsByRoute(route)
	if err != nil {
		return nil, err
	}

	return func(s *sql.Selector) {
		s.Where(query.Predicate(s.C(dbledgercreditvoidrecord.FieldFboSubAccountID)))
	}, nil
}
