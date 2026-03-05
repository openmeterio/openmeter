package quickstart

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
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

func initClient(t *testing.T) *api.ClientWithResponses {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	client, err := api.NewClientWithResponses(address)
	require.NoError(t, err)

	return client
}

func openmeterAddress(t *testing.T) string {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	return strings.TrimRight(address, "/")
}

func createMeterV3(t *testing.T, address string, body apiv3.CreateMeterRequest) *apiv3.Meter {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		address+"/openmeter/meters",
		bytes.NewReader(payload),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Invalid status code [response_body=%s]", string(responseBody))

	var created apiv3.Meter
	require.NoError(t, json.Unmarshal(responseBody, &created))

	return &created
}

func queryMeterV3(t require.TestingT, address string, meterID string, body apiv3.MeterQueryRequest) *apiv3.MeterQueryResult {
	if helper, ok := any(t).(interface{ Helper() }); ok {
		helper.Helper()
	}

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		address+"/openmeter/meters/"+meterID,
		bytes.NewReader(payload),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Invalid status code [response_body=%s]", string(responseBody))

	var result apiv3.MeterQueryResult
	require.NoError(t, json.Unmarshal(responseBody, &result))

	return &result
}

func TestQuickstart(t *testing.T) {
	client := initClient(t)
	address := openmeterAddress(t)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	v1MeterSlug := "quickstart_v1_" + suffix
	v3MeterKey := "quickstart_v3_" + suffix

	v1MeterName := "Quickstart V1 Meter"
	v1CreateResp, err := client.CreateMeterWithResponse(context.Background(), api.CreateMeterJSONRequestBody{
		Slug:        v1MeterSlug,
		Name:        lo.ToPtr(v1MeterName),
		Aggregation: api.MeterAggregationCount,
		EventType:   "request",
		GroupBy: &map[string]string{
			"method": "$.method",
			"route":  "$.route",
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, v1CreateResp.StatusCode(), "Invalid status code [response_body=%s]", string(v1CreateResp.Body))

	v3Meter := createMeterV3(t, address, apiv3.CreateMeterRequest{
		Key:         v3MeterKey,
		Name:        "Quickstart V3 Meter",
		Aggregation: apiv3.MeterAggregationCount,
		EventType:   "request",
		Dimensions: &map[string]string{
			"method": "$.method",
			"route":  "$.route",
		},
	})

	// TODO: read these from JSON files to make it easier to keep things in sync
	{
		ev := cloudevents.New()
		ev.SetID("00001")
		ev.SetSource("service-0")
		ev.SetType("request")
		ev.SetSubject("customer-1")
		ev.SetTime(time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC))
		_ = ev.SetData("application/json", map[string]string{
			"method":      "GET",
			"route":       "/hello",
			"duration_ms": "40",
		})

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.IngestEventWithResponse(context.Background(), ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}, 30*time.Second, time.Second)
	}

	{
		ev := cloudevents.New()
		ev.SetID("00002")
		ev.SetSource("service-0")
		ev.SetType("request")
		ev.SetSubject("customer-1")
		ev.SetTime(time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC))
		_ = ev.SetData("application/json", map[string]string{
			"method":      "GET",
			"route":       "/hello",
			"duration_ms": "40",
		})

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.IngestEventWithResponse(context.Background(), ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}, 30*time.Second, time.Second)
	}

	{
		ev := cloudevents.New()
		ev.SetID("00003")
		ev.SetSource("service-0")
		ev.SetType("request")
		ev.SetSubject("customer-1")
		ev.SetTime(time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC))
		_ = ev.SetData("application/json", map[string]string{
			"method":      "GET",
			"route":       "/hello",
			"duration_ms": "40",
		})

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.IngestEventWithResponse(context.Background(), ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}, 30*time.Second, time.Second)
	}

	t.Run("v1", func(t *testing.T) {
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			windowSize := meter.WindowSizeHour

			resp, err := client.QueryMeterWithResponse(context.Background(), v1MeterSlug, &api.QueryMeterParams{
				WindowSize: lo.ToPtr(api.WindowSize(windowSize)),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			require.Len(t, resp.JSON200.Data, 2)
			assert.Equal(t, float64(2), resp.JSON200.Data[0].Value)
			assert.Equal(t, float64(1), resp.JSON200.Data[1].Value)
		}, 30*time.Second, time.Second)
	})

	t.Run("v3", func(t *testing.T) {
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			granularity := apiv3.MeterQueryGranularity("PT1H")
			resp := queryMeterV3(t, address, v3Meter.Id, apiv3.MeterQueryRequest{
				Granularity: &granularity,
			})

			require.Len(t, resp.Data, 2)
			assert.Equal(t, apiv3.Numeric("2"), resp.Data[0].Value)
			assert.Equal(t, apiv3.Numeric("1"), resp.Data[1].Value)
		}, 30*time.Second, time.Second)
	})
}
