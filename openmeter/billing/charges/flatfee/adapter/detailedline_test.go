package adapter

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfeedetailedline "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeedetailedline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestFlatFeeDetailedLineAdapter(t *testing.T) {
	suite.Run(t, new(FlatFeeDetailedLineAdapterSuite))
}

type FlatFeeDetailedLineAdapterSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *entdb.Client
	adapter  flatfee.Adapter
}

type newDetailedLineInput struct {
	Charge                 flatfee.Charge
	ServicePeriod          timeutil.ClosedPeriod
	ChildUniqueReferenceID string
	Quantity               int64
	Description            *string
}

func (s *FlatFeeDetailedLineAdapterSuite) SetupSuite() {
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

func (s *FlatFeeDetailedLineAdapterSuite) TearDownSuite() {
	s.dbClient.Close()
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

func (s *FlatFeeDetailedLineAdapterSuite) TestUpsertDetailedLinesReplacesAndSoftDeletesByChildUniqueReferenceID() {
	ctx := s.T().Context()
	namespace := "flatfee-detailedline-adapter"
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
					SettlementMode:        productcatalog.InvoiceOnlySettlementMode,
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(10),
					ProRating: productcatalog.ProRatingConfig{
						Enabled: false,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
				InitialStatus:        flatfee.StatusActive,
				AmountAfterProration: alpacadecimal.NewFromInt(10),
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(createdCharges, 1)

	charge := createdCharges[0]

	initialLines := flatfee.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "keep",
			Quantity:               1,
			Description:            lo.ToPtr("old description"),
		}),
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "delete",
			Quantity:               2,
			Description:            lo.ToPtr("delete me"),
		}),
	}
	s.Require().NoError(s.adapter.UpsertDetailedLines(ctx, charge.GetChargeID(), initialLines))

	replacementLines := flatfee.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "keep",
			Quantity:               3,
		}),
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "new",
			Quantity:               4,
			Description:            lo.ToPtr("new description"),
		}),
	}
	s.Require().NoError(s.adapter.UpsertDetailedLines(ctx, charge.GetChargeID(), replacementLines))

	fetchedCharge, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDetailedLines,
		},
	})
	s.Require().NoError(err)
	s.True(fetchedCharge.Realizations.DetailedLines.IsPresent())
	s.Len(fetchedCharge.Realizations.DetailedLines.OrEmpty(), 2)
	s.Equal("keep", fetchedCharge.Realizations.DetailedLines.OrEmpty()[0].ChildUniqueReferenceID)
	s.Equal("new", fetchedCharge.Realizations.DetailedLines.OrEmpty()[1].ChildUniqueReferenceID)
	s.Equal(float64(3), fetchedCharge.Realizations.DetailedLines.OrEmpty()[0].Quantity.InexactFloat64())
	s.Nil(fetchedCharge.Realizations.DetailedLines.OrEmpty()[0].Description)

	keptRow, err := s.dbClient.ChargeFlatFeeDetailedLine.Query().
		Where(
			dbchargeflatfeedetailedline.NamespaceEQ(namespace),
			dbchargeflatfeedetailedline.ChargeIDEQ(charge.ID),
			dbchargeflatfeedetailedline.ChildUniqueReferenceIDEQ("keep"),
			dbchargeflatfeedetailedline.DeletedAtIsNil(),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.Equal("keep", keptRow.PricerReferenceID)

	newRow, err := s.dbClient.ChargeFlatFeeDetailedLine.Query().
		Where(
			dbchargeflatfeedetailedline.NamespaceEQ(namespace),
			dbchargeflatfeedetailedline.ChargeIDEQ(charge.ID),
			dbchargeflatfeedetailedline.ChildUniqueReferenceIDEQ("new"),
			dbchargeflatfeedetailedline.DeletedAtIsNil(),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.Equal("new", newRow.PricerReferenceID)

	deletedRow, err := s.dbClient.ChargeFlatFeeDetailedLine.Query().
		Where(
			dbchargeflatfeedetailedline.NamespaceEQ(namespace),
			dbchargeflatfeedetailedline.ChargeIDEQ(charge.ID),
			dbchargeflatfeedetailedline.ChildUniqueReferenceIDEQ("delete"),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.NotNil(deletedRow.DeletedAt)
}

func (s *FlatFeeDetailedLineAdapterSuite) createCustomer(namespace string) string {
	s.T().Helper()

	customer, err := s.dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer").
		Save(s.T().Context())
	s.Require().NoError(err)

	return customer.ID
}

func (s *FlatFeeDetailedLineAdapterSuite) newDetailedLine(input newDetailedLineInput) flatfee.DetailedLine {
	s.T().Helper()

	totalAmount := alpacadecimal.NewFromFloat(0.1).Mul(alpacadecimal.NewFromInt(input.Quantity))

	return flatfee.DetailedLine{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			Namespace:   input.Charge.Namespace,
			Name:        "Detailed line",
			Description: input.Description,
		}),
		ServicePeriod:          input.ServicePeriod,
		Currency:               input.Charge.Intent.Currency,
		ChildUniqueReferenceID: input.ChildUniqueReferenceID,
		PaymentTerm:            input.Charge.Intent.PaymentTerm,
		PerUnitAmount:          alpacadecimal.NewFromFloat(0.1),
		Quantity:               alpacadecimal.NewFromInt(input.Quantity),
		Category:               stddetailedline.CategoryRegular,
		Totals: totals.Totals{
			Amount:       totalAmount,
			ChargesTotal: totalAmount,
			Total:        totalAmount,
		},
	}
}
