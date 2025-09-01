package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/pkg/convert"
)

// TestEntitlementParitySuite validates that subject-based APIs (v1) and customer-based APIs (v2)
// provide equivalent functionality. We go step-by-step to keep changes small and focused.
func TestEntitlementParitySuite(t *testing.T) {
	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	meterSlug := "entitlement_parity_meter"

	// Test data
	customerKey := fmt.Sprintf("parity_cust_%d", time.Now().Unix())
	subjectKey := customerKey + "-subject"

	// Create customer and subject mapping
	cust := CreateCustomerWithSubject(t, client, customerKey, subjectKey)
	customerID := cust.Id

	// Create two features to use across both flows
	var feature1ID string
	{
		randKey := fmt.Sprintf("entitlement_parity_feature_1_%d", time.Now().Unix())
		resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Name:      "Entitlement Parity Feature",
			MeterSlug: convert.ToPointer(meterSlug),
			Key:       randKey,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
		feature1ID = resp.JSON201.Id
	}

	var feature2ID string
	{
		randKey := fmt.Sprintf("entitlement_parity_feature_2_%d", time.Now().Unix())
		resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Name:      "Entitlement Parity Feature",
			MeterSlug: convert.ToPointer(meterSlug),
			Key:       randKey,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
		feature2ID = resp.JSON201.Id
	}

	// Common usage period for both requests
	month := &api.RecurringPeriodInterval{}
	require.NoError(t, month.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))
	anchor := convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC))

	t.Run("Create Entitlement parity (subject vs customer)", func(t *testing.T) {
		// Subject-based create (v1)
		var subjectEntitlementID string
		{
			metered := api.EntitlementMeteredCreateInputs{
				Type:      "metered",
				FeatureId: &feature1ID,
				UsagePeriod: api.RecurringPeriodCreateInput{
					Anchor:   anchor,
					Interval: *month,
				},
			}
			body := &api.CreateEntitlementJSONRequestBody{}
			require.NoError(t, body.FromEntitlementMeteredCreateInputs(metered))

			resp, err := client.CreateEntitlementWithResponse(ctx, subjectKey, *body)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

			m, err := resp.JSON201.AsEntitlementMetered()
			require.NoError(t, err)
			require.Equal(t, subjectKey, m.SubjectKey)
			require.Equal(t, feature1ID, m.FeatureId)
			subjectEntitlementID = m.Id
		}

		// Customer-based create (v2)
		{
			metered := api.EntitlementMeteredCreateInputs{
				Type:      "metered",
				FeatureId: &feature2ID,
				UsagePeriod: api.RecurringPeriodCreateInput{
					Anchor:   anchor,
					Interval: *month,
				},
			}
			var body api.CreateCustomerEntitlementV2JSONRequestBody
			require.NoError(t, body.FromEntitlementMeteredCreateInputs(metered))

			resp, err := client.CreateCustomerEntitlementV2WithResponse(ctx, customerID, body)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

			v2, err := resp.JSON201.AsEntitlementMeteredV2()
			require.NoError(t, err)
			require.Equal(t, feature2ID, v2.FeatureId)
			require.Equal(t, customerID, v2.CustomerId)

			// Basic shape parity: same feature and usage period anchor
			require.NotNil(t, v2.UsagePeriod)
			require.Equal(t, anchor.Format(time.RFC3339), v2.UsagePeriod.Anchor.Format(time.RFC3339))

			_ = subjectEntitlementID // reserved for later parity checks (e.g., value/history)
		}
	})

	t.Run("List Entitlements parity (subject vs customer)", func(t *testing.T) {
		// Subject list (v1)
		var subjectHas bool
		{
			resp, err := client.ListSubjectEntitlementsWithResponse(ctx, subjectKey, &api.ListSubjectEntitlementsParams{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			require.NotNil(t, resp.JSON200)

			for _, item := range *resp.JSON200 {
				if m, err := item.AsEntitlementMetered(); err == nil {
					if m.FeatureId == feature1ID {
						subjectHas = true
						break
					}
				}
			}
		}

		// Customer list (v2)
		var customerHas bool
		{
			resp, err := client.ListCustomerEntitlementsV2WithResponse(ctx, customerID, &api.ListCustomerEntitlementsV2Params{
				Page:     lo.ToPtr(1),
				PageSize: lo.ToPtr(100),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			require.NotNil(t, resp.JSON200)

			for _, item := range resp.JSON200.Items {
				if v2, err := item.AsEntitlementMeteredV2(); err == nil {
					if v2.FeatureId == feature2ID {
						customerHas = true
						break
					}
				}
			}
		}

		assert.True(t, subjectHas, "subject entitlement list should include feature")
		assert.True(t, customerHas, "customer entitlement list should include feature")
	})
}
