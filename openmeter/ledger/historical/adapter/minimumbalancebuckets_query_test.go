package adapter

import (
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestMinimumBalanceBucketsQuery_SQLTemplate(t *testing.T) {
	at := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	query, args, err := (minimumBalanceBucketsQuery{query: ledger.MinimumBalanceBucketQuery{
		BalanceBucketQuery: ledger.BalanceBucketQuery{
			Namespace: "ns-test",
			Filters:   ledger.Filters{AccountID: lo.ToPtr("account"), AsOf: &at},
			GroupBy:   []string{ledger.BalanceBucketGroupBySourceChargeID},
		},
		ExcludeFutureAnnotationFilters: map[string]string{"movement": "ignored"},
	}}).SQL()
	require.NoError(t, err)
	expectedSQL := `
WITH "movements" AS (SELECT
        "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id",
        ($1::timestamptz) AS "effective_at",
        SUM("ledger_entries"."amount") AS "amount",
        MIN("ledger_entries"."created_at") AS "oldest_matching_entry_created_at"
    FROM "ledger_entries"
    JOIN "ledger_transactions" AS "transactions"
        ON "ledger_entries"."transaction_id" = "transactions"."id"
    WHERE "ledger_entries"."namespace" = $2 AND EXISTS (SELECT "ledger_sub_accounts"."id"
        FROM "ledger_sub_accounts"
        WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
            AND "ledger_sub_accounts"."account_id" = $3) AND "transactions"."booked_at" <= $4
    GROUP BY "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id"
    UNION ALL
    SELECT
        "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id",
        "transactions"."booked_at" AS "effective_at",
        (SUM(CASE WHEN (NOT ("transactions"."annotations"->'movement' IS NOT NULL))
            OR "transactions"."annotations"->>'movement' <> $5 THEN "ledger_entries"."amount" ELSE 0 END)) AS "amount",
        MIN("ledger_entries"."created_at") AS "oldest_matching_entry_created_at"
    FROM "ledger_entries"
    JOIN "ledger_transactions" AS "transactions"
        ON "ledger_entries"."transaction_id" = "transactions"."id"
    WHERE "ledger_entries"."namespace" = $6 AND EXISTS (SELECT "ledger_sub_accounts"."id"
        FROM "ledger_sub_accounts"
        WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
            AND "ledger_sub_accounts"."account_id" = $7) AND "transactions"."booked_at" > $8
    GROUP BY "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id", "transactions"."booked_at"),
"running_balances" AS (SELECT
        "sub_account_id", "source_charge_id", "effective_at",
        "oldest_matching_entry_created_at",
        (SUM("amount") OVER (PARTITION BY "sub_account_id", "source_charge_id"
            ORDER BY "effective_at")) AS "balance"
    FROM "movements"),
"balance_buckets" AS (SELECT
        "running_balances"."sub_account_id",
        "running_balances"."source_charge_id",
        NULL AS "spend_charge_id",
        NULL AS "collection_origin_id",
        (CASE WHEN MIN("effective_at") > $9 THEN LEAST(MIN(balance), 0) ELSE MIN(balance) END) AS "sum_amount",
        MIN("oldest_matching_entry_created_at") AS "oldest_matching_entry_created_at"
    FROM "running_balances"
    GROUP BY "sub_account_id", "source_charge_id")
SELECT
    "balance_buckets"."sub_account_id",
    "balance_buckets"."source_charge_id",
    "balance_buckets"."spend_charge_id",
    "balance_buckets"."collection_origin_id",
    "balance_buckets"."sum_amount",
    "balance_buckets"."oldest_matching_entry_created_at",
    "sub_accounts"."route_id",
    "accounts"."account_type",
    "routes"."routing_key_version",
    "routes"."routing_key",
    "routes"."currency",
    "routes"."cost_basis_currency",
    "routes"."tax_code",
    "routes"."tax_behavior",
    "routes"."filters",
    "routes"."cost_basis",
    "routes"."credit_priority",
    "routes"."transaction_authorization_status"
FROM "balance_buckets"
JOIN "ledger_sub_accounts" AS "sub_accounts"
    ON "balance_buckets"."sub_account_id" = "sub_accounts"."id"
JOIN "ledger_accounts" AS "accounts"
    ON "sub_accounts"."account_id" = "accounts"."id"
JOIN "ledger_sub_account_routes" AS "routes"
    ON "sub_accounts"."route_id" = "routes"."id"
ORDER BY
    "balance_buckets"."sub_account_id",
    "balance_buckets"."source_charge_id",
    "balance_buckets"."spend_charge_id",
    "balance_buckets"."collection_origin_id"
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{at, "ns-test", "account", at, "ignored", "ns-test", "account", at, at}, args)
}

func TestMinimumBalanceBucketsQuery_ScopedEntriesSQLTemplate(t *testing.T) {
	q := newMinimumBalanceBucketsTestQuery()
	scope, err := q.scopedEntries()
	require.NoError(t, err)

	query, args := scope.Query()
	expectedSQL := `
SELECT * FROM "ledger_entries"
JOIN "ledger_transactions" AS "transactions"
    ON "ledger_entries"."transaction_id" = "transactions"."id"
WHERE "ledger_entries"."namespace" = $1 AND EXISTS (SELECT "ledger_sub_accounts"."id"
    FROM "ledger_sub_accounts"
    WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
        AND "ledger_sub_accounts"."account_id" = $2)
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{"ns-test", "account"}, args)
}

