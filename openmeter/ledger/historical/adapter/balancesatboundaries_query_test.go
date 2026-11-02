package adapter

import (
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
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
