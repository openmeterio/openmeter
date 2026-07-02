package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/convert"
)

// entKind selects which entitlement type the seeded features use. It is the third
// benchmark axis (alongside customer and feature counts) and is what makes the
// benchmark representative of the production cost: boolean entitlements skip the
// metered balance path (ClickHouse), while metered entitlements force the
// per-entitlement usage query inside GetAccess that dominates real latency.
type entKind string

const (
	// kindBoolean grants every feature a boolean entitlement. GetAccess short-circuits
	// the balance calculation, so this is the cheap, algorithmic-only baseline.
	kindBoolean entKind = "boolean"
	// kindMetered grants every feature a metered entitlement, so GetAccess runs a
	// ClickHouse usage query per entitlement — the production-representative path.
	kindMetered entKind = "metered"
	// kindMixed grants alternating boolean/metered entitlements (~50/50), modeling a
	// realistic tenant where only some features are usage-metered.
	kindMixed entKind = "mixed"
)

// selectedKinds reads GOV_BENCH_KIND and returns the entitlement kinds to benchmark.
// Default is boolean only, so the existing `make -C e2e bench-governance` behavior
// (and its baseline numbers) is unchanged. Set GOV_BENCH_KIND=metered|mixed|all to
// measure the metered balance path.
func selectedKinds() []entKind {
	switch os.Getenv("GOV_BENCH_KIND") {
	case "metered":
		return []entKind{kindMetered}
	case "mixed":
		return []entKind{kindMixed}
	case "all":
		return []entKind{kindBoolean, kindMetered, kindMixed}
	default:
		return []entKind{kindBoolean}
	}
}

