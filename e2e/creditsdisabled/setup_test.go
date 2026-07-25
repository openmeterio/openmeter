package creditsdisabled

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
)

func initClient(t testing.TB) *api.ClientWithResponses {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	client, err := api.NewClientWithResponses(address)
	require.NoError(t, err)

	return client
}

func createCustomerWithSubject(t *testing.T, client *api.ClientWithResponses, customerKey string, subjectKey string) *api.Customer {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

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
	require.NotNil(t, resp.JSON201)
	require.NotNil(t, resp.JSON201.UsageAttribution)

	require.Equal(t, []string{subjectKey}, resp.JSON201.UsageAttribution.SubjectKeys)

	subjectResp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
		api.SubjectUpsert{Key: subjectKey},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, subjectResp.StatusCode(), "Invalid status code [response_body=%s]", string(subjectResp.Body))

	return resp.JSON201
}
