package adapter

import (
	"entgo.io/ent/dialect/sql"
	"github.com/lib/pq"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// featureKeyFilterPredicate builds the feature restriction predicate for
// listing charges. A keyed filter uses the Postgres array overlap operator
// (&&): the charge matches when its restriction includes ANY of the requested
// keys, and unrestricted charges (NULL or empty restriction) never match a
// keyed filter. An unrestricted charge is stored as NULL when the filters are
// omitted, but an explicit empty features list persists as '{}' — the
// exists predicates treat both as unrestricted.
func featureKeyFilterPredicate(f *creditpurchase.FeatureKeyFilter) predicate.ChargeCreditPurchase {
	if f == nil {
		return nil
	}

	if f.Exists != nil {
		if *f.Exists {
			return dbchargecreditpurchase.And(
				dbchargecreditpurchase.FeatureFiltersNotNil(),
				dbchargecreditpurchase.FeatureFiltersNEQ(pq.StringArray{}),
			)
		}

		return dbchargecreditpurchase.Or(
			dbchargecreditpurchase.FeatureFiltersIsNil(),
			dbchargecreditpurchase.FeatureFiltersEQ(pq.StringArray{}),
		)
	}

	if len(f.In) == 0 {
		return nil
	}

	return func(s *sql.Selector) {
		s.Where(sql.P(func(b *sql.Builder) {
			b.Ident(s.C(dbchargecreditpurchase.FieldFeatureFilters)).WriteString(" && ").Arg(pq.StringArray(f.In))
		}))
	}
}
