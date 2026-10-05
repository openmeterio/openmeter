package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBreakageLedgerReferenceConstraintTiming(t *testing.T) {
	const (
		versionBefore = 20260925133042
		versionAfter  = 20261005135427
	)

	runner{
		stops: stops{
			{
				version:   versionBefore,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					requireBreakageReferenceTiming(t, db, false)
				},
			},
			{
				version:   versionAfter,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Ledger references may be unresolved until the caller posts its group.
					requireBreakageReferenceTiming(t, db, true)
				},
			},
			{
				version:   versionBefore,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					requireBreakageReferenceTiming(t, db, false)
				},
			},
		},
	}.Test(t)
}

func requireBreakageReferenceTiming(t *testing.T, db *sql.DB, deferred bool) {
	t.Helper()

	var count int
	var deferrable, initiallyDeferred bool
	err := db.QueryRowContext(t.Context(), `
		SELECT COUNT(*), BOOL_AND(condeferrable), BOOL_AND(condeferred)
		FROM pg_constraint
		WHERE conrelid = 'ledger_breakage_records'::regclass
		  AND confrelid IN ('ledger_entries'::regclass, 'ledger_transactions'::regclass, 'ledger_transaction_groups'::regclass)
	`).Scan(&count, &deferrable, &initiallyDeferred)
	require.NoError(t, err)
	require.Equal(t, 5, count)
	require.Equal(t, deferred, deferrable)
	require.Equal(t, deferred, initiallyDeferred)

	// Plan/release and subaccount references remain immediate.
	err = db.QueryRowContext(t.Context(), `
		SELECT COUNT(*) FROM pg_constraint
		WHERE conrelid = 'ledger_breakage_records'::regclass
		  AND contype = 'f'
		  AND confrelid NOT IN ('ledger_entries'::regclass, 'ledger_transactions'::regclass, 'ledger_transaction_groups'::regclass)
		  AND NOT condeferrable AND NOT condeferred
	`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 4, count)
}
