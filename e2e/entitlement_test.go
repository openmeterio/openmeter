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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func TestEntitlementWithUniqueCountAggregation(t *testing.T) {
	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	meterSlug := "entitlement_uc_meter"
	subject := "ent_customer"
	var featureId string
	var entitlementId string

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

		featureId = *resp.JSON201.Id
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
		entitlementId = *metered.Id
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
		}, time.Minute, time.Second)
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
		subject := "ent_customer_2"
		var featureId string
		var entitlementId string

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

			featureId = *resp.JSON201.Id
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
			entitlementId = *metered.Id
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
