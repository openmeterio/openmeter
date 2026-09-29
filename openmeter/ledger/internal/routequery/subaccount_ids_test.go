package routequery

import (
	"testing"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestSubAccountIDsByRouteSQL(t *testing.T) {
	tests := []struct {
		name     string
		route    ledger.RouteFilter
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "exact feature route",
			route: ledger.RouteFilter{
				Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
				Features: mo.Some([]string{"feature-b", "feature-a"}),
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND "lsar"."filters"->'features' = $2::jsonb`,
			wantArgs: []any{
				"USD",
				`["feature-a","feature-b"]`,
			},
		},
		{
			name: "unrestricted exact route",
			route: ledger.RouteFilter{
				Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
				Features: mo.Some[[]string](nil),
			},
			wantSQL:  `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND "lsar"."filters"->'features' IS NULL`,
			wantArgs: []any{"USD"},
		},
		{
			name: "match feature route",
			route: ledger.RouteFilter{
				Currency:     currencies.NewCurrencyReference(currencyx.Code("USD")),
				MatchFeature: "feature-a",
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND ("lsar"."filters"->'features' IS NULL OR "lsar"."filters"->'features' @> $2::jsonb)`,
			wantArgs: []any{
				"USD",
				`["feature-a"]`,
			},
		},
		{
			name: "without plan restrictions",
			route: ledger.RouteFilter{
				MatchPlan: mo.Some[*ledger.PlanFilter](nil),
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE COALESCE("lsar"."filters"->'plans', '[]'::jsonb) = '[]'::jsonb`,
		},
		{
			name: "match plan key",
			route: ledger.RouteFilter{
				Currency:  currencies.NewCurrencyReference(currencyx.Code("USD")),
				MatchPlan: mo.Some(&ledger.PlanFilter{Key: "pro"}),
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND (COALESCE("lsar"."filters"->'plans', '[]'::jsonb) = '[]'::jsonb OR jsonb_path_exists("lsar"."filters", $2::jsonpath, $3::jsonb))`,
			wantArgs: []any{
				"USD",
				`$.plans[*] ? (@.key == $key)`,
				`{"key":"pro"}`,
			},
		},
		{
			name: "match plan version",
			route: ledger.RouteFilter{
				MatchPlan: mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}),
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE COALESCE("lsar"."filters"->'plans', '[]'::jsonb) = '[]'::jsonb OR jsonb_path_exists("lsar"."filters", $1::jsonpath, $2::jsonb)`,
			wantArgs: []any{
				`$.plans[*] ? (@.key == $key && (!exists(@.version) || @.version == null || @.version.eq == $version || @.version.in[*] == $version || @.version.gte <= $version || @.version.lte >= $version))`,
				`{"key":"pro","version":2}`,
			},
		},
		{
			name: "match feature and plan version",
			route: ledger.RouteFilter{
				MatchFeature: "feature-a",
				MatchPlan:    mo.Some(&ledger.PlanFilter{Key: "pro", Version: &ledger.VersionFilter{Eq: lo.ToPtr(2)}}),
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE (COALESCE("lsar"."filters"->'plans', '[]'::jsonb) = '[]'::jsonb OR jsonb_path_exists("lsar"."filters", $1::jsonpath, $2::jsonb)) AND ("lsar"."filters"->'features' IS NULL OR "lsar"."filters"->'features' @> $3::jsonb)`,
			wantArgs: []any{
				`$.plans[*] ? (@.key == $key && (!exists(@.version) || @.version == null || @.version.eq == $version || @.version.in[*] == $version || @.version.gte <= $version || @.version.lte >= $version))`,
				`{"key":"pro","version":2}`,
				`["feature-a"]`,
			},
		},
		{
			name: "plan key stays a bound JSON value",
			route: ledger.RouteFilter{
				MatchPlan: mo.Some(&ledger.PlanFilter{Key: `pro'"\beta`}),
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE COALESCE("lsar"."filters"->'plans', '[]'::jsonb) = '[]'::jsonb OR jsonb_path_exists("lsar"."filters", $1::jsonpath, $2::jsonb)`,
			wantArgs: []any{
				`$.plans[*] ? (@.key == $key)`,
				`{"key":"pro'\"\\beta"}`,
			},
		},
		{
			name: "unresolved custom currency code prefix",
			route: ledger.RouteFilter{
				Currency: currencies.NewCurrencyReference(currencyx.Code("ACME")),
			},
			wantSQL:  `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" LIKE $1`,
			wantArgs: []any{"custom|v1|ACME|%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := NewSubAccountIDsByRoute(tt.route)
			require.NoError(t, err)
			gotSQL, gotArgs := query.sql()

			require.Equal(t, tt.wantSQL, gotSQL)
			require.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}
