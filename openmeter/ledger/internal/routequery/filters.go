package routequery

import (
	"encoding/json"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// ExactFeaturesPredicate compares only the feature dimension of a route.
func ExactFeaturesPredicate(column func(string) string, features []string) *sql.Predicate {
	features = ledger.FeatureFilters(features).Normalize()
	encoded, _ := json.Marshal(features)
	return sql.P(func(b *sql.Builder) {
		b.Ident(column("filters")).WriteString("->'features'")
		if len(features) == 0 {
			b.WriteString(" IS NULL")
		} else {
			b.WriteString(" = ").Arg(string(encoded)).WriteString("::jsonb")
		}
	})
}

// MatchFeaturePredicate includes unrestricted routes and routes containing the feature.
func MatchFeaturePredicate(column func(string) string, feature string) *sql.Predicate {
	encoded, _ := json.Marshal([]string{feature})
	return sql.Or(ExactFeaturesPredicate(column, nil), sql.P(func(b *sql.Builder) {
		b.Ident(column("filters")).WriteString("->'features' @> ").Arg(string(encoded)).WriteString("::jsonb")
	}))
}

// ExactFiltersPredicate compares complete normalized route restriction sets.
func ExactFiltersPredicate(column func(string) string, filters ledger.CreditFilters) *sql.Predicate {
	encoded, _ := json.Marshal(filters.Normalize())
	return sql.P(func(b *sql.Builder) {
		b.Ident(column("filters")).WriteString(" - 'schema_version' = ").Arg(string(encoded)).WriteString("::jsonb - 'schema_version'")
	})
}
