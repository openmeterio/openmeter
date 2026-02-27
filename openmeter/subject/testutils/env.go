package testutils

import (
	"crypto/rand"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementpgadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	productcatalogadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

func NewTestULID(t *testing.T) string {
	t.Helper()

	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

var NewTestNamespace = NewTestULID

type TestEnv struct {
	Logger             *slog.Logger
	Tracer             trace.Tracer
	SubjectService     subject.Service
	CustomerService    customer.Service
	EntitlementAdapter entitlement.EntitlementRepo
	FeatureService     feature.FeatureConnector

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

	tracer := noop.NewTracerProvider().Tracer("test_env")

	// Init database
	db := testutils.InitPostgresDB(t)
	client := db.EntDriver.Client()

	// Init event publisher
	publisher := eventbus.NewMock(t)

	// Init meter service
	meterAdapter, err := meteradapter.New(nil)
	require.NoErrorf(t, err, "initializing meter adapter must not fail")
	require.NotNilf(t, meterAdapter, "meter adapter must not be nil")

	// Init feature service
	featureAdapter := productcatalogadapter.NewPostgresFeatureRepo(client, logger)
	featureService := feature.NewFeatureConnector(featureAdapter, meterAdapter, publisher, nil)

	// Entitlement Adapter
	entitlementDBAdapter := entitlementpgadapter.NewPostgresEntitlementRepo(client)
	require.NotNilf(t, entitlementDBAdapter, "entitlement adapter must not be nil")

	// Init subject service
	subjectAdapter, err := subjectadapter.New(client)
	require.NoErrorf(t, err, "initializing subject adapter must not fail")
	require.NotNilf(t, subjectAdapter, "subject adapter must not be nil")

	subjectService, err := subjectservice.New(subjectAdapter)
	require.NoErrorf(t, err, "initializing subject service must not fail")
	require.NotNilf(t, subjectService, "subject service must not be nil")

	// Init Customer service
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: client,
		Logger: logger,
	})
	require.NoErrorf(t, err, "initializing customer adapter must not fail")
	require.NotNilf(t, customerAdapter, "customer adapter must not be nil")

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: publisher,
	})
	require.NoErrorf(t, err, "initializing subject service must not fail")
	require.NotNilf(t, customerService, "subject service must not be nil")

	return &TestEnv{
		Logger:             logger,
		Tracer:             tracer,
		FeatureService:     featureService,
		SubjectService:     subjectService,
		CustomerService:    customerService,
		EntitlementAdapter: entitlementDBAdapter,
		Client:             client,
		db:                 db,
		close:              sync.Once{},
	}
}
