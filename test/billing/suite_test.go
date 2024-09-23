package billing_test

import (
	"context"
	"log/slog"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerrepository "github.com/openmeterio/openmeter/openmeter/customer/repository"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

type BaseSuite struct {
	suite.Suite

	TestDB   *testutils.TestDB
	DBClient *db.Client

	BillingRepo    billing.Repository
	BillingService billing.Service

	CustomerService customer.Service
}

func (s *BaseSuite) SetupSuite() {
	t := s.T()
	t.Log("setup suite")

	s.TestDB = testutils.InitPostgresDB(t)

	// init db
	dbClient := db.NewClient(db.Driver(s.TestDB.EntDriver.Driver()))

	s.Require().NoError(dbClient.Schema.Create(context.Background()))

	// setup invoicing stack

	customerRepo, err := customerrepository.New(customerrepository.Config{
		Client: dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)
	s.CustomerService = customer.Service(customerRepo)

	customerService, err := customer.NewService(customer.Config{
		Repository: customerRepo,
	})
	require.NoError(t, err)
	s.CustomerService = customerService

	billingRepo, err := adapter.New(adapter.Config{
		Client: dbClient,
	})
	require.NoError(t, err)
	s.BillingRepo = billingRepo

	billingService, err := service.New(service.Config{
		Repository: billingRepo,
	})
	require.NoError(t, err)
	s.BillingService = billingService
}

func (s *BaseSuite) TearDownSuite() {
	s.TestDB.EntDriver.Close()
	s.TestDB.PGDriver.Close()
}