func TestMinimumBalanceBucketsQuery_OpeningBalancesSQLTemplate(t *testing.T) {
	q := newMinimumBalanceBucketsTestQuery()
	scope, err := q.scopedEntries()
	require.NoError(t, err)

	query, args := q.openingBalances(scope).Query()
	expectedSQL := `
SELECT
    "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id",
    ($1::timestamptz) AS "effective_at",
    SUM("ledger_entries"."amount") AS "amount",
    MIN("ledger_entries"."created_at") AS "oldest_matching_entry_created_at"
FROM "ledger_entries"
JOIN "ledger_transactions" AS "transactions"
    ON "ledger_entries"."transaction_id" = "transactions"."id"
WHERE "ledger_entries"."namespace" = $2 AND EXISTS (SELECT "ledger_sub_accounts"."id"
    FROM "ledger_sub_accounts"
    WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
        AND "ledger_sub_accounts"."account_id" = $3) AND "transactions"."booked_at" <= $4
GROUP BY "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id"
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{*q.query.Filters.AsOf, "ns-test", "account", *q.query.Filters.AsOf}, args)
}

func TestMinimumBalanceBucketsQuery_FutureMovementsSQLTemplate(t *testing.T) {
	t.Run("with future exclusions", func(t *testing.T) {
		q := newMinimumBalanceBucketsTestQuery()
		scope, err := q.scopedEntries()
		require.NoError(t, err)

		query, args := q.futureMovements(scope).Query()
		expectedSQL := `
SELECT
    "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id",
    "transactions"."booked_at" AS "effective_at",
    (SUM(CASE WHEN (NOT ("transactions"."annotations"->'movement' IS NOT NULL))
        OR "transactions"."annotations"->>'movement' <> $1 THEN "ledger_entries"."amount" ELSE 0 END)) AS "amount",
    MIN("ledger_entries"."created_at") AS "oldest_matching_entry_created_at"
FROM "ledger_entries"
JOIN "ledger_transactions" AS "transactions"
    ON "ledger_entries"."transaction_id" = "transactions"."id"
WHERE "ledger_entries"."namespace" = $2 AND EXISTS (SELECT "ledger_sub_accounts"."id"
    FROM "ledger_sub_accounts"
    WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
        AND "ledger_sub_accounts"."account_id" = $3) AND "transactions"."booked_at" > $4
GROUP BY "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id", "transactions"."booked_at"
`
		require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
		require.Equal(t, []any{"ignored", "ns-test", "account", *q.query.Filters.AsOf}, args)
	})

	t.Run("without future exclusions", func(t *testing.T) {
		q := newMinimumBalanceBucketsTestQuery()
		q.query.ExcludeFutureAnnotationFilters = nil
		scope, err := q.scopedEntries()
		require.NoError(t, err)

		query, args := q.futureMovements(scope).Query()
		expectedSQL := `
SELECT
    "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id",
    "transactions"."booked_at" AS "effective_at",
    (SUM("ledger_entries"."amount")) AS "amount",
    MIN("ledger_entries"."created_at") AS "oldest_matching_entry_created_at"
FROM "ledger_entries"
JOIN "ledger_transactions" AS "transactions"
    ON "ledger_entries"."transaction_id" = "transactions"."id"
WHERE "ledger_entries"."namespace" = $1 AND EXISTS (SELECT "ledger_sub_accounts"."id"
    FROM "ledger_sub_accounts"
    WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
        AND "ledger_sub_accounts"."account_id" = $2) AND "transactions"."booked_at" > $3
GROUP BY "ledger_entries"."sub_account_id", "ledger_entries"."source_charge_id", "transactions"."booked_at"
`
		require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
		require.Equal(t, []any{"ns-test", "account", *q.query.Filters.AsOf}, args)
	})
}

func TestMinimumBalanceBucketsQuery_RunningBalancesSQLTemplate(t *testing.T) {
	q := newMinimumBalanceBucketsTestQuery()
	query, args := q.runningBalances().Query()
	expectedSQL := `
SELECT
    "sub_account_id", "source_charge_id", "effective_at",
    "oldest_matching_entry_created_at",
    (SUM("amount") OVER (PARTITION BY "sub_account_id", "source_charge_id"
        ORDER BY "effective_at")) AS "balance"
FROM "movements"
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Empty(t, args)
}

