package adapter

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/amountdiscount"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbchargeflatfee "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfee"
	dbchargeflatfeerundetailedline "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerundetailedline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestFlatFeeDetailedLineAdapter(t *testing.T) {
	suite.Run(t, new(FlatFeeDetailedLineAdapterSuite))
}

type FlatFeeDetailedLineAdapterSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *entdb.Client
	adapter  flatfee.Adapter

	taxCodeEnv *taxcodetestutils.TestEnv
}

type newDetailedLineInput struct {
	Charge                 flatfee.Charge
	ServicePeriod          timeutil.ClosedPeriod
	ChildUniqueReferenceID string
	Quantity               int64
	Description            *string
	AmountDiscounts        mo.Option[amountdiscount.AmountDiscounts]
}

func (s *FlatFeeDetailedLineAdapterSuite) SetupSuite() {
	t := s.T()

	s.testDB = testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	s.dbClient = entdb.NewClient(entdb.Driver(s.testDB.EntDriver.Driver()))

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
	s.taxCodeEnv = taxcodetestutils.NewTestEnvFromClient(t, s.dbClient, slog.Default())
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
	taxCodeID := s.taxCodeEnv.CreateTaxCode(s.T(), namespace).ID

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
						ManagedBy:  billing.SubscriptionManagedLine,
						CustomerID: customerID,
						Currency:   currenciestestutils.NewFiatCurrency(s.T(), "USD"),
						TaxConfig: productcatalog.TaxCodeConfig{
							TaxCodeID: taxCodeID,
						},
					},
					IntentMutableFields: flatfee.IntentMutableFields{
						IntentMutableFields: chargesmeta.IntentMutableFields{
							Name:              "flat-fee-charge",
							ServicePeriod:     servicePeriod,
							FullServicePeriod: servicePeriod,
							BillingPeriod:     servicePeriod,
						},
						InvoiceAt:             servicePeriod.To,
						PaymentTerm:           productcatalog.InAdvancePaymentTerm,
						AmountBeforeProration: alpacadecimal.NewFromInt(10),
						ProRating: productcatalog.ProRatingConfig{
							Enabled: false,
							Mode:    productcatalog.ProRatingModeProratePrices,
						},
					},
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				},
				InitialStatus:        flatfee.StatusCreated,
				AmountAfterProration: alpacadecimal.NewFromInt(10),
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(createdCharges, 1)

	charge := createdCharges[0]
	run, err := s.adapter.CreateCurrentRun(ctx, flatfee.CreateCurrentRunInput{
		Charge:               charge.ChargeBase,
		ServicePeriod:        servicePeriod,
		AmountAfterProration: alpacadecimal.NewFromInt(10),
	})
	s.Require().NoError(err)
	runID := run.ID

	initialLines := flatfee.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "keep",
			Quantity:               1,
			Description:            lo.ToPtr("old description"),
			AmountDiscounts: mo.Some(amountdiscount.AmountDiscounts{
				{
					ChildUniqueReferenceID: "maximum-spend",
					Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
					Amount:                 alpacadecimal.NewFromFloat(0.03),
				},
			}),
		}),
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "delete",
			Quantity:               2,
			Description:            lo.ToPtr("delete me"),
		}),
	}
	s.Require().NoError(s.adapter.UpsertDetailedLines(ctx, runID, initialLines))

	initialKeepRow, err := s.dbClient.ChargeFlatFeeRunDetailedLine.Query().
		Where(
			dbchargeflatfeerundetailedline.NamespaceEQ(namespace),
			dbchargeflatfeerundetailedline.RunIDEQ(runID.ID),
			dbchargeflatfeerundetailedline.ChildUniqueReferenceIDEQ("keep"),
			dbchargeflatfeerundetailedline.DeletedAtIsNil(),
		).
		Only(ctx)
	s.Require().NoError(err)
	initialKeepDiscounts, ok := initialKeepRow.AmountDiscounts.Get()
	s.Require().True(ok)
	s.Require().Len(initialKeepDiscounts, 1)
	s.Equal(float64(0.03), initialKeepDiscounts[0].Amount.InexactFloat64())

	replacementLines := flatfee.DetailedLines{
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "keep",
			Quantity:               3,
			AmountDiscounts:        mo.Some(amountdiscount.AmountDiscounts{}),
		}),
		s.newDetailedLine(newDetailedLineInput{
			Charge:                 charge,
			ServicePeriod:          servicePeriod,
			ChildUniqueReferenceID: "new",
			Quantity:               4,
			Description:            lo.ToPtr("new description"),
			AmountDiscounts: mo.Some(amountdiscount.AmountDiscounts{
				{
					ChildUniqueReferenceID: "maximum-spend-reversal",
					Description:            lo.ToPtr("reverse maximum spend"),
					Reason:                 billing.NewDiscountReasonFrom(billing.MaximumSpendDiscount{}),
					Amount:                 alpacadecimal.NewFromFloat(-0.02),
					RoundingAmount:         alpacadecimal.NewFromFloat(-0.01),
				},
			}),
		}),
	}
	s.Require().NoError(s.adapter.UpsertDetailedLines(ctx, runID, replacementLines))

	fetchedCharge, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDetailedLines,
		},
	})
	s.Require().NoError(err)
	s.Require().NotNil(fetchedCharge.Realizations.CurrentRun)
	s.True(fetchedCharge.Realizations.CurrentRun.DetailedLines.IsPresent())
	s.Len(fetchedCharge.Realizations.CurrentRun.DetailedLines.OrEmpty(), 2)
	s.Equal("keep", fetchedCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].ChildUniqueReferenceID)
	s.Equal("new", fetchedCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[1].ChildUniqueReferenceID)
	s.Equal(float64(3), fetchedCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].Quantity.InexactFloat64())
	s.Nil(fetchedCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].Description)
	keptDiscounts, ok := fetchedCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].AmountDiscounts.Get()
	s.Require().True(ok)
	s.Empty(keptDiscounts)
	newDiscounts, ok := fetchedCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[1].AmountDiscounts.Get()
	s.Require().True(ok)
	s.Require().Len(newDiscounts, 1)
	s.Equal(float64(-0.02), newDiscounts[0].Amount.InexactFloat64())
	s.Equal(float64(-0.01), newDiscounts[0].RoundingAmount.InexactFloat64())

	dbCharge, err := s.dbClient.ChargeFlatFee.Query().
		Where(
			dbchargeflatfee.NamespaceEQ(namespace),
			dbchargeflatfee.IDEQ(charge.ID),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(dbCharge.CurrentRealizationRunID)
	s.Equal(runID.ID, *dbCharge.CurrentRealizationRunID)

	keptRow, err := s.dbClient.ChargeFlatFeeRunDetailedLine.Query().
		Where(
			dbchargeflatfeerundetailedline.NamespaceEQ(namespace),
			dbchargeflatfeerundetailedline.RunIDEQ(runID.ID),
			dbchargeflatfeerundetailedline.ChildUniqueReferenceIDEQ("keep"),
			dbchargeflatfeerundetailedline.DeletedAtIsNil(),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.Equal("keep", keptRow.PricerReferenceID)
	keptRowDiscounts, ok := keptRow.AmountDiscounts.Get()
	s.Require().True(ok)
	s.Require().NotNil(keptRowDiscounts)
	s.Empty(keptRowDiscounts)

	_, err = s.dbClient.ChargeFlatFeeRunDetailedLine.UpdateOneID(keptRow.ID).
		ClearAmountDiscounts().
		Save(ctx)
	s.Require().NoError(err)

	fetchedCharge, err = s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDetailedLines,
		},
	})
	s.Require().NoError(err)
	s.True(fetchedCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].AmountDiscounts.IsAbsent())

	newRow, err := s.dbClient.ChargeFlatFeeRunDetailedLine.Query().
		Where(
			dbchargeflatfeerundetailedline.NamespaceEQ(namespace),
			dbchargeflatfeerundetailedline.RunIDEQ(runID.ID),
			dbchargeflatfeerundetailedline.ChildUniqueReferenceIDEQ("new"),
			dbchargeflatfeerundetailedline.DeletedAtIsNil(),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.Equal("new", newRow.PricerReferenceID)

	deletedRow, err := s.dbClient.ChargeFlatFeeRunDetailedLine.Query().
		Where(
			dbchargeflatfeerundetailedline.NamespaceEQ(namespace),
			dbchargeflatfeerundetailedline.RunIDEQ(runID.ID),
			dbchargeflatfeerundetailedline.ChildUniqueReferenceIDEQ("delete"),
		).
		Only(ctx)
	s.Require().NoError(err)
	s.NotNil(deletedRow.DeletedAt)

	s.Require().NoError(s.adapter.DetachCurrentRun(ctx, charge.GetChargeID()))

	fetchedCharge, err = s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
		Expands: chargesmeta.Expands{
			chargesmeta.ExpandRealizations,
			chargesmeta.ExpandDetailedLines,
		},
	})
	s.Require().NoError(err)
	s.Nil(fetchedCharge.Realizations.CurrentRun)
	s.Require().Len(fetchedCharge.Realizations.PriorRuns, 1)
	s.True(fetchedCharge.Realizations.PriorRuns[0].DetailedLines.IsPresent())
	s.Len(fetchedCharge.Realizations.PriorRuns[0].DetailedLines.OrEmpty(), 2)
}

