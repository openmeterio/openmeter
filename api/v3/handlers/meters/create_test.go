package meters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	meterservice "github.com/openmeterio/openmeter/openmeter/meter/service"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type rejectHyphenatedKeyValidator struct{}

func (rejectHyphenatedKeyValidator) ValidateJSONPath(_ context.Context, jsonPath string) (bool, error) {
	return jsonPath != "$.transaction-count", nil
}

func TestCreateMeterJSONPathValidation(t *testing.T) {
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() {
		require.NoError(t, database.EntDriver.Close())
		require.NoError(t, database.PGDriver.Close())
	})

	meterAdapter, err := meteradapter.New(meteradapter.Config{
		Client: database.EntDriver.Client(),
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)

	namespaceManager, err := namespace.NewManager(namespace.ManagerConfig{DefaultNamespace: "default"})
	require.NoError(t, err)

	service := meterservice.NewManage(meterAdapter, eventbus.NewMock(t), namespaceManager, nil, rejectHyphenatedKeyValidator{})

	createMeter := func(t *testing.T, body string) *httptest.ResponseRecorder {
		t.Helper()

		ns := ulid.Make().String()
		h := New(func(context.Context) (string, error) { return ns, nil }, service, nil, nil)

		request := httptest.NewRequest(http.MethodPost, "/api/v3/openmeter/meters", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		h.CreateMeter().ServeHTTP(response, request)

		return response
	}

	t.Run("value property the event store cannot evaluate is a bad request", func(t *testing.T) {
		// When a meter is created with a value property ClickHouse cannot parse.
		response := createMeter(t, `{"key":"transactions","name":"Transactions","event_type":"transaction","aggregation":"sum","value_property":"$.transaction-count"}`)

		// Then the client gets a 400 naming the path.
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), "invalid value property JSONPath: $.transaction-count")
	})

	t.Run("dimension the event store cannot evaluate is a bad request", func(t *testing.T) {
		// When a meter is created with a valid value property but an unparseable dimension.
		response := createMeter(t, `{"key":"transactions","name":"Transactions","event_type":"transaction","aggregation":"sum","value_property":"$.amount","dimensions":{"count":"$.transaction-count"}}`)

		// Then the client gets a 400 naming the dimension.
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), "invalid group by count JSONPath: $.transaction-count")
	})

	t.Run("valid value property and dimensions are created", func(t *testing.T) {
		// When a meter is created with paths ClickHouse can parse, including bracket notation.
		response := createMeter(t, `{"key":"transactions","name":"Transactions","event_type":"transaction","aggregation":"sum","value_property":"$['transaction-count']","dimensions":{"region":"$.region"}}`)

		// Then it is created.
		require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	})
}
