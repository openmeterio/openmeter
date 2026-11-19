package adapter_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	syncadapter "github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// BenchmarkListSubscriptionsToReconcile measures the actual adapter query with
// 200,000 subscriptions in ten namespaces and 100 customers: 100,000 running,
// 10,000 ended within the lookback, 70,000 ended a year earlier and 20,000 deleted.
// Seeding, VACUUM ANALYZE, warmup, and EXPLAIN are outside the timed section.
// Set SUBSCRIPTION_SYNC_BENCH_PLAN_DIR to save the exact SQL and execution plans.
func BenchmarkListSubscriptionsToReconcile(b *testing.B) {
	b.StopTimer()
	ctx := b.Context()
	testDB := testutils.InitPostgresDB(b, testutils.PostgresDBStateAtlasMigrated)
	b.Cleanup(func() { testDB.Close(b) })
	db := testDB.PGDriver.DB()
	for i := 0; i < 100; i++ {
		err := testDB.EntDriver.Client().Customer.Create().
			SetID(fmt.Sprintf("%026d", i+1)).SetNamespace(fmt.Sprintf("bench-%d", i%10)).
			SetName("Synthetic benchmark customer").Exec(ctx)
		require.NoError(b, err)
	}

	asOf := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// The group is taken from (i-1)/100 so it is independent of the namespace
	// and customer, which cycle on i-1:
	//   0-9   running (active_to NULL)
	//   10    ended twelve hours before asOf, inside the default lookback
	//   11-17 ended a year before asOf
	//   18-19 deleted
	_, err := db.ExecContext(ctx, `
INSERT INTO subscriptions
    (id, namespace, customer_id, created_at, updated_at, deleted_at, active_from, active_to,
     name, currency, billing_anchor, billing_cadence, pro_rating_config, settlement_mode, cost_basis_mode)
SELECT lpad(i::text, 26, '0'), 'bench-' || ((i-1) % 10), lpad((((i-1) % 100)+1)::text, 26, '0'),
       '2025-01-01T00:00:00Z'::timestamptz + ((i-1)/10) * interval '1 second',
       '2026-01-01'::timestamptz,
       CASE WHEN ((i-1)/100) % 20 >= 18 THEN '2026-01-01'::timestamptz END,
       '2024-06-01'::timestamptz,
       CASE WHEN ((i-1)/100) % 20 = 10 THEN $1::timestamptz - interval '12 hours'
            WHEN ((i-1)/100) % 20 BETWEEN 11 AND 17 THEN $1::timestamptz - interval '1 year'
       END,
       'Synthetic subscription ' || i, 'USD',
       '2025-01-01'::timestamptz, 'P1M', '{"enabled":true,"mode":"prorate_prices"}',
       'credit_then_invoice', 'dynamic'
FROM generate_series(1, 200000) AS i`, asOf)
	require.NoError(b, err)
	_, err = db.ExecContext(ctx, "VACUUM ANALYZE subscriptions")
	require.NoError(b, err)

	adapter, err := syncadapter.New(syncadapter.Config{Client: testDB.EntDriver.Client()})
	require.NoError(b, err)

	for _, scenario := range []struct {
		name string
		// withState is the share (per thousand) of non-deleted subscriptions that have a state row.
		withState int
		// due is the share (per thousand) of running subscriptions with a state row whose deadline has passed.
		due int
		// changed is the share (per thousand) of rows whose subscription changed after the last sync.
		changed int
	}{
		// First run after rollout: nothing has a recorded outcome yet.
		{name: "rollout_no_state"},
		{name: "rollout_half_done", withState: 500},
		// Regular operation: a few deadlines pass and a few change events are lost between runs.
		{name: "steady", withState: 1000, due: 5, changed: 1},
		{name: "nothing_due", withState: 1000},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			_, err := db.ExecContext(ctx, "TRUNCATE subscription_sync_states_v2")
			require.NoError(b, err)
			_, err = db.ExecContext(ctx, `
INSERT INTO subscription_sync_states_v2
    (id, namespace, subscription_id, created_at, updated_at, last_success_at, last_success_state,
     next_reconciliation_at, last_synced_subs_revision, current_subs_revision)
SELECT id, namespace, id, '2026-01-01', '2026-01-01', '2026-01-01',
       CASE WHEN active_to IS NULL THEN 'active' ELSE 'ended' END,
       CASE WHEN active_to IS NULL THEN
           CASE WHEN position % 1000 < $3 THEN $2::timestamptz - interval '1 hour'
                ELSE $2::timestamptz + interval '10 days' END
       END,
       lpad('1', 26, '0'),
       CASE WHEN (position + 500) % 1000 < $4 THEN lpad('2', 26, '0') ELSE lpad('1', 26, '0') END
FROM (
    SELECT id, namespace, active_to, row_number() OVER (ORDER BY md5(id)) AS position
    FROM subscriptions WHERE deleted_at IS NULL
) ranked
WHERE position <= 180000 * $1::bigint / 1000`, scenario.withState, asOf, scenario.due, scenario.changed)
			require.NoError(b, err)
			_, err = db.ExecContext(ctx, "VACUUM ANALYZE subscription_sync_states_v2")
			require.NoError(b, err)

			for _, selection := range []struct {
				name       string
				namespaces []string
				customers  []string
				cursor     *paginationv2.Cursor
				forceSync  bool
			}{
				{name: "global_first"},
				{
					name: "global_late",
					cursor: &paginationv2.Cursor{
						Time: createdAt.Add(14999 * time.Second),
						ID:   fmt.Sprintf("%026d", 150000),
					},
				},
				{name: "namespace", namespaces: []string{"bench-9"}},
				{
					name:       "namespace_late",
					namespaces: []string{"bench-9"},
					cursor: &paginationv2.Cursor{
						Time: createdAt.Add(14999 * time.Second),
						ID:   fmt.Sprintf("%026d", 150000),
					},
				},
				{name: "customer", namespaces: []string{"bench-9"}, customers: []string{fmt.Sprintf("%026d", 100)}},
				{name: "forced_global_first", forceSync: true},
			} {
				b.Run(selection.name, func(b *testing.B) {
					input := subscriptionsync.ListSubscriptionsToReconcileInput{
						Namespaces: selection.namespaces,
						Customers:  selection.customers,
						AsOf:       asOf,
						Lookback:   24 * time.Hour,
						ForceSync:  selection.forceSync,
						Cursor:     selection.cursor,
						Limit:      10000,
					}
					page, err := adapter.ListSubscriptionsToReconcile(b.Context(), input)
					require.NoError(b, err)
					switch {
					case selection.forceSync, scenario.withState == 0:
						require.NotEmpty(b, page.Items)
					case scenario.withState == 1000 && scenario.due == 0 && scenario.changed == 0:
						require.Empty(b, page.Items)
					}

					query, args := syncadapter.SubscriptionsToReconcileQuery(dialect.Postgres, input)
					var plan string
					err = db.QueryRowContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+query, args...).Scan(&plan)
					require.NoError(b, err)
					if dir := os.Getenv("SUBSCRIPTION_SYNC_BENCH_PLAN_DIR"); dir != "" {
						require.NoError(b, os.MkdirAll(dir, 0o755))
						name := strings.ReplaceAll(b.Name(), "/", "_")
						require.NoError(b, os.WriteFile(filepath.Join(dir, name+".json"), []byte(plan), 0o644))
						require.NoError(b, os.WriteFile(filepath.Join(dir, name+".sql"),
							[]byte(fmt.Sprintf("%s\n-- Args: %v\n", query, args)), 0o644))
					}

					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						page, err = adapter.ListSubscriptionsToReconcile(b.Context(), input)
						if err != nil {
							b.Fatal(err)
						}
					}

					b.StopTimer()
					b.ReportMetric(float64(len(page.Items)), "rows/op")
				})
			}
		})
	}
}
