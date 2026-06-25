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
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestFlatFeeIntentOverrideAdapter(t *testing.T) {
	suite.Run(t, new(FlatFeeIntentOverrideAdapterSuite))
}

type FlatFeeIntentOverrideAdapterSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *entdb.Client
	adapter  flatfee.Adapter

	taxCodeEnv *taxcodetestutils.TestEnv
}

func (s *FlatFeeIntentOverrideAdapterSuite) SetupSuite() {
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

	s.taxCodeEnv = taxcodetestutils.NewTestEnvFromClient(t, s.dbClient, slog.Default())
	s.adapter = a
}

func (s *FlatFeeIntentOverrideAdapterSuite) TearDownSuite() {
	s.dbClient.Close()
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

func (s *FlatFeeIntentOverrideAdapterSuite) TestUpdateAndReadIntentOverride() {
	ctx := s.T().Context()
	namespace := "flatfee-intentoverride-adapter"
	charge := s.createCharge(namespace)
	overrideTaxCodeID := s.taxCodeEnv.CreateTaxCode(s.T(), namespace).ID

	overrideServicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
	}
	overrideFullServicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	overrideBillingPeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
	}
	overrideInvoiceAt := time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC)
	amountBeforeProration := alpacadecimal.NewFromInt(42)
	paymentTerm := productcatalog.InAdvancePaymentTerm
	proRating := productcatalog.ProRatingConfig{
		Enabled: true,
		Mode:    productcatalog.ProRatingModeProratePrices,
	}

	charge.Intent.BaseLayer.IntentDeletedAt = lo.ToPtr(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	charge.Intent.OverrideLayer = &flatfee.IntentMutableFields{
		IntentMutableFields: chargesmeta.IntentMutableFields{
			Name:        "manual flat fee",
			Description: lo.ToPtr("manual description"),
			Metadata: models.Metadata{
				"source": "manual",
			},
			TaxConfig: productcatalog.TaxCodeConfig{
				Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
				TaxCodeID: overrideTaxCodeID,
			},
			ServicePeriod:     overrideServicePeriod,
			FullServicePeriod: overrideFullServicePeriod,
			BillingPeriod:     overrideBillingPeriod,
		},
		InvoiceAt:             overrideInvoiceAt,
		FeatureKey:            "manual-feature",
		PaymentTerm:           paymentTerm,
		ProRating:             proRating,
		AmountBeforeProration: amountBeforeProration,
		PercentageDiscounts: lo.ToPtr(productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(10),
		}),
	}

	_, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	s.Require().ErrorContains(err, "override does not exist")

	chargeWithoutOverride := charge.ChargeBase
	chargeWithoutOverride.Intent.OverrideLayer = nil
	updated, err := s.adapter.UpdateCharge(ctx, chargeWithoutOverride)
	s.Require().NoError(err)
	s.NotNil(updated.Intent.BaseLayer.IntentDeletedAt)
	fetchedBeforeOverrideCreate, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedBeforeOverrideCreate.Intent.OverrideLayer)
	s.NotNil(fetchedBeforeOverrideCreate.DeletedAt)

	updated.Intent.OverrideLayer = charge.Intent.OverrideLayer
	updated, err = s.adapter.CreateChargeOverride(ctx, updated)
	s.Require().NoError(err)
	s.Nil(updated.DeletedAt)
	s.requireOverrideMatches(updated.Intent.OverrideLayer, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideInvoiceAt, overrideTaxCodeID)

	_, err = s.adapter.CreateChargeOverride(ctx, updated)
	s.Require().Error(err)

	overrideInvoiceAt = time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)
	updated.Intent.OverrideLayer.InvoiceAt = overrideInvoiceAt
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	s.requireOverrideMatches(updated.Intent.OverrideLayer, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideInvoiceAt, overrideTaxCodeID)

	updated.Intent.OverrideLayer.Description = nil
	updated.Intent.OverrideLayer.Metadata = nil
	updated.Intent.OverrideLayer.TaxConfig.Behavior = nil
	updated.Intent.OverrideLayer.TaxConfig.TaxCodeID = overrideTaxCodeID
	updated.Intent.OverrideLayer.FeatureKey = ""
	updated.Intent.OverrideLayer.PercentageDiscounts = nil
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	s.Require().NotNil(updated.Intent.OverrideLayer)
	s.Nil(updated.Intent.OverrideLayer.Description)
	s.Nil(updated.Intent.OverrideLayer.Metadata)
	s.Nil(updated.Intent.OverrideLayer.TaxConfig.Behavior)
	s.Equal(overrideTaxCodeID, updated.Intent.OverrideLayer.TaxConfig.TaxCodeID)
	s.Empty(updated.Intent.OverrideLayer.FeatureKey)
	s.Nil(updated.Intent.OverrideLayer.PercentageDiscounts)

	fetched, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Require().NotNil(fetched.Intent.OverrideLayer)
	s.Nil(fetched.Intent.OverrideLayer.Description)
	s.Nil(fetched.Intent.OverrideLayer.Metadata)
	s.Nil(fetched.Intent.OverrideLayer.TaxConfig.Behavior)
	s.Equal(overrideTaxCodeID, fetched.Intent.OverrideLayer.TaxConfig.TaxCodeID)
	s.Empty(fetched.Intent.OverrideLayer.FeatureKey)
	s.Nil(fetched.Intent.OverrideLayer.PercentageDiscounts)

	fetchedByIDs, err := s.adapter.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: namespace,
		IDs:       []string{charge.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedByIDs, 1)
	s.Require().NotNil(fetchedByIDs[0].Intent.OverrideLayer)
	s.Nil(fetchedByIDs[0].Intent.OverrideLayer.Description)
	s.Nil(fetchedByIDs[0].Intent.OverrideLayer.Metadata)
	s.Nil(fetchedByIDs[0].Intent.OverrideLayer.TaxConfig.Behavior)
	s.Equal(overrideTaxCodeID, fetchedByIDs[0].Intent.OverrideLayer.TaxConfig.TaxCodeID)
	s.Empty(fetchedByIDs[0].Intent.OverrideLayer.FeatureKey)
	s.Nil(fetchedByIDs[0].Intent.OverrideLayer.PercentageDiscounts)

	cleared, err := s.adapter.DeleteChargeOverride(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.Nil(cleared.Intent.OverrideLayer)
	s.NotNil(cleared.DeletedAt)

	fetchedAfterClear, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedAfterClear.Intent.OverrideLayer)
	s.NotNil(fetchedAfterClear.DeletedAt)
}

