package subscription_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	billingworkersubscription "github.com/openmeterio/openmeter/openmeter/billing/worker/subscription"
	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	pcsubscriptionservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	subscription "github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type testDeps struct {
	subscriptiontestutils.SubscriptionDependencies
	pcSubscriptionService       pcsubscription.PlanSubscriptionService
	subscriptionService         subscription.Service
	subscriptionWorkflowService subscriptionworkflow.Service
	workerHandler               *billingworkersubscription.Handler
	billingService              billing.Service
	sandboxApp                  app.App
	cleanup                     func(t *testing.T) // Cleanup function
}

type setupConfig struct{}

func setup(t *testing.T, _ setupConfig) testDeps {
	t.Helper()

	// Let's build the dependencies
	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	require.NotNil(t, dbDeps)

	publisher := eventbus.NewMock(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	pcSubsService := pcsubscriptionservice.New(pcsubscriptionservice.Config{
		WorkflowService:     deps.WorkflowService,
		SubscriptionService: deps.SubscriptionService,
		PlanService:         deps.PlanService,
		Logger:              testutils.NewLogger(t),
		CustomerService:     deps.CustomerService,
	})

	// App
	appAdapter, err := appadapter.New(appadapter.Config{
		Client:  deps.DBDeps.DBClient,
		BaseURL: "http://localhost:8888",
	})
	require.NoError(t, err)

	appService, err := appservice.New(appservice.Config{
		Adapter:   appAdapter,
		Publisher: publisher,
	})
	require.NoError(t, err)

	billingAdapter, err := billingadapter.New(billingadapter.Config{
		Client: deps.DBDeps.DBClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	billingService, err := billingservice.New(billingservice.Config{
		Adapter:             billingAdapter,
		CustomerService:     deps.CustomerService,
		AppService:          appService,
		Logger:              slog.Default(),
		FeatureService:      deps.FeatureConnector,
		MeterService:        deps.MeterService,
		StreamingConnector:  deps.MockStreamingConnector,
		Publisher:           publisher,
		AdvancementStrategy: billing.ForegroundAdvancementStrategy,
	})
	require.NoError(t, err)

	invoiceCalculator := invoicecalc.NewMockableCalculator(t, billingService.InvoiceCalculator())

	billingService = billingService.WithInvoiceCalculator(invoiceCalculator)

	workerHandler, err := billingworkersubscription.New(billingworkersubscription.Config{
		BillingService:      billingService,
		Logger:              slog.Default(),
		Tracer:              noop.NewTracerProvider().Tracer("test"),
		TxCreator:           billingAdapter,
		SubscriptionService: deps.SubscriptionService,
	})
	require.NoError(t, err)

	// OpenMeter sandbox (registration as side-effect)
	_, err = appsandbox.NewMockableFactory(t, appsandbox.Config{
		AppService:     appService,
		BillingService: billingService,
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = appService.CreateApp(ctx,
		app.CreateAppInput{
			Name:        "Test Sandbox",
			Description: "Test Sandbox app",
			Type:        app.AppTypeSandbox,
			Namespace:   "test-namespace",
		})

	require.NoError(t, err)

	// Create sandbox app
	sandboxAppBase, err := appService.CreateApp(ctx,
		app.CreateAppInput{
			Name:        "Sandbox",
			Description: "Sandbox app",
			Type:        app.AppTypeSandbox,
			Namespace:   "test-namespace",
		})

	require.NoError(t, err)

	sandboxApp, err := appService.GetApp(ctx, app.GetAppInput{
		Namespace: "test-namespace",
		ID:        sandboxAppBase.ID,
	})
	require.NoError(t, err)

	return testDeps{
		SubscriptionDependencies:    deps,
		pcSubscriptionService:       pcSubsService,
		subscriptionService:         deps.SubscriptionService,
		subscriptionWorkflowService: deps.WorkflowService,
		cleanup:                     dbDeps.Cleanup,
		workerHandler:               workerHandler,
		billingService:              billingService,
		sandboxApp:                  sandboxApp,
	}
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
				Interval: lo.Must(isodate.String("PT0S").Parse()),
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance: true,
				DraftPeriod: lo.Must(isodate.String("P1D").Parse()),
				DueAfter:    lo.Must(isodate.String("P1W").Parse()),
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