func TestMinimumBalanceBucketsQuery_MinimumBalancesSQLTemplate(t *testing.T) {
	q := newMinimumBalanceBucketsTestQuery()
	query, args := q.minimumBalances().Query()
	expectedSQL := `
SELECT
    "running_balances"."sub_account_id",
    "running_balances"."source_charge_id",
    NULL AS "spend_charge_id",
    NULL AS "collection_origin_id",
    (CASE WHEN MIN("effective_at") > $1 THEN LEAST(MIN(balance), 0) ELSE MIN(balance) END) AS "sum_amount",
    MIN("oldest_matching_entry_created_at") AS "oldest_matching_entry_created_at"
FROM "running_balances"
GROUP BY "sub_account_id", "source_charge_id"
`
	require.Equal(t, strings.Join(strings.Fields(expectedSQL), " "), query)
	require.Equal(t, []any{*q.query.Filters.AsOf}, args)
}

func TestMinimumBalanceBucketsQuery_BucketColumns(t *testing.T) {
	for _, tc := range []struct {
		name      string
		groupBy   []string
		columns   []string
		qualified []string
	}{
		{
			name:      "subaccount only",
			columns:   []string{"sub_account_id"},
			qualified: []string{`"ledger_entries"."sub_account_id"`},
		},
		{
			name:      "source",
			groupBy:   []string{ledger.BalanceBucketGroupBySourceChargeID},
			columns:   []string{"sub_account_id", "source_charge_id"},
			qualified: []string{`"ledger_entries"."sub_account_id"`, `"ledger_entries"."source_charge_id"`},
		},
		{
			name: "all provenance dimensions",
			groupBy: []string{
				ledger.BalanceBucketGroupBySourceChargeID,
				ledger.BalanceBucketGroupBySpendChargeID,
				ledger.BalanceBucketGroupByCollectionOriginID,
			},
			columns: []string{"sub_account_id", "source_charge_id", "spend_charge_id", "collection_origin_id"},
			qualified: []string{
				`"ledger_entries"."sub_account_id"`,
				`"ledger_entries"."source_charge_id"`,
				`"ledger_entries"."spend_charge_id"`,
				`"ledger_entries"."collection_origin_id"`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := newMinimumBalanceBucketsTestQuery()
			q.query.GroupBy = tc.groupBy

			require.Equal(t, tc.columns, q.bucketColumns())
			require.Equal(t, tc.qualified, q.entryBucketColumns())
		})
	}
}

func newMinimumBalanceBucketsTestQuery() minimumBalanceBucketsQuery {
	at := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

	return minimumBalanceBucketsQuery{query: ledger.MinimumBalanceBucketQuery{
		BalanceBucketQuery: ledger.BalanceBucketQuery{
			Namespace: "ns-test",
			Filters:   ledger.Filters{AccountID: lo.ToPtr("account"), AsOf: &at},
			GroupBy:   []string{ledger.BalanceBucketGroupBySourceChargeID},
		},
		ExcludeFutureAnnotationFilters: map[string]string{"movement": "ignored"},
	}}
}
