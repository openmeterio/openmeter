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

// MatchPlanPredicate selects the same plan view as Route.Matches, including
// credits without plan restrictions. A nil plan selects only those credits.
func MatchPlanPredicate(column func(string) string, plan *ledger.PlanFilter) *sql.Predicate {
	unrestricted := sql.P(func(b *sql.Builder) {
		b.WriteString("COALESCE(").Ident(column("filters")).WriteString("->'plans', '[]'::jsonb) = '[]'::jsonb")
	})
	if plan == nil {
		return unrestricted
	}

	path := "$.plans[*] ? (@.key == $key)"
	variables := map[string]any{"key": plan.Key}
	if plan.Version != nil && plan.Version.Eq != nil {
		path = "$.plans[*] ? (@.key == $key && (!exists(@.version) || @.version == null || @.version.eq == $version || @.version.in[*] == $version || @.version.gte <= $version || @.version.lte >= $version))"
		variables["version"] = *plan.Version.Eq
	}
	encoded, _ := json.Marshal(variables)

	return sql.Or(unrestricted, sql.P(func(b *sql.Builder) {
		b.WriteString("jsonb_path_exists(").Ident(column("filters")).WriteString(", ").Arg(path).
			WriteString("::jsonpath, ").Arg(string(encoded)).WriteString("::jsonb)")
	}))
}

// ExactFiltersPredicate compares complete normalized route restriction sets.
func ExactFiltersPredicate(column func(string) string, filters ledger.CreditFilters) *sql.Predicate {
	encoded, _ := json.Marshal(filters.Normalize())
	return sql.P(func(b *sql.Builder) {
		b.Ident(column("filters")).WriteString(" - 'schema_version' = ").Arg(string(encoded)).WriteString("::jsonb - 'schema_version'")
	})
}
