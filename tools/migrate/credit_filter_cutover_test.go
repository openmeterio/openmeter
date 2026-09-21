package migrate_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger/crediteligibility"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestCreditFilterStorageCutover(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer testDB.PGDriver.Close()
	conn, err := testDB.PGDriver.DB().Conn(t.Context())
	require.NoError(t, err)
	defer conn.Close()
	tables := []struct{ table, legacy, filters string }{
		{"ledger_sub_account_routes", "features", "filters"},
		{"charge_credit_purchases", "feature_filters", "filters"},
	}
	// Given legacy rows, unrestricted rows, and an envelope already dual-written.
	for _, tc := range tables {
		_, err = conn.ExecContext(t.Context(), fmt.Sprintf(`CREATE TEMP TABLE %s (id integer PRIMARY KEY, routing_key text DEFAULT 'unchanged', %s text[], %s jsonb);
  INSERT INTO %s (id, %s, %s) VALUES
   (1, NULL, NULL), (2, ARRAY[]::text[], NULL), (3, ARRAY['input','output'], NULL),
   (4, ARRAY['stale'], '{"schema_version":1,"features":["existing"]}');`, tc.table, tc.legacy, tc.filters, tc.table, tc.legacy, tc.filters))
		require.NoError(t, err)
	}
	// When backfilled and rerun, existing envelopes and accounting identities survive.
	up := readMigration(t, "20260921133354_credit_filter_cutover.up.sql")
	_, err = conn.ExecContext(t.Context(), up)
	require.NoError(t, err)
	_, err = conn.ExecContext(t.Context(), up)
	require.NoError(t, err)
	for _, tc := range tables {
		for i, want := range []string{`{"schema_version":1}`, `{"schema_version":1}`, `{"schema_version":1,"features":["input","output"]}`, `{"schema_version":1,"features":["existing"]}`} {
			var filters, key string
			err = conn.QueryRowContext(t.Context(), fmt.Sprintf(`SELECT %s::text, routing_key FROM %s WHERE id=$1`, tc.filters, tc.table), i+1).Scan(&filters, &key)
			require.NoError(t, err)
			require.JSONEq(t, want, filters)
			var decoded crediteligibility.Filters
			require.NoError(t, json.Unmarshal([]byte(filters), &decoded))
			require.Equal(t, crediteligibility.FiltersVersion1, decoded.Version)
			require.Equal(t, "unchanged", key)
		}
		_, err = conn.ExecContext(t.Context(), fmt.Sprintf(`INSERT INTO %s (id,%s) VALUES (5,'{"schema_version":1,"features":["new"]}')`, tc.table, tc.filters))
		require.NoError(t, err)
		_, err = conn.ExecContext(t.Context(), fmt.Sprintf(`INSERT INTO %s (id) VALUES (6)`, tc.table))
		require.ErrorContains(t, err, "not-null constraint")
	}
	// Then rollback reconstructs feature projections, including rows written after cutover.
	down := readMigration(t, "20260921133354_credit_filter_cutover.down.sql")
	_, err = conn.ExecContext(t.Context(), down)
	require.NoError(t, err)
	for _, tc := range tables {
		var features string
		err = conn.QueryRowContext(t.Context(), fmt.Sprintf(`SELECT %s::text FROM %s WHERE id=5`, tc.legacy, tc.table)).Scan(&features)
		require.NoError(t, err)
		require.Equal(t, "{new}", features)
	}
	// Unsupported versions and dimensions must never be reinterpreted on rollback.
	for _, tc := range tables {
		for _, filters := range []string{
			`{"schema_version":2,"features":["new"]}`,
			`{"features":["new"]}`,
			`{"schema_version":1,"plans":[{"key":"pro"}]}`,
		} {
			_, err = conn.ExecContext(t.Context(), fmt.Sprintf(`UPDATE %s SET filters=$1 WHERE id=5`, tc.table), filters)
			require.NoError(t, err)
			_, err = conn.ExecContext(t.Context(), down)
			require.ErrorContains(t, err, "cannot restore feature-only storage")
			_, err = conn.ExecContext(t.Context(), "ROLLBACK")
			require.NoError(t, err)
		}
		_, err = conn.ExecContext(t.Context(), fmt.Sprintf(`UPDATE %s SET filters='{"schema_version":1,"features":["new"]}' WHERE id=5`, tc.table))
		require.NoError(t, err)
	}
}
