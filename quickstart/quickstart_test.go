package quickstart

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	//go:embed examples/first.json
	firstExample []byte

	//go:embed examples/second.json
	secondExample []byte

	//go:embed examples/third.json
	thirdExample []byte
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

func TestQuickstart(t *testing.T) {
	client := initClient(t)

	{
		var ev cloudevents.Event

		err := json.Unmarshal(firstExample, &ev)
		require.NoError(t, err)

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.IngestEventWithResponse(context.Background(), ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}, 30*time.Second, time.Second)
	}

	{
		var ev cloudevents.Event

		err := json.Unmarshal(secondExample, &ev)
		require.NoError(t, err)

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.IngestEventWithResponse(context.Background(), ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}, 30*time.Second, time.Second)
	}

	{
		var ev cloudevents.Event

		err := json.Unmarshal(thirdExample, &ev)
		require.NoError(t, err)

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.IngestEventWithResponse(context.Background(), ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}, 30*time.Second, time.Second)
	}

	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		windowSize := models.WindowSizeHour

		resp, err := client.QueryMeterWithResponse(context.Background(), "api_requests_total", &api.QueryMeterParams{
			WindowSize: &windowSize,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())

		require.Len(t, resp.JSON200.Data, 2)
		assert.Equal(t, float64(2), resp.JSON200.Data[0].Value)
		assert.Equal(t, float64(1), resp.JSON200.Data[1].Value)
	}, 30*time.Second, time.Second)
}
