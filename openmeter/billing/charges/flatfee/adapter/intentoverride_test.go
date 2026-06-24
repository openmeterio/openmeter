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

	charge.Intent.IntentDeletedAt = lo.ToPtr(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	charge.IntentOverride = &flatfee.IntentOverride{
		Name:        "manual flat fee",
		Description: lo.ToPtr("manual description"),
		Metadata: models.Metadata{
			"source": "manual",
		},
		TaxBehavior:           lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID:             &overrideTaxCodeID,
		ServicePeriod:         overrideServicePeriod,
		FullServicePeriod:     overrideFullServicePeriod,
		BillingPeriod:         overrideBillingPeriod,
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
	chargeWithoutOverride.IntentOverride = nil
	updated, err := s.adapter.UpdateCharge(ctx, chargeWithoutOverride)
	s.Require().NoError(err)
	s.NotNil(updated.Intent.IntentDeletedAt)
	fetchedBeforeOverrideCreate, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedBeforeOverrideCreate.IntentOverride)
	s.NotNil(fetchedBeforeOverrideCreate.DeletedAt)

	updated.IntentOverride = charge.IntentOverride
	updated, err = s.adapter.CreateChargeOverride(ctx, updated)
	s.Require().NoError(err)
	s.Nil(updated.DeletedAt)
	s.requireOverrideMatches(updated.IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideInvoiceAt, overrideTaxCodeID)
	s.Equal(overrideInvoiceAt, updated.GetMergedIntent().InvoiceAt)

	_, err = s.adapter.CreateChargeOverride(ctx, updated)
	s.Require().Error(err)

	overrideInvoiceAt = time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)
	updated.IntentOverride.InvoiceAt = overrideInvoiceAt
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	s.requireOverrideMatches(updated.IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideInvoiceAt, overrideTaxCodeID)
	s.Equal(overrideInvoiceAt, updated.GetMergedIntent().InvoiceAt)

	updated.IntentOverride.Description = nil
	updated.IntentOverride.Metadata = nil
	updated.IntentOverride.TaxBehavior = nil
	updated.IntentOverride.TaxCodeID = nil
	updated.IntentOverride.FeatureKey = ""
	updated.IntentOverride.PercentageDiscounts = nil
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	s.Require().NotNil(updated.IntentOverride)
	s.Nil(updated.IntentOverride.Description)
	s.Nil(updated.IntentOverride.Metadata)
	s.Nil(updated.IntentOverride.TaxBehavior)
	s.Nil(updated.IntentOverride.TaxCodeID)
	s.Empty(updated.IntentOverride.FeatureKey)
	s.Nil(updated.IntentOverride.PercentageDiscounts)

	fetched, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Require().NotNil(fetched.IntentOverride)
	s.Nil(fetched.IntentOverride.Description)
	s.Nil(fetched.IntentOverride.Metadata)
	s.Nil(fetched.IntentOverride.TaxBehavior)
	s.Nil(fetched.IntentOverride.TaxCodeID)
	s.Empty(fetched.IntentOverride.FeatureKey)
	s.Nil(fetched.IntentOverride.PercentageDiscounts)
	s.Equal(overrideInvoiceAt, fetched.GetMergedIntent().InvoiceAt)

	fetchedByIDs, err := s.adapter.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: namespace,
		IDs:       []string{charge.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedByIDs, 1)
	s.Require().NotNil(fetchedByIDs[0].IntentOverride)
	s.Nil(fetchedByIDs[0].IntentOverride.Description)
	s.Nil(fetchedByIDs[0].IntentOverride.Metadata)
	s.Nil(fetchedByIDs[0].IntentOverride.TaxBehavior)
	s.Nil(fetchedByIDs[0].IntentOverride.TaxCodeID)
	s.Empty(fetchedByIDs[0].IntentOverride.FeatureKey)
	s.Nil(fetchedByIDs[0].IntentOverride.PercentageDiscounts)
	s.Equal(overrideInvoiceAt, fetchedByIDs[0].GetMergedIntent().InvoiceAt)

	cleared, err := s.adapter.DeleteChargeOverride(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.Nil(cleared.IntentOverride)
	s.NotNil(cleared.DeletedAt)

	fetchedAfterClear, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedAfterClear.IntentOverride)
	s.NotNil(fetchedAfterClear.DeletedAt)
}

func (s *FlatFeeIntentOverrideAdapterSuite) TestDeleteChargeWithIntentOverrideDeletesOverrideIntent() {
	ctx := s.T().Context()
	namespace := "flatfee-intentoverride-delete"
	charge := s.createCharge(namespace)
	deletedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(deletedAt)
	defer clock.UnFreeze()

	charge.IntentOverride = &flatfee.IntentOverride{
		Name:                  "manual flat fee",
		ServicePeriod:         charge.Intent.ServicePeriod,
		FullServicePeriod:     charge.Intent.FullServicePeriod,
		BillingPeriod:         charge.Intent.BillingPeriod,
		InvoiceAt:             charge.Intent.InvoiceAt,
		PaymentTerm:           charge.Intent.PaymentTerm,
		ProRating:             charge.Intent.ProRating,
		AmountBeforeProration: charge.Intent.AmountBeforeProration,
	}

	_, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	s.Require().ErrorContains(err, "override does not exist")

	updated := charge.ChargeBase
	updated.IntentOverride = nil
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	updated.IntentOverride = charge.IntentOverride
	updated, err = s.adapter.CreateChargeOverride(ctx, updated)
	s.Require().NoError(err)
	s.Require().NotNil(updated.IntentOverride)
	s.Nil(updated.Intent.IntentDeletedAt)
	s.Nil(updated.IntentOverride.IntentDeletedAt)
	s.Nil(updated.DeletedAt)

	s.Require().NoError(s.adapter.DeleteCharge(ctx, flatfee.Charge{ChargeBase: updated}))

	fetched, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Equal(flatfee.StatusDeleted, fetched.Status)
	s.Nil(fetched.Intent.IntentDeletedAt)
	s.Require().NotNil(fetched.IntentOverride)
	s.Require().NotNil(fetched.IntentOverride.IntentDeletedAt)
	s.Require().NotNil(fetched.DeletedAt)
	s.Equal(deletedAt, *fetched.IntentOverride.IntentDeletedAt)
	s.Equal(deletedAt, *fetched.DeletedAt)
}

func (s *FlatFeeIntentOverrideAdapterSuite) TestOverrideNotPresentIgnoresStaleOverrideColumns() {
	ctx := s.T().Context()
	namespace := "flatfee-intentoverride-stale"
	charge := s.createCharge(namespace)

	_, err := s.dbClient.ChargeFlatFee.UpdateOneID(charge.ID).
		SetOverrideName("stale manual name").
		SetOverrideFeatureKey("stale-feature").
		Save(ctx)
	s.Require().NoError(err)

	fetched, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetched.IntentOverride)
}

func (s *FlatFeeIntentOverrideAdapterSuite) requireOverrideMatches(
	override *flatfee.IntentOverride,
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
	s.Require().NotNil(override.TaxBehavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *override.TaxBehavior)
	s.Equal(taxCodeID, lo.FromPtr(override.TaxCodeID))
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
				Intent: flatfee.Intent{
					Intent: chargesmeta.Intent{
						Name:       "flat-fee-charge",
						ManagedBy:  billing.SubscriptionManagedLine,
						CustomerID: customerID,
						Currency:   currencyx.Code("USD"),
						TaxConfig: productcatalog.TaxCodeConfig{
							TaxCodeID: taxCodeID,
						},
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
	s.Nil(createdCharges[0].IntentOverride)

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
