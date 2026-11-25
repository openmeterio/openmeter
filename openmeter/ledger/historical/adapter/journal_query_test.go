package adapter

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func TestAccountBookingCoverageQuery(t *testing.T) {
	at := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	earlier := at.Add(-time.Hour)
	query, args := (accountBookingCoverageQuery{input: ledger.AccountBookingCoverageInput{
		AccountEntryScope: ledger.AccountEntryScope{Namespace: "ns", AccountIDs: []string{"fbo", "receivable"}},
		AsOf:              at, Earlier: []time.Time{earlier},
	}}).SQL()

	expected := `
SELECT COUNT(*), MIN("tx"."booked_at") FILTER (WHERE "tx"."booked_at" > $1),
       COUNT(*) FILTER (WHERE "tx"."booked_at" <= $2),
       COUNT(*) FILTER (WHERE "tx"."booked_at" <= $3)
FROM "ledger_entries"
JOIN "ledger_transactions" AS "tx" ON "ledger_entries"."transaction_id" = "tx"."id"
WHERE "ledger_entries"."namespace" = $4 AND EXISTS
(SELECT "ledger_sub_accounts"."id" FROM "ledger_sub_accounts"
 WHERE ("ledger_entries"."sub_account_id" = "ledger_sub_accounts"."id"
 AND "ledger_sub_accounts"."namespace" = $5) AND "ledger_sub_accounts"."account_id" IN ($6, $7))`
	require.Equal(t, strings.Join(strings.Fields(expected), " "), query)
	require.Equal(t, []any{at, at, earlier, "ns", "ns", "fbo", "receivable"}, args)
}
