package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

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
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
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

	s.TestDB = testutils.InitPostgresDB(t)

	// init db
	dbClient := db.NewClient(db.Driver(s.TestDB.EntDriver.Driver()))
	s.DBClient = dbClient

	if os.Getenv("TEST_DISABLE_ATLAS") != "" {
		s.Require().NoError(dbClient.Schema.Create(context.Background()))
	} else {
		s.Require().NoError(migrate.Up(s.TestDB.URL))
	}

	// setup invoicing stack

	// Meter repo

	s.MockStreamingConnector = streamingtestutils.NewMockStreamingConnector(t)

	meterAdapter, err := meteradapter.New(nil)
	require.NoError(t, err)

	s.MeterAdapter = meterAdapter

	// Entitlement
	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbClient,
		StreamingConnector: s.MockStreamingConnector,
		Logger:             slog.Default(),
		MeterService:       s.MeterAdapter,
		Publisher:          eventbus.NewMock(t),
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: isodate.String("P1D"),
		},
	})

	// Feature
	s.FeatureRepo = entitlementRegistry.FeatureRepo
	s.FeatureService = entitlementRegistry.Feature

	// Customer

	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:              customerAdapter,
		EntitlementConnector: entitlementRegistry.Entitlement,
		Publisher:            eventbus.NewMock(t),
	})
	require.NoError(t, err)
	s.CustomerService = customerService

	// App
	appAdapter, err := appadapter.New(appadapter.Config{
		Client:  dbClient,
		BaseURL: "http://localhost:8888",
	})
	require.NoError(t, err)

	appService, err := appservice.New(appservice.Config{
		Adapter: appAdapter,
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
		Adapter:             billingAdapter,
		CustomerService:     s.CustomerService,
		AppService:          s.AppService,
		Logger:              slog.Default(),
		FeatureService:      s.FeatureService,
		MeterService:        s.MeterAdapter,
		StreamingConnector:  s.MockStreamingConnector,
		Publisher:           eventbus.NewMock(s.T()),
		AdvancementStrategy: billing.ForegroundAdvancementStrategy,
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
}

func (s *BaseSuite) InstallSandboxApp(t *testing.T, ns string) app.App {
	ctx := context.Background()
	_, err := s.AppService.CreateApp(ctx,
		app.CreateAppInput{
			Name:        "Sandbox",
			Description: "Sandbox app",
			Type:        app.AppTypeSandbox,
			Namespace:   ns,
		})

	require.NoError(t, err)

	defaultApp, err := s.AppService.GetDefaultApp(ctx, app.GetDefaultAppInput{
		Namespace: ns,
		Type:      app.AppTypeSandbox,
	})

	require.NoError(t, err)
	return defaultApp
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
			UsageAttribution: customer.CustomerUsageAttribution{
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

		switch line.Type {
		case billing.InvoiceLineTypeFee:
			s.T().Logf("fee  [%s..%s] childUniqueReferenceID: %s, invoiceAt: %s, qty: %s, unit price: %s (total=%s) %s\n",
				line.Period.Start.Format(time.RFC3339),
				line.Period.End.Format(time.RFC3339),
				lo.FromPtrOr(line.ChildUniqueReferenceID, "null"),
				line.InvoiceAt.Format(time.RFC3339),
				line.FlatFee.Quantity.String(),
				line.FlatFee.PerUnitAmount.String(),
				line.Totals.Total.String(),
				deleted)
		case billing.InvoiceLineTypeUsageBased:
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
}
