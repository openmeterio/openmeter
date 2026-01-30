package customer

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	customerhooks "github.com/openmeterio/openmeter/openmeter/customer/service/hooks"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entcustomervalidator "github.com/openmeterio/openmeter/openmeter/entitlement/validators/customer"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	meterservice "github.com/openmeterio/openmeter/openmeter/meter/service"
	addonrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	addonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	planpkg "github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	planaddonrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/adapter"
	planaddonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/service"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	subject "github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	subjecthooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddonrepo "github.com/openmeterio/openmeter/openmeter/subscription/addon/repo"
	subscriptionaddonservice "github.com/openmeterio/openmeter/openmeter/subscription/addon/service"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	annotationhook "github.com/openmeterio/openmeter/openmeter/subscription/hooks/annotations"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptioncustomer "github.com/openmeterio/openmeter/openmeter/subscription/validators/customer"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	subscriptionworkflowservice "github.com/openmeterio/openmeter/openmeter/subscription/workflow/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	App() app.Service
	Customer() customer.Service
	Subscription() subscription.Service
	SubscriptionWorkflow() subscriptionworkflow.Service
	Entitlement() entitlement.Service
	Feature() feature.FeatureConnector
	Subject() subject.Service
	Plan() planpkg.Service
	Billing() billing.Service
	Meter() *meteradapter.Adapter

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	app                  app.Service
	customer             customer.Service
	subscription         subscription.Service
	subscriptionWorkflow subscriptionworkflow.Service
	entitlement          entitlement.Service
	feature              feature.FeatureConnector
	subject              subject.Service
	meter                *meteradapter.Adapter
	plan                 planpkg.Service
	billing              billing.Service

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) App() app.Service {
	return n.app
}

func (n testEnv) Customer() customer.Service {
	return n.customer
}

func (n testEnv) Subscription() subscription.Service {
	return n.subscription
}

func (n testEnv) SubscriptionWorkflow() subscriptionworkflow.Service {
	return n.subscriptionWorkflow
}

func (n testEnv) Entitlement() entitlement.Service {
	return n.entitlement
}

func (n testEnv) Feature() feature.FeatureConnector {
	return n.feature
}

func (n testEnv) Subject() subject.Service {
	return n.subject
}

func (n testEnv) Plan() planpkg.Service {
	return n.plan
}

func (n testEnv) Billing() billing.Service {
	return n.billing
}

func (n testEnv) Meter() *meteradapter.Adapter {
	return n.meter
}

const (
	DefaultPostgresHost = "127.0.0.1"
)

