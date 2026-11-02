package adapter

import (
	stdsql "database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	accountadapter "github.com/openmeterio/openmeter/openmeter/ledger/account/adapter"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func BenchmarkBalancesAtBoundaries(b *testing.B) {
	for _, historySize := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("history_%d", historySize), func(b *testing.B) {
			db, queries := seedBalanceBoundaryHistory(b, historySize)
			for _, pageSize := range []int{1, 50, 500} {
				b.Run(fmt.Sprintf("page_%d", pageSize), func(b *testing.B) {
					input := ledger.GetBalancesAtBoundariesInput{Queries: queries[:2*pageSize]}
					for _, variant := range []struct {
						name  string
						build func() (string, []any, error)
					}{
						{name: "repeated_sums", build: func() (string, []any, error) { return (balancesAtBoundariesQuery{input: input}).independentSums() }},
						{name: "shared_scan", build: (balancesAtBoundariesQuery{input: input}).SQL},
					} {
						b.Run(variant.name, func(b *testing.B) {
							query, args, err := variant.build()
							require.NoError(b, err)
							b.ResetTimer()
							b.RunParallel(func(pb *testing.PB) {
								for pb.Next() {
									rows, err := db.QueryContext(b.Context(), query, args...)
									if err != nil {
										b.Error(err)
										return
									}

									count := 0
									for rows.Next() {
										var amount stdsql.NullString
										var idx int
										if err := rows.Scan(&amount, &idx); err != nil {
											b.Error(err)
										}

										count++
									}

									if err := rows.Err(); err != nil {
										b.Error(err)
									}

									rows.Close()
									if count != len(input.Queries) {
										b.Errorf("got %d boundaries, want %d", count, len(input.Queries))
									}
								}
							})
						})
					}
				})
			}
		})
	}
}

func seedBalanceBoundaryHistory(t testing.TB, historySize int) (*stdsql.DB, []ledger.Query) {
	t.Helper()
	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() { db.Close(t) })
	ctx := t.Context()
	client := db.EntDriver.Client()
	repo := NewRepo(client)
	accounts := accountadapter.NewRepo(client)
	namespace := "boundary-benchmark"
	currency := currencies.NewCurrencyReference(currencyx.Code("USD"))
	var accountIDs, subAccountIDs []string
	for _, accountType := range []ledger.AccountType{ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerReceivable} {
		account, err := accounts.CreateAccount(ctx, ledgeraccount.CreateAccountInput{Namespace: namespace, Type: accountType})
		require.NoError(t, err)
		sub, err := accounts.EnsureSubAccount(ctx, ledgeraccount.CreateSubAccountInput{
			Namespace: namespace, AccountID: account.ID.ID, Route: ledger.Route{Currency: currency},
		})
		require.NoError(t, err)
		accountIDs = append(accountIDs, account.ID.ID)
		subAccountIDs = append(subAccountIDs, sub.ID)
	}

	group, err := repo.CreateTransactionGroup(ctx, ledgerhistorical.CreateTransactionGroupInput{Namespace: namespace})
	require.NoError(t, err)
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	_, err = db.PGDriver.DB().ExecContext(ctx, `INSERT INTO ledger_transactions
		(id, namespace, created_at, updated_at, booked_at, group_id)
		SELECT lpad(n::text, 26, '0'), $1, $2, $2, $2::timestamptz + n * interval '1 minute', $3
		FROM generate_series(1, $4::integer) AS n`, namespace, start, group.ID, historySize)
	require.NoError(t, err)
	_, err = db.PGDriver.DB().ExecContext(ctx, `INSERT INTO ledger_entries
		(id, namespace, created_at, updated_at, amount, sub_account_id, transaction_id, identity_key, schema_version)
		SELECT prefix || substr(id, 2), namespace, created_at, updated_at,
			(CASE WHEN id::bigint % 10 = 1 THEN 10 ELSE -1 END) * sign, subaccount, id, '', 1
		FROM ledger_transactions CROSS JOIN (VALUES ('1', 1, $1::character(26)), ('2', -1, $2::character(26)))
		AS sides(prefix, sign, subaccount)`, subAccountIDs[0], subAccountIDs[1])
	require.NoError(t, err)
	_, err = db.PGDriver.DB().ExecContext(ctx, "ANALYZE")
	require.NoError(t, err)

	queries := make([]ledger.Query, 0, 1000)
	for row := range 500 {
		cursor := ledger.TransactionCursor{
			BookedAt: start.Add(time.Duration(historySize-row) * time.Minute), CreatedAt: start,
		}
		cursor.ID.Namespace = namespace
		cursor.ID.ID = fmt.Sprintf("%026d", historySize-row)
		for idx, accountID := range accountIDs {
			route := ledger.RouteFilter{Currency: currency}
			if idx == 1 {
				route.CostBasis = mo.Some[*alpacadecimal.Decimal](nil)
			}

			queries = append(queries, ledger.Query{Namespace: namespace, Filters: ledger.Filters{
				AccountID: lo.ToPtr(accountID), Route: route, After: &cursor,
			}})
		}
	}

	return db.PGDriver.DB(), queries
}