// BenchmarkGovernanceQuery measures end-to-end latency of
// POST /api/v3/openmeter/governance/query against a running stack
// (OPENMETER_ADDRESS). It is a benchmark, so it only runs under `go test -bench`;
// a plain `go test ./e2e/...` skips it entirely.
//
// What it is good for: a realistic baseline and a before/after signal for
// performance work — it exercises the real router, the OAS layer, real Postgres,
// and the real entitlement GetAccess fan-out over HTTP.
//
// Entitlement kind (GOV_BENCH_KIND) selects what GetAccess actually does:
//   - boolean (default): the balance path is short-circuited — algorithmic baseline.
//   - metered / mixed:   GetAccess runs a ClickHouse usage query per metered
//     entitlement, the larger production cost. This is the path the deferred
//     resolveAccess parallelization is meant to speed up, so measuring it here is
//     what tells us whether that optimization moves the ceiling.
//
// Caveat: the metered path is exercised by the per-entitlement usage query inside
// GetAccess; this fixture does NOT ingest usage events (that would require waiting
// for ClickHouse ingestion and is flaky/slow). So row volume is minimal — the cost
// measured is the per-entitlement query overhead × N customers, which is exactly the
// serial structure parallelization targets, not absolute production row-scan time.
//
// The ClickHouse query is issued even with no grants and no events: with zero grants
// the burndown engine still produces a single full-period phase (credit/engine
// burnphase.go) and runBetweenResets calls QueryUsage for it (credit/engine run.go).
// So there is no zero-grant/zero-usage short-circuit that would collapse the metered
// path back to the boolean one — verified, which is why seeding grants or events is
// unnecessary to make the metered benchmark meaningful. Ingesting events would only
// add row-scan volume on top, material only at production data scale this harness is
// not built to seed.
//
// Sizes scale customers x features. The customer-count axis drives the GetAccess
// fan-out (the dominant cost); the feature-count axis drives the per-customer
// feature-access map. By default the diagonal (10/50/100% of the 100x100 spec
// ceiling) plus a 1x1 fixed-overhead baseline is run. Set GOV_BENCH_FULL_MATRIX=1
// to run the full 3x3 matrix, which isolates the two axes.
//
// Seeding is heavy (100x100 = ~10k entitlement creates over HTTP) and runs once
// per sub-benchmark, outside the timed loop.
func BenchmarkGovernanceQuery(b *testing.B) {
	client := initClient(b) // skips when OPENMETER_ADDRESS is unset
	v3 := newV3Client(b)

	type size struct {
		name      string
		customers int
		// features is how many features/entitlements are SEEDED per customer (the real cost
		// driver: GetAccess computes a balance for every one of a customer's entitlements).
		features int
		// queryFeatures is how many of those features the query actually REQUESTS; 0 means all.
		// When queryFeatures < features the query is "selective": the caller asks about a few
		// features of a customer that has many entitlements. This decouples what's requested (M)
		// from what's computed (N) to expose the over-compute — GetAccess currently resolves all
		// N regardless of M. The default cells keep queryFeatures==features (N==M), which HIDES
		// the over-compute; only GOV_BENCH_OVERCOMPUTE cells reveal it.
		queryFeatures int
	}

	// Diagonal + fixed-overhead baseline (default).
	sizes := []size{
		{"customers=1/features=1", 1, 1, 0},
		{"customers=10/features=10", 10, 10, 0},
		{"customers=50/features=50", 50, 50, 0},
		{"customers=100/features=100", 100, 100, 0},
	}

	// Full 3x3 matrix isolates the customer axis from the feature axis.
	if os.Getenv("GOV_BENCH_FULL_MATRIX") != "" {
		sizes = nil
		sizes = append(sizes, size{"customers=1/features=1", 1, 1, 0})
		for _, c := range []int{10, 50, 100} {
			for _, f := range []int{10, 50, 100} {
				sizes = append(sizes, size{fmt.Sprintf("customers=%d/features=%d", c, f), c, f, 0})
			}
		}
	}

	// Over-compute cells: seed N entitlements/customer but query only M<N features. Reveals that
	// GetAccess computes all N balances (N ClickHouse queries) while the result uses only M — a
	// selective query against a customer with many entitlements. customers=1 gives one clean trace
	// (N GetEntitlementBalance spans, M-feature result); customers=10 gives a latency signal.
	if os.Getenv("GOV_BENCH_OVERCOMPUTE") != "" {
		sizes = []size{
			{"customers=1/features=50/query=10", 1, 50, 10},
			{"customers=10/features=50/query=10", 10, 50, 10},
		}
	}

	for _, kind := range selectedKinds() {
		for _, s := range sizes {
			b.Run(fmt.Sprintf("kind=%s/%s", kind, s.name), func(b *testing.B) {
				custKeys, featKeys := seedGovernanceFixture(b, client, s.customers, s.features, kind)

				// Query all seeded features unless the cell requests a selective subset.
				queryKeys := featKeys
				if s.queryFeatures > 0 && s.queryFeatures < len(featKeys) {
					queryKeys = featKeys[:s.queryFeatures]
				}

				reqBody := apiv3.GovernanceQueryRequest{
					Customer: apiv3.GovernanceQueryRequestCustomers{Keys: custKeys},
					Feature:  &apiv3.GovernanceQueryRequestFeatures{Keys: queryKeys},
				}

				// Warm-up + correctness gate: a wrong result (e.g. missing customers)
				// would make the latency number meaningless.
				status, resp, problem := v3.QueryGovernance(reqBody)
				require.Equalf(b, http.StatusOK, status, "governance query failed: %+v", problem)
				require.Lenf(b, resp.Data, s.customers, "expected %d resolved customers", s.customers)
				require.Lenf(b, resp.Data[0].Features, len(queryKeys), "expected %d features per customer", len(queryKeys))

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					status, _, _ := v3.QueryGovernance(reqBody)
					if status != http.StatusOK {
						b.Fatalf("governance query returned %d", status)
					}
				}
				b.StopTimer()
			})
		}
	}
}

// seedGovernanceFixture creates nFeatures features and nCustomers customers, granting
// each customer one entitlement per feature (nCustomers x nFeatures entitlements). The
// entitlement type is chosen per feature by kind: boolean for kindBoolean, metered for
// kindMetered, and alternating boolean/metered (~50/50) for kindMixed. Metered features
// are backed by a single SUM meter created once per run. Keys carry a per-run unique
// prefix so repeated runs against the same database do not collide. Returns the customer
// keys and feature keys to query.
func seedGovernanceFixture(b *testing.B, client *api.ClientWithResponses, nCustomers, nFeatures int, kind entKind) (custKeys, featKeys []string) {
	b.Helper()
	ctx := b.Context()
	run := uniqueKey("gov_bench")

	// isMetered reports whether feature i is metered, given the kind.
	isMetered := func(i int) bool {
		switch kind {
		case kindMetered:
			return true
		case kindMixed:
			return i%2 == 1
		default: // kindBoolean
			return false
		}
	}

	// Any metered feature needs a meter to point at; create one SUM meter for the run.
	var meterSlug string
	for i := 0; i < nFeatures; i++ {
		if isMetered(i) {
			meterSlug = createGovernanceMeter(b, client, ctx, run)
			break
		}
	}

	featKeys = make([]string, 0, nFeatures)
	featMetered := make([]bool, 0, nFeatures)
	for i := 0; i < nFeatures; i++ {
		fkey := fmt.Sprintf("%s_feat_%d", run, i)
		body := api.CreateFeatureJSONRequestBody{Key: fkey, Name: fkey}
		if isMetered(i) {
			body.MeterSlug = convert.ToPointer(meterSlug)
		}
		resp, err := client.CreateFeatureWithResponse(ctx, body)
		require.NoError(b, err)
		require.Equalf(b, http.StatusCreated, resp.StatusCode(), "create feature: %s", resp.Body)
		featKeys = append(featKeys, fkey)
		featMetered = append(featMetered, isMetered(i))
	}

	custKeys = make([]string, 0, nCustomers)
	for c := 0; c < nCustomers; c++ {
		ckey := fmt.Sprintf("%s_cust_%d", run, c)
		skey := ckey + "_subj"

		subResp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{api.SubjectUpsert{Key: skey}})
		require.NoError(b, err)
		require.Equalf(b, http.StatusOK, subResp.StatusCode(), "upsert subject: %s", subResp.Body)

		custResp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Key:              lo.ToPtr(ckey),
			Name:             ckey,
			UsageAttribution: &api.CustomerUsageAttribution{SubjectKeys: []string{skey}},
		})
		require.NoError(b, err)
		require.Equalf(b, http.StatusCreated, custResp.StatusCode(), "create customer: %s", custResp.Body)
		custID := custResp.JSON201.Id

		for i, fkey := range featKeys {
			if featMetered[i] {
				grantMeteredEntitlement(b, client, ctx, custID, fkey)
			} else {
				grantBooleanEntitlement(b, client, ctx, custID, fkey)
			}
		}
		custKeys = append(custKeys, ckey)
	}

	return custKeys, featKeys
}

