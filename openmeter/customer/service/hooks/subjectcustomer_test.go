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
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

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

func TestCustomerProvisioner_EnsureCustomer(t *testing.T) {
	// Setup test environment
	env := NewTestEnv(t)
	t.Cleanup(func() {
		env.Close(t)
	})

	// Run database migrations
	env.DBSchemaMigrate(t)

	// Get new namespace ID
	namespace := NewTestNamespace(t)

	provisioner := CustomerProvisioner{
		customer:         env.CustomerService,
		customerOverride: nil,
		logger:           env.Logger,
		tracer:           env.Tracer,
	}

	ctx := t.Context()

	t.Run("Create", func(t *testing.T) {
		sub, err := env.SubjectService.Create(ctx, subject.CreateInput{
			Namespace:        namespace,
			Key:              "acme-inc",
			DisplayName:      lo.ToPtr("ACME Inc."),
			StripeCustomerId: lo.ToPtr("cus_abcdefgh"),
		})
		require.NoError(t, err, "creating subject should not fail")
		assert.NotNilf(t, sub, "subject must not be nil")

		cus, err := provisioner.getCustomerForSubject(ctx, &sub)
		require.ErrorAsf(t, err, new(*models.GenericNotFoundError), "error must be not found error")
		assert.Nilf(t, cus, "customer must be nil")

		cus, err = provisioner.EnsureCustomer(ctx, &sub)
		require.NoError(t, err, "provisioning customer should not fail")
		assert.NotNilf(t, cus, "customer must not be nil")

		cus, err = env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &customer.CustomerID{
				Namespace: cus.Namespace,
				ID:        cus.ID,
			},
		})
		require.NoErrorf(t, err, "getting customer for subject should not fail")
		assert.NotNilf(t, cus, "customer must not be nil")
		AssertSubjectCustomerStrictEqual(t, &sub, cus)

		t.Run("Update", func(t *testing.T) {
			sub, err = env.SubjectService.Update(ctx, subject.UpdateInput{
				ID:        sub.Id,
				Namespace: sub.Namespace,
				DisplayName: subject.OptionalNullable[string]{
					Value: lo.ToPtr("ACME2 Inc."),
					IsSet: true,
				},
				StripeCustomerId: subject.OptionalNullable[string]{
					Value: lo.ToPtr("cus_12345678"),
					IsSet: true,
				},
				Metadata: subject.OptionalNullable[map[string]interface{}]{
					Value: lo.ToPtr(map[string]interface{}{
						"foo": "bar",
						"bar": 1,
						"baz": false,
					}),
					IsSet: true,
				},
			})
			require.NoError(t, err, "updating subject should not fail")
			assert.NotNilf(t, sub, "subject must not be nil")

			cus, err = env.CustomerService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
				CustomerID: customer.CustomerID{
					Namespace: cus.Namespace,
					ID:        cus.ID,
				},
				CustomerMutate: customer.CustomerMutate{
					Key:  lo.ToPtr("example-corp"),
					Name: "Example Corporation",
					Metadata: lo.ToPtr(models.Metadata{
						"buz": "qux",
					}),
					UsageAttribution: cus.UsageAttribution,
				},
			})
			require.NoError(t, err, "updating customer should not fail")
			assert.NotNilf(t, sub, "customer must not be nil")

			cus, err = provisioner.EnsureCustomer(ctx, &sub)
			require.NoError(t, err, "provisioning customer should not fail")
			assert.NotNilf(t, cus, "customer must not be nil")

			cus, err = env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &customer.CustomerID{
					Namespace: cus.Namespace,
					ID:        cus.ID,
				},
			})
			require.NoErrorf(t, err, "getting customer for subject should not fail")
			assert.NotNilf(t, cus, "customer must not be nil")
			AssertSubjectCustomerEqual(t, &sub, cus)
		})
	})

	t.Run("Conflict", func(t *testing.T) {
		t.Run("CustomerUsageAttributionMismatch", func(t *testing.T) {
			sub, err := env.SubjectService.Create(ctx, subject.CreateInput{
				Namespace:        namespace,
				Key:              "org-1",
				DisplayName:      lo.ToPtr("Org. 1"),
				StripeCustomerId: lo.ToPtr("cus_abcdefgh"),
				Metadata: lo.ToPtr(map[string]interface{}{
					"foo": "bar",
					"bar": 1,
					"baz": true,
				}),
			})
			require.NoError(t, err, "creating subject should not fail")
			assert.NotNilf(t, sub, "subject must not be nil")

			cusForSubject, err := provisioner.EnsureCustomer(ctx, &sub)
			require.NoError(t, err, "provisioning customer should not fail")
			assert.NotNilf(t, cusForSubject, "customer must not be nil")

			_, err = env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
				Namespace: namespace,
				CustomerMutate: customer.CustomerMutate{
					Key:         lo.ToPtr(sub.Key),
					Name:        "Org. 1",
					Description: nil,
					UsageAttribution: &customer.CustomerUsageAttribution{
						SubjectKeys: []string{
							"not-" + sub.Key,
						},
					},
					PrimaryEmail:   nil,
					Currency:       nil,
					BillingAddress: nil,
					Metadata: &models.Metadata{
						"baz": "qux",
					},
					Annotation: nil,
				},
			})

			require.True(t, models.IsGenericConflictError(err), "creating customer should fail with conflict")
		})

		t.Run("CustomerKeyMismatch", func(t *testing.T) {
			sub, err := env.SubjectService.Create(ctx, subject.CreateInput{
				Namespace:        namespace,
				Key:              "org-10",
				DisplayName:      lo.ToPtr("Org. 10"),
				StripeCustomerId: lo.ToPtr("cus_abcdefgh"),
				Metadata: lo.ToPtr(map[string]interface{}{
					"foo": "bar",
					"bar": 1,
					"baz": true,
				}),
			})
			require.NoError(t, err, "creating subject should not fail")
			assert.NotNilf(t, sub, "subject must not be nil")

			cus, err := env.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
				Namespace: namespace,
				CustomerMutate: customer.CustomerMutate{
					Key:         lo.ToPtr("not-" + sub.Key),
					Name:        "Org. 30",
					Description: nil,
					UsageAttribution: &customer.CustomerUsageAttribution{
						SubjectKeys: []string{
							sub.Key,
						},
					},
					PrimaryEmail:   nil,
					Currency:       nil,
					BillingAddress: nil,
					Metadata: &models.Metadata{
						"baz": "qux",
					},
					Annotation: nil,
				},
			})
			require.NoError(t, err, "creating customer should not fail")
			assert.NotNilf(t, cus, "customer must not be nil")

			cus, err = provisioner.EnsureCustomer(ctx, &sub)
			require.NoError(t, err, "provisioning customer should not fail")
			assert.NotNilf(t, cus, "customer must not be nil")

			cus, err = env.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &customer.CustomerID{
					Namespace: cus.Namespace,
					ID:        cus.ID,
				},
			})
			require.NoErrorf(t, err, "getting customer for subject should not fail")
			assert.NotNilf(t, cus, "customer must not be nil")
			AssertSubjectCustomerEqual(t, &sub, cus)
		})
	})
}

