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
	var feature1Key string
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
		feature1Key = randKey
	}

	var feature2ID string
	var feature2Key string
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
		feature2Key = randKey
	}

	// Common usage period for both requests
	month := &api.RecurringPeriodInterval{}
	require.NoError(t, month.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))
	anchor := convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC))

	var subjectEntitlementID string
	var customerEntitlementFeatureKey string

	t.Run("Create Entitlement parity (subject vs customer)", func(t *testing.T) {
		// Subject-based create (v1)
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
			metered := api.EntitlementMeteredV2CreateInputs{
				Type:      "metered",
				FeatureId: &feature2ID,
				UsagePeriod: api.RecurringPeriodCreateInput{
					Anchor:   anchor,
					Interval: *month,
				},
			}
			var body api.CreateCustomerEntitlementV2JSONRequestBody
			require.NoError(t, body.FromEntitlementMeteredV2CreateInputs(metered))

			resp, err := client.CreateCustomerEntitlementV2WithResponse(ctx, customerID, body)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

			v2, err := resp.JSON201.AsEntitlementMeteredV2()
			require.NoError(t, err)
			require.Equal(t, feature2ID, v2.FeatureId)
			require.Equal(t, customerID, v2.CustomerId)

			customerEntitlementFeatureKey = v2.FeatureKey

			// Basic shape parity: same feature and usage period anchor
			require.NotNil(t, v2.UsagePeriod)
			require.Equal(t, anchor.Format(time.RFC3339), v2.UsagePeriod.Anchor.Format(time.RFC3339))
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

	t.Run("Grants parity (create and list)", func(t *testing.T) {
		grantAmount := 50.0
		effectiveAt := time.Now().Truncate(time.Minute)

		// v1: create grant for subject entitlement
		{
			resp, err := client.CreateGrantWithResponse(ctx, subjectKey, subjectEntitlementID, api.EntitlementGrantCreateInput{
				Amount:      grantAmount,
				EffectiveAt: effectiveAt,
				Expiration:  api.ExpirationPeriod{Duration: "MONTH", Count: 1},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
		}

		// v2: create grant for customer entitlement via feature key
		{
			resp, err := client.CreateCustomerEntitlementGrantV2WithResponse(ctx, customerID, customerEntitlementFeatureKey, api.CreateCustomerEntitlementGrantV2JSONRequestBody{
				Amount:      grantAmount,
				EffectiveAt: effectiveAt,
				Expiration:  &api.ExpirationPeriod{Duration: "MONTH", Count: 1},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
		}

		// Cross-API: create grant for v1 entitlement using v2 API (by feature key)
		{
			resp, err := client.CreateCustomerEntitlementGrantV2WithResponse(ctx, customerID, feature1Key, api.CreateCustomerEntitlementGrantV2JSONRequestBody{
				Amount:      grantAmount,
				EffectiveAt: effectiveAt,
				Expiration:  &api.ExpirationPeriod{Duration: "MONTH", Count: 1},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
		}

		// Cross-API: create grant for v2 entitlement using v1 API (by feature key)
		{
			resp, err := client.CreateGrantWithResponse(ctx, subjectKey, feature2Key, api.EntitlementGrantCreateInput{
				Amount:      grantAmount,
				EffectiveAt: effectiveAt,
				Expiration:  api.ExpirationPeriod{Duration: "MONTH", Count: 1},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
		}

		// v1 list grants for subject entitlement (by ID)
		var v1Grants []api.EntitlementGrant
		{
			resp, err := client.ListEntitlementGrantsWithResponse(ctx, subjectKey, subjectEntitlementID, &api.ListEntitlementGrantsParams{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			require.NotNil(t, resp.JSON200)
			v1Grants = *resp.JSON200
		}

		// v2 list grants for subject entitlement (by feature key)
		var v2GrantsForV1Entitlement api.GrantV2PaginatedResponse
		{
			resp, err := client.ListCustomerEntitlementGrantsV2WithResponse(ctx, customerID, feature1Key, &api.ListCustomerEntitlementGrantsV2Params{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			require.NotNil(t, resp.JSON200)
			v2GrantsForV1Entitlement = *resp.JSON200
		}

		// v2 list grants for v2 entitlement (by feature key)
		var v2Grants api.GrantV2PaginatedResponse
		{
			resp, err := client.ListCustomerEntitlementGrantsV2WithResponse(ctx, customerID, customerEntitlementFeatureKey, &api.ListCustomerEntitlementGrantsV2Params{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			require.NotNil(t, resp.JSON200)
			v2Grants = *resp.JSON200
		}

		// v1 list grants for v2 entitlement (by feature key)
		var v1GrantsForV2Entitlement []api.EntitlementGrant
		{
			resp, err := client.ListEntitlementGrantsWithResponse(ctx, subjectKey, feature2Key, &api.ListEntitlementGrantsParams{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
			require.NotNil(t, resp.JSON200)
			v1GrantsForV2Entitlement = *resp.JSON200
		}

		// Parity assertions: both lists contain at least one grant in all views
		assert.GreaterOrEqual(t, len(v1Grants), 1)
		assert.GreaterOrEqual(t, v2Grants.TotalCount, 1)
		assert.GreaterOrEqual(t, v2GrantsForV1Entitlement.TotalCount, 1)
		assert.GreaterOrEqual(t, len(v1GrantsForV2Entitlement), 1)
	})

	t.Run("Usage parity (create and list)", func(t *testing.T) {
		// This can take up to a minute which might be considered slow
		if !shouldRunSlowTests(t) {
			t.Skip("Skipping slow test, please reenable when we have a second resolution for entitlements")
		}
		// Report usage for parity meter to validate values
		t.Run("Report usage (parity meter)", func(t *testing.T) {
			now := time.Now()

			uniqueEventCount := 10
			for i := 0; i < uniqueEventCount*2; i++ {
				timestamp := gofakeit.DateRange(now, now.Add(-5*time.Second))
				value := fmt.Sprintf("%v", math.Floor(float64(i)/2))

				ev := cloudevents.New()
				ev.SetID(gofakeit.UUID())
				ev.SetSource("my-app")
				ev.SetType("entitlement_parity")
				ev.SetSubject(subjectKey)
				ev.SetTime(timestamp)
				_ = ev.SetData("application/json", map[string]string{
					"value": value,
				})

				resp, err := client.IngestEventWithResponse(ctx, ev)
				require.NoError(t, err)
				require.Equal(t, http.StatusNoContent, resp.StatusCode())
			}

			// NOTE: This is a temporary workaround to avoid data race condition in (assert|require).EventuallyWithTf function which
			// was not triggered before `testify` v1.11.1. Which includes the change (https://github.com/stretchr/testify/pull/1427)
			// that makes the EventuallyWithTf return early triggering the race condition in our case where the result of the condition
			// from the previous tick is used to decide the result of the test.
			// Remove this when the race condition is fixed in `testify`.
			// Bare in mind his still can fail if sink-worker is is not running or the ingestion of the events takes longer than the sleep time.
			time.Sleep(5 * time.Second)

			// Wait for events to be processed
			assert.EventuallyWithT(t, func(t *assert.CollectT) {
				resp, err := client.QueryMeterWithResponse(ctx, meterSlug, &api.QueryMeterParams{
					To: convert.ToPointer(time.Now().Truncate(time.Minute)),
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode())
				require.Len(t, resp.JSON200.Data, 1)
				assert.Equal(t, float64(uniqueEventCount), resp.JSON200.Data[0].Value)
				// We have to assert for more than a minute as entitlement value checks have 60 second granularity
			}, 62*time.Second, time.Second)
		})

		// Value parity (cross-API)
		t.Run("Value parity (v1 vs v2, cross-api)", func(t *testing.T) {
			now := time.Now().Truncate(time.Minute)

			// v1 value for subject entitlement by ID
			var v1ValueForSubjectByID float64
			{
				resp, err := client.GetEntitlementValueWithResponse(ctx, subjectKey, subjectEntitlementID, &api.GetEntitlementValueParams{Time: &now})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
				require.NotNil(t, resp.JSON200.Balance)
				v1ValueForSubjectByID = *resp.JSON200.Balance
			}

			// v1 value for v2 entitlement (by feature key)
			var v1ValueForV2ByKey float64
			{
				resp, err := client.GetEntitlementValueWithResponse(ctx, subjectKey, feature2Key, &api.GetEntitlementValueParams{Time: &now})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
				require.NotNil(t, resp.JSON200.Balance)
				v1ValueForV2ByKey = *resp.JSON200.Balance
			}

			// v2 value for subject entitlement (by feature key)
			var v2ValueForV1ByKey float64
			{
				resp, err := client.GetCustomerEntitlementValueV2WithResponse(ctx, customerID, feature1Key, &api.GetCustomerEntitlementValueV2Params{Time: &now})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
				require.NotNil(t, resp.JSON200.Balance)
				v2ValueForV1ByKey = *resp.JSON200.Balance
			}

			// v2 value for customer entitlement (by feature key)
			var v2ValueForV2ByKey float64
			{
				resp, err := client.GetCustomerEntitlementValueV2WithResponse(ctx, customerID, customerEntitlementFeatureKey, &api.GetCustomerEntitlementValueV2Params{Time: &now})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
				require.NotNil(t, resp.JSON200.Balance)
				v2ValueForV2ByKey = *resp.JSON200.Balance
			}

			// Parity: ensure that cross-API reads are consistent (we don't assert absolute numbers here, just equality across views)
			assert.Equal(t, v1ValueForSubjectByID, v2ValueForV1ByKey, "v1(ID) vs v2(by feature1Key) should match")
			assert.Equal(t, v1ValueForV2ByKey, v2ValueForV2ByKey, "v1(by feature2Key) vs v2(by feature2Key) should match")
		})
	})

	t.Run("Annotations and metadata parity (create and get)", func(t *testing.T) {
		t.Run("Annotations created with V2 API should show up in V1 API", func(t *testing.T) {
			createGrantResponse, err := client.CreateCustomerEntitlementGrantV2WithResponse(ctx, customerID, feature1Key, api.CreateCustomerEntitlementGrantV2JSONRequestBody{
				Amount:      100,
				EffectiveAt: time.Now().Truncate(time.Minute).Add(time.Minute),
				Expiration:  nil,
				Annotations: &api.Annotations{
					"some_annotation": "some_annotation_value",
				},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, createGrantResponse.StatusCode(), "Invalid status code [response_body=%s]", string(createGrantResponse.Body))

			getGrantResponse, err := client.ListEntitlementGrantsWithResponse(ctx, subjectKey, feature1Key, &api.ListEntitlementGrantsParams{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, getGrantResponse.StatusCode(), "Invalid status code [response_body=%s]", string(getGrantResponse.Body))
			require.NotNil(t, getGrantResponse.JSON200)
			require.GreaterOrEqual(t, len(lo.FromPtr(getGrantResponse.JSON200)), 1, "Invalid number of grants [response_body=%s]", string(getGrantResponse.Body))

			var found *api.EntitlementGrant

			for _, grant := range lo.FromPtr(getGrantResponse.JSON200) {
				if grant.Id == createGrantResponse.JSON201.Id {
					found = &grant
					break
				}
			}

			require.NotNil(t, found, "Grant not found [response_body=%s]", string(getGrantResponse.Body))
			require.Equal(t, "some_annotation_value", lo.FromPtr(found.Annotations)["some_annotation"])
		})
	})
}

func TestEntitlementDifferences(t *testing.T) {
	client := initClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	meterSlug := "entitlement_parity_meter"

	// Test data
	customerKey := fmt.Sprintf("parity_cust_%d_2", time.Now().Unix())
	subjectKey := customerKey + "-subject"

	// Create customer and subject mapping
	cust := CreateCustomerWithSubject(t, client, customerKey, subjectKey)
	customerID := cust.Id

	// Create a feature to use across both flows
	var featureID string
	var feature1Key string
	{
		randKey := fmt.Sprintf("entitlement_parity_feature_1_%d_2", time.Now().Unix())
		resp, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Name:      "Entitlement Parity Feature",
			MeterSlug: convert.ToPointer(meterSlug),
			Key:       randKey,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))
		featureID = resp.JSON201.Id
		feature1Key = randKey
	}

	// Usage period for requests
	month := &api.RecurringPeriodInterval{}
	require.NoError(t, month.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))
	anchor := convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC))

	t.Run("New API should be able to create grants without expiration", func(t *testing.T) {
		t.Run("Should create entitlement through V2 API with default grants", func(t *testing.T) {
			metered := api.EntitlementMeteredV2CreateInputs{
				Type:      "metered",
				FeatureId: &featureID,
				UsagePeriod: api.RecurringPeriodCreateInput{
					Anchor:   anchor,
					Interval: *month,
				},
				Grants: &[]api.EntitlementGrantCreateInputV2{
					{
						Amount:      100,
						EffectiveAt: time.Now().Truncate(time.Minute).Add(time.Minute),
						Expiration:  nil,
					},
				},
			}
			var body api.CreateCustomerEntitlementV2JSONRequestBody
			require.NoError(t, body.FromEntitlementMeteredV2CreateInputs(metered))

			// Let's create an entitlement with a grant
			entRes, err := client.CreateCustomerEntitlementV2WithResponse(ctx, customerID, body)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, entRes.StatusCode(), "Invalid status code [response_body=%s]", string(entRes.Body))

			v2EntGrants, err := client.ListCustomerEntitlementGrantsV2WithResponse(ctx, customerID, feature1Key, &api.ListCustomerEntitlementGrantsV2Params{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, v2EntGrants.StatusCode(), "Invalid status code [response_body=%s]", string(v2EntGrants.Body))

			require.NotNil(t, v2EntGrants.JSON200)
			require.GreaterOrEqual(t, len(v2EntGrants.JSON200.Items), 1, "Invalid number of grants [response_body=%s]", string(v2EntGrants.Body))

			for _, grant := range v2EntGrants.JSON200.Items {
				require.Nil(t, grant.Expiration)
			}
		})

		t.Run("Old API should still be able to list grants without expiration with filled-in dummy values", func(t *testing.T) {
			v1EntGrants, err := client.ListEntitlementGrantsWithResponse(ctx, subjectKey, feature1Key, &api.ListEntitlementGrantsParams{})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, v1EntGrants.StatusCode(), "Invalid status code [response_body=%s]", string(v1EntGrants.Body))
			require.NotNil(t, v1EntGrants.JSON200)

			require.GreaterOrEqual(t, len(lo.FromPtr(v1EntGrants.JSON200)), 1, "Invalid number of grants [response_body=%s]", string(v1EntGrants.Body))
			for _, grant := range lo.FromPtr(v1EntGrants.JSON200) {
				require.NotNil(t, grant.Expiration)
			}
		})
	})
}