// createGovernanceMeter creates a single SUM meter for the benchmark run and returns its
// slug. All metered features in the run share this meter; the meter exists only so
// metered entitlements resolve a usage query — no events are ingested against it.
func createGovernanceMeter(b *testing.B, client *api.ClientWithResponses, ctx context.Context, run string) string {
	b.Helper()

	slug := run + "_meter"
	resp, err := client.CreateMeterWithResponse(ctx, api.MeterCreate{
		Slug:          slug,
		Name:          convert.ToPointer(slug),
		Aggregation:   api.MeterAggregationSum,
		EventType:     run + "_event",
		ValueProperty: convert.ToPointer("$.value"),
	})
	require.NoError(b, err)
	// The meter API returns 200 (not 201) on create.
	require.Equalf(b, http.StatusOK, resp.StatusCode(), "create meter: %s", resp.Body)

	return slug
}

// grantBooleanEntitlement creates a boolean entitlement for the given customer and
// feature key via the V2 customer-entitlement endpoint.
func grantBooleanEntitlement(b *testing.B, client *api.ClientWithResponses, ctx context.Context, custID, featureKey string) {
	b.Helper()

	var body api.CreateCustomerEntitlementV2JSONRequestBody
	require.NoError(b, body.FromEntitlementBooleanCreateInputs(api.EntitlementBooleanCreateInputs{
		Type:       api.EntitlementBooleanCreateInputsTypeBoolean,
		FeatureKey: lo.ToPtr(featureKey),
	}))

	resp, err := client.CreateCustomerEntitlementV2WithResponse(ctx, custID, body)
	require.NoError(b, err)
	require.Equalf(b, http.StatusCreated, resp.StatusCode(), "create boolean entitlement: %s", resp.Body)
}

// grantMeteredEntitlement creates a metered entitlement for the given customer and
// feature key via the V2 customer-entitlement endpoint. The metered type is what forces
// GetAccess to run a balance/usage query (ClickHouse) when governance resolves access.
func grantMeteredEntitlement(b *testing.B, client *api.ClientWithResponses, ctx context.Context, custID, featureKey string) {
	b.Helper()

	month := &api.RecurringPeriodInterval{}
	require.NoError(b, month.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

	var body api.CreateCustomerEntitlementV2JSONRequestBody
	require.NoError(b, body.FromEntitlementMeteredV2CreateInputs(api.EntitlementMeteredV2CreateInputs{
		Type:       "metered",
		FeatureKey: lo.ToPtr(featureKey),
		UsagePeriod: api.RecurringPeriodCreateInput{
			// Anchor at seed time so only the current usage period is evaluated. A fixed past
			// anchor would let the balance engine accrue a monthly reset period per elapsed
			// month, drifting benchmark cost over time and breaking cross-run comparisons.
			Anchor:   convert.ToPointer(time.Now().UTC()),
			Interval: *month,
		},
	}))

	resp, err := client.CreateCustomerEntitlementV2WithResponse(ctx, custID, body)
	require.NoError(b, err)
	require.Equalf(b, http.StatusCreated, resp.StatusCode(), "create metered entitlement: %s", resp.Body)
}
