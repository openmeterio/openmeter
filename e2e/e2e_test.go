package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
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

func TestMain(m *testing.M) {
	wait := os.Getenv("TEST_WAIT_ON_START")

	if b, err := strconv.ParseBool(wait); err == nil && b {
		// Make sure OpenMeter is ready
		// TODO: replace this with some sort of health check
		time.Sleep(15 * time.Second)
	}

	os.Exit(m.Run())
}

func TestIngest(t *testing.T) {
	client := initClient(t)

	// Make clickhouse's job easier by sending events within a fix time range
	now := time.Now()

	var sum int

	for i := 0; i < 1000; i++ {
		// Make clickhouse's job easier by sending events within a fix time range
		timestamp := gofakeit.DateRange(now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour))
		duration := gofakeit.Number(1, 100)

		sum += duration

		ev := cloudevents.New()
		ev.SetID(gofakeit.UUID())
		ev.SetSource("my-app")
		ev.SetType("ingest")
		ev.SetSubject("customer-1")
		ev.SetTime(timestamp)
		_ = ev.SetData("application/json", map[string]string{
			"duration_ms": fmt.Sprintf("%d", duration),
		})

		resp, err := client.IngestEventWithResponse(context.Background(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode())
	}

	// Wait for events to be processed
	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		resp, err := client.QueryMeterWithResponse(context.Background(), "ingest", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.Len(t, resp.JSON200.Data, 1)
		assert.Equal(t, float64(sum), resp.JSON200.Data[0].Value)
	}, time.Minute, time.Second)
}

func TestBatchIngest(t *testing.T) {
	client := initClient(t)

	// Make clickhouse's job easier by sending events within a fix time range
	now := time.Now()

	var sum int

	var events []cloudevents.Event

	for i := 0; i < 1000; i++ {
		// Make clickhouse's job easier by sending events within a fix time range
		timestamp := gofakeit.DateRange(now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour))
		duration := gofakeit.Number(1, 1000)

		sum += duration

		ev := cloudevents.New()
		ev.SetID(gofakeit.UUID())
		ev.SetSource("my-app")
		ev.SetType("batchingest")
		ev.SetSubject("customer-1")
		ev.SetTime(timestamp)
		_ = ev.SetData("application/json", map[string]string{
			"duration_ms": fmt.Sprintf("%d", duration),
		})

		events = append(events, ev)
	}

	resp, err := client.IngestEventBatchWithResponse(context.Background(), events)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())

	// Wait for events to be processed
	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		resp, err := client.QueryMeterWithResponse(context.Background(), "batchingest", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.Len(t, resp.JSON200.Data, 1)
		assert.Equal(t, float64(sum), resp.JSON200.Data[0].Value)
	}, time.Minute, time.Second)
}

func TestDedupe(t *testing.T) {
	client := initClient(t)

	// Make clickhouse's job easier by sending events within a fix time range
	now := time.Now()

	for i := 0; i < 1000; i++ {
		// Make clickhouse's job easier by sending events within a fix time range
		timestamp := gofakeit.DateRange(now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour))

		ev := cloudevents.New()
		ev.SetID("52f44f66-020f-4fa9-a733-102a8ef6f515")
		ev.SetSource("my-app")
		ev.SetType("dedupe")
		ev.SetSubject("customer-1")
		ev.SetTime(timestamp)
		_ = ev.SetData("application/json", map[string]string{})

		resp, err := client.IngestEventWithResponse(context.Background(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode())
	}

	// Wait for events to be processed
	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		resp, err := client.QueryMeterWithResponse(context.Background(), "dedupe", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.Len(t, resp.JSON200.Data, 1)
		assert.Equal(t, 1.0, resp.JSON200.Data[0].Value)
	}, time.Minute, time.Second)
}

