package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
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
	// Make sure OpenMeter is ready
	// TODO: replace this with some sort of health check
	time.Sleep(15 * time.Second)

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
	}, 30*time.Second, time.Second)
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
	}, 30*time.Second, time.Second)
}

func TestDedupe(t *testing.T) {
	client := initClient(t)

	// Make clickhouse's job easier by sending events within a fix time range
	now := time.Now()

	var firstDuration int

	for i := 0; i < 1000; i++ {
		// Make clickhouse's job easier by sending events within a fix time range
		timestamp := gofakeit.DateRange(now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour))
		duration := gofakeit.Number(1, 1000)

		if firstDuration == 0 {
			firstDuration = duration
		}

		ev := cloudevents.New()
		ev.SetID("52f44f66-020f-4fa9-a733-102a8ef6f515")
		ev.SetSource("my-app")
		ev.SetType("dedupe")
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
		resp, err := client.QueryMeterWithResponse(context.Background(), "dedupe", nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.Len(t, resp.JSON200.Data, 1)
		assert.Equal(t, float64(firstDuration), resp.JSON200.Data[0].Value)
	}, 30*time.Second, time.Second)
}

func TestQuery(t *testing.T) {
	client := initClient(t)

	// Reproducible random data
	faker := gofakeit.New(8675309)

	paths := []string{"/", "/about", "/users", "/contact"}

	timestamp := faker.DateRange(time.Date(2023, time.May, 6, 0, 0, 0, 0, time.UTC), faker.FutureDate().UTC()).UTC().Truncate(time.Second)

	const customerCount = 5

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
	time.Sleep(10 * time.Second)

	t.Run("Total", func(t *testing.T) {
		t.Parallel()

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
		}, 30*time.Second, time.Second)
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
			}, 30*time.Second, time.Second)
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
			}, 30*time.Second, time.Second)
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
			}, 30*time.Second, time.Second)
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
		}, 30*time.Second, time.Second)
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
