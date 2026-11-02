package adapter

import (
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestBalancesAtBoundariesQuery_SQLTemplate(t *testing.T) {
	at := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	cursor := ledger.TransactionCursor{BookedAt: at, CreatedAt: at.Add(time.Hour), ID: models.NamespacedID{Namespace: "ns", ID: "transaction"}}
	input := ledger.GetBalancesAtBoundariesInput{Queries: []ledger.Query{
		{Namespace: "ns", Filters: ledger.Filters{AccountID: lo.ToPtr("account"), AsOf: &at}},
		{Namespace: "ns", Filters: ledger.Filters{AccountID: lo.ToPtr("account"), After: &cursor}},
	}}
	query, args, err := (balancesAtBoundariesQuery{input: input}).SQL()
	require.NoError(t, err)

	expected := `
WITH "movements" AS (SELECT ($1::integer) AS "scope_index", "ledger_entries"."amount",
        "transactions"."booked_at", "transactions"."created_at",
        "transactions"."id" AS "transaction_id", NULL::integer AS boundary_index
    FROM "ledger_entries"
    JOIN "ledger_transactions" AS "transactions"
        ON "ledger_entries"."transaction_id" = "transactions"."id"
    WHERE ("ledger_entries"."namespace" = $2 AND EXISTS (SELECT "ledger_sub_accounts"."id" FROM "ledger_sub_accounts"
        WHERE "ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
            AND "ledger_sub_accounts"."account_id" = $3))
        AND "transactions"."booked_at" <= $4
    UNION ALL
    SELECT "scope_index", 0::numeric AS amount, "booked_at", "created_at",
        "transaction_id", "boundary_index"
    FROM (VALUES ($5::integer, $6::timestamptz, NULL::timestamptz, NULL::bpchar, $7::integer),
        ($8::integer, $9::timestamptz, $10::timestamptz, $11::bpchar, $12::integer))
        AS boundaries(scope_index, booked_at, created_at, transaction_id, boundary_index)),
"running_balances" AS (SELECT "boundary_index", (SUM("amount") OVER (PARTITION BY "scope_index"
        ORDER BY booked_at, created_at NULLS LAST, transaction_id NULLS LAST,
            boundary_index NULLS FIRST)) AS "balance"
    FROM "movements")
SELECT "balance", "boundary_index" FROM "running_balances" WHERE "boundary_index" IS NOT NULL`
	require.Equal(t, strings.Join(strings.Fields(expected), " "), query)
	require.Equal(t, []any{0, "ns", "account", at, 0, at, 0, 0, at, cursor.CreatedAt, cursor.ID.ID, 1}, args)
}

func TestBalancesAtBoundariesQuerySharesHistoryAcrossPage(t *testing.T) {
	at := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	var input ledger.GetBalancesAtBoundariesInput
	for row := range 500 {
		for _, account := range []string{"fbo", "receivable"} {
			input.Queries = append(input.Queries, ledger.Query{Namespace: "ns", Filters: ledger.Filters{
				AccountID: lo.ToPtr(account), AsOf: lo.ToPtr(at.Add(time.Duration(row) * time.Minute)),
			}})
		}
	}

	query, _, err := (balancesAtBoundariesQuery{input: input}).SQL()
	require.NoError(t, err)
	// Page size adds boundary markers, not another ledger scan for each row.
	require.Equal(t, 2, strings.Count(query, `FROM "ledger_entries"`))
	require.NotNil(t, input.Queries[0].Filters.AsOf)
}

func TestBalanceBoundaryScope_MovementsSQLTemplate(t *testing.T) {
	at := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	from := at.Add(-24 * time.Hour)
	scope := balanceBoundaryScope{
		query: ledger.Query{
			Namespace: "ns",
			Filters: ledger.Filters{
				TransactionID: lo.ToPtr("transaction"),
				Provenance: ledger.ProvenanceFilter{
					SourceChargeID: mo.Some(lo.ToPtr("source")),
				},
				BookedAtPeriod: &timeutil.OpenPeriod{From: &from, To: &at},
			},
		},
		latestAt: at,
	}
	selector, err := scope.movements(2)
	require.NoError(t, err)
	query, args := selector.Query()

	expected := `
SELECT ($1::integer) AS "scope_index", "ledger_entries"."amount",
    "transactions"."booked_at", "transactions"."created_at",
    "transactions"."id" AS "transaction_id", NULL::integer AS boundary_index
FROM "ledger_entries"
JOIN "ledger_transactions" AS "transactions"
    ON "ledger_entries"."transaction_id" = "transactions"."id"
WHERE ((("ledger_entries"."namespace" = $2 AND "ledger_entries"."transaction_id" = $3)
    AND "ledger_entries"."source_charge_id" = $4)
    AND EXISTS (SELECT "ledger_transactions"."id" FROM "ledger_transactions"
        WHERE ("ledger_entries"."transaction_id" = "ledger_transactions"."id"
            AND "ledger_transactions"."booked_at" >= $5) AND "ledger_transactions"."booked_at" < $6))
    AND "transactions"."booked_at" <= $7`
	require.Equal(t, strings.Join(strings.Fields(expected), " "), query)
	require.Equal(t, []any{2, "ns", "transaction", "source", from, at, at}, args)
}

func TestBalancesAtBoundariesQuery_BoundariesSQLTemplate(t *testing.T) {
	at := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	later := at.Add(time.Hour)
	cursor := ledger.TransactionCursor{
		BookedAt: at, CreatedAt: later,
		ID: models.NamespacedID{Namespace: "ns", ID: "transaction"},
	}
	q := balancesAtBoundariesQuery{input: ledger.GetBalancesAtBoundariesInput{Queries: []ledger.Query{
		{Namespace: "ns", Filters: ledger.Filters{AsOf: &later}},
		{Namespace: "ns", Filters: ledger.Filters{After: &cursor}},
		{Namespace: "ns", Filters: ledger.Filters{AsOf: &later}},
	}}}
	query, args := q.boundaries([]int{1, 0, 1}).Query()

	// Markers keep request order and duplicates; cursor markers carry all three
	// ordering fields, while timestamp markers sort after that time's entries.
	expected := `
SELECT "scope_index", 0::numeric AS amount, "booked_at", "created_at", "transaction_id", "boundary_index"
FROM (VALUES
    ($1::integer, $2::timestamptz, NULL::timestamptz, NULL::bpchar, $3::integer),
    ($4::integer, $5::timestamptz, $6::timestamptz, $7::bpchar, $8::integer),
    ($9::integer, $10::timestamptz, NULL::timestamptz, NULL::bpchar, $11::integer))
    AS boundaries(scope_index, booked_at, created_at, transaction_id, boundary_index)`
	require.Equal(t, strings.Join(strings.Fields(expected), " "), query)
	require.Equal(t, []any{1, later, 0, 0, at, later, "transaction", 1, 1, later, 2}, args)
}

func TestBalancesAtBoundariesQuery_IndependentScopesSQLTemplate(t *testing.T) {
	at := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	createdAt := at.Add(time.Hour)
	cursor := ledger.TransactionCursor{
		BookedAt: at, CreatedAt: createdAt,
		ID: models.NamespacedID{Namespace: "cursor-ns", ID: "transaction"},
	}
	q := balancesAtBoundariesQuery{input: ledger.GetBalancesAtBoundariesInput{Queries: []ledger.Query{
		{Namespace: "timestamp-ns", Filters: ledger.Filters{AsOf: &at}},
		{Namespace: "cursor-ns", Filters: ledger.Filters{After: &cursor}},
	}}}
	query, args, err := q.SQL()
	require.NoError(t, err)

	expected := `
SELECT SUM("ledger_entries"."amount") AS "sum_amount", $1::integer AS boundary_index
FROM "ledger_entries"
WHERE "ledger_entries"."namespace" = $2 AND EXISTS (SELECT "ledger_transactions"."id" FROM "ledger_transactions"
    WHERE "ledger_entries"."transaction_id" = "ledger_transactions"."id"
        AND "ledger_transactions"."booked_at" <= $3)
UNION ALL
SELECT SUM("ledger_entries"."amount") AS "sum_amount", $4::integer AS boundary_index
FROM "ledger_entries"
WHERE "ledger_entries"."namespace" = $5 AND EXISTS (SELECT "ledger_transactions"."id" FROM "ledger_transactions"
    WHERE "ledger_entries"."transaction_id" = "ledger_transactions"."id"
        AND ("ledger_transactions"."booked_at" < $6 OR ("ledger_transactions"."booked_at" = $7
            AND ("ledger_transactions"."created_at" < $8 OR ("ledger_transactions"."created_at" = $9
                AND "ledger_transactions"."id" <= $10)))))`
	require.Equal(t, strings.Join(strings.Fields(expected), " "), query)
	require.Equal(t, []any{0, "timestamp-ns", at, 1, "cursor-ns", at, at, createdAt, createdAt, "transaction"}, args)
}

func TestBalancesAtBoundariesQuery_MultipleScopesSQLTemplate(t *testing.T) {
	at := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	later := at.Add(time.Hour)
	q := balancesAtBoundariesQuery{input: ledger.GetBalancesAtBoundariesInput{Queries: []ledger.Query{
		{Namespace: "first", Filters: ledger.Filters{AsOf: &at}},
		{Namespace: "second", Filters: ledger.Filters{AsOf: &at}},
		{Namespace: "first", Filters: ledger.Filters{AsOf: &later}},
	}}}
	query, args, err := q.SQL()
	require.NoError(t, err)

	// Each scope reads through its own latest boundary. Placeholder numbering
	// continues across both scans and the boundary markers.
	expected := `
WITH "movements" AS (SELECT ($1::integer) AS "scope_index", "ledger_entries"."amount",
        "transactions"."booked_at", "transactions"."created_at",
        "transactions"."id" AS "transaction_id", NULL::integer AS boundary_index
    FROM "ledger_entries"
    JOIN "ledger_transactions" AS "transactions"
        ON "ledger_entries"."transaction_id" = "transactions"."id"
    WHERE "ledger_entries"."namespace" = $2 AND "transactions"."booked_at" <= $3
    UNION ALL
    SELECT ($4::integer) AS "scope_index", "ledger_entries"."amount",
        "transactions"."booked_at", "transactions"."created_at",
        "transactions"."id" AS "transaction_id", NULL::integer AS boundary_index
    FROM "ledger_entries"
    JOIN "ledger_transactions" AS "transactions"
        ON "ledger_entries"."transaction_id" = "transactions"."id"
    WHERE "ledger_entries"."namespace" = $5 AND "transactions"."booked_at" <= $6
    UNION ALL
    SELECT "scope_index", 0::numeric AS amount, "booked_at", "created_at", "transaction_id", "boundary_index"
    FROM (VALUES ($7::integer, $8::timestamptz, NULL::timestamptz, NULL::bpchar, $9::integer),
        ($10::integer, $11::timestamptz, NULL::timestamptz, NULL::bpchar, $12::integer),
        ($13::integer, $14::timestamptz, NULL::timestamptz, NULL::bpchar, $15::integer))
        AS boundaries(scope_index, booked_at, created_at, transaction_id, boundary_index)),
"running_balances" AS (SELECT "boundary_index", (SUM("amount") OVER (PARTITION BY "scope_index"
        ORDER BY booked_at, created_at NULLS LAST, transaction_id NULLS LAST, boundary_index NULLS FIRST)) AS "balance"
    FROM "movements")
SELECT "balance", "boundary_index" FROM "running_balances" WHERE "boundary_index" IS NOT NULL`
	require.Equal(t, strings.Join(strings.Fields(expected), " "), query)
	require.Equal(t, []any{0, "first", later, 1, "second", at, 0, at, 0, 1, at, 1, 0, later, 2}, args)
}