func TestQuery(t *testing.T) {
	client := initClient(t)

	// Reproducible random data
	const customerCount = 5
	paths := []string{"/", "/about", "/users", "/contact"}
	faker := gofakeit.New(8675309)
	timestamp := faker.DateRange(time.Date(2023, time.May, 6, 0, 0, 0, 0, time.UTC), faker.FutureDate().UTC()).UTC().Truncate(time.Second)

	t.Run("Total", func(t *testing.T) {
		var events []cloudevents.Event

		newEvents := func(fn func(ev *cloudevents.Event)) []cloudevents.Event {
			var events []cloudevents.Event

			for i := 0; i < customerCount; i++ {
				ev := cloudevents.New()

				ev.SetID(faker.UUID())
				ev.SetSource("my-app")
				ev.SetType("query")
				ev.SetSubject(fmt.Sprintf("customer-%d", i+1))

				fn(&ev)

				events = append(events, ev)
			}

			return events
		}

		newTimedEvents := func(timestamp time.Time) []cloudevents.Event {
			method := faker.HTTPMethod()
			path := paths[faker.Number(0, len(paths)-1)]

			return newEvents(func(ev *cloudevents.Event) {
				ev.SetTime(timestamp)
				_ = ev.SetData("application/json", map[string]string{
					"duration_ms": "100",
					"method":      method,
					"path":        path,
				})
			})
		}

		// First event
		{
			events = append(events, newTimedEvents(timestamp)...)
		}

		// Plus one minute
		{
			events = append(events, newTimedEvents(timestamp.Add(time.Minute))...)
		}

		// Plus one hour
		{
			events = append(events, newTimedEvents(timestamp.Add(time.Hour))...)
		}

		// Plus one day
		{
			events = append(events, newTimedEvents(timestamp.Add(24*time.Hour))...)
		}

		{
			resp, err := client.IngestEventBatchWithResponse(context.Background(), events)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}

		// Wait for events to be processed
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.QueryMeterWithResponse(context.Background(), "query", nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			expected := &api.MeterQueryResult{
				Data: []models.MeterQueryRow{
					{
						Value:       customerCount * 4 * 100,
						WindowStart: timestamp.Truncate(time.Minute),
						WindowEnd:   timestamp.Add(24 * time.Hour).Truncate(time.Minute).Add(time.Minute),
						GroupBy:     map[string]*string{},
					},
				},
			}

			assert.Equal(t, expected, resp.JSON200)
		}, time.Minute, time.Second)
	})

	t.Run("WindowSize", func(t *testing.T) {
		t.Parallel()

		t.Run("Minute", func(t *testing.T) {
			t.Parallel()

			windowSize := models.WindowSizeMinute

			// Wait for events to be processed
			assert.EventuallyWithT(t, func(t *assert.CollectT) {
				resp, err := client.QueryMeterWithResponse(context.Background(), "query", &api.QueryMeterParams{
					WindowSize: &windowSize,
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode())

				expected := &api.MeterQueryResult{
					Data: []models.MeterQueryRow{
						{
							Value:       customerCount * 100,
							WindowStart: timestamp.Truncate(time.Minute),
							WindowEnd:   timestamp.Truncate(time.Minute).Add(time.Minute),
							GroupBy:     map[string]*string{},
						},
						{
							Value:       customerCount * 100,
							WindowStart: timestamp.Add(time.Minute).Truncate(time.Minute),
							WindowEnd:   timestamp.Add(time.Minute).Truncate(time.Minute).Add(time.Minute),
							GroupBy:     map[string]*string{},
						},
						{
							Value:       customerCount * 100,
							WindowStart: timestamp.Add(time.Hour).Truncate(time.Minute),
							WindowEnd:   timestamp.Add(time.Hour).Truncate(time.Minute).Add(time.Minute),
							GroupBy:     map[string]*string{},
						},
						{
							Value:       customerCount * 100,
							WindowStart: timestamp.Add(24 * time.Hour).Truncate(time.Minute),
							WindowEnd:   timestamp.Add(24 * time.Hour).Truncate(time.Minute).Add(time.Minute),
							GroupBy:     map[string]*string{},
						},
					},
					WindowSize: &windowSize,
				}

				assert.Equal(t, expected, resp.JSON200)
			}, time.Minute, time.Second)
		})

		t.Run("Hour", func(t *testing.T) {
			t.Parallel()

			windowSize := models.WindowSizeHour

			// Wait for events to be processed
			assert.EventuallyWithT(t, func(t *assert.CollectT) {
				resp, err := client.QueryMeterWithResponse(context.Background(), "query", &api.QueryMeterParams{
					WindowSize: &windowSize,
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode())

				expected := &api.MeterQueryResult{
					Data: []models.MeterQueryRow{
						{
							Value:       customerCount * 2 * 100,
							WindowStart: timestamp.Truncate(time.Hour),
							WindowEnd:   timestamp.Truncate(time.Hour).Add(time.Hour),
							GroupBy:     map[string]*string{},
						},
						{
							Value:       customerCount * 100,
							WindowStart: timestamp.Add(time.Hour).Truncate(time.Hour),
							WindowEnd:   timestamp.Add(time.Hour).Truncate(time.Hour).Add(time.Hour),
							GroupBy:     map[string]*string{},
						},
						{
							Value:       customerCount * 100,
							WindowStart: timestamp.Add(24 * time.Hour).Truncate(time.Hour),
							WindowEnd:   timestamp.Add(24 * time.Hour).Truncate(time.Hour).Add(time.Hour),
							GroupBy:     map[string]*string{},
						},
					},
					WindowSize: &windowSize,
				}

				assert.Equal(t, expected, resp.JSON200)
			}, time.Minute, time.Second)
		})

		t.Run("Day", func(t *testing.T) {
			t.Parallel()

			windowSize := models.WindowSizeDay

			// Wait for events to be processed
			assert.EventuallyWithT(t, func(t *assert.CollectT) {
				resp, err := client.QueryMeterWithResponse(context.Background(), "query", &api.QueryMeterParams{
					WindowSize: &windowSize,
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode())

				expected := &api.MeterQueryResult{
					Data: []models.MeterQueryRow{
						{
							Value:       customerCount * 3 * 100,
							WindowStart: timestamp.Truncate(24 * time.Hour),
							WindowEnd:   timestamp.Truncate(24 * time.Hour).Add(24 * time.Hour),
							GroupBy:     map[string]*string{},
						},
						{
							Value:       customerCount * 100,
							WindowStart: timestamp.Add(24 * time.Hour).Truncate(24 * time.Hour),
							WindowEnd:   timestamp.Add(24 * time.Hour).Truncate(24 * time.Hour).Add(24 * time.Hour),
							GroupBy:     map[string]*string{},
						},
					},
					WindowSize: &windowSize,
				}

				assert.Equal(t, expected, resp.JSON200)
			}, time.Minute, time.Second)
		})
	})

	t.Run("Subject", func(t *testing.T) {
		t.Parallel()

		// TODO: randomize?
		// TODO: make sure we have enough subject
		subject := []string{"customer-1", "customer-2"}

		// Wait for events to be processed
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.QueryMeterWithResponse(context.Background(), "query", &api.QueryMeterParams{
				Subject: &subject,
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			expected := &api.MeterQueryResult{
				Data: []models.MeterQueryRow{
					{
						Value:       4 * 100,
						WindowStart: timestamp.Truncate(time.Minute),
						WindowEnd:   timestamp.Truncate(time.Minute).Add(24*time.Hour + time.Minute),
						Subject:     &subject[1],
						GroupBy:     map[string]*string{},
					},
					{
						Value:       4 * 100,
						WindowStart: timestamp.Truncate(time.Minute),
						WindowEnd:   timestamp.Truncate(time.Minute).Add(24*time.Hour + time.Minute),
						Subject:     &subject[0],
						GroupBy:     map[string]*string{},
					},
				},
			}

			assert.Equal(t, expected, resp.JSON200)
		}, time.Minute, time.Second)
	})

	// TODO: improve group by tests by adding more than one parameter
	//
	// Note: this test breaks if any of the randomization parameters are changed
	// TODO: Fix query ordering first
	// t.Run("GroupBy", func(t *testing.T) {
	// 	t.Parallel()
	//
	// 	resp, err := client.QueryMeterWithResponse(context.Background(), "query", &api.QueryMeterParams{
	// 		GroupBy: &[]string{"method"},
	// 	})
	// 	require.NoError(t, err)
	// 	require.Equal(t, http.StatusOK, resp.StatusCode())
	//
	// 	expected := &api.MeterQueryResult{
	// 		Data: []models.MeterQueryRow{
	// 			{
	// 				Value:       4 * 100,
	// 				WindowStart: timestamp.Truncate(time.Minute),
	// 				WindowEnd:   timestamp.Truncate(time.Minute).Add(24*time.Hour + time.Minute),
	// 				GroupBy:     map[string]*string{},
	// 			},
	// 			{
	// 				Value:       4 * 100,
	// 				WindowStart: timestamp.Truncate(time.Minute),
	// 				WindowEnd:   timestamp.Truncate(time.Minute).Add(24*time.Hour + time.Minute),
	// 				GroupBy:     map[string]*string{},
	// 			},
	// 		},
	// 	}
	//
	// 	assert.Equal(t, expected, resp.JSON200)
	// })

	// TODO: add tests for group by and subject
}

func TestCredit(t *testing.T) {
	client := initClient(t)
	subject := "customer-1"
	meterSlug := "credit_test_meter"
	var featureId *api.FeatureId

	const waitTime = time.Second * 4

	t.Run("Create Feature", func(t *testing.T) {
		randKey := ulid.Make().String()
		resp, err := client.CreateFeatureWithResponse(context.Background(), api.CreateFeatureJSONRequestBody{
			Name:      "Credit Test Feature",
			MeterSlug: convert.ToPointer("credit_test_meter"),
			Key:       randKey,
			MeterGroupByFilters: &map[string]string{
				"model": "gpt-4",
			},
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		featureId = resp.JSON201.Id

		expected := &api.Feature{
			Id:        featureId,
			Name:      "Credit Test Feature",
			Key:       randKey,
			MeterSlug: convert.ToPointer("credit_test_meter"),
			MeterGroupByFilters: &map[string]string{
				"model": "gpt-4",
			},
		}

		require.NotEmpty(t, *resp.JSON201.CreatedAt)
		require.NotEmpty(t, *resp.JSON201.UpdatedAt)
		resp.JSON201.CreatedAt = nil
		resp.JSON201.UpdatedAt = nil

		require.Equal(t, expected, resp.JSON201)
	})

	var entitlementId *string
	var eCreatedAt *time.Time
	t.Run("Create a Entitlement", func(t *testing.T) {
		resp, err := client.CreateEntitlementWithResponse(context.Background(), subject, api.CreateEntitlementJSONRequestBody{
			Type:      "metered",
			FeatureId: *featureId,
			UsagePeriod: api.RecurringPeriodCreateInputs{
				Anchor:   time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				Interval: "MONTHLY",
			},
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		metered, err := resp.JSON201.AsEntitlementMetered()
		require.NoError(t, err)

		require.Equal(t, metered.SubjectKey, subject)
		entitlementId = metered.Id
		eCreatedAt = metered.CreatedAt
	})

	t.Run("Create for same subject and feature", func(t *testing.T) {
		resp, err := client.CreateEntitlementWithResponse(context.Background(), subject, api.CreateEntitlementJSONRequestBody{
			Type:      "metered",
			FeatureId: *featureId,
			UsagePeriod: api.RecurringPeriodCreateInputs{
				Anchor:   time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				Interval: "DAILY",
			},
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		require.NotEmpty(t, resp.ApplicationproblemJSON409.Extensions.ConflictingEntityId)
		require.Equal(t, *entitlementId, resp.ApplicationproblemJSON409.Extensions.ConflictingEntityId)
	})

	t.Run("Create Grant", func(t *testing.T) {
		effectiveAt := time.Now().Truncate(time.Minute)

		priority := 1
		maxRolloverAmount := 100.0

		// Create grant
		resp, err := client.CreateGrantWithResponse(context.Background(), subject, *entitlementId, api.EntitlementGrantCreateInput{
			Amount:      100,
			EffectiveAt: effectiveAt,
			Expiration: api.ExpirationPeriod{
				Duration: "MONTH",
				Count:    1,
			},
			Priority:          &priority,
			MaxRolloverAmount: &maxRolloverAmount,
			Recurrence: &api.RecurringPeriodCreateInputs{
				Anchor:   time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				Interval: "YEARLY",
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", resp.Body)

		require.NotEmpty(t, resp.JSON201.UpdatedAt)
		require.NotEmpty(t, resp.JSON201.ExpiresAt)
		require.NotEmpty(t, resp.JSON201.CreatedAt)
		require.NotEmpty(t, resp.JSON201.NextRecurrence)

		expected := &api.EntitlementGrant{
			Id:                resp.JSON201.Id,
			Amount:            100,
			EntitlementId:     entitlementId,
			Priority:          &priority,
			EffectiveAt:       effectiveAt.UTC(),
			MaxRolloverAmount: &maxRolloverAmount,
			SubjectKey:        &subject,
			Expiration: api.ExpirationPeriod{
				Duration: "MONTH",
				Count:    1,
			},
			Recurrence: &api.RecurringPeriodCreateInputs{
				Anchor:   time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				Interval: "YEARLY",
			},
		}

		resp.JSON201.CreatedAt = nil
		resp.JSON201.UpdatedAt = nil
		resp.JSON201.ExpiresAt = nil
		resp.JSON201.NextRecurrence = nil

		require.Equal(t, expected, resp.JSON201)
	})

	t.Run("Ingest usage", func(t *testing.T) {
		// Reproducible random data
		faker := gofakeit.New(8675309)
		var events []cloudevents.Event

		newEvent := func(timestamp string, model string) cloudevents.Event {
			ts, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				t.Fatal(err)
			}

			ev := cloudevents.New()
			ev.SetID(faker.UUID())
			ev.SetSource("credit-test")
			ev.SetType("credit_event")
			ev.SetTime(ts)
			ev.SetSubject(subject)
			_ = ev.SetData("application/json", map[string]string{
				"model": model,
			})

			return ev
		}

		et := eCreatedAt.Add(time.Second * 15)

		// First event
		{
			events = append(events, newEvent(et.Format(time.RFC3339), "gpt-4"))
		}

		// Irrelevant event (does not affect credit because of model mismatch)
		{
			events = append(events, newEvent(et.Format(time.RFC3339), "gpt-3"))
		}

		// Ingore events
		{
			resp, err := client.IngestEventBatchWithResponse(context.Background(), events)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}

		// Wait for events to be processed
		testutils.EventuallyWithTf(t, func(c *assert.CollectT, saveErr func(err any)) {
			resp, err := client.QueryMeterWithResponse(context.Background(), meterSlug, &api.QueryMeterParams{
				Subject: &[]string{subject},
			})
			saveErr(err)
			assert.NoError(c, err)
			assert.Equal(c, http.StatusOK, resp.StatusCode())

			require.GreaterOrEqual(t, len(resp.JSON200.Data), 1)

			// As we invested two events with a count meter
			assert.NotNil(c, resp.JSON200)
			if resp.JSON200 != nil {
				assert.Len(c, resp.JSON200.Data, 1)
				if len(resp.JSON200.Data) > 0 {
					assert.Equal(c, 2.0, resp.JSON200.Data[0].Value)
				}
			}
		}, waitTime, time.Second)
	})

	t.Run("Entitlement Value", func(t *testing.T) {
		// Get grants
		grantListResp, err := client.ListGrantsWithResponse(context.Background(), &api.ListGrantsParams{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, grantListResp.StatusCode())
		require.NotNil(t, grantListResp.JSON200)
		require.Len(t, *grantListResp.JSON200, 1)

		// Get feature
		featureListResp, err := client.ListFeaturesWithResponse(context.Background(), nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, featureListResp.StatusCode())
		require.NotNil(t, featureListResp.JSON200)
		require.Len(t, *featureListResp.JSON200, 1)

		resp, err := client.GetEntitlementValueWithResponse(context.Background(), subject, *entitlementId, &api.GetEntitlementValueParams{
			Time: convert.ToPointer(eCreatedAt.Add(time.Minute)),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		expected := &api.EntitlementValue{
			Balance:   convert.ToPointer(99.0),
			HasAccess: convert.ToPointer(true),
			Overage:   convert.ToPointer(0.0),
			Usage:     convert.ToPointer(1.0),
		}

		assert.Equal(t, expected, resp.JSON200)
	})

	t.Run("Reset", func(t *testing.T) {
		// we have to wait for a minute to pass so we can reset
		time.Sleep(time.Minute)
		effectiveAt := time.Now().Truncate(time.Minute)

		// Get grants
		grantListResp, err := client.ListGrantsWithResponse(context.Background(), &api.ListGrantsParams{})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, grantListResp.StatusCode())
		require.NotNil(t, grantListResp.JSON200)
		require.Len(t, *grantListResp.JSON200, 1)

		// Get feature
		featureListResp, err := client.ListFeaturesWithResponse(context.Background(), nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, featureListResp.StatusCode())
		require.NotNil(t, featureListResp.JSON200)
		require.Len(t, *featureListResp.JSON200, 1)

		// Reset usage
		resetResp, err := client.ResetEntitlementUsageWithResponse(context.Background(), subject, *entitlementId, api.ResetEntitlementUsageJSONRequestBody{
			EffectiveAt: &effectiveAt,
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resetResp.StatusCode())
	})
}
