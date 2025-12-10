package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	customerservicehooks "github.com/openmeterio/openmeter/openmeter/customer/service/hooks"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	subjecthooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type BaseSuite struct {
	suite.Suite
	*require.Assertions

	TestDB   *testutils.TestDB
	DBClient *db.Client

	BillingAdapter    billing.Adapter
	BillingService    billing.Service
	InvoiceCalculator *invoicecalc.MockableInvoiceCalculator

	FeatureService         feature.FeatureConnector
	FeatureRepo            feature.FeatureRepo
	MeterAdapter           *meteradapter.TestAdapter
	MockStreamingConnector *streamingtestutils.MockStreamingConnector

	CustomerService customer.Service
	SubjectService  subject.Service

	AppService app.Service
	SandboxApp *appsandbox.MockableFactory
}

// GetUniqueNamespace returns a unique namespace with the given prefix
func (s *BaseSuite) GetUniqueNamespace(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, ulid.Make().String())
}

func (b *BaseSuite) GetSubscriptionMixInDependencies() SubscriptionMixInDependencies {
	return SubscriptionMixInDependencies{
		DBClient:               b.DBClient,
		FeatureRepo:            b.FeatureRepo,
		FeatureService:         b.FeatureService,
		CustomerService:        b.CustomerService,
		MeterAdapter:           b.MeterAdapter,
		MockStreamingConnector: b.MockStreamingConnector,
	}
}

func (s *BaseSuite) SetupSuite() {
	t := s.T()
	t.Log("setup suite")
	s.Assertions = require.New(t)
	publisher := eventbus.NewMock(t)

	s.TestDB = testutils.InitPostgresDB(t)

	// init db
	dbClient := db.NewClient(db.Driver(s.TestDB.EntDriver.Driver()))
	s.DBClient = dbClient

	if os.Getenv("TEST_DISABLE_ATLAS") != "" {
		s.Require().NoError(dbClient.Schema.Create(context.Background()))
	} else {
		migrator, err := migrate.New(migrate.MigrateOptions{
			ConnectionString: s.TestDB.URL,
			Migrations:       migrate.OMMigrationsConfig,
			Logger:           testutils.NewLogger(t),
		})
		s.NoError(err)

		defer migrator.CloseOrLogError()

		s.NoError(migrator.Up())
	}

	// setup invoicing stack

	// Meter repo

	s.MockStreamingConnector = streamingtestutils.NewMockStreamingConnector(t)

	meterAdapter, err := meteradapter.New(nil)
	require.NoError(t, err)

	s.MeterAdapter = meterAdapter

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	// Entitlement
	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbClient,
		StreamingConnector: s.MockStreamingConnector,
		Logger:             slog.Default(),
		MeterService:       s.MeterAdapter,
		Publisher:          publisher,
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Locker: locker,
	})

	// Feature
	s.FeatureRepo = entitlementRegistry.FeatureRepo
	s.FeatureService = entitlementRegistry.Feature

	// Subject
	subjectAdapter, err := subjectadapter.New(dbClient)
	require.NoError(t, err)

	subjectService, err := subjectservice.New(subjectAdapter)
	require.NoError(t, err)
	s.SubjectService = subjectService

	// Customer

	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: publisher,
	})
	require.NoError(t, err)
	s.CustomerService = customerService

	// App
	appAdapter, err := appadapter.New(appadapter.Config{
		Client: dbClient,
	})
	require.NoError(t, err)

	appService, err := appservice.New(appservice.Config{
		Adapter:   appAdapter,
		Publisher: publisher,
	})
	require.NoError(t, err)
	s.AppService = appService

	// Billing
	billingAdapter, err := billingadapter.New(billingadapter.Config{
		Client: dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)
	s.BillingAdapter = billingAdapter

	billingService, err := billingservice.New(billingservice.Config{
		Adapter:                      billingAdapter,
		CustomerService:              s.CustomerService,
		AppService:                   s.AppService,
		Logger:                       slog.Default(),
		FeatureService:               s.FeatureService,
		MeterService:                 s.MeterAdapter,
		StreamingConnector:           s.MockStreamingConnector,
		Publisher:                    publisher,
		AdvancementStrategy:          billing.ForegroundAdvancementStrategy,
		MaxParallelQuantitySnapshots: 2,
	})
	require.NoError(t, err)

	s.InvoiceCalculator = invoicecalc.NewMockableCalculator(t, billingService.InvoiceCalculator())

	s.BillingService = billingService.WithInvoiceCalculator(s.InvoiceCalculator)

	// OpenMeter sandbox (registration as side-effect)
	sandboxApp, err := appsandbox.NewMockableFactory(t, appsandbox.Config{
		AppService:     appService,
		BillingService: s.BillingService,
	})
	require.NoError(t, err)

	s.SandboxApp = sandboxApp

	// Hooks

	// Subject hooks

	subjectCustomerHook, err := subjecthooks.NewCustomerSubjectHook(subjecthooks.CustomerSubjectHookConfig{
		Subject: subjectService,
		Logger:  slog.Default(),
		Tracer:  noop.NewTracerProvider().Tracer("test_env"),
	})
	require.NoError(t, err)
	customerService.RegisterHooks(subjectCustomerHook)

	// customer hooks
	customerSubjectHook, err := customerservicehooks.NewSubjectCustomerHook(customerservicehooks.SubjectCustomerHookConfig{
		Customer:         customerService,
		CustomerOverride: billingService,
		Logger:           slog.Default(),
		Tracer:           noop.NewTracerProvider().Tracer("test_env"),
	})
	require.NoError(t, err)
	subjectService.RegisterHooks(customerSubjectHook)

	entitlementValidatorHook, err := customerservicehooks.NewEntitlementValidatorHook(customerservicehooks.EntitlementValidatorHookConfig{
		EntitlementService: entitlementRegistry.Entitlement,
	})
	require.NoError(t, err)
	customerService.RegisterHooks(entitlementValidatorHook)
}