func NewTestEnv(t *testing.T, ctx context.Context) (TestEnv, error) {
	logger := slog.Default().WithGroup("customer")
	publisher := eventbus.NewMock(t)

	// Initialize postgres driver
	dbDeps := subscriptiontestutils.SetupDBDeps(t)

	// Streaming
	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)

	// Meter
	meterAdapter, err := meteradapter.New(meteradapter.Config{
		Client: dbDeps.DBClient,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create meter adapter: %w", err)
	}

	meterService := meterservice.New(meterAdapter)

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	// Customer
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbDeps.DBClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer adapter: %w", err)
	}

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: publisher,
	})
	if err != nil {
		return nil, err
	}

	// Entitlement
	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbDeps.DBClient,
		StreamingConnector: streamingConnector,
		Logger:             logger,
		MeterService:       meterService,
		CustomerService:    customerService,
		Publisher:          publisher,
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Locker: locker,
		Tracer: noop.NewTracerProvider().Tracer("test_env"),
	})

	// Customer hooks

	entValidator, err := entcustomervalidator.NewValidator(entitlementRegistry.EntitlementRepo)
	if err != nil {
		return nil, err
	}

	customerService.RegisterRequestValidator(entValidator)

	// Subject
	subjectAdapter, err := subjectadapter.New(dbDeps.DBClient)
	if err != nil {
		return nil, err
	}

	subjectService, err := subjectservice.New(subjectAdapter)
	if err != nil {
		return nil, err
	}

	subjectCustomerHook, err := customerhooks.NewSubjectCustomerHook(customerhooks.SubjectCustomerHookConfig{
		Customer:         customerService,
		CustomerOverride: noopCustomerOverrideService{},
		Logger:           logger,
		Tracer:           noop.NewTracerProvider().Tracer("test_env"),
	})
	if err != nil {
		return nil, err
	}

	subjectService.RegisterHooks(subjectCustomerHook)

	customerSubjectHook, err := subjecthooks.NewCustomerSubjectHook(subjecthooks.CustomerSubjectHookConfig{
		Subject: subjectService,
		Logger:  logger,
		Tracer:  noop.NewTracerProvider().Tracer("test_env"),
	})
	if err != nil {
		return nil, err
	}

	customerService.RegisterHooks(customerSubjectHook)

	// Plan
	planAdapter, err := planadapter.New(planadapter.Config{
		Client: dbDeps.DBClient,
		Logger: logger.WithGroup("plan"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plan adapter: %w", err)
	}

	planService, err := planservice.New(planservice.Config{
		Adapter:   planAdapter,
		Feature:   entitlementRegistry.Feature,
		Logger:    logger.WithGroup("plan"),
		Publisher: publisher,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create plan service: %w", err)
	}

	// App
	appAdapter, err := appadapter.New(appadapter.Config{
		Client: dbDeps.DBClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app adapter: %w", err)
	}

	appService, err := appservice.New(appservice.Config{
		Adapter:   appAdapter,
		Publisher: publisher,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app service: %w", err)
	}

	// Subscription
	ffService := ffx.NewTestContextService(ffx.AccessConfig{
		subscription.MultiSubscriptionEnabledFF: false,
	})

	subRepo := subscriptionrepo.NewSubscriptionRepo(dbDeps.DBClient)
	subPhaseRepo := subscriptionrepo.NewSubscriptionPhaseRepo(dbDeps.DBClient)
	subItemRepo := subscriptionrepo.NewSubscriptionItemRepo(dbDeps.DBClient)

	entitlementAdapter := subscriptionentitlement.NewSubscriptionEntitlementAdapter(
		entitlementRegistry.Entitlement,
		subItemRepo,
		subPhaseRepo,
	)

	svc, err := subscriptionservice.New(subscriptionservice.ServiceConfig{
		SubscriptionRepo:      subRepo,
		SubscriptionPhaseRepo: subPhaseRepo,
		SubscriptionItemRepo:  subItemRepo,
		CustomerService:       customerService,
		EntitlementAdapter:    entitlementAdapter,
		FeatureService:        entitlementRegistry.Feature,
		TransactionManager:    subItemRepo,
		Publisher:             publisher,
		Lockr:                 locker,
		FeatureFlags:          ffService,
	})
	require.NoError(t, err)

	addonRepo, err := addonrepo.New(addonrepo.Config{
		Client: dbDeps.DBClient,
		Logger: logger,
	})
	require.NoError(t, err)

	addonService, err := addonservice.New(addonservice.Config{
		Adapter:   addonRepo,
		Logger:    logger,
		Publisher: publisher,
		Feature:   entitlementRegistry.Feature,
	})
	require.NoError(t, err)

	planAddonRepo, err := planaddonrepo.New(planaddonrepo.Config{
		Client: dbDeps.DBClient,
		Logger: logger,
	})
	require.NoError(t, err)

	planAddonService, err := planaddonservice.New(planaddonservice.Config{
		Adapter:   planAddonRepo,
		Logger:    logger,
		Plan:      planService,
		Addon:     addonService,
		Publisher: publisher,
	})
	require.NoError(t, err)
	subAddRepo := subscriptionaddonrepo.NewSubscriptionAddonRepo(dbDeps.DBClient)
	subAddQtyRepo := subscriptionaddonrepo.NewSubscriptionAddonQuantityRepo(dbDeps.DBClient)

	subAddSvc, err := subscriptionaddonservice.NewService(subscriptionaddonservice.Config{
		TxManager:        subItemRepo,
		Logger:           logger,
		AddonService:     addonService,
		SubService:       svc,
		SubAddRepo:       subAddRepo,
		SubAddQtyRepo:    subAddQtyRepo,
		PlanAddonService: planAddonService,
		Publisher:        publisher,
	})
	require.NoError(t, err)

	annotationCleanupHook, err := annotationhook.NewAnnotationCleanupHook(svc, subRepo, logger)
	require.NoError(t, err)
	require.NoError(t, svc.RegisterHook(annotationCleanupHook))

	workflowSvc := subscriptionworkflowservice.NewWorkflowService(subscriptionworkflowservice.WorkflowServiceConfig{
		Service:            svc,
		CustomerService:    customerService,
		TransactionManager: subItemRepo,
		AddonService:       subAddSvc,
		Logger:             logger.With("subsystem", "subscription.workflow.service"),
		Lockr:              locker,
		FeatureFlags:       ffService,
	})

	subsCustValidator, err := subscriptioncustomer.NewValidator(svc, customerService)
	if err != nil {
		return nil, err
	}

	customerService.RegisterRequestValidator(subsCustValidator)

	// Billing
	billingAdapter, err := billingadapter.New(billingadapter.Config{
		Client: dbDeps.DBClient,
		Logger: logger.WithGroup("billing"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create billing adapter: %w", err)
	}

	billingService, err := billingservice.New(billingservice.Config{
		Adapter:                      billingAdapter,
		CustomerService:              customerService,
		AppService:                   appService,
		Logger:                       logger.WithGroup("billing"),
		FeatureService:               entitlementRegistry.Feature,
		MeterService:                 meterService,
		StreamingConnector:           streamingConnector,
		Publisher:                    publisher,
		AdvancementStrategy:          billing.ForegroundAdvancementStrategy,
		MaxParallelQuantitySnapshots: 2,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create billing service: %w", err)
	}

	// Set up app sandbox listing
	_, err = appsandbox.NewMockableFactory(t, appsandbox.Config{
		AppService:     appService,
		BillingService: billingService,
	})
	require.NoError(t, err)

	closerFunc := func() error {
		dbDeps.Cleanup(t)
		return nil
	}

	return &testEnv{
		app:                  appService,
		customer:             customerService,
		closerFunc:           closerFunc,
		entitlement:          entitlementRegistry.Entitlement,
		feature:              entitlementRegistry.Feature,
		subscription:         svc,
		subscriptionWorkflow: workflowSvc,
		subject:              subjectService,
		plan:                 planService,
		billing:              billingService,
		meter:                meterAdapter,
	}, nil
}

type noopCustomerOverrideService struct {
	billing.CustomerOverrideService
}

func minimalCreateProfileInputTemplate(appID app.AppID) billing.CreateProfileInput {
	return billing.CreateProfileInput{
		Name:      "Awesome Profile",
		Default:   true,
		Namespace: "test-namespace",

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				// We set the interval to 0 so that the invoice is collected immediately, testcases
				// validating the collection logic can set a different interval
				Interval: lo.Must(datetime.ISODurationString("PT0S").Parse()),
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance: true,
				DraftPeriod: lo.Must(datetime.ISODurationString("P1D").Parse()),
				DueAfter:    lo.Must(datetime.ISODurationString("P1W").Parse()),
			},
			Payment: billing.PaymentConfig{
				CollectionMethod: billing.CollectionMethodChargeAutomatically,
			},
			Tax: billing.WorkflowTaxConfig{
				Enabled:  true,
				Enforced: false,
			},
		},

		Supplier: billing.SupplierContact{
			Name: "Awesome Supplier",
			Address: models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
		},

		Apps: billing.CreateProfileAppsInput{
			Invoicing: appID,
			Payment:   appID,
			Tax:       appID,
		},
	}
}

func (s *CustomerHandlerTestSuite) installSandboxApp(t *testing.T, ns string) app.App {
	appBase, err := s.Env.App().CreateApp(t.Context(),
		app.CreateAppInput{
			Name:        "Sandbox",
			Description: "Sandbox app",
			Type:        app.AppTypeSandbox,
			Namespace:   ns,
		})

	require.NoError(t, err)

	sandboxApp, err := s.Env.App().GetApp(t.Context(), app.GetAppInput{
		Namespace: ns,
		ID:        appBase.ID,
	})
	require.NoError(t, err)

	return sandboxApp
}

func (s *CustomerHandlerTestSuite) createDefaultProfile(t *testing.T, app app.App, ns string) *billing.Profile {
	clonedCreateProfileInput := minimalCreateProfileInputTemplate(app.GetID())
	clonedCreateProfileInput.Namespace = ns

	profile, err := s.Env.Billing().CreateProfile(t.Context(), clonedCreateProfileInput)
	require.NoError(t, err)

	return profile
}
