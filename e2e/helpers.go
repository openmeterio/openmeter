package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
)

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