func (s *FlatFeeIntentOverrideAdapterSuite) TestDeleteChargeWithIntentOverrideDeletesOverrideIntent() {
	ctx := s.T().Context()
	namespace := "flatfee-intentoverride-delete"
	charge := s.createCharge(namespace)
	deletedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(deletedAt)
	defer clock.UnFreeze()

	charge.Intent.OverrideLayer = &flatfee.IntentMutableFields{
		IntentMutableFields: chargesmeta.IntentMutableFields{
			Name:              "manual flat fee",
			TaxConfig:         charge.Intent.BaseLayer.TaxConfig,
			ServicePeriod:     charge.Intent.BaseLayer.ServicePeriod,
			FullServicePeriod: charge.Intent.BaseLayer.FullServicePeriod,
			BillingPeriod:     charge.Intent.BaseLayer.BillingPeriod,
		},
		InvoiceAt:             charge.Intent.BaseLayer.InvoiceAt,
		PaymentTerm:           charge.Intent.BaseLayer.PaymentTerm,
		ProRating:             charge.Intent.BaseLayer.ProRating,
		AmountBeforeProration: charge.Intent.BaseLayer.AmountBeforeProration,
	}

	_, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	s.Require().ErrorContains(err, "override does not exist")

	updated := charge.ChargeBase
	updated.Intent.OverrideLayer = nil
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	updated.Intent.OverrideLayer = charge.Intent.OverrideLayer
	updated, err = s.adapter.CreateChargeOverride(ctx, updated)
	s.Require().NoError(err)
	s.Require().NotNil(updated.Intent.OverrideLayer)
	s.Nil(updated.Intent.BaseLayer.IntentDeletedAt)
	s.Nil(updated.Intent.OverrideLayer.IntentDeletedAt)
	s.Nil(updated.DeletedAt)

	s.Require().NoError(s.adapter.DeleteCharge(ctx, flatfee.Charge{ChargeBase: updated}))

	fetched, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Equal(flatfee.StatusDeleted, fetched.Status)
	s.Nil(fetched.Intent.BaseLayer.IntentDeletedAt)
	s.Require().NotNil(fetched.Intent.OverrideLayer)
	s.Require().NotNil(fetched.Intent.OverrideLayer.IntentDeletedAt)
	s.Require().NotNil(fetched.DeletedAt)
	s.Equal(deletedAt, *fetched.Intent.OverrideLayer.IntentDeletedAt)
	s.Equal(deletedAt, *fetched.DeletedAt)
}

