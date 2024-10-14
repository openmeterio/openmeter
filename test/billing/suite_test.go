package billing_test

import (
	"context"
	"log/slog"
	"os"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

type BaseSuite struct {
	suite.Suite

	TestDB   *testutils.TestDB
	DBClient *db.Client

	BillingAdapter billing.Adapter
	BillingService billing.Service

	CustomerService customer.Service
}

func (s *BaseSuite) SetupSuite() {
	t := s.T()
	t.Log("setup suite")

	s.TestDB = testutils.InitPostgresDB(t)

	// init db
	dbClient := db.NewClient(db.Driver(s.TestDB.EntDriver.Driver()))

	if os.Getenv("TEST_DISABLE_ATLAS") != "" {
		s.Require().NoError(dbClient.Schema.Create(context.Background()))
	} else {
		s.Require().NoError(migrate.Up(s.TestDB.URL))
	}

	// setup invoicing stack

	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)
	s.CustomerService = customer.Service(customerAdapter)

	customerService, err := customerservice.New(customerservice.Config{
		Adapter: customerAdapter,
	})
	require.NoError(t, err)
	s.CustomerService = customerService

	billingAdapter, err := billingadapter.New(billingadapter.Config{
		Client: dbClient,
	})
	require.NoError(t, err)
	s.BillingAdapter = billingAdapter

	billingService, err := billingservice.New(billingservice.Config{
		Adapter:         billingAdapter,
		CustomerService: s.CustomerService,
	})
	require.NoError(t, err)
	s.BillingService = billingService
}

func (s *BaseSuite) TearDownSuite() {
	s.TestDB.EntDriver.Close()
	s.TestDB.PGDriver.Close()
}
