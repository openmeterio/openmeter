package e2e

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/convert"
)

type customerIOTA int

func (c customerIOTA) Key() string {
	return fmt.Sprintf("customer-%d", c)
}

const (
	customer1 customerIOTA = iota
	customer2
	customer3
)

func TestIngest(t *testing.T) {
	client := initClient(t)

	// ensure subject exists
	{
		resp, err := client.UpsertSubjectWithResponse(context.Background(), api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{
				Key: "customer-1",
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}

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

// Send an event with content type application/json.
// We treat request as application/cloudevents+json if it's a single event
// and treat application/cloudevents-batch+json if it's an array of events.
func TestIngestContentTypeApplicationJSON(t *testing.T) {
	client := initClient(t)

	// ensure subject exists
	{
		resp, err := client.UpsertSubjectWithResponse(context.Background(), api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{
				Key: "customer-1",
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}

	tm := time.Now().Add(-time.Hour).Format(time.RFC3339)
	eventType := "ingest_content_type_application_json"

	// Send a single event
	{
		payload := fmt.Sprintf(`{
			"specversion" : "1.0",
			"id": "%s",
			"source": "my-app",
			"type": "%s",
			"subject": "customer-1",
			"time": "%s",
			"data": { "duration_ms": "100" }
		}`, ulid.Make().String(), eventType, tm)

		resp, err := client.IngestEventsWithBody(context.Background(), "application/json", strings.NewReader(payload))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	}

	// Send an array of events
	{
		payload := fmt.Sprintf(`[
			{
				"specversion" : "1.0",
				"id": "%s",
				"source": "my-app",
				"type": "%s",
				"subject": "customer-1",
				"time": "%s",
				"data": { "duration_ms": "100" }
			},
			{
				"specversion" : "1.0",
				"id": "%s",
				"source": "my-app",
				"type": "%s",
				"subject": "customer-1",
				"time": "%s",
				"data": { "duration_ms": "100" }
			}
		]`,
			ulid.Make().String(), eventType, tm,
			ulid.Make().String(), eventType, tm,
		)

		resp, err := client.IngestEventsWithBody(context.Background(), "application/json", strings.NewReader(payload))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	}

	// Wait for events to be processed
	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		resp, err := client.QueryMeterWithResponse(context.Background(), eventType, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.Len(t, resp.JSON200.Data, 1)
		assert.Equal(t, 300.0, resp.JSON200.Data[0].Value)
	}, time.Minute, time.Second)
}

func TestBatchIngest(t *testing.T) {
	client := initClient(t)

	// ensure subject exists
	{
		resp, err := client.UpsertSubjectWithResponse(context.Background(), api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{
				Key: "customer-1",
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}

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

func TestInvalidIngest(t *testing.T) {
	client := initClient(t)

	// Make clickhouse's job easier by sending events within a fix time range
	timeBase := time.Now().Add(-time.Hour)
	timeIdx := 0
	eventType := "ingest_invalid"
	subject := eventType
	meterKey := eventType

	// ensure subject exists
	{
		resp, err := client.UpsertSubjectWithResponse(context.Background(), api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{
				Key: subject,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}

	getTime := func() time.Time {
		timeIdx++
		// Event list is in reverse order by time
		return timeBase.Add(time.Duration(-timeIdx) * time.Minute)
	}

	// Send an event with unsupported data content type: xml
	{
		ev := cloudevents.New()
		ev.SetID(ulid.Make().String())
		ev.SetSource("my-app")
		ev.SetType(eventType)
		ev.SetSubject(subject)
		ev.SetTime(getTime())
		_ = ev.SetData(cloudevents.ApplicationXML, []byte("<xml></xml>"))

		resp, err := client.IngestEventWithResponse(context.Background(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode())
	}

	// Send an event where data is a string
	{
		payload := fmt.Sprintf(`{
			"specversion" : "1.0",
			"id": "%s",
			"source": "my-app",
			"type": "%s",
			"subject": "%s",
			"time": "%s",
			"data": "string"
		}`, ulid.Make().String(), eventType, subject, getTime().Format(time.RFC3339))

		resp, err := client.IngestEventsWithBody(context.Background(), "application/cloudevents+json", strings.NewReader(payload))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	}

	// Send an event where data is null
	// null data should be treated as no data in CloudEvents and should be accepted
	{
		payload := fmt.Sprintf(`{
				"specversion" : "1.0",
				"id": "%s",
				"source": "my-app",
				"type": "%s",
				"subject": "%s",
				"time": "%s",
				"data": null
			}`, ulid.Make().String(), eventType, subject, getTime().Format(time.RFC3339))

		resp, err := client.IngestEventsWithBody(context.Background(), "application/cloudevents+json", strings.NewReader(payload))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode)
		resp.Body.Close()
	}

	// Send an event without data
	{
		ev := cloudevents.New()
		ev.SetID(ulid.Make().String())
		ev.SetSource("my-app")
		ev.SetType(eventType)
		ev.SetSubject(subject)
		ev.SetTime(getTime())

		resp, err := client.IngestEventWithResponse(context.Background(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode())
	}

	// Send an event with empty data
	{
		ev := cloudevents.New()
		ev.SetID(ulid.Make().String())
		ev.SetSource("my-app")
		ev.SetType(eventType)
		ev.SetSubject(subject)
		ev.SetTime(getTime())
		err := ev.SetData(cloudevents.ApplicationJSON, map[string]string{})
		require.NoError(t, err)

		resp, err := client.IngestEventWithResponse(context.Background(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode())
	}

	// Send an event with a NaN value (will skip from aggregation)
	{
		ev := cloudevents.New()
		ev.SetID(ulid.Make().String())
		ev.SetSource("my-app")
		ev.SetType(eventType)
		ev.SetSubject(subject)
		ev.SetTime(getTime())
		_ = ev.SetData(cloudevents.ApplicationJSON, map[string]string{
			"duration_ms": "NaN",
		})

		resp, err := client.IngestEventWithResponse(context.Background(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode())
	}

	// Send an event with a Inf value (will skip from aggregation)
	{
		ev := cloudevents.New()
		ev.SetID(ulid.Make().String())
		ev.SetSource("my-app")
		ev.SetType(eventType)
		ev.SetSubject(subject)
		ev.SetTime(getTime())
		_ = ev.SetData(cloudevents.ApplicationJSON, map[string]string{
			"duration_ms": "Inf",
		})

		resp, err := client.IngestEventWithResponse(context.Background(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode())
	}

	// Send a valid event, this is what we should get back
	{
		ev := cloudevents.New()
		ev.SetID(ulid.Make().String())
		ev.SetSource("my-app")
		ev.SetType(eventType)
		ev.SetSubject(subject)
		ev.SetTime(getTime())
		_ = ev.SetData(cloudevents.ApplicationJSON, map[string]string{
			"duration_ms": "100",
		})

		resp, err := client.IngestEventWithResponse(context.Background(), ev)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, resp.StatusCode())
	}

	// Wait for events to be processed
	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		resp, err := client.QueryMeterWithResponse(context.Background(), meterKey, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.Len(t, resp.JSON200.Data, 1)
		assert.Equal(t, 100.0, resp.JSON200.Data[0].Value)
	}, time.Minute, time.Second)

	// List events with has error should return the invalid events
	resp, err := client.ListEventsWithResponse(context.Background(), &api.ListEventsParams{
		Subject: &subject,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.NotNil(t, resp.JSON200)

	events := *resp.JSON200
	require.Len(t, events, 6)

	// unsupported data content gets rejected with a bad request so it should not be in the list

	// non json data should be rejected with a bad request so it should not be in the list

	// null data should have processing error
	require.NotNil(t, events[0].ValidationError)
	require.Contains(t, *events[0].ValidationError, `invalid event: null and missing value property`)

	// missing data should have processing error
	// we only validate events against meters in the processing pipeline so this is an async error
	require.NotNil(t, events[1].ValidationError)
	require.Contains(t, *events[1].ValidationError, `invalid event: null and missing value property`)

	// empty data should have processing error as it does not have the required value property
	require.NotNil(t, events[2].ValidationError)
	require.Contains(t, *events[2].ValidationError, `invalid event: missing value property: "$.duration_ms"`)

	// nan data should have processing error as it does not have the required value property
	require.NotNil(t, events[3].ValidationError)
	require.Contains(t, *events[3].ValidationError, `invalid event: value cannot be NaN`)

	// inf data should have processing error as it does not have the required value property
	require.NotNil(t, events[4].ValidationError)
	require.Contains(t, *events[4].ValidationError, `invalid event: value cannot be infinity`)
}

func TestDedupe(t *testing.T) {
	client := initClient(t)

	// ensure subject exists
	{
		resp, err := client.UpsertSubjectWithResponse(context.Background(), api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{
				Key: "customer-1",
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}

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
	startOfDay := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	client := initClient(t)

	// Reproducible random data
	const customerCount = 5

	// ensure subjects exist
	for i := 1; i <= customerCount; i++ {
		resp, err := client.UpsertSubjectWithResponse(context.Background(), api.UpsertSubjectJSONRequestBody{
			api.SubjectUpsert{Key: fmt.Sprintf("customer-%d", i)},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}
	paths := []string{"/", "/about", "/users", "/contact"}
	faker := gofakeit.New(8675309)
	randTime := faker.DateRange(time.Date(2023, time.May, 6, 0, 0, 0, 0, time.UTC), faker.FutureDate().UTC())
	timestamp := startOfDay(randTime.UTC()).UTC().Truncate(time.Second)

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
			events = append(events, newTimedEvents(timestamp.AddDate(0, 0, 1))...)
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
				Data: []api.MeterQueryRow{
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

			windowSize := meter.WindowSizeMinute

			// Wait for events to be processed
			assert.EventuallyWithT(t, func(t *assert.CollectT) {
				resp, err := client.QueryMeterWithResponse(context.Background(), "query", &api.QueryMeterParams{
					WindowSize: lo.ToPtr(api.WindowSize(windowSize)),
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode())

				expected := &api.MeterQueryResult{
					Data: []api.MeterQueryRow{
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
					WindowSize: lo.ToPtr(api.WindowSize(windowSize)),
				}

				assert.Equal(t, expected, resp.JSON200)
			}, time.Minute, time.Second)
		})

		t.Run("Hour", func(t *testing.T) {
			t.Parallel()

			windowSize := meter.WindowSizeHour

			// Wait for events to be processed
			assert.EventuallyWithT(t, func(t *assert.CollectT) {
				resp, err := client.QueryMeterWithResponse(context.Background(), "query", &api.QueryMeterParams{
					WindowSize: lo.ToPtr(api.WindowSize(windowSize)),
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode())

				expected := &api.MeterQueryResult{
					Data: []api.MeterQueryRow{
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
					WindowSize: lo.ToPtr(api.WindowSize(windowSize)),
				}

				assert.Equal(t, expected, resp.JSON200)
			}, time.Minute, time.Second)
		})

		t.Run("Day", func(t *testing.T) {
			t.Parallel()

			windowSize := meter.WindowSizeDay

			// Wait for events to be processed
			assert.EventuallyWithT(t, func(t *assert.CollectT) {
				resp, err := client.QueryMeterWithResponse(context.Background(), "query", &api.QueryMeterParams{
					WindowSize: lo.ToPtr(api.WindowSize(windowSize)),
				})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, resp.StatusCode())

				expected := &api.MeterQueryResult{
					Data: []api.MeterQueryRow{
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
					WindowSize: lo.ToPtr(api.WindowSize(windowSize)),
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
				Data: []api.MeterQueryRow{
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
}

func TestQueryDSTTransition(t *testing.T) {
	client := initClient(t)

	// Test DST transitions with DAY window size in America/Los_Angeles timezone
	// Verify that window boundaries are continuous across DST changes
	// DST start: March 9, 2025 at 2am -> 3am (spring forward)
	// DST end: November 2, 2025 at 2am -> 1am (fall back)

	losAngeles, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)

	eventType := "query"
	subject := "customer-dst"

	t.Run("DST_Spring_Forward", func(t *testing.T) {
		// Test March 9, 2025 DST transition (spring forward: 2am -> 3am)
		// Test 3 days before and 3 days after the transition
		var events []cloudevents.Event

		// Generate events for March 6-12, 2025 (3 days before, transition day, 3 days after)
		for day := 6; day <= 12; day++ {
			baseTime := time.Date(2025, time.March, day, 10, 0, 0, 0, losAngeles)
			ev := cloudevents.New()
			ev.SetID(ulid.Make().String())
			ev.SetSource("dst-test")
			ev.SetType(eventType)
			ev.SetSubject(subject)
			ev.SetTime(baseTime)
			_ = ev.SetData("application/json", map[string]interface{}{
				"duration_ms": "100",
			})
			events = append(events, ev)
		}

		// Ingest events
		{
			resp, err := client.IngestEventBatchWithResponse(context.Background(), events)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}

		// Query with DAY window size in Los Angeles timezone
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			from := time.Date(2025, time.March, 6, 0, 0, 0, 0, losAngeles)
			to := time.Date(2025, time.March, 13, 0, 0, 0, 0, losAngeles)

			resp, err := client.QueryMeterWithResponse(context.Background(), eventType, &api.QueryMeterParams{
				Subject:        &[]string{subject},
				WindowSize:     lo.ToPtr(api.WindowSizeDay),
				WindowTimeZone: lo.ToPtr("America/Los_Angeles"),
				From:           lo.ToPtr(from),
				To:             lo.ToPtr(to),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			require.NotNil(t, resp.JSON200)
			require.Len(t, resp.JSON200.Data, 7) // March 6-12

			// Verify window continuity - each window's end should equal the next window's start
			for i := 0; i < len(resp.JSON200.Data)-1; i++ {
				currentWindowEnd := resp.JSON200.Data[i].WindowEnd.In(losAngeles)
				nextWindowStart := resp.JSON200.Data[i+1].WindowStart.In(losAngeles)
				assert.Equal(t, currentWindowEnd, nextWindowStart,
					"Gap detected: window %d ends at %v but window %d starts at %v",
					i, currentWindowEnd, i+1, nextWindowStart)
			}

			// Verify all windows start at midnight in LA timezone
			for i, row := range resp.JSON200.Data {
				windowStart := row.WindowStart.In(losAngeles)
				assert.Equal(t, 0, windowStart.Hour(), "Day %d window doesn't start at midnight", i)
				assert.Equal(t, 0, windowStart.Minute(), "Day %d window doesn't start at midnight", i)
				assert.Equal(t, 0, windowStart.Second(), "Day %d window doesn't start at midnight", i)
			}

			// Verify the DST transition day (March 9) windows are correct
			dstDay := resp.JSON200.Data[3] // March 9 is index 3 (6,7,8,9)
			assert.Equal(t, time.Date(2025, time.March, 9, 0, 0, 0, 0, losAngeles), dstDay.WindowStart.In(losAngeles))
			assert.Equal(t, time.Date(2025, time.March, 10, 0, 0, 0, 0, losAngeles), dstDay.WindowEnd.In(losAngeles))
		}, time.Minute, time.Second)
	})

	t.Run("DST_Fall_Back", func(t *testing.T) {
		// Test November 2, 2025 DST transition (fall back: 2am -> 1am)
		// Test 3 days before and 3 days after the transition
		var events []cloudevents.Event

		// Generate events for October 30 - November 5, 2025 (3 days before, transition day, 3 days after)
		days := []struct {
			month int
			day   int
		}{
			{int(time.October), 30},
			{int(time.October), 31},
			{int(time.November), 1},
			{int(time.November), 2},
			{int(time.November), 3},
			{int(time.November), 4},
			{int(time.November), 5},
		}

		for _, d := range days {
			baseTime := time.Date(2025, time.Month(d.month), d.day, 10, 0, 0, 0, losAngeles)
			ev := cloudevents.New()
			ev.SetID(ulid.Make().String())
			ev.SetSource("dst-test")
			ev.SetType(eventType)
			ev.SetSubject(subject)
			ev.SetTime(baseTime)
			_ = ev.SetData("application/json", map[string]interface{}{
				"duration_ms": "100",
			})
			events = append(events, ev)
		}

		// Ingest events
		{
			resp, err := client.IngestEventBatchWithResponse(context.Background(), events)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}

		// Query with DAY window size in Los Angeles timezone
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			from := time.Date(2025, time.October, 30, 0, 0, 0, 0, losAngeles)
			to := time.Date(2025, time.November, 6, 0, 0, 0, 0, losAngeles)

			resp, err := client.QueryMeterWithResponse(context.Background(), eventType, &api.QueryMeterParams{
				Subject:        &[]string{subject},
				WindowSize:     lo.ToPtr(api.WindowSizeDay),
				WindowTimeZone: lo.ToPtr("America/Los_Angeles"),
				From:           lo.ToPtr(from),
				To:             lo.ToPtr(to),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			require.NotNil(t, resp.JSON200)
			require.Len(t, resp.JSON200.Data, 7) // October 30 - November 5

			// Verify window continuity - each window's end should equal the next window's start
			for i := 0; i < len(resp.JSON200.Data)-1; i++ {
				currentWindowEnd := resp.JSON200.Data[i].WindowEnd.In(losAngeles)
				nextWindowStart := resp.JSON200.Data[i+1].WindowStart.In(losAngeles)
				assert.Equal(t, currentWindowEnd, nextWindowStart,
					"Gap detected: window %d ends at %v but window %d starts at %v",
					i, currentWindowEnd, i+1, nextWindowStart)
			}

			// Verify all windows start at midnight in LA timezone
			for i, row := range resp.JSON200.Data {
				windowStart := row.WindowStart.In(losAngeles)
				assert.Equal(t, 0, windowStart.Hour(), "Day %d window doesn't start at midnight", i)
				assert.Equal(t, 0, windowStart.Minute(), "Day %d window doesn't start at midnight", i)
				assert.Equal(t, 0, windowStart.Second(), "Day %d window doesn't start at midnight", i)
			}

			// Verify the DST transition day (November 2) windows are correct
			dstDay := resp.JSON200.Data[3] // November 2 is index 3 (Oct 30, 31, Nov 1, 2)
			assert.Equal(t, time.Date(2025, time.November, 2, 0, 0, 0, 0, losAngeles), dstDay.WindowStart.In(losAngeles))
			assert.Equal(t, time.Date(2025, time.November, 3, 0, 0, 0, 0, losAngeles), dstDay.WindowEnd.In(losAngeles))
		}, time.Minute, time.Second)
	})
}

func TestCredit(t *testing.T) {
	client := initClient(t)
	meterSlug := "credit_test_meter"
	var featureId string
	var featureKey string

	const waitTime = time.Second * 30

	apiMONTH := &api.RecurringPeriodInterval{}
	require.NoError(t, apiMONTH.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumMONTH))

	apiYEAR := &api.RecurringPeriodInterval{}
	require.NoError(t, apiYEAR.FromRecurringPeriodIntervalEnum(api.RecurringPeriodIntervalEnumYEAR))

	customerKey := customer1.Key()
	subjectKey := customerKey

	// Let's create a customer with a subject
	CreateCustomerWithSubject(t, client, customerKey, subjectKey)

	t.Run("Create Feature", func(t *testing.T) {
		randKey := fmt.Sprintf("credit_test_feature_%d", time.Now().Unix())
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
		featureKey = resp.JSON201.Key

		expected := &api.Feature{
			Id:        featureId,
			Name:      "Credit Test Feature",
			Key:       randKey,
			MeterSlug: convert.ToPointer("credit_test_meter"),
			MeterGroupByFilters: &map[string]string{
				"model": "gpt-4",
			},
			AdvancedMeterGroupByFilters: &map[string]api.FilterString{
				"model": {
					Eq: convert.ToPointer("gpt-4"),
				},
			},
		}

		require.NotEmpty(t, resp.JSON201.CreatedAt)
		require.NotEmpty(t, resp.JSON201.UpdatedAt)

		// Exclude created_at and update_at from comparison
		resp.JSON201.CreatedAt = time.Time{}
		resp.JSON201.UpdatedAt = time.Time{}
		require.Equal(t, expected, resp.JSON201)
	})

	var entitlementId string
	var eCreatedAt time.Time
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
		resp, err := client.CreateEntitlementWithResponse(context.Background(), subjectKey, *body)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		metered, err := resp.JSON201.AsEntitlementMetered()
		require.NoError(t, err)

		require.Equal(t, metered.SubjectKey, subjectKey)
		entitlementId = metered.Id
		eCreatedAt = metered.CreatedAt
	})

	t.Run("Create for same subject and feature", func(t *testing.T) {
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
		resp, err := client.CreateEntitlementWithResponse(context.Background(), subjectKey, *body)

		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		require.NotNil(t, resp.ApplicationproblemJSON409)
		require.NotEmpty(t, resp.ApplicationproblemJSON409.Extensions["conflictingEntityId"])
		require.Equal(t, entitlementId, resp.ApplicationproblemJSON409.Extensions["conflictingEntityId"])
	})

	t.Run("Create a Entitlement With Default Grants", func(t *testing.T) {
		randCustomerKey := ulid.Make().String()
		randSubjectKey := ulid.Make().String()

		CreateCustomerWithSubject(t, client, randCustomerKey, randSubjectKey)

		measureUsageFrom := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
		muf := &api.MeasureUsageFrom{}
		err := muf.FromMeasureUsageFromTime(measureUsageFrom)
		require.NoError(t, err)
		meteredEntitlement := api.EntitlementMeteredCreateInputs{
			Type:      "metered",
			FeatureId: &featureId,
			UsagePeriod: api.RecurringPeriodCreateInput{
				Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
				Interval: *apiMONTH,
			},
			MeasureUsageFrom:        muf,
			IssueAfterReset:         convert.ToPointer(100.0),
			IssueAfterResetPriority: convert.ToPointer[uint8](6),
		}
		body := &api.CreateEntitlementJSONRequestBody{}
		err = body.FromEntitlementMeteredCreateInputs(meteredEntitlement)
		require.NoError(t, err)
		resp, err := client.CreateEntitlementWithResponse(context.Background(), randSubjectKey, *body)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		metered, err := resp.JSON201.AsEntitlementMetered()
		require.NoError(t, err)

		require.Equal(t, randSubjectKey, metered.SubjectKey)
		require.Equal(t, measureUsageFrom, metered.MeasureUsageFrom)

		// fetch grants for entitlement
		grantListResp, err := client.ListEntitlementGrantsWithResponse(context.Background(), randSubjectKey, metered.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, grantListResp.StatusCode())
		require.NotNil(t, grantListResp.JSON200)
		require.Len(t, *grantListResp.JSON200, 1)

		require.Equal(t, *metered.IssueAfterReset, (*grantListResp.JSON200)[0].Amount)
		require.Equal(t, metered.IssueAfterResetPriority, (*grantListResp.JSON200)[0].Priority)
		require.Equal(t, metered.Id, (*grantListResp.JSON200)[0].EntitlementId)
		require.Equal(t, api.Annotations{
			"issueAfterReset": true,
		}, *(*grantListResp.JSON200)[0].Annotations)
	})
	t.Run("Create a Entitlement With MeasureUsageFrom enum", func(t *testing.T) {
		randSubject := ulid.Make().String()
		randCustomerKey := ulid.Make().String()

		CreateCustomerWithSubject(t, client, randCustomerKey, randSubject)

		periodAnchor := time.Now().Truncate(time.Minute).Add(-time.Hour).In(time.UTC)
		muf := &api.MeasureUsageFrom{}
		err := muf.FromMeasureUsageFromPreset(api.MeasureUsageFromPresetCurrentPeriodStart)
		require.NoError(t, err)
		meteredEntitlement := api.EntitlementMeteredCreateInputs{
			Type:      "metered",
			FeatureId: &featureId,
			UsagePeriod: api.RecurringPeriodCreateInput{
				Anchor:   &periodAnchor,
				Interval: *apiMONTH,
			},
			MeasureUsageFrom: muf,
		}
		body := &api.CreateEntitlementJSONRequestBody{}
		err = body.FromEntitlementMeteredCreateInputs(meteredEntitlement)
		require.NoError(t, err)
		resp, err := client.CreateEntitlementWithResponse(context.Background(), randSubject, *body)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		metered, err := resp.JSON201.AsEntitlementMetered()
		require.NoError(t, err)

		require.Equal(t, randSubject, metered.SubjectKey)
		require.Equal(t, periodAnchor, metered.MeasureUsageFrom)
	})

	t.Run("Create Grant", func(t *testing.T) {
		effectiveAt := time.Now().Truncate(time.Minute)

		priority := uint8(1)
		maxRolloverAmount := 100.0
		minRolloverAmount := 0.0

		// Create grant
		resp, err := client.CreateGrantWithResponse(context.Background(), subjectKey, entitlementId, api.EntitlementGrantCreateInput{
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
				Interval: *apiYEAR,
			},
			Metadata: &api.Metadata{
				"some_key": "some_value",
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
			MinRolloverAmount: &minRolloverAmount,
			Expiration: api.ExpirationPeriod{
				Duration: "MONTH",
				Count:    1,
			},
			Recurrence: &api.RecurringPeriod{
				Anchor:      time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				Interval:    *apiYEAR,
				IntervalISO: "P1Y",
			},
			Metadata: &api.Metadata{
				"some_key": "some_value",
			},
		}

		// Exclude timestamps from comparison
		resp.JSON201.CreatedAt = time.Time{}
		resp.JSON201.UpdatedAt = time.Time{}
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
			ev.SetSubject(subjectKey)
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

		// NOTE: This is a temporary workaround to avoid data race condition in (assert|require).EventuallyWithTf function which
		// was not triggered before `testify` v1.11.1. Which includes the change (https://github.com/stretchr/testify/pull/1427)
		// that makes the EventuallyWithTf return early triggering the race condition in our case where the result of the condition
		// from the previous tick is used to decide the result of the test.
		// Remove this when the race condition is fixed in `testify`.
		// Bare in mind his still can fail if sink-worker is is not running or the ingestion of the events takes longer than the sleep time.
		time.Sleep(5 * time.Second)

		// Wait for events to be processed, fail on network errors
		testutils.EventuallyWithTf(t, func(c *assert.CollectT, saveErr func(err any)) {
			resp, err := client.QueryMeterWithResponse(context.Background(), meterSlug, &api.QueryMeterParams{
				Subject: &[]string{subjectKey},
			})
			saveErr(err)
			require.NoError(c, err)
			require.Equal(c, http.StatusOK, resp.StatusCode())

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
		grantListResp, err := client.ListEntitlementGrantsWithResponse(context.Background(), subjectKey, entitlementId, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, grantListResp.StatusCode())
		require.NotNil(t, grantListResp.JSON200)
		require.Len(t, *grantListResp.JSON200, 1)

		// Get feature
		featureListResp, err := client.ListFeaturesWithResponse(context.Background(), nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, featureListResp.StatusCode())
		require.NotNil(t, featureListResp.JSON200)
		features, err := featureListResp.JSON200.AsListFeaturesResult0()
		require.NoError(t, err)
		require.Len(t, features, 1)

		resp, err := client.GetEntitlementValueWithResponse(context.Background(), subjectKey, entitlementId, &api.GetEntitlementValueParams{
			Time: convert.ToPointer(eCreatedAt.Add(time.Minute * 2)),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		expected := &api.EntitlementValue{
			Balance:                   convert.ToPointer(99.0),
			HasAccess:                 true,
			Overage:                   convert.ToPointer(0.0),
			Usage:                     convert.ToPointer(1.0),
			TotalAvailableGrantAmount: convert.ToPointer(100.0),
		}

		assert.Equal(t, expected, resp.JSON200)
	})

	t.Run("Slow: Reset testing", func(t *testing.T) {
		// Entitlements are one minute rounded, so we need to wait for a minute to pass so we can reset, if we
		// really want we can run this on main. (if we have a second resolution we can just enable it)
		if !shouldRunSlowTests(t) {
			t.Skip("Skipping slow test, please reenable when we have a second resolution for entitlements")
		}

		t.Run("Reset", func(t *testing.T) {
			// we have to wait for a minute to pass so we can reset
			time.Sleep(time.Minute)
			effectiveAt := time.Now().Truncate(time.Minute)

			// Get grants
			grantListResp, err := client.ListEntitlementGrantsWithResponse(context.Background(), subjectKey, entitlementId, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, grantListResp.StatusCode())
			require.NotNil(t, grantListResp.JSON200)
			require.Len(t, *grantListResp.JSON200, 1)

			// Get feature
			featureListResp, err := client.ListFeaturesWithResponse(context.Background(), nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, featureListResp.StatusCode())
			require.NotNil(t, featureListResp.JSON200)
			features, err := featureListResp.JSON200.AsListFeaturesResult0()
			require.NoError(t, err)
			require.Len(t, features, 1)

			// Reset usage
			resetResp, err := client.ResetEntitlementUsageWithResponse(context.Background(), subjectKey, entitlementId, api.ResetEntitlementUsageJSONRequestBody{
				EffectiveAt: &effectiveAt,
			})

			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resetResp.StatusCode())
		})

		customer2Key := customer2.Key()
		subject2 := customer2Key
		// we have to wait after the reset
		time.Sleep(time.Minute)
		time.Sleep(time.Second * 10)

		t.Run("Create entitlement with automatic grant issuing", func(t *testing.T) {
			CreateCustomerWithSubject(t, client, customer2Key, subject2)

			meteredEntitlement := api.EntitlementMeteredCreateInputs{
				Type:      "metered",
				FeatureId: &featureId,
				UsagePeriod: api.RecurringPeriodCreateInput{
					Anchor:   convert.ToPointer(time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
					Interval: *apiMONTH,
				},
				IssueAfterReset: convert.ToPointer(50.0),
			}
			body := &api.CreateEntitlementJSONRequestBody{}
			err := body.FromEntitlementMeteredCreateInputs(meteredEntitlement)
			require.NoError(t, err)
			resp, err := client.CreateEntitlementWithResponse(context.Background(), subject2, *body)

			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

			metered, err := resp.JSON201.AsEntitlementMetered()
			require.NoError(t, err)

			require.Equal(t, metered.SubjectKey, subject2)

			// fetch grants for entitlement
			grantListResp, err := client.ListEntitlementGrantsWithResponse(context.Background(), subject2, metered.Id, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, grantListResp.StatusCode())
			require.NotNil(t, grantListResp.JSON200)
			require.Len(t, *grantListResp.JSON200, 1)
		})
	})

	t.Run("Override previous entitlement", func(t *testing.T) {
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

		subject := customer3.Key()

		CreateCustomerWithSubject(t, client, subject, subject)

		// create an entitlement
		resp, err := client.CreateEntitlementWithResponse(context.Background(), subject, *body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		metered, err := resp.JSON201.AsEntitlementMetered()
		require.NoError(t, err)

		entId := metered.Id

		// Override entitlement
		resp2, err := client.OverrideEntitlementWithResponse(context.Background(), subject, entId, *body)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "Invalid status code [response_body=%s]", string(resp.Body))

		require.NotNil(t, resp2)
		require.NotNil(t, resp2.JSON201)

		metered, err = resp2.JSON201.AsEntitlementMetered()
		require.NoError(t, err)

		require.Equal(t, metered.SubjectKey, subject)
		require.NotEqual(t, metered.Id, entId)
	})

	t.Run("List entitlements", func(t *testing.T) {
		// should return 2 entitlements for subject for feature
		resp, err := client.ListEntitlementsWithResponse(context.Background(), &api.ListEntitlementsParams{
			EntitlementType: &[]string{"metered"},
			Subject:         &[]string{subjectKey},
			Feature:         &[]string{featureKey},
			Page:            convert.ToPointer(1),
			PageSize:        convert.ToPointer(10),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.NotNil(t, resp.JSON200)
		// should be a paginated response
		resBody, err := resp.JSON200.AsEntitlementPaginatedResponse()
		require.NoError(t, err)

		require.Equal(t, 1, resBody.Page)
		require.Equal(t, 1, resBody.TotalCount)
		require.Equal(t, 1, len(resBody.Items))

		// should return 0 entitlements due to unused types
		resp, err = client.ListEntitlementsWithResponse(context.Background(), &api.ListEntitlementsParams{
			EntitlementType: &[]string{"static", "boolean"},
			Subject:         &[]string{subjectKey},
			Feature:         &[]string{featureKey},
			Page:            convert.ToPointer(1),
			PageSize:        convert.ToPointer(10),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.NotNil(t, resp.JSON200)
		// should be a paginated response
		resBody, err = resp.JSON200.AsEntitlementPaginatedResponse()
		require.NoError(t, err)

		require.Equal(t, 1, resBody.Page)
		require.Equal(t, 0, resBody.TotalCount)
		require.Equal(t, 0, len(resBody.Items))

		// should return 400 for invalid type
		resp, err = client.ListEntitlementsWithResponse(context.Background(), &api.ListEntitlementsParams{
			EntitlementType: &[]string{"INVALID_STR"},
			Subject:         &[]string{subjectKey},
			Feature:         &[]string{featureKey},
			Page:            convert.ToPointer(1),
			PageSize:        convert.ToPointer(10),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode())
	})
}
