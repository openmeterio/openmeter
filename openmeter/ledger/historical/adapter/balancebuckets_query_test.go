package adapter

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestBalanceBucketsQuery_SQL(t *testing.T) {
	bookedFrom := time.Now().UTC().Add(-1 * time.Hour)
	txID := "01TESTTXID1234567890123456"

	q := balanceBucketsQuery{
		query: ledger.BalanceBucketQuery{
			Namespace: "ns-test",
			Filters: ledger.Filters{
				TransactionID: &txID,
				BookedAtPeriod: &timeutil.OpenPeriod{
					From: &bookedFrom,
				},
				Route: ledger.RouteFilter{
					Currency:       currencyx.Code("USD"),
					CostBasis:      mo.Some(lo.ToPtr(mustDecimal(t, "0.70"))),
					CreditPriority: lo.ToPtr(7),
				},
			},
			GroupBy: []string{
				ledger.BalanceBucketGroupBySourceChargeID,
				ledger.BalanceBucketGroupBySpendChargeID,
			},
		},
	}

	sqlStr, args, err := q.SQL()
	require.NoError(t, err)

	require.Equal(t, `WITH "balance_buckets" AS (SELECT "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id", "ledger_entries"."spend_charge_id", SUM("ledger_entries"."amount") AS "sum_amount" FROM "ledger_entries" WHERE (("ledger_entries"."namespace" = $1 AND "ledger_entries"."transaction_id" = $2) AND EXISTS (SELECT "ledger_transactions"."id" FROM "ledger_transactions" WHERE "ledger_entries"."transaction_id" = "ledger_transactions"."id" AND "ledger_transactions"."booked_at" >= $3)) AND EXISTS (SELECT "ledger_sub_accounts"."id" FROM "ledger_sub_accounts" WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id" AND EXISTS (SELECT "ledger_sub_account_routes"."id" FROM "ledger_sub_account_routes" WHERE (("ledger_sub_accounts"."route_id" = "ledger_sub_account_routes"."id" AND "ledger_sub_account_routes"."currency" = $4) AND "ledger_sub_account_routes"."credit_priority" = $5) AND "ledger_sub_account_routes"."cost_basis" = $6)) GROUP BY "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id", "ledger_entries"."spend_charge_id") SELECT "balance_buckets"."sub_account_id", "balance_buckets"."source_charge_id", "balance_buckets"."spend_charge_id", "balance_buckets"."sum_amount", "sub_accounts"."route_id", "accounts"."account_type", "routes"."routing_key_version", "routes"."routing_key", "routes"."currency", "routes"."exchange_source_currency", "routes"."tax_code", "routes"."tax_behavior", "routes"."features", "routes"."cost_basis", "routes"."credit_priority", "routes"."transaction_authorization_status" FROM "balance_buckets" JOIN "ledger_sub_accounts" AS "sub_accounts" ON "balance_buckets"."sub_account_id" = "sub_accounts"."id" JOIN "ledger_accounts" AS "accounts" ON "sub_accounts"."account_id" = "accounts"."id" JOIN "ledger_sub_account_routes" AS "routes" ON "sub_accounts"."route_id" = "routes"."id" ORDER BY "balance_buckets"."sub_account_id", "balance_buckets"."source_charge_id", "balance_buckets"."spend_charge_id"`, sqlStr)
	require.Equal(t, []any{
		"ns-test",
		txID,
		bookedFrom,
		"USD",
		7,
		mustDecimal(t, "0.7"),
	}, args)
}

func TestBalanceBucketsQuery_SQLUngroupedDimensions(t *testing.T) {
	q := balanceBucketsQuery{
		query: ledger.BalanceBucketQuery{
			Namespace: "ns-test",
		},
	}

	sqlStr, args, err := q.SQL()
	require.NoError(t, err)

	require.Equal(t, `WITH "balance_buckets" AS (SELECT "ledger_entries"."sub_account_id", NULL AS "source_charge_id", NULL AS "spend_charge_id", SUM("ledger_entries"."amount") AS "sum_amount" FROM "ledger_entries" WHERE "ledger_entries"."namespace" = $1 GROUP BY "ledger_entries"."sub_account_id") SELECT "balance_buckets"."sub_account_id", "balance_buckets"."source_charge_id", "balance_buckets"."spend_charge_id", "balance_buckets"."sum_amount", "sub_accounts"."route_id", "accounts"."account_type", "routes"."routing_key_version", "routes"."routing_key", "routes"."currency", "routes"."exchange_source_currency", "routes"."tax_code", "routes"."tax_behavior", "routes"."features", "routes"."cost_basis", "routes"."credit_priority", "routes"."transaction_authorization_status" FROM "balance_buckets" JOIN "ledger_sub_accounts" AS "sub_accounts" ON "balance_buckets"."sub_account_id" = "sub_accounts"."id" JOIN "ledger_accounts" AS "accounts" ON "sub_accounts"."account_id" = "accounts"."id" JOIN "ledger_sub_account_routes" AS "routes" ON "sub_accounts"."route_id" = "routes"."id" ORDER BY "balance_buckets"."sub_account_id", "balance_buckets"."source_charge_id", "balance_buckets"."spend_charge_id"`, sqlStr)
	require.Equal(t, []any{"ns-test"}, args)
}
