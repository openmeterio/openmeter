package hooks

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

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

func Test_CustomerSubjectHook(t *testing.T) {
	// Setup test environment
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := NewTestNamespace(t)

	ctx := t.Context()

	hook, err := NewCustomerSubjectHook(CustomerSubjectHookConfig{
		Subject: env.SubjectService,
		Logger:  env.Logger,
	})
	require.NoError(t, err, "creating customer subject provisioner hook should not fail")
	require.NotNilf(t, hook, "customer subject provisioner hook must not be nil")

	env.CustomerService.RegisterHooks(hook)

	t.Run("Create", func(t *testing.T) {
		t.Run("WithExistingSubject", func(t *testing.T) {
			sub, err := env.SubjectService.Create(ctx, subject.CreateInput{
				Namespace:   namespace,
				Key:         "example-inc",
				DisplayName: lo.ToPtr("Example Inc."),
			})
			require.NoError(t, err, "creating subject should not fail")
			assert.NotNilf(t, sub, "subject must not be nil")

			cus, err := env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
				Namespace: namespace,
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("example-inc"),
					Name: "Example Inc.",
					UsageAttribution: customer.CustomerUsageAttribution{
						SubjectKeys: []string{
							"example-inc",
						},
					},
				},
			})
			require.NoErrorf(t, err, "getting customer for subject should not fail")
			assert.NotNilf(t, cus, "customer must not be nil")

			for _, subKey := range cus.UsageAttribution.SubjectKeys {
				t.Run("GetSubject", func(t *testing.T) {
					sub, err = env.SubjectService.GetByKey(ctx, models.NamespacedKey{
						Namespace: namespace,
						Key:       subKey,
					})
					require.NoError(t, err, "getting subject should not fail")
					assert.NotNilf(t, sub, "subject must not be nil")

					assert.Equalf(t, subKey, sub.Key, "subject key must be equal to usage attribution key")
				})
			}
		})

		t.Run("WithoutExistingSubject", func(t *testing.T) {
			cus, err := env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
				Namespace: namespace,
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("acme-inc"),
					Name: "ACME Inc.",
					UsageAttribution: customer.CustomerUsageAttribution{
						SubjectKeys: []string{
							"acme-inc",
						},
					},
				},
			})
			require.NoErrorf(t, err, "getting customer for subject should not fail")
			assert.NotNilf(t, cus, "customer must not be nil")

			for _, subKey := range cus.UsageAttribution.SubjectKeys {
				t.Run("GetSubject", func(t *testing.T) {
					sub, err := env.SubjectService.GetByKey(ctx, models.NamespacedKey{
						Namespace: namespace,
						Key:       subKey,
					})
					require.NoError(t, err, "getting subject should not fail")
					assert.NotNilf(t, sub, "subject must not be nil")

					assert.Equalf(t, subKey, sub.Key, "subject key must be equal to usage attribution key")
				})
			}
		})
	})
}

func NewTestULID(t *testing.T) string {
	t.Helper()

	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

var NewTestNamespace = NewTestULID

type TestEnv struct {
	Logger          *slog.Logger
	SubjectService  subject.Service
	CustomerService customer.Service

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

	// Init event publisher
	publisher := eventbus.NewMock(t)

	// Init meter service
	meterAdapter, err := meteradapter.New(nil)
	require.NoErrorf(t, err, "initializing meter adapter must not fail")
	require.NotNilf(t, meterAdapter, "meter adapter must not be nil")

	// Init subject service
	subjectAdapter, err := subjectadapter.New(client)
	require.NoErrorf(t, err, "initializing subject adapter must not fail")
	require.NotNilf(t, subjectAdapter, "subject adapter must not be nil")

	subjectService, err := subjectservice.New(subjectAdapter)
	require.NoErrorf(t, err, "initializing subject service must not fail")
	require.NotNilf(t, subjectAdapter, "subject service must not be nil")

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
	require.NotNilf(t, subjectAdapter, "subject service must not be nil")

	return &TestEnv{
		Logger:          logger,
		SubjectService:  subjectService,
		CustomerService: customerService,
		Client:          client,
		db:              db,
		close:           sync.Once{},
	}
}
