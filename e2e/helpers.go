package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
)

// initE2EPostgresPool opens a direct Postgres connection alongside OPENMETER_ADDRESS, for
// e2e checks and fixtures that need to read or seed data the public API doesn't expose
// (ledger account mappings, bulk namespace seeding). Requires OPENMETER_E2E_POSTGRES_URL,
// or the local compose stack's default exposed port.
func initE2EPostgresPool(t testing.TB) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("OPENMETER_E2E_POSTGRES_URL")
	if dsn == "" {
		address := os.Getenv("OPENMETER_ADDRESS")
		if !strings.Contains(address, "localhost:38888") && !strings.Contains(address, "127.0.0.1:38888") {
			t.Skipf("this e2e check requires OPENMETER_E2E_POSTGRES_URL or local compose stack at localhost:38888, got %q", address)
		}

		dsn = "postgres://postgres:postgres@127.0.0.1:35432/postgres?sslmode=disable"
	}

	pool, err := pgxpool.New(t.Context(), dsn)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	require.NoError(t, pool.Ping(t.Context()))

	return pool
}

// This will not be needed once we get rid of subjects. Cloud middleware already handles subject / customer creation.
func CreateCustomerWithSubject(t *testing.T, client *api.ClientWithResponses, customerKey string, subjectKey string) *api.Customer {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// First create the customer
	resp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
		Name:         fmt.Sprintf("Test Customer %s", customerKey),
		Currency:     lo.ToPtr("USD"),
		Key:          lo.ToPtr(customerKey),
		PrimaryEmail: lo.ToPtr(fmt.Sprintf("test-%s@test.com", customerKey)),
		UsageAttribution: &api.CustomerUsageAttribution{
			SubjectKeys: []string{subjectKey},
		},
	})

	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

	require.Equal(t, []string{subjectKey}, resp.JSON201.UsageAttribution.SubjectKeys)

	// Then create the subject
	{
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: subjectKey},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
	}

	return resp.JSON201
}

func GetMeterIDBySlug(t *testing.T, client *api.ClientWithResponses, meterSlug string) string {
	t.Helper()

	resp, err := client.GetMeterWithResponse(context.Background(), meterSlug)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
	require.NotNil(t, resp.JSON200)

	return resp.JSON200.Id
}
