package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
)

func TestListEventsByCustomerAttribution(t *testing.T) {
	client := initClient(t)
	suffix := ulid.Make().String()
	customerKey := fmt.Sprintf("streaming-customer-%s", suffix)
	subjectKeys := []string{
		fmt.Sprintf("streaming-subject-1-%s", suffix),
		fmt.Sprintf("streaming-subject-'2'-%s", suffix),
	}

	// Given a customer whose usage can be attributed by its key or either subject key.
	customerResponse, err := client.CreateCustomerWithResponse(t.Context(), api.CreateCustomerJSONRequestBody{
		Name:     fmt.Sprintf("Streaming E2E Customer %s", suffix),
		Currency: lo.ToPtr(api.CurrencyCode("USD")),
		Key:      &customerKey,
		UsageAttribution: &api.CustomerUsageAttribution{
			SubjectKeys: subjectKeys,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, customerResponse.StatusCode(), "response body: %s", customerResponse.Body)
	require.NotNil(t, customerResponse.JSON201)
	customerID := customerResponse.JSON201.Id
	require.NotEmpty(t, customerID)

	attributionKeys := append([]string{customerKey}, subjectKeys...)
	eventIDs := make([]string, 0, len(attributionKeys))
	eventTime := time.Now().UTC()

	// When usage events are sent using every attribution key.
	for i, attributionKey := range attributionKeys {
		eventID := ulid.Make().String()
		eventIDs = append(eventIDs, eventID)

		event := cloudevents.New()
		event.SetID(eventID)
		event.SetSource("openmeter-e2e-streaming")
		event.SetType(fmt.Sprintf("streaming-e2e-%s", suffix))
		event.SetSubject(attributionKey)
		event.SetTime(eventTime.Add(time.Duration(i) * time.Millisecond))
		require.NoError(t, event.SetData(cloudevents.ApplicationJSON, map[string]string{
			"attributionKey": attributionKey,
		}))

		response, err := client.IngestEventWithResponse(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), "response body: %s", response.Body)
	}

	// Then filtering by customer ID returns every attributed event, while filtering by an
	// attribution key returns the event sent with that key.
	attributionFilters := []struct {
		name             string
		params           api.ListEventsParams
		expectedEventIDs []string
	}{
		{
			name: "Should filter events by customer ID",
			params: api.ListEventsParams{
				CustomerId: &[]string{customerID},
			},
			expectedEventIDs: eventIDs,
		},
		{
			name: "Should filter events by customer key",
			params: api.ListEventsParams{
				Subject: &customerKey,
			},
			expectedEventIDs: []string{eventIDs[0]},
		},
		{
			name: "Should filter events by subject key 1",
			params: api.ListEventsParams{
				Subject: &subjectKeys[0],
			},
			expectedEventIDs: []string{eventIDs[1]},
		},
		{
			name: "Should filter events by subject key 2",
			params: api.ListEventsParams{
				Subject: &subjectKeys[1],
			},
			expectedEventIDs: []string{eventIDs[2]},
		},
	}

	for _, attributionFilter := range attributionFilters {
		t.Run(attributionFilter.name, func(t *testing.T) {
			params := attributionFilter.params
			params.From = lo.ToPtr(eventTime.Add(-time.Minute))
			params.Limit = lo.ToPtr(100)

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				response, err := client.ListEventsWithResponse(t.Context(), &params)
				require.NoError(collect, err)
				require.Equal(collect, http.StatusOK, response.StatusCode(), "response body: %s", response.Body)
				require.NotNil(collect, response.JSON200)

				events := *response.JSON200
				require.Len(collect, events, len(attributionFilter.expectedEventIDs))

				gotEventIDs := make([]string, 0, len(events))
				for _, event := range events {
					gotEventIDs = append(gotEventIDs, event.Event.ID())
					require.NotNil(collect, event.CustomerId)
					assert.Equal(collect, customerID, *event.CustomerId)
				}

				assert.ElementsMatch(collect, attributionFilter.expectedEventIDs, gotEventIDs)
			}, time.Minute, time.Second)
		})
	}
}
