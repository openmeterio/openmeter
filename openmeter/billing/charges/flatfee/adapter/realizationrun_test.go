package adapter

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestFlatFeeRealizationRunAdapter(t *testing.T) {
	suite.Run(t, new(FlatFeeRealizationRunAdapterSuite))
}

type FlatFeeRealizationRunAdapterSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *entdb.Client
	adapter  flatfee.Adapter
}

func (s *FlatFeeRealizationRunAdapterSuite) SetupSuite() {
	t := s.T()

	s.testDB = testutils.InitPostgresDB(t)
	s.dbClient = entdb.NewClient(entdb.Driver(s.testDB.EntDriver.Driver()))

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: s.testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           slog.Default(),
	})
	require.NoError(t, err)
	defer migrator.CloseOrLogError()
	require.NoError(t, migrator.Up())

	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: s.dbClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	a, err := New(Config{
		Client:      s.dbClient,
		Logger:      slog.Default(),
		MetaAdapter: metaAdapter,
	})
	require.NoError(t, err)

	s.adapter = a
}

func (s *FlatFeeRealizationRunAdapterSuite) TearDownSuite() {
	s.dbClient.Close()
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

func (s *FlatFeeRealizationRunAdapterSuite) TestCreateCurrentRunFailsWhenCurrentRunAlreadyAttached() {
	ctx := s.T().Context()
	namespace := "flatfee-current-run-adapter"
	customerID := s.createCustomer(namespace)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	createdCharges, err := s.adapter.CreateCharges(ctx, flatfee.CreateChargesInput{
		Namespace: namespace,
		Intents: []flatfee.IntentWithInitialStatus{
			{
				Intent: flatfee.Intent{
					Intent: chargesmeta.Intent{
						Name:              "flat-fee-charge",
						ManagedBy:         billing.SubscriptionManagedLine,
						CustomerID:        customerID,
						Currency:          currencyx.Code("USD"),
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             servicePeriod.To,
					SettlementMode:        productcatalog.CreditThenInvoiceSettlementMode,
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(10),
					ProRating: productcatalog.ProRatingConfig{
						Enabled: false,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
				InitialStatus:        flatfee.StatusCreated,
				AmountAfterProration: alpacadecimal.NewFromInt(10),
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(createdCharges, 1)

	run, err := s.adapter.CreateCurrentRun(ctx, flatfee.CreateCurrentRunInput{
		Charge:               createdCharges[0].ChargeBase,
		ServicePeriod:        servicePeriod,
		AmountAfterProration: alpacadecimal.NewFromInt(10),
	})
	s.Require().NoError(err)
	s.Nil(run.LineID)
	s.Nil(run.InvoiceID)

	_, err = s.adapter.CreateCurrentRun(ctx, flatfee.CreateCurrentRunInput{
		Charge:               createdCharges[0].ChargeBase,
		ServicePeriod:        servicePeriod,
		AmountAfterProration: alpacadecimal.NewFromInt(10),
	})
	s.Require().ErrorContains(err, "already has current run")
}

func (s *FlatFeeRealizationRunAdapterSuite) createCustomer(namespace string) string {
	s.T().Helper()

	customer, err := s.dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer").
		Save(s.T().Context())
	s.Require().NoError(err)

	return customer.ID
}
