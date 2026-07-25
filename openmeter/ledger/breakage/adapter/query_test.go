package adapter

import (
	"testing"

	"github.com/lib/pq"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestExpiredRecordRouteQuerySQL(t *testing.T) {
	tests := []struct {
		name     string
		query    expiredRecordRouteQuery
		wantSQL  string
		wantArgs []any
	}{
		{
			name: "exact feature route",
			query: expiredRecordRouteQuery{
				Route: ledger.RouteFilter{
					Currency: currencyx.Code("USD"),
					Features: mo.Some([]string{"feature-b", "feature-a"}),
				},
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND "lsar"."features" = $2`,
			wantArgs: []any{
				"USD",
				pq.StringArray{"feature-a", "feature-b"},
			},
		},
		{
			name: "match feature route",
			query: expiredRecordRouteQuery{
				Route: ledger.RouteFilter{
					Currency:     currencyx.Code("USD"),
					MatchFeature: "feature-a",
				},
			},
			wantSQL: `SELECT "lsa"."id" FROM "ledger_sub_accounts" AS "lsa" JOIN "ledger_sub_account_routes" AS "lsar" ON "lsa"."route_id" = "lsar"."id" WHERE "lsar"."currency" = $1 AND ("lsar"."features" IS NULL OR "lsar"."features" @> $2)`,
			wantArgs: []any{
				"USD",
				pq.StringArray{"feature-a"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := tt.query.SQL()

			require.Equal(t, tt.wantSQL, gotSQL)
			require.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}
