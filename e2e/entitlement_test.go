package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func TestEntitlementV2(t *testing.T) {
	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	meterSlug := "entitlement_v2_meter"
	subject := "ent_customer_v2"

	// ensure subject exists
	{
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: subject},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}

	apiMONTH := &api.RecurringPeriodInterval{}
	require.NoError(t, apiMONTH.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

	apiYEAR := &api.RecurringPeriodInterval{}
	require.NoError(t, apiYEAR.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumYEAR))

	// V2: Create entitlement via Customer API and list them (with real customer)
	t.Run("V2 Create and List Customer Entitlements", func(t *testing.T) {
		// Set up dedicated subject and customer
		v2Subject := "ent_customer_v2"
		{
			resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{api.SubjectUpsert{Key: v2Subject}})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())
		}
		var v2CustomerID string
		{
			resp, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
				Name:             "Entitlement V2 Customer",
				UsageAttribution: &api.CustomerUsageAttribution{SubjectKeys: []string{v2Subject}},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", resp.Body)
			v2CustomerID = resp.JSON201.Id
		}

		// Create feature for V2 flow
		var v2FeatureId string
		{
			randKey := fmt.Sprintf("entitlement_v2_list_feature_%d", time.Now().Unix())
			resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
				Name:      "Entitlement V2 List Feature",
				MeterSlug: convert.ToPointer(meterSlug),
				Key:       randKey,
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			v2FeatureId = resp.JSON201.Id
		}

		// Create entitlement via customer V2 endpoint
		{
			iv := &api.RecurringPeriodInterval{}
			require.NoError(t, iv.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))
			me := api.EntitlementMeteredV2CreateInputs{
				Type:      "metered",
				FeatureId: &v2FeatureId,
				UsagePeriod: api.RecurringPeriodCreateInput{
					Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
					Interval: *iv,
				},
			}
			var body api.CreateCustomerEntitlementV2JSONRequestBody
			require.NoError(t, body.FromEntitlementMeteredV2CreateInputs(me))

			res, err := client.CreateCustomerEntitlementV2WithResponse(ctx, v2CustomerID, body)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))

			v2, err := res.JSON201.AsEntitlementMeteredV2()
			require.NoError(t, err)
			require.Equal(t, v2FeatureId, v2.FeatureId)
			require.Equal(t, v2CustomerID, v2.CustomerId)
		}

		// List customer entitlements V2
		{
			resp, err := client.ListCustomerEntitlementsV2WithResponse(ctx, v2CustomerID, &api.ListCustomerEntitlementsV2Params{
				Page:     lo.ToPtr(1),
				PageSize: lo.ToPtr(10),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			require.NotNil(t, resp.JSON200)
			require.GreaterOrEqual(t, resp.JSON200.TotalCount, 1)
		}
	})

	// V2: Validate error mapping when customer does not exist
	t.Run("V2 Create with missing customer should map to 404", func(t *testing.T) {
		iv := &api.RecurringPeriodInterval{}
		require.NoError(t, iv.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))
		me := api.EntitlementMeteredV2CreateInputs{
			Type:       "metered",
			FeatureKey: lo.ToPtr("nonexistent_feature_key"),
			UsagePeriod: api.RecurringPeriodCreateInput{
				Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
				Interval: *iv,
			},
		}
		var body api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, body.FromEntitlementMeteredV2CreateInputs(me))

		// Use random customer id to ensure not found
		randomID := fmt.Sprintf("missing-%d", time.Now().UnixNano())
		res, err := client.CreateCustomerEntitlementV2WithResponse(ctx, randomID, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, res.StatusCode(), "expected 404 mapping, got %d", res.StatusCode())
	})
}

