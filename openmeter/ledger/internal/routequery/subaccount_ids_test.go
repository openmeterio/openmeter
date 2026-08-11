package routequery

import (
	"testing"

	"github.com/lib/pq"
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
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND "lsar"."features" = $2`,
			wantArgs: []any{
				"USD",
				pq.StringArray{"feature-a", "feature-b"},
			},
		},
		{
			name: "unrestricted exact route",
			route: ledger.RouteFilter{
				Currency: currencies.NewCurrencyReference(currencyx.Code("USD")),
				Features: mo.Some[[]string](nil),
			},
			wantSQL:  `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND "lsar"."features" IS NULL`,
			wantArgs: []any{"USD"},
		},
		{
			name: "match feature route",
			route: ledger.RouteFilter{
				Currency:     currencies.NewCurrencyReference(currencyx.Code("USD")),
				MatchFeature: "feature-a",
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND ("lsar"."features" IS NULL OR "lsar"."features" @> $2)`,
			wantArgs: []any{
				"USD",
				pq.StringArray{"feature-a"},
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
