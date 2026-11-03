package adapter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestRepo_ListTransactions_QueryScope(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	var queries []string
	client := entdb.NewClient(
		entdb.Driver(env.db.EntDriver.Driver()),
		entdb.Debug(),
		entdb.Log(func(args ...any) {
			message := fmt.Sprint(args...)
			if _, query, ok := strings.Cut(message, `FROM "ledger_transactions" WHERE `); ok {
				queries = append(queries, query)
			}
		}),
	)
	reader := NewRepo(client)

	for _, tc := range []struct {
		name  string
		input ledger.ListTransactionsInput
		query string
	}{
		{
			name:  "unfiltered listing needs no entry lookup",
			input: ledger.ListTransactionsInput{Namespace: "query-ns", Limit: 10},
			query: `"ledger_transactions"."namespace" = $1
				ORDER BY "ledger_transactions"."booked_at" DESC, "ledger_transactions"."created_at" DESC, "ledger_transactions"."id" DESC
				LIMIT 11 args=[query-ns]`,
		},
		{
			name: "funded candidates scope both account lookups",
			input: ledger.ListTransactionsInput{
				Namespace: "query-ns",
				Limit:     10,
				EntryFilter: ledger.TransactionEntryFilter{
					AccountIDs: []string{"fbo", "receivable"},
				},
			},
			query: `"ledger_transactions"."namespace" = $1 AND EXISTS (SELECT "ledger_entries"."transaction_id" FROM "ledger_entries"
				WHERE ("ledger_transactions"."id" = "ledger_entries"."transaction_id" AND EXISTS (SELECT "ledger_sub_accounts"."id" FROM "ledger_sub_accounts"
					WHERE ("ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
						AND "ledger_sub_accounts"."account_id" IN ($2, $3)) AND "ledger_sub_accounts"."namespace" = $4))
					AND "ledger_entries"."namespace" = $5)
				ORDER BY "ledger_transactions"."booked_at" DESC, "ledger_transactions"."created_at" DESC, "ledger_transactions"."id" DESC
				LIMIT 11 args=[query-ns fbo receivable query-ns query-ns]`,
		},
		{
			name: "consumed candidates also scope the net movement query",
			input: ledger.ListTransactionsInput{
				Namespace: "query-ns",
				Limit:     10,
				EntryFilter: ledger.TransactionEntryFilter{
					AccountIDs: []string{"fbo"},
				},
				CreditMovement: ledger.ListTransactionsCreditMovementNegative,
			},
			query: `("ledger_transactions"."namespace" = $1 AND EXISTS (SELECT "ledger_entries"."transaction_id" FROM "ledger_entries"
				WHERE ("ledger_transactions"."id" = "ledger_entries"."transaction_id" AND EXISTS (SELECT "ledger_sub_accounts"."id" FROM "ledger_sub_accounts"
					WHERE ("ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
						AND "ledger_sub_accounts"."account_id" IN ($2)) AND "ledger_sub_accounts"."namespace" = $3))
					AND "ledger_entries"."namespace" = $4)) AND "ledger_transactions"."id" IN (SELECT "le"."transaction_id" FROM "ledger_entries" AS "le"
				JOIN "ledger_sub_accounts" AS "lsa" ON "le"."sub_account_id" = "lsa"."id"
				JOIN "ledger_accounts" AS "la" ON "lsa"."account_id" = "la"."id"
				WHERE (("la"."account_type" = $5 AND "le"."namespace" = $6) AND "lsa"."namespace" = $7)
					AND "lsa"."account_id" IN ($8)
				GROUP BY "le"."transaction_id" HAVING SUM(amount) < 0)
				ORDER BY "ledger_transactions"."booked_at" DESC, "ledger_transactions"."created_at" DESC, "ledger_transactions"."id" DESC
				LIMIT 11 args=[query-ns fbo query-ns query-ns customer_fbo query-ns query-ns fbo]`,
		},
		{
			name: "positive movement without an account filter stays namespace scoped",
			input: ledger.ListTransactionsInput{
				Namespace:      "query-ns",
				Limit:          10,
				CreditMovement: ledger.ListTransactionsCreditMovementPositive,
			},
			query: `"ledger_transactions"."namespace" = $1 AND "ledger_transactions"."id" IN (SELECT "le"."transaction_id" FROM "ledger_entries" AS "le"
				JOIN "ledger_sub_accounts" AS "lsa" ON "le"."sub_account_id" = "lsa"."id"
				JOIN "ledger_accounts" AS "la" ON "lsa"."account_id" = "la"."id"
				WHERE ("la"."account_type" = $2 AND "le"."namespace" = $3) AND "lsa"."namespace" = $4
				GROUP BY "le"."transaction_id" HAVING SUM(amount) > 0)
				ORDER BY "ledger_transactions"."booked_at" DESC, "ledger_transactions"."created_at" DESC, "ledger_transactions"."id" DESC
				LIMIT 11 args=[query-ns customer_fbo query-ns query-ns]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: an empty ledger; check SQL because correct results alone
			// cannot catch a missing index-prefix predicate.
			queries = nil

			// when: the adapter builds the transaction listing query.
			page, err := reader.ListTransactions(t.Context(), tc.input)
			require.NoError(t, err)

			// then: every filtered entry/subaccount lookup binds namespace.
			require.Empty(t, page.Items)
			require.Equal(t, []string{strings.Join(strings.Fields(tc.query), " ")}, queries)
		})
	}
}
