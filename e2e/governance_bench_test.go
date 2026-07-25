package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// BenchmarkGovernanceQuery measures end-to-end latency of
// POST /api/v3/openmeter/governance/query against a running stack
// (OPENMETER_ADDRESS). It is a benchmark, so it only runs under `go test -bench`;
// a plain `go test ./e2e/...` skips it entirely.
//
// What it is good for: a realistic baseline and a before/after signal for
// performance work — it exercises the real router, the OAS layer, real Postgres,
// and the real entitlement GetAccess fan-out over HTTP.
//
// What it is NOT: a production-latency oracle. Entitlements seeded here are
// boolean, so GetAccess skips the metered balance path (ClickHouse). That path is
// the larger production cost and is instrumented separately (entitlement package).
// A metered variant — seeding usage events and waiting for ClickHouse ingestion —
// is the follow-up for measuring it.
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
		features  int
	}

	// Diagonal + fixed-overhead baseline (default).
	sizes := []size{
		{"customers=1/features=1", 1, 1},
		{"customers=10/features=10", 10, 10},
		{"customers=50/features=50", 50, 50},
		{"customers=100/features=100", 100, 100},
	}

	// Full 3x3 matrix isolates the customer axis from the feature axis.
	if os.Getenv("GOV_BENCH_FULL_MATRIX") != "" {
		sizes = nil
		sizes = append(sizes, size{"customers=1/features=1", 1, 1})
		for _, c := range []int{10, 50, 100} {
			for _, f := range []int{10, 50, 100} {
				sizes = append(sizes, size{fmt.Sprintf("customers=%d/features=%d", c, f), c, f})
			}
		}
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			custKeys, featKeys := seedGovernanceFixture(b, client, s.customers, s.features)

			reqBody := v3sdk.GovernanceQueryRequest{
				Customer: v3sdk.GovernanceQueryRequestCustomers{Keys: custKeys},
				Feature:  &v3sdk.GovernanceQueryRequestFeatures{Keys: featKeys},
			}

			// Warm-up + correctness gate: a wrong result (e.g. missing customers)
			// would make the latency number meaningless.
			resp, err := v3.Governance.QueryAccess(b.Context(), reqBody, v3sdk.GovernanceQueryResultListParams{})
			v3.requireStatus(http.StatusOK, err)
			require.Lenf(b, resp.Data, s.customers, "expected %d resolved customers", s.customers)
			require.Lenf(b, resp.Data[0].Features, s.features, "expected %d features per customer", s.features)

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
		})
	}
}

// seedGovernanceFixture creates nFeatures boolean features and nCustomers
// customers, granting each customer a boolean entitlement for every feature
// (nCustomers x nFeatures entitlements). Keys carry a per-run unique prefix so
// repeated runs against the same database do not collide. Returns the customer
// keys and feature keys to query.
func seedGovernanceFixture(b *testing.B, client *api.ClientWithResponses, nCustomers, nFeatures int) (custKeys, featKeys []string) {
	b.Helper()
	ctx := b.Context()
	run := uniqueKey("gov_bench")

	featKeys = make([]string, 0, nFeatures)
	for i := 0; i < nFeatures; i++ {
		fkey := fmt.Sprintf("%s_feat_%d", run, i)
		resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Key:  fkey,
			Name: fkey,
		})
		require.NoError(b, err)
		require.Equalf(b, http.StatusCreated, resp.StatusCode(), "create feature: %s", resp.Body)
		featKeys = append(featKeys, fkey)
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

		for _, fkey := range featKeys {
			grantBooleanEntitlement(b, client, ctx, custID, fkey)
		}
		custKeys = append(custKeys, ckey)
	}

	return custKeys, featKeys
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