func AssertSubjectCustomerStrictEqual(t *testing.T, s *subject.Subject, c *customer.Customer) {
	t.Helper()

	assert.Equalf(t, s.Key, lo.FromPtr(c.Key), "subject key must be equal to customer key")

	AssertSubjectCustomerEqual(t, s, c)
}

func AssertSubjectCustomerEqual(t *testing.T, s *subject.Subject, c *customer.Customer) {
	t.Helper()

	assert.Equalf(t, s.Namespace, c.Namespace, "subject namespace must be equal to customer namespace")
	assert.Containsf(t, c.UsageAttribution.SubjectKeys, s.Key, "customer usage attribute must contain subject key")
	assert.Equalf(t, lo.FromPtr(s.DisplayName), c.Name, "subject display name must be equal to customer display name")

	sm := MetadataFromMap(s.Metadata)
	cm := lo.FromPtr(c.Metadata)
	for k, v := range sm {
		vv, ok := cm[k]
		assert.Truef(t, ok, "customer metadata must contain subject metadata key %s", k)
		assert.Equalf(t, v, vv, "customer metadata value must be equal to subject metadata value")
	}
}

func NewTestULID(t *testing.T) string {
	t.Helper()

	return ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String()
}

var NewTestNamespace = NewTestULID

type TestEnv struct {
	Logger          *slog.Logger
	Tracer          trace.Tracer
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
		Tracer:          tracer,
		SubjectService:  subjectService,
		CustomerService: customerService,
		Client:          client,
		db:              db,
		close:           sync.Once{},
	}
}

func TestMetadataFromMap(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected models.Metadata
	}{
		{
			name: "string",
			metadata: map[string]interface{}{
				"foo": "bar",
			},
			expected: models.Metadata{
				"foo": "bar",
			},
		},
		{
			name: "skip",
			metadata: map[string]interface{}{
				"a":  true,
				"b":  "baz",
				"c":  nil,
				"d1": 1,
				"d2": int8(2),
				"d3": int16(3),
				"d4": int32(4),
				"d5": int64(5),
				"e":  []string{"bar", "buzz"},
				"f1": float32(3.14),
				"f2": 3.14,
				"g": map[string]interface{}{
					"foo": "bar",
					"bar": "baz",
				},
				"h": lo.ToPtr(true),
			},
			expected: models.Metadata{
				"a":  "true",
				"b":  "baz",
				"d1": "1",
				"d2": "2",
				"d3": "3",
				"d4": "4",
				"d5": "5",
				"e":  `"bar","buzz"`,
				"f1": "3.14",
				"f2": "3.14",
				"g":  `"bar"="baz","foo"="bar"`,
				"h":  "true",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := MetadataFromMap(test.metadata)

			assert.Equalf(t, test.expected, actual, "expected: %v, actual: %v", test.expected, actual)
		})
	}
}