func TestEntitlementWithUniqueCountAggregation(t *testing.T) {
	// This takes a minute to run in itself due to Entitlements being one minute rounded and we need to wait in the last
	// test for the minute to pass.
	if !shouldRunSlowTests(t) {
		t.Skip("Skipping slow test, please reenable when we have a second resolution for entitlements")
	}

	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	meterSlug := "entitlement_uc_meter"
	subject := "ent_customer"
	var featureId string
	var entitlementId string

	// ensure subject exists
	{
		resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: subject},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}

	apiMONTH := &api.RecurringPeriodInterval{}
	require.NoError(t, apiMONTH.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

	apiYEAR := &api.RecurringPeriodInterval{}
	require.NoError(t, apiYEAR.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumYEAR))

	t.Run("Create Feature", func(t *testing.T) {
		randKey := fmt.Sprintf("entitlement_uc_test_feature_%d", time.Now().Unix())
		resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Name:      "Entitlement Test Feature",
			MeterSlug: convert.ToPointer(meterSlug),
			Key:       randKey,
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		featureId = resp.JSON201.Id
	})

	t.Run("Create a Entitlement", func(t *testing.T) {
		meteredEntitlement := api.EntitlementMeteredCreateInputs{
			Type:      "metered",
			FeatureId: &featureId,
			UsagePeriod: api.RecurringPeriodCreateInput{
				Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
				Interval: *apiMONTH,
			},
		}
		body := &api.CreateEntitlementJSONRequestBody{}
		err := body.FromEntitlementMeteredCreateInputs(meteredEntitlement)
		require.NoError(t, err)
		resp, err := client.CreateEntitlementWithResponse(ctx, subject, *body)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		metered, err := resp.JSON201.AsEntitlementMetered()
		require.NoError(t, err)

		require.Equal(t, metered.SubjectKey, subject)
		entitlementId = metered.Id
	})

	t.Run("Create entitlement via Customer V2 API", func(t *testing.T) {
		// Ensure subject exists and matches a customer mapping (use same subject)
		{
			resp, err := client.UpsertSubjectWithResponse(ctx, api.UpsertSubjectJSONRequestBody{api.SubjectUpsert{Key: subject}})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())
		}

		// Create a feature for V2 test
		var v2FeatureId string
		{
			randKey := fmt.Sprintf("entitlement_v2_test_feature_%d", time.Now().Unix())
			resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
				Name:      "Entitlement V2 Test Feature",
				MeterSlug: convert.ToPointer(meterSlug),
				Key:       randKey,
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			v2FeatureId = resp.JSON201.Id
		}

		// Create entitlement via customer V2 endpoint
		{
			apiMONTH := &api.RecurringPeriodInterval{}
			require.NoError(t, apiMONTH.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

			meteredEntitlement := api.EntitlementMeteredV2CreateInputs{
				Type:      "metered",
				FeatureId: &v2FeatureId,
				UsagePeriod: api.RecurringPeriodCreateInput{
					Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
					Interval: *apiMONTH,
				},
			}

			// Build union body for V2
			var createBody api.CreateCustomerEntitlementV2JSONRequestBody
			require.NoError(t, createBody.FromEntitlementMeteredV2CreateInputs(meteredEntitlement))

			// Use customerIdOrKey that maps 1:1 to subject
			res, err := client.CreateCustomerEntitlementV2WithResponse(ctx, subject, createBody)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, res.StatusCode(), "Invalid status code [response_body=%s]", string(res.Body))

			// Validate V2 union response minimally
			v2, err := res.JSON201.AsEntitlementMeteredV2()
			require.NoError(t, err)
			require.Equal(t, v2FeatureId, v2.FeatureId)
			// CustomerKey may be nil if we resolved by customerId only; assert customerId presence
			require.Equal(t, subject, v2.CustomerId)
		}
	})

	grantAmount := 100.0
	t.Run("Create Grant", func(t *testing.T) {
		effectiveAt := time.Now().Truncate(time.Minute)

		priority := uint8(1)
		maxRolloverAmount := 100.0
		minRolloverAmount := 0.0

		// Create grant
		resp, err := client.CreateGrantWithResponse(ctx, subject, entitlementId, api.EntitlementGrantCreateInput{
			Amount:      grantAmount,
			EffectiveAt: effectiveAt,
			Expiration: api.ExpirationPeriod{
				Duration: "MONTH",
				Count:    1,
			},
			Priority:          &priority,
			MaxRolloverAmount: &maxRolloverAmount,
			MinRolloverAmount: &minRolloverAmount,
			Recurrence: &api.RecurringPeriodCreateInput{
				Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
				Interval: *apiYEAR,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", resp.Body)
	})

	uniqueEventCount := 10
	t.Run("Report usage", func(t *testing.T) {
		now := time.Now()

		for i := 0; i < uniqueEventCount*2; i++ {
			timestamp := gofakeit.DateRange(now, now.Add(time.Second))

			ev := cloudevents.New()
			ev.SetID(gofakeit.UUID())
			ev.SetSource("my-app")
			ev.SetType("credit_event_uc")
			ev.SetSubject(subject)
			ev.SetTime(timestamp)
			_ = ev.SetData("application/json", map[string]string{
				// Let's have 50% of the events with the same value
				"value": fmt.Sprintf("%v", math.Floor(float64(i)/2)),
			})

			resp, err := client.IngestEventWithResponse(ctx, ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}

		// Wait for events to be processed
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.QueryMeterWithResponse(ctx, meterSlug, &api.QueryMeterParams{
				To: convert.ToPointer(time.Now().Truncate(time.Minute)),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			require.Len(t, resp.JSON200.Data, 1)
			assert.Equal(t, float64(uniqueEventCount), resp.JSON200.Data[0].Value)
		}, 2*time.Minute, time.Second)
	})

	t.Run("Should calculate usage correctly", func(t *testing.T) {
		resp, err := client.GetEntitlementValueWithResponse(ctx, subject, entitlementId, &api.GetEntitlementValueParams{
			Time: convert.ToPointer(time.Now().Truncate(time.Minute)),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.NotNil(t, resp.JSON200.Balance)

		assert.Equal(t, grantAmount-float64(uniqueEventCount), *resp.JSON200.Balance)
		assert.Equal(t, float64(uniqueEventCount), *resp.JSON200.Usage)
	})

	t.Run("Should not count usage of previous period twice", func(t *testing.T) {
		// let's wait till the next minute so we can reset
		currMinute := time.Now().Truncate(time.Minute)
		waitUntil := currMinute.Add(time.Minute + time.Second)
		time.Sleep(time.Until(waitUntil))

		effectiveAt := time.Now().Truncate(time.Minute)

		// Reset usage
		_, err := client.ResetEntitlementUsageWithResponse(ctx, subject, entitlementId, api.ResetEntitlementUsageJSONRequestBody{
			EffectiveAt: &effectiveAt,
		})
		require.NoError(t, err)

		resp, err := client.GetEntitlementValueWithResponse(ctx, subject, entitlementId, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.NotNil(t, resp.JSON200.Balance)

		// Grant can roll over with full amount
		assert.Equal(t, grantAmount-float64(uniqueEventCount), *resp.JSON200.Balance)
		assert.Equal(t, float64(0), *resp.JSON200.Usage)
	})
}

func TestEntitlementISOUsagePeriod(t *testing.T) {
	t.Run("Should create entitlement with ISO usage period", func(t *testing.T) {
		client := initClient(t)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		meterSlug := "entitlement_uc_meter"
		customer := "ent_customer_2"
		subject := customer + "-subject"
		var featureId string
		var entitlementId string

		CreateCustomerWithSubject(t, client, customer, subject)

		iv2w := &api.RecurringPeriodInterval{}
		require.Nil(t, iv2w.FromRecurringPeriodInterval0("P2W"))

		t.Run("Create Feature", func(t *testing.T) {
			randKey := fmt.Sprintf("entitlement_uc_test_feature_%d", time.Now().Unix())
			resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
				Name:      "Entitlement Test Feature",
				MeterSlug: convert.ToPointer(meterSlug),
				Key:       randKey,
			})

			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

			featureId = resp.JSON201.Id
		})

		t.Run("Create a Entitlement", func(t *testing.T) {
			meteredEntitlement := api.EntitlementMeteredCreateInputs{
				Type:      "metered",
				FeatureId: &featureId,
				UsagePeriod: api.RecurringPeriodCreateInput{
					Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
					Interval: *iv2w,
				},
			}
			body := &api.CreateEntitlementJSONRequestBody{}
			err := body.FromEntitlementMeteredCreateInputs(meteredEntitlement)
			require.NoError(t, err)
			resp, err := client.CreateEntitlementWithResponse(ctx, subject, *body)

			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

			metered, err := resp.JSON201.AsEntitlementMetered()
			require.NoError(t, err)

			require.Equal(t, metered.SubjectKey, subject)
			entitlementId = metered.Id
		})

		t.Run("Create Grant", func(t *testing.T) {
			effectiveAt := time.Now().Truncate(time.Minute)

			priority := uint8(1)
			maxRolloverAmount := 100.0
			minRolloverAmount := 0.0

			// Create grant
			resp, err := client.CreateGrantWithResponse(ctx, subject, entitlementId, api.EntitlementGrantCreateInput{
				Amount:      100,
				EffectiveAt: effectiveAt,
				Expiration: api.ExpirationPeriod{
					Duration: "MONTH",
					Count:    1,
				},
				Priority:          &priority,
				MaxRolloverAmount: &maxRolloverAmount,
				MinRolloverAmount: &minRolloverAmount,
				Recurrence: &api.RecurringPeriodCreateInput{
					Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
					Interval: *iv2w,
				},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", resp.Body)
		})
	})
}

func TestEntitlementWithLatestAggregation(t *testing.T) {
	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	meterSlug := "entitlement_latest_meter"
	subject := "ent_latest_customer"
	customer := "ent_latest_customer"
	var featureId string
	var entitlementId string

	CreateCustomerWithSubject(t, client, customer, subject)

	apiMONTH := &api.RecurringPeriodInterval{}
	require.NoError(t, apiMONTH.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

	t.Run("Create Meter with LATEST aggregation", func(t *testing.T) {
		resp, err := client.CreateMeterWithResponse(ctx, api.MeterCreate{
			Slug:          meterSlug,
			Name:          convert.ToPointer("Latest Aggregation Meter"),
			Description:   convert.ToPointer("Meter for testing LATEST aggregation type"),
			Aggregation:   api.MeterAggregationLatest,
			EventType:     "latest_event",
			ValueProperty: convert.ToPointer("$.value"),
		})

		require.NoError(t, err)
		// meter API returns 200 instead of 201
		require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
	})

	t.Run("Create Feature", func(t *testing.T) {
		randKey := fmt.Sprintf("entitlement_latest_test_feature_%d", time.Now().Unix())
		resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Name:      "Entitlement Latest Test Feature",
			MeterSlug: convert.ToPointer(meterSlug),
			Key:       randKey,
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		featureId = resp.JSON201.Id
	})

	t.Run("Create Entitlement", func(t *testing.T) {
		mu := api.MeasureUsageFrom{}
		require.NoError(t, mu.FromMeasureUsageFromTime(time.Now().Add(-10*time.Minute).Truncate(time.Minute)))

		meteredEntitlement := api.EntitlementMeteredCreateInputs{
			Type:      "metered",
			FeatureId: &featureId,
			UsagePeriod: api.RecurringPeriodCreateInput{
				Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
				Interval: *apiMONTH,
			},
			MeasureUsageFrom: &mu,
		}
		body := &api.CreateEntitlementJSONRequestBody{}
		err := body.FromEntitlementMeteredCreateInputs(meteredEntitlement)
		require.NoError(t, err)
		resp, err := client.CreateEntitlementWithResponse(ctx, subject, *body)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		metered, err := resp.JSON201.AsEntitlementMetered()
		require.NoError(t, err)

		require.Equal(t, metered.SubjectKey, subject)
		entitlementId = metered.Id
	})

	grantAmount := 1000.0
	t.Run("Create Grant", func(t *testing.T) {
		effectiveAt := time.Now().Add(-9 * time.Minute).Truncate(time.Minute)

		priority := uint8(1)

		// Create grant
		resp, err := client.CreateGrantWithResponse(ctx, subject, entitlementId, api.EntitlementGrantCreateInput{
			Amount:      grantAmount,
			EffectiveAt: effectiveAt,
			Expiration: api.ExpirationPeriod{
				Duration: "MONTH",
				Count:    1,
			},
			Priority: &priority,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", resp.Body)
	})

	t.Run("Report usage events with different values", func(t *testing.T) {
		now := time.Now()

		// Send multiple events with different values - LATEST should use the most recent value
		events := []struct {
			value     float64
			timestamp time.Time
		}{
			{100.0, now.Add(-5 * time.Minute)}, // Earlier event
			{200.0, now.Add(-3 * time.Minute)}, // Middle event
			{150.0, now.Add(-1 * time.Minute)}, // Latest event (this should be used)
		}

		for i, event := range events {
			ev := cloudevents.New()
			ev.SetID(gofakeit.UUID())
			ev.SetSource("my-app")
			ev.SetType("latest_event")
			ev.SetSubject(subject)
			ev.SetTime(event.timestamp)
			_ = ev.SetData("application/json", map[string]interface{}{
				"value": event.value,
			})

			resp, err := client.IngestEventWithResponse(ctx, ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode(), "Failed to ingest event %d", i)
		}

		// Wait for events to be processed
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.QueryMeterWithResponse(ctx, meterSlug, &api.QueryMeterParams{
				To: convert.ToPointer(time.Now().Truncate(time.Minute)),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			require.Len(t, resp.JSON200.Data, 1)
			// For LATEST aggregation, should return the most recent value (150.0)
			assert.Equal(t, 150.0, resp.JSON200.Data[0].Value)
		}, 2*time.Minute, time.Second)
	})

	t.Run("Should calculate balance correctly with LATEST aggregation", func(t *testing.T) {
		resp, err := client.GetEntitlementValueWithResponse(ctx, subject, entitlementId, &api.GetEntitlementValueParams{
			Time: convert.ToPointer(time.Now()),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.NotNil(t, resp.JSON200.Balance, "got body %s", string(resp.Body))
		require.NotNil(t, resp.JSON200.Usage, "got body %s", string(resp.Body))

		// Balance should be grant amount minus the latest usage value (150.0)
		assert.Equal(t, grantAmount-150.0, *resp.JSON200.Balance)
		assert.Equal(t, 150.0, *resp.JSON200.Usage)
	})

	t.Run("Should handle new latest value correctly", func(t *testing.T) {
		if !shouldRunSlowTests(t) {
			t.Skip("Skipping slow test, please reenable when we have a second resolution for entitlements")
		}

		// Send a new event with a different value
		newLatestValue := 300.0

		ev := cloudevents.New()
		ev.SetID(gofakeit.UUID())
		ev.SetSource("my-app")
		ev.SetType("latest_event")
		ev.SetSubject(subject)
		ev.SetTime(time.Now())
		_ = ev.SetData("application/json", map[string]interface{}{
			"value": newLatestValue,
		})

		resp, err := client.IngestEventWithResponse(ctx, ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode())

		// Wait for event to be processed
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.GetEntitlementValueWithResponse(ctx, subject, entitlementId, &api.GetEntitlementValueParams{
				Time: convert.ToPointer(time.Now().Truncate(time.Minute)),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			require.NotNil(t, resp.JSON200.Balance)
			require.NotNil(t, resp.JSON200.Usage)

			// Balance should now reflect the new latest value
			assert.Equal(t, grantAmount-newLatestValue, *resp.JSON200.Balance)
			assert.Equal(t, newLatestValue, *resp.JSON200.Usage)
		}, 2*time.Minute, time.Second)
	})
}

func TestEntitlementStaticConfigEncoding(t *testing.T) {
	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subject := "ent_static_config_customer"
	customer := "ent_static_config_customer"
	var customerID string
	var featureId string
	var entitlementId string

	c := CreateCustomerWithSubject(t, client, customer, subject)
	customerID = c.Id

	t.Run("Create Feature", func(t *testing.T) {
		randKey := fmt.Sprintf("entitlement_static_config_feature_%d", time.Now().Unix())
		resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Name: "Entitlement Static Config Feature",
			Key:  randKey,
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		featureId = resp.JSON201.Id
	})

	expectedConfigJSON := `{"integrations":["github","gitlab"],"maxUsers":10}`
	// Config needs to be a JSON-encoded string (not raw JSON object)
	// because the API expects: {"config": "{...}"} not {"config": {...}}
	configAsJSONString, err := json.Marshal(expectedConfigJSON)
	require.NoError(t, err)

	t.Run("Create Static Entitlement with Config (V1)", func(t *testing.T) {
		staticEntitlement := api.EntitlementStaticCreateInputs{
			Type:      "static",
			FeatureId: &featureId,
			Config:    configAsJSONString,
		}
		body := &api.CreateEntitlementJSONRequestBody{}
		err := body.FromEntitlementStaticCreateInputs(staticEntitlement)
		require.NoError(t, err)
		resp, err := client.CreateEntitlementWithResponse(ctx, subject, *body)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		static, err := resp.JSON201.AsEntitlementStatic()
		require.NoError(t, err)
		entitlementId = static.Id

		// The config comes as json.RawMessage which is a JSON-encoded string
		// Unmarshal it to get the actual JSON content
		var configStr string
		require.NoError(t, json.Unmarshal(static.Config, &configStr))
		require.JSONEq(t, expectedConfigJSON, configStr, "Config should be valid JSON, not double-encoded")
	})

	t.Run("Get Entitlement by ID and Check Config (V1)", func(t *testing.T) {
		resp, err := client.GetEntitlementByIdWithResponse(ctx, entitlementId)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		static, err := resp.JSON200.AsEntitlementStatic()
		require.NoError(t, err)

		// Verify config is not double-encoded when retrieved
		var configStr string
		require.NoError(t, json.Unmarshal(static.Config, &configStr))
		require.JSONEq(t, expectedConfigJSON, configStr, "Retrieved config should match and not be double-encoded")
	})

	t.Run("Get Entitlement Value and Check Config (V1)", func(t *testing.T) {
		resp, err := client.GetEntitlementValueWithResponse(ctx, subject, entitlementId, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		if resp.JSON200.Config != nil {
			// The config string should be valid JSON
			require.JSONEq(t, expectedConfigJSON, *resp.JSON200.Config, "EntitlementValue config should not be double-encoded")
		} else {
			t.Fatal("Config should not be nil in entitlement value")
		}
	})

	// Test V2 API
	t.Run("Create Static Entitlement with Config (V2)", func(t *testing.T) {
		// Create a new feature for V2 test
		var v2FeatureId string
		{
			randKey := fmt.Sprintf("entitlement_static_config_v2_feature_%d", time.Now().Unix())
			resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
				Name: "Entitlement Static Config V2 Feature",
				Key:  randKey,
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode())
			v2FeatureId = resp.JSON201.Id
		}

		// Create static entitlement via V2 API using V1 static create inputs
		staticEntitlement := api.EntitlementStaticCreateInputs{
			Type:      "static",
			FeatureId: &v2FeatureId,
			Config:    configAsJSONString,
		}
		var body api.CreateCustomerEntitlementV2JSONRequestBody
		require.NoError(t, body.FromEntitlementStaticCreateInputs(staticEntitlement))

		resp, err := client.CreateCustomerEntitlementV2WithResponse(ctx, customer, body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		v2Static, err := resp.JSON201.AsEntitlementStaticV2()
		require.NoError(t, err)

		// The config should be a valid JSON string, not double-encoded
		var configStr string
		require.NoError(t, json.Unmarshal(v2Static.Config, &configStr))
		require.JSONEq(t, expectedConfigJSON, configStr, "V2 Config should be valid JSON, not double-encoded")
	})

	// Test V3 API (Entitlement Access)
	var v3FeatureKey string
	t.Run("Create Feature for V3", func(t *testing.T) {
		v3FeatureKey = fmt.Sprintf("entitlement_static_config_v3_feature_%d", time.Now().Unix())
		resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Name: "Entitlement Static Config V3 Feature",
			Key:  v3FeatureKey,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode())
	})

	t.Run("Create Static Entitlement for V3", func(t *testing.T) {
		staticEntitlement := api.EntitlementStaticCreateInputs{
			Type:       "static",
			FeatureKey: &v3FeatureKey,
			Config:     configAsJSONString,
		}
		body := &api.CreateEntitlementJSONRequestBody{}
		require.NoError(t, body.FromEntitlementStaticCreateInputs(staticEntitlement))
		resp, err := client.CreateEntitlementWithResponse(ctx, subject, *body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode())
	})

	// Raw HTTP tests to verify actual format (independent of Go client)
	baseURL := getBaseURL(t)
	httpClient := &http.Client{}

	t.Run("Raw HTTP: Get Entitlement by ID (V1)", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/entitlements/%s", baseURL, entitlementId), nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		// Verify config is a string in the JSON response
		configValue, ok := result["config"]
		require.True(t, ok, "config field should exist")
		configStr, ok := configValue.(string)
		require.True(t, ok, "config should be a JSON string, not an object")
		require.JSONEq(t, expectedConfigJSON, configStr, "Raw HTTP: config should not be double-encoded")
	})

	t.Run("Raw HTTP: Get Entitlement Value (V1)", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/subjects/%s/entitlements/%s/value", baseURL, subject, entitlementId), nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		// Verify config is a string in the JSON response
		configValue, ok := result["config"]
		require.True(t, ok, "config field should exist")
		configStr, ok := configValue.(string)
		require.True(t, ok, "config should be a JSON string, not an object")
		require.JSONEq(t, expectedConfigJSON, configStr, "Raw HTTP: EntitlementValue config should not be double-encoded")
	})

	t.Run("Raw HTTP: List Customer Entitlements (V2)", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v2/customers/%s/entitlements", baseURL, customer), nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		items, ok := result["items"].([]interface{})
		require.True(t, ok, "items should be an array")
		require.NotEmpty(t, items, "should have at least one entitlement")

		// Find a static entitlement with config
		var foundConfig bool
		for _, item := range items {
			entitlement, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if entitlement["type"] == "static" {
				if configValue, ok := entitlement["config"]; ok {
					configStr, ok := configValue.(string)
					require.True(t, ok, "V2 config should be a JSON string, not an object")
					require.JSONEq(t, expectedConfigJSON, configStr, "Raw HTTP V2: config should not be double-encoded")
					foundConfig = true
					break
				}
			}
		}
		require.True(t, foundConfig, "should find a static entitlement with config")
	})

	t.Run("Raw HTTP: Check Entitlement Access (V3)", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v3/openmeter/customers/%s/entitlement-access", baseURL, customerID), nil)
		require.NoError(t, err)

		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "V3 endpoint should be available")

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		data, ok := result["data"].([]interface{})
		require.True(t, ok, "data should be an array")
		require.NotEmpty(t, data, "should have at least one access result")

		// Find the static entitlement we created
		var foundConfig bool
		for _, item := range data {
			access, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			// Check if this is our v3 feature
			if fk, ok := access["feature_key"].(string); ok && fk == v3FeatureKey {
				configValue, ok := access["config"]
				require.True(t, ok, "config field should exist for static entitlement")
				configStr, ok := configValue.(string)
				require.True(t, ok, "V3 config should be a JSON string, not an object")
				require.JSONEq(t, expectedConfigJSON, configStr, "Raw HTTP V3: config should not be double-encoded")
				foundConfig = true
				break
			}
		}
		require.True(t, foundConfig, "should find the v3 static entitlement with config")
	})
}

// getBaseURL returns the base URL for raw HTTP requests
func getBaseURL(t *testing.T) string {
	t.Helper()
	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}
	return address
}