func (s *BaseSuite) InstallSandboxApp(t *testing.T, ns string) app.App {
	ctx := context.Background()
	appBase, err := s.AppService.CreateApp(ctx,
		app.CreateAppInput{
			Name:        "Sandbox",
			Description: "Sandbox app",
			Type:        app.AppTypeSandbox,
			Namespace:   ns,
		})

	require.NoError(t, err)

	sandboxApp, err := s.AppService.GetApp(ctx, app.GetAppInput{
		Namespace: ns,
		ID:        appBase.ID,
	})
	require.NoError(t, err)

	return sandboxApp
}

func (s *BaseSuite) CreateTestCustomer(ns string, subjectKey string) *customer.Customer {
	s.T().Helper()

	customer, err := s.CustomerService.CreateCustomer(context.Background(), customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			BillingAddress: &models.Address{
				Country:    lo.ToPtr(models.CountryCode("US")),
				PostalCode: lo.ToPtr("12345"),
			},
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{subjectKey},
			},
		},
	})

	s.NoError(err)
	return customer
}

func (s *BaseSuite) TearDownSuite() {
	s.TestDB.EntDriver.Close()
	s.TestDB.PGDriver.Close()
}

func (s *BaseSuite) DebugDumpInvoice(h string, i billing.Invoice) {
	s.T().Log(h)

	l := i.Lines.OrEmpty()

	slices.SortFunc(l, func(l1, l2 *billing.Line) int {
		if l1.Period.Start.Before(l2.Period.Start) {
			return -1
		} else if l1.Period.Start.After(l2.Period.Start) {
			return 1
		}
		return 0
	})

	for _, line := range i.Lines.OrEmpty() {
		deleted := ""
		if line.DeletedAt != nil {
			deleted = " (deleted)"
		}

		priceJson, err := json.Marshal(line.UsageBased.Price)
		s.NoError(err)

		s.T().Logf("usage[%s..%s] childUniqueReferenceID: %s, invoiceAt: %s, qty: %s, price: %s (total=%s) %s\n",
			line.Period.Start.Format(time.RFC3339),
			line.Period.End.Format(time.RFC3339),
			lo.FromPtrOr(line.ChildUniqueReferenceID, "null"),
			line.InvoiceAt.Format(time.RFC3339),
			line.UsageBased.Quantity,
			string(priceJson),
			line.Totals.Total.String(),
			deleted)
	}
}

type DraftInvoiceInput struct {
	Namespace string
	Customer  *customer.Customer
}

