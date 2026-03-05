package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
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

func GetMeterIDBySlug(t *testing.T, client *api.ClientWithResponses, meterSlug string) string {
	t.Helper()

	resp, err := client.GetMeterWithResponse(context.Background(), meterSlug)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
	require.NotNil(t, resp.JSON200)

	return resp.JSON200.Id
}

func QueryMeterV3(t *testing.T, meterID string, body apiv3.MeterQueryRequest) (int, *apiv3.MeterQueryResult, error) {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		strings.TrimRight(address, "/")+"/openmeter/meters/"+meterID,
		bytes.NewReader(payload),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.ReadAll(resp.Body)
		return resp.StatusCode, nil, nil
	}

	var result apiv3.MeterQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, &result, nil
}
