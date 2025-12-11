package adapter

import (
	"crypto/rand"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func Test_Adapter(t *testing.T) {
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	env.DBSchemaMigrate(t)

	namespace := NewTestNamespace(t)

	t.Run("Meter", func(t *testing.T) {
		t.Run("List", func(t *testing.T) {
			meterInputs := []meter.CreateMeterInput{
				{
					Namespace:     namespace,
					Name:          "Test meter 1",
					Key:           "test-meter-1",
					Aggregation:   meter.MeterAggregationSum,
					EventType:     "om.meter",
					ValueProperty: lo.ToPtr("$.value"),
					GroupBy: map[string]string{
						"group":   "$.group",
						"group_2": "$.group_2",
					},
				},
				{
					Namespace:     namespace,
					Name:          "Test meter 2",
					Key:           "test-meter-2",
					Aggregation:   meter.MeterAggregationCount,
					EventType:     "om.meter",
					ValueProperty: nil,
					GroupBy: map[string]string{
						"group":   "$.group",
						"group_2": "$.group_2",
					},
				},
			}

			for _, input := range meterInputs {
				_, err := env.Meter.CreateMeter(t.Context(), input)
				require.NoErrorf(t, err, "creating meter must not fail")
			}

			t.Run("FilterByEventTypes", func(t *testing.T) {
				out, err := env.Meter.ListMeters(t.Context(), meter.ListMetersParams{
					Page: pagination.Page{
						PageSize:   100,
						PageNumber: 1,
					},
					Namespace: namespace,
					EventTypes: lo.ToPtr([]string{
						"om.meter",
					}),
				})
				require.NoErrorf(t, err, "listing meters must not fail")

				require.Lenf(t, out.Items, 2, "expected 2 meters with event type om.meter, got %d", len(out.Items))

				for _, m := range out.Items {
					assert.Equalf(t, m.EventType, "om.meter", "expected meter event type om.meter, got %s", m.EventType)
				}
			})
		})
	})
}

type TestEnv struct {
	Logger *slog.Logger
	Meter  *Adapter

	Client *entdb.Client
	db     *testutils.TestDB
	close  sync.Once
}

func (e *TestEnv) DBSchemaMigrate(t *testing.T) {
	t.Helper()

	require.NotNilf(t, e.db, "database must be initialized")

	err := e.db.EntDriver.Client().Schema.Create(t.Context())
	require.NoErrorf(t, err, "schema migration must not fail")
}

func (e *TestEnv) Close(t *testing.T) {
	t.Helper()

	e.close.Do(func() {
		if e.db != nil {
			if err := e.db.EntDriver.Close(); err != nil {
				t.Errorf("failed to close ent driver: %v", err)
			}

			if err := e.db.PGDriver.Close(); err != nil {
				t.Errorf("failed to postgres driver: %v", err)
			}
		}

		if e.Client != nil {
			if err := e.Client.Close(); err != nil {
				t.Errorf("failed to close ent client: %v", err)
			}
		}
	})
}

func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	// Init logger
	logger := testutils.NewDiscardLogger(t)

	// Init database
	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	// Init meter service
	meterAdapter, err := New(Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing meter adapter must not fail")
	require.NotNilf(t, meterAdapter, "meter adapter must not be nil")

	return &TestEnv{
		Logger: logger,
		Meter:  meterAdapter,
		db:     db,
		Client: client,
	}
}

func NewTestULID(t *testing.T) string {
	t.Helper()

	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

var NewTestNamespace = NewTestULID