func (i DraftInvoiceInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Customer == nil {
		return errors.New("customer is required")
	}

	if err := i.Customer.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *BaseSuite) CreateGatheringInvoice(t *testing.T, ctx context.Context, in DraftInvoiceInput) {
	s.NoError(in.Validate())

	namespace := in.Customer.Namespace

	now := clock.Now()
	invoiceAt := now.Add(-time.Second)
	periodEnd := now.Add(-24 * time.Hour)
	periodStart := periodEnd.Add(-24 * 30 * time.Hour)
	// Given we have a default profile for the namespace

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: in.Customer.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []*billing.Line{
				billing.NewFlatFeeLine(
					billing.NewFlatFeeLineInput{
						Namespace:     namespace,
						Period:        billing.Period{Start: periodStart, End: periodEnd},
						InvoiceAt:     invoiceAt,
						ManagedBy:     billing.ManuallyManagedLine,
						Name:          "Test item1",
						PerUnitAmount: alpacadecimal.NewFromFloat(100),
						Currency:      currencyx.Code(currency.USD),
						Metadata: map[string]string{
							"key": "value",
						},
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					},
				),
				billing.NewFlatFeeLine(
					billing.NewFlatFeeLineInput{
						Namespace:     namespace,
						Period:        billing.Period{Start: periodStart, End: periodEnd},
						InvoiceAt:     invoiceAt,
						ManagedBy:     billing.ManuallyManagedLine,
						Name:          "Test item2",
						PerUnitAmount: alpacadecimal.NewFromFloat(200),
						Currency:      currencyx.Code(currency.USD),
						Metadata: map[string]string{
							"key": "value",
						},
						PaymentTerm: productcatalog.InArrearsPaymentTerm,
					},
				),
			},
		})

	require.NoError(s.T(), err)
	require.Len(s.T(), res.Lines, 2)
	line1ID := res.Lines[0].ID
	line2ID := res.Lines[1].ID
	require.NotEmpty(s.T(), line1ID)
	require.NotEmpty(s.T(), line2ID)
}

func (s *BaseSuite) CreateDraftInvoice(t *testing.T, ctx context.Context, in DraftInvoiceInput) billing.Invoice {
	s.NoError(in.Validate())

	s.CreateGatheringInvoice(t, ctx, in)

	now := clock.Now()
	invoice, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customer.CustomerID{
			ID:        in.Customer.ID,
			Namespace: in.Customer.Namespace,
		},
		AsOf: lo.ToPtr(now),
	})

	require.NoError(t, err)
	require.Len(t, invoice, 1)
	require.Len(t, invoice[0].Lines.MustGet(), 2)

	return invoice[0]
}

type TestFeature struct {
	Cleanup func()
	Feature feature.Feature
}

func (s *BaseSuite) SetupApiRequestsTotalFeature(ctx context.Context, ns string) TestFeature {
	apiRequestsTotalMeterSlug := "api-requests-total"

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "API Requests Total",
			},
			Key:           apiRequestsTotalMeterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	})
	s.NoError(err, "Replacing meters must not return error")

	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotalMeterSlug, 0, time.Now())

	apiRequestsTotalFeatureKey := "api-requests-total"

	apiRequestsTotalFeature, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: ns,
		Name:      "api-requests-total",
		Key:       apiRequestsTotalFeatureKey,
		MeterSlug: lo.ToPtr("api-requests-total"),
	})
	s.NoError(err)

	return TestFeature{
		Cleanup: func() {
			err = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
			s.NoError(err, "failed to replace meters")

			s.MockStreamingConnector.Reset()
		},
		Feature: apiRequestsTotalFeature,
	}
}

type BillingProfileEditFn func(p *billing.CreateProfileInput)

type BillingProfileProvisionOptions struct {
	editFn BillingProfileEditFn
}

type BillingProfileProvisionOption func(*BillingProfileProvisionOptions)

func WithBillingProfileEditFn(editFn BillingProfileEditFn) BillingProfileProvisionOption {
	return func(opts *BillingProfileProvisionOptions) {
		opts.editFn = editFn
	}
}

func WithProgressiveBilling() BillingProfileProvisionOption {
	return WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Invoicing.ProgressiveBilling = true
	})
}

func WithCollectionInterval(period datetime.ISODuration) BillingProfileProvisionOption {
	return WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
		p.WorkflowConfig.Collection.Interval = period
	})
}

func (s *BaseSuite) ProvisionBillingProfile(ctx context.Context, ns string, appID app.AppID, opts ...BillingProfileProvisionOption) *billing.Profile {
	provisionOpts := BillingProfileProvisionOptions{}

	for _, opt := range opts {
		opt(&provisionOpts)
	}

	clonedCreateProfileInput := minimalCreateProfileInputTemplate(appID)
	clonedCreateProfileInput.Namespace = ns

	if provisionOpts.editFn != nil {
		provisionOpts.editFn(&clonedCreateProfileInput)
	}

	profile, err := s.BillingService.CreateProfile(ctx, clonedCreateProfileInput)
	s.NoError(err)

	return profile
}

func ExpectJSONEqual(t *testing.T, exp, actual any) {
	t.Helper()

	aJSON, err := json.Marshal(exp)
	require.NoError(t, err)

	bJSON, err := json.Marshal(actual)
	require.NoError(t, err)

	require.JSONEq(t, string(aJSON), string(bJSON))
}