func (s *FlatFeeIntentOverrideAdapterSuite) requireOverrideMatches(
	override *flatfee.IntentMutableFields,
	servicePeriod timeutil.ClosedPeriod,
	fullServicePeriod timeutil.ClosedPeriod,
	billingPeriod timeutil.ClosedPeriod,
	invoiceAt time.Time,
	taxCodeID string,
) {
	s.T().Helper()

	s.Require().NotNil(override)
	s.Equal("manual flat fee", override.Name)
	s.Equal("manual description", lo.FromPtr(override.Description))
	s.Equal(models.Metadata{"source": "manual"}, override.Metadata)
	s.Require().NotNil(override.TaxConfig.Behavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *override.TaxConfig.Behavior)
	s.Equal(taxCodeID, override.TaxConfig.TaxCodeID)
	s.Equal("manual-feature", override.FeatureKey)
	s.Equal(servicePeriod, override.ServicePeriod)
	s.Equal(fullServicePeriod, override.FullServicePeriod)
	s.Equal(billingPeriod, override.BillingPeriod)
	s.Equal(invoiceAt, override.InvoiceAt)
	s.Equal(productcatalog.InAdvancePaymentTerm, override.PaymentTerm)
	s.True(override.ProRating.Enabled)
	s.Equal(productcatalog.ProRatingModeProratePrices, override.ProRating.Mode)
	s.Equal(float64(42), override.AmountBeforeProration.InexactFloat64())
	s.Require().NotNil(override.PercentageDiscounts)
	s.Equal(models.NewPercentage(10), override.PercentageDiscounts.Percentage)
}

func (s *FlatFeeIntentOverrideAdapterSuite) createCharge(namespace string) flatfee.Charge {
	s.T().Helper()

	customerID := s.createCustomer(namespace)
	taxCodeID := s.taxCodeEnv.CreateTaxCode(s.T(), namespace).ID
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	createdCharges, err := s.adapter.CreateCharges(s.T().Context(), flatfee.CreateChargesInput{
		Namespace: namespace,
		Intents: []flatfee.IntentWithInitialStatus{
			{
				Intent: flatfee.OverridableIntent{
					Intent: chargesmeta.Intent{
						ManagedBy:  billing.SubscriptionManagedLine,
						CustomerID: customerID,
						Currency:   currencyx.Code("USD"),
					},
					BaseLayer: flatfee.IntentMutableFields{
						IntentMutableFields: chargesmeta.IntentMutableFields{
							Name: "flat-fee-charge",
							TaxConfig: productcatalog.TaxCodeConfig{
								TaxCodeID: taxCodeID,
							},
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
	s.Nil(createdCharges[0].Intent.OverrideLayer)

	return createdCharges[0]
}

func (s *FlatFeeIntentOverrideAdapterSuite) createCustomer(namespace string) string {
	s.T().Helper()

	customer, err := s.dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer").
		Save(s.T().Context())
	s.Require().NoError(err)

	return customer.ID
}