func (s *FlatFeeDetailedLineAdapterSuite) TestFetchDetailedLinesWithoutRuns() {
	ctx := s.T().Context()

	charge, err := s.adapter.FetchDetailedLines(ctx, flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			ManagedResource: chargesmeta.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "flatfee-detailedline-adapter",
				},
				ID: "charge-id",
			},
		},
	})
	s.Require().NoError(err)
	s.Nil(charge.Realizations.CurrentRun)
	s.Empty(charge.Realizations.PriorRuns)
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
	baseIntent := input.Charge.Intent.GetBaseIntent()

	return flatfee.DetailedLine{
		Base: stddetailedline.Base{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   input.Charge.Namespace,
				Name:        "Detailed line",
				Description: input.Description,
			}),
			ServicePeriod:          input.ServicePeriod,
			ChildUniqueReferenceID: input.ChildUniqueReferenceID,
			PaymentTerm:            baseIntent.PaymentTerm,
			PerUnitAmount:          alpacadecimal.NewFromFloat(0.1),
			Quantity:               alpacadecimal.NewFromInt(input.Quantity),
			Category:               stddetailedline.CategoryRegular,
			Totals: totals.Totals{
				Amount:       totalAmount,
				ChargesTotal: totalAmount,
				Total:        totalAmount,
			},
		},
		AmountDiscounts: input.AmountDiscounts,
	}
}
