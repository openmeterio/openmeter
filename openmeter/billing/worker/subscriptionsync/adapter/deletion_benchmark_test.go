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

// BenchmarkListDeletedSubscriptionsToReconcile measures the actual adapter query
// with 200,000 subscriptions (160,000 deleted), ten namespaces and 100 customers.
// Seeding, VACUUM ANALYZE, warmup, and EXPLAIN are outside the timed section.
// Set SUBSCRIPTION_SYNC_BENCH_PLAN_DIR to save the exact SQL and execution plans.
func BenchmarkListDeletedSubscriptionsToReconcile(b *testing.B) {
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

	_, err := db.ExecContext(ctx, `
INSERT INTO subscriptions
    (id, namespace, customer_id, created_at, updated_at, deleted_at, active_from,
     name, currency, billing_anchor, billing_cadence, pro_rating_config, settlement_mode, cost_basis_mode)
SELECT lpad(i::text, 26, '0'), 'bench-' || ((i-1) % 10), lpad((((i-1) % 100)+1)::text, 26, '0'),
       '2025-01-01'::timestamptz + ((i-1)/10) * interval '1 second',
       '2026-01-01'::timestamptz,
       CASE WHEN ((i-1)/100) % 5 != 0 THEN '2026-01-01'::timestamptz END,
       '2025-01-01'::timestamptz, 'Synthetic subscription ' || i, 'USD',
       '2025-01-01'::timestamptz, 'P1M', '{"enabled":true,"mode":"prorate_prices"}',
       'credit_then_invoice', 'dynamic'
FROM generate_series(1, 200000) AS i`)
	require.NoError(b, err)
	_, err = db.ExecContext(ctx, "VACUUM ANALYZE subscriptions")
	require.NoError(b, err)

	adapter, err := syncadapter.New(syncadapter.Config{Client: testDB.EntDriver.Client()})
	require.NoError(b, err)
	now := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, completion := range []struct {
		name      string
		completed int
	}{
		{name: "backlog", completed: 0},
		{name: "90_percent_done", completed: 900},
		{name: "99_9_percent_done", completed: 999},
		{name: "all_done", completed: 1000},
	} {
		b.Run(completion.name, func(b *testing.B) {
			_, err := db.ExecContext(ctx, "TRUNCATE subscription_sync_states_v2")
			require.NoError(b, err)
			_, err = db.ExecContext(ctx, `
INSERT INTO subscription_sync_states_v2
    (id, namespace, subscription_id, created_at, updated_at, last_success_at, last_success_state)
SELECT id, namespace, id, '2026-01-01', '2026-01-01', '2026-01-01', 'subscription_deleted'
FROM (
    SELECT id, namespace, row_number() OVER (ORDER BY md5(id)) AS position
    FROM subscriptions WHERE deleted_at IS NOT NULL
) ranked
WHERE position <= 160000 * $1 / 1000`, completion.completed)
			require.NoError(b, err)
			_, err = db.ExecContext(ctx, "VACUUM ANALYZE subscription_sync_states_v2")
			require.NoError(b, err)

			for _, selection := range []struct {
				name       string
				namespaces []string
				customers  []string
				cursor     *paginationv2.Cursor
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
			} {
				b.Run(selection.name, func(b *testing.B) {
					input := subscriptionsync.ListDeletedSubscriptionsToReconcileInput{
						AsOf:       now,
						Namespaces: selection.namespaces,
						Customers:  selection.customers,
						Cursor:     selection.cursor,
						Limit:      10000,
					}
					page, err := adapter.ListDeletedSubscriptionsToReconcile(b.Context(), input)
					require.NoError(b, err)
					switch completion.completed {
					case 1000:
						require.Empty(b, page.Items)
					case 0:
						require.NotEmpty(b, page.Items)
					}

					query, args := syncadapter.DeletedSubscriptionsToReconcileQuery(dialect.Postgres, input)
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
						page, err = adapter.ListDeletedSubscriptionsToReconcile(b.Context(), input)
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
