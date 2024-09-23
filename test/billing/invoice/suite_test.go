package invoice_test

import (
	"context"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing/invoice"
	invoicerepository "github.com/openmeterio/openmeter/openmeter/billing/invoice/repository"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customerrepository "github.com/openmeterio/openmeter/openmeter/customer/repository"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BaseSuite struct {
	suite.Suite

	TestDB   *testutils.TestDB
	DBClient *db.Client

	InvoiceRepo    invoice.Repository
	InvoiceService invoice.Service

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

	invoiceRepo, err := invoicerepository.New(invoicerepository.Config{
		Client: dbClient,
	})
	require.NoError(t, err)
	s.InvoiceRepo = invoiceRepo

	invoiceService, err := invoice.NewService(invoice.Config{
		Repository:      invoiceRepo,
		CustomerService: customerService,
	})
	require.NoError(t, err)
	s.InvoiceService = invoiceService
}

func (s *BaseSuite) TearDownSuite() {
	s.TestDB.EntDriver.Close()
	s.TestDB.PGDriver.Close()
}
