package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
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

func TestIngest(t *testing.T) {
	client := initClient(t)

	timestamp, _ := time.Parse(time.RFC3339, "2023-12-04T08:37:23.151Z")
	var sum int

	for i := 0; i < 1000; i++ {
		timestamp := timestamp.Add(time.Second)
		duration := i + 1
		sum += duration

		ev := cloudevents.New()
		ev.SetID(faker.UUIDHyphenated())
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
	time.Sleep(15 * time.Second)

	resp, err := client.QueryMeterWithResponse(context.Background(), "ingest", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	require.Len(t, resp.JSON200.Data, 1)
	assert.Equal(t, float64(sum), resp.JSON200.Data[0].Value)
}

func TestDedupe(t *testing.T) {
	client := initClient(t)

	timestamp, _ := time.Parse(time.RFC3339, "2023-12-04T08:37:23.151Z")
	var sum int

	for i := 0; i < 1000; i++ {
		timestamp := timestamp.Add(time.Second)
		duration := i + 1
		sum += duration

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
	time.Sleep(15 * time.Second)

	resp, err := client.QueryMeterWithResponse(context.Background(), "dedupe", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())

	require.Len(t, resp.JSON200.Data, 1)
	assert.Equal(t, float64(1), resp.JSON200.Data[0].Value)
}
