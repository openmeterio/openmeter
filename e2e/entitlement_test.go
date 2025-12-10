package e2e

import (
	"context"
	"fmt"
	"math"
	"net/http"
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
