package e2e

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

const (
	defaultGovernanceBenchNamespaceDecoys = 100_000
	governanceBenchNamespaceDecoysEnv     = "GOV_BENCH_NAMESPACE_DECOYS"
)

// BenchmarkGovernanceQueryNamespaceScale isolates the effect of total namespace
// size on governance query latency, independent of the customers/features axes
// BenchmarkGovernanceQuery already covers. It fixes a small, constant query
// load (100 customers, matching the OAS customer-key cap, x 1 feature) so the
// entitlement GetAccess fan-out cost — the dominant cost per
// BenchmarkGovernanceQuery's doc comment — stays flat across sub-benchmarks,
// and varies only how many OTHER customers/subjects exist in the namespace
// (decoys, never queried, seeded directly via SQL for speed, not HTTP).
//
// This targets the customer usage-attribution resolution path specifically:
// pre-UNION-ALL, a large decoy count made GetCustomersByUsageAttribution
// seq-scan the customers table (see #4684 and the follow-up bulk fix); this
// benchmark shows whether decoy count still moves total request latency
// post-fix. Decoy counts accumulate across sub-benchmarks (0 -> 10k -> N), so
// each step only seeds the incremental delta.
//
// Requires direct Postgres access alongside OPENMETER_ADDRESS (see
// initE2EPostgresPool) — creating decoys over HTTP would dominate the
// benchmark's own setup time long before showing anything about the query
// path. Set GOV_BENCH_NAMESPACE_DECOYS to change the top decoy count from the
// default 100,000.
func BenchmarkGovernanceQueryNamespaceScale(b *testing.B) {
	client := initClient(b)
	v3 := newV3Client(b)
	pool := initE2EPostgresPool(b)

	const (
		queryCustomers = 100
		queryFeatures  = 1
	)

	custKeys, featKeys := seedGovernanceFixture(b, client, queryCustomers, queryFeatures)
	namespace := getCustomerNamespaceByKey(b, pool, custKeys[0])
	decoyRun := uniqueKey("gov_bench_ns_decoy")

	decoyCounts := []int{0, 10_000, governanceBenchNamespaceDecoyCount(b)}

	reqBody := v3sdk.GovernanceQueryRequest{
		Customer: v3sdk.GovernanceQueryRequestCustomers{Keys: custKeys},
		Feature:  &v3sdk.GovernanceQueryRequestFeatures{Keys: featKeys},
	}

	seeded := 0
	for _, target := range decoyCounts {
		b.Run(fmt.Sprintf("decoys=%d", target), func(b *testing.B) {
			if target > seeded {
				seedNamespaceDecoys(b, pool, namespace, decoyRun, seeded, target)
				seeded = target
			}

			// Warm-up + correctness gate: a wrong result (e.g. missing customers)
			// would make the latency number meaningless.
			resp, err := v3.Governance.QueryAccess(b.Context(), reqBody, v3sdk.GovernanceQueryResultListParams{})
			v3.requireStatus(http.StatusOK, err)
			require.Lenf(b, resp.Data, queryCustomers, "expected %d resolved customers", queryCustomers)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := v3.Governance.QueryAccess(b.Context(), reqBody, v3sdk.GovernanceQueryResultListParams{}); err != nil {
					b.Fatalf("governance query failed: %v", err)
				}
				if s := v3.statuses.last(); s != http.StatusOK {
					b.Fatalf("governance query returned %d", s)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(target), "decoys")
		})
	}
}

func governanceBenchNamespaceDecoyCount(b *testing.B) int {
	b.Helper()

	value := os.Getenv(governanceBenchNamespaceDecoysEnv)
	if value == "" {
		return defaultGovernanceBenchNamespaceDecoys
	}

	count, err := strconv.Atoi(value)
	if err != nil || count < 1 {
		b.Fatalf("%s must be a positive integer, got %q", governanceBenchNamespaceDecoysEnv, value)
	}

	return count
}

func getCustomerNamespaceByKey(tb testing.TB, pool *pgxpool.Pool, key string) string {
	tb.Helper()

	var namespace string
	err := pool.QueryRow(
		tb.Context(),
		`SELECT namespace FROM customers WHERE key = $1`,
		key,
	).Scan(&namespace)
	require.NoError(tb, err)

	return namespace
}

// seedNamespaceDecoys bulk-inserts decoy customers (and one subject key each)
// directly via SQL, numbered (from, to] under the given per-benchmark-run
// prefix, so repeated calls with a growing `to` only seed the incremental
// delta, and separate benchmark runs never collide on the same key even if
// the database was not reset in between. Decoys are never included in a
// governance query; they exist purely to grow the namespace the resolution
// query has to search. ANALYZE keeps planner stats current, matching the
// customer adapter's own usage-attribution benchmark.
func seedNamespaceDecoys(tb testing.TB, pool *pgxpool.Pool, namespace, runPrefix string, from, to int) {
	tb.Helper()

	n := to - from
	ids := make([]string, n)
	keys := make([]string, n)
	subjectKeys := make([]string, n)

	for i := range n {
		ids[i] = ulid.Make().String()
		keys[i] = fmt.Sprintf("%s_%d", runPrefix, from+i)
		subjectKeys[i] = keys[i] + "_subj"
	}

	rctx := tb.Context()

	_, err := pool.Exec(rctx, `
		INSERT INTO customers (id, namespace, created_at, updated_at, name, key)
		SELECT id, $1, now(), now(), key, key
		FROM unnest($2::text[], $3::text[]) AS t(id, key)
	`, namespace, ids, keys)
	require.NoError(tb, err)

	_, err = pool.Exec(rctx, `
		INSERT INTO customer_subjects (namespace, subject_key, created_at, customer_id)
		SELECT $1, subject_key, now(), customer_id
		FROM unnest($2::text[], $3::text[]) AS t(subject_key, customer_id)
	`, namespace, subjectKeys, ids)
	require.NoError(tb, err)

	_, err = pool.Exec(rctx, "ANALYZE customers")
	require.NoError(tb, err)
	_, err = pool.Exec(rctx, "ANALYZE customer_subjects")
	require.NoError(tb, err)
}
