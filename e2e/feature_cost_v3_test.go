package e2e

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

func TestV3QueryFeatureCostRepresentations(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)
	key := uniqueKey("feature_cost")
	from := time.Now().UTC().Truncate(time.Minute)
	to := from.Add(time.Minute)

	// Given a metered feature with a fixed unit cost and an attributed customer.
	meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
		Key:           key,
		Name:          "Feature Cost Meter",
		Aggregation:   v3sdk.MeterAggregationSum,
		EventType:     key,
		EventsFrom:    &from,
		ValueProperty: lo.ToPtr("$.value"),
		Dimensions:    &map[string]string{"model": "$.model"},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, meter)
	unitCost, err := v3sdk.FeatureUnitCostFromFeatureManualUnitCost(v3sdk.FeatureManualUnitCost{Amount: "0.000000125"})
	require.NoError(t, err)
	feature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
		Key:      key,
		Name:     "Feature Cost",
		Meter:    &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		UnitCost: &unitCost,
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, feature)
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:              key,
		Name:             "CSV, \"Customer\"",
		UsageAttribution: &v3sdk.CustomerUsageAttribution{SubjectKeys: []string{key}},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	for _, usage := range []struct {
		model string
		value string
	}{
		{model: "model,one", value: "150"},
		{model: "model,one", value: "100"},
		{model: "model-two", value: "40"},
	} {
		ev := event.New()
		ev.SetID(ulid.Make().String())
		ev.SetSource("e2e")
		ev.SetType(meter.EventType)
		ev.SetSubject(key)
		ev.SetTime(from)
		require.NoError(t, ev.SetData("application/json", map[string]string{"model": usage.model, "value": usage.value}))
		response, err := v1.IngestEventWithResponse(t.Context(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), "body: %s", response.Body)
	}

	query := v3sdk.MeterQueryRequest{
		From:              &from,
		To:                &to,
		Granularity:       lo.ToPtr(v3sdk.MeterQueryGranularityMinute),
		GroupByDimensions: &[]string{"model", "customer_id", "subject"},
		Filters: &v3sdk.MeterQueryFilters{
			Dimensions: &map[string]v3sdk.QueryFilterStringMapItemInput{
				"customer_id": {Eq: lo.ToPtr(customer.ID)},
			},
		},
	}
	wantJSON := &v3sdk.FeatureCostQueryResult{
		From: &from,
		To:   &to,
		Data: []v3sdk.FeatureCostQueryRow{
			{
				From: from, To: to, Usage: "250", Cost: v3sdk.NullableValue(v3sdk.Numeric("0.00003125")), Currency: "USD",
				Dimensions: map[string]string{"model": "model,one", "customer_id": customer.ID, "subject": key},
			},
			{
				From: from, To: to, Usage: "40", Cost: v3sdk.NullableValue(v3sdk.Numeric("0.000005")), Currency: "USD",
				Dimensions: map[string]string{"model": "model-two", "customer_id": customer.ID, "subject": key},
			},
		},
	}

	// Wait for the real ingestion/sink pipeline before checking both representations.
	ctx := t.Context()
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		result, err := c.Features.QueryCost(ctx, feature.ID, &query)
		require.NoError(collect, err)
		assert.Equal(collect, wantJSON, result)
	}, time.Minute, time.Second)

	body, err := json.Marshal(query)
	require.NoError(t, err)
	for _, tt := range []struct {
		name   string
		accept string
	}{
		{name: "default JSON"},
		{name: "explicit JSON", accept: "application/json"},
		{name: "CSV", accept: "text/csv"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// When the same request goes through the real feature-cost HTTP route.
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, c.baseURL+"/features/"+feature.ID+"/cost/query", bytes.NewReader(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			response, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer response.Body.Close()
			raw, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode, "body: %s", raw)

			// Then JSON keeps its contract and CSV carries the same rows and exact amounts.
			if tt.accept != "text/csv" {
				require.Equal(t, "application/json", response.Header.Get("Content-Type"))
				var result v3sdk.FeatureCostQueryResult
				require.NoError(t, json.Unmarshal(raw, &result))
				require.Equal(t, wantJSON, &result)
				return
			}
			require.Equal(t, "text/csv", response.Header.Get("Content-Type"))
			require.Equal(t, "attachment; filename="+feature.Key+"-cost.csv", response.Header.Get("Content-Disposition"))
			records, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
			require.NoError(t, err)
			wantCSV := [][]string{{"from", "to", "subject", "customer_id", "customer_key", "customer_name", "model", "usage", "cost", "currency", "detail"}}
			for _, row := range wantJSON.Data {
				wantCSV = append(wantCSV, []string{
					row.From.Format(time.RFC3339), row.To.Format(time.RFC3339), row.Dimensions["subject"],
					row.Dimensions["customer_id"], customer.Key, customer.Name, row.Dimensions["model"],
					row.Usage, row.Cost.MustGet(), row.Currency, lo.FromPtr(row.Detail),
				})
			}
			require.Equal(t, wantCSV, records)
		})
	}
}
