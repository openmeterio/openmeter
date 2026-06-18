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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/intentoverride"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
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
	amountBeforeProration := alpacadecimal.NewFromInt(42)
	paymentTerm := productcatalog.InAdvancePaymentTerm
	proRating := productcatalog.ProRatingConfig{
		Enabled: true,
		Mode:    productcatalog.ProRatingModeProratePrices,
	}

	charge.IntentOverride = &intentoverride.FlatFee{
		OverrideBase: intentoverride.OverrideBase{
			Kind:        intentoverride.KindEdit,
			Name:        lo.ToPtr("manual flat fee"),
			Description: mo.Some(lo.ToPtr("manual description")),
			Metadata: &models.Metadata{
				"source": "manual",
			},
			TaxBehavior:       mo.Some(lo.ToPtr(productcatalog.InclusiveTaxBehavior)),
			TaxCodeID:         &overrideTaxCodeID,
			ServicePeriod:     &overrideServicePeriod,
			FullServicePeriod: &overrideFullServicePeriod,
			BillingPeriod:     &overrideBillingPeriod,
		},
		FeatureKey:            mo.Some(lo.ToPtr("manual-feature")),
		PaymentTerm:           &paymentTerm,
		ProRating:             &proRating,
		AmountBeforeProration: &amountBeforeProration,
		PercentageDiscounts: mo.Some(lo.ToPtr(productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(10),
		})),
	}

	updated, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	s.Require().NoError(err)
	s.requireOverrideMatches(updated.IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideTaxCodeID)

	fetched, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.requireOverrideMatches(fetched.IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideTaxCodeID)

	fetchedByIDs, err := s.adapter.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: namespace,
		IDs:       []string{charge.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedByIDs, 1)
	s.requireOverrideMatches(fetchedByIDs[0].IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideTaxCodeID)

	fetched.ChargeBase.IntentOverride = &intentoverride.FlatFee{
		OverrideBase: intentoverride.OverrideBase{
			Kind:        intentoverride.KindEdit,
			Description: mo.Some((*string)(nil)),
			TaxBehavior: mo.Some((*productcatalog.TaxBehavior)(nil)),
		},
		FeatureKey:          mo.Some((*string)(nil)),
		PercentageDiscounts: mo.Some((*productcatalog.PercentageDiscount)(nil)),
	}
	clearedValues, err := s.adapter.UpdateCharge(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.requireExplicitClearOverrideMatches(clearedValues.IntentOverride)

	fetchedClearedValues, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.requireExplicitClearOverrideMatches(fetchedClearedValues.IntentOverride)

	rawClearedValues, err := s.dbClient.ChargeFlatFee.Get(ctx, charge.ID)
	s.Require().NoError(err)
	s.Nil(rawClearedValues.OverrideName)
	s.Require().NotNil(rawClearedValues.OverrideDescription)
	s.Empty(*rawClearedValues.OverrideDescription)
	s.Nil(rawClearedValues.OverrideMetadata)
	s.Require().NotNil(rawClearedValues.OverrideTaxBehavior)
	s.Empty(*rawClearedValues.OverrideTaxBehavior)
	s.Nil(rawClearedValues.OverrideTaxCodeID)
	s.Nil(rawClearedValues.OverrideServicePeriodFrom)
	s.Nil(rawClearedValues.OverrideServicePeriodTo)
	s.Nil(rawClearedValues.OverrideFullServicePeriodFrom)
	s.Nil(rawClearedValues.OverrideFullServicePeriodTo)
	s.Nil(rawClearedValues.OverrideBillingPeriodFrom)
	s.Nil(rawClearedValues.OverrideBillingPeriodTo)
	s.Require().NotNil(rawClearedValues.OverrideFeatureKey)
	s.Empty(*rawClearedValues.OverrideFeatureKey)
	s.Nil(rawClearedValues.OverridePaymentTerm)
	s.Nil(rawClearedValues.OverrideProRating)
	s.Nil(rawClearedValues.OverrideAmountBeforeProration)
	s.Require().NotNil(rawClearedValues.OverridePercentageDiscounts)
	s.Nil(rawClearedValues.OverridePercentageDiscounts.Value)

	fetchedClearedValues.ChargeBase.IntentOverride = nil
	cleared, err := s.adapter.UpdateCharge(ctx, fetchedClearedValues.ChargeBase)
	s.Require().NoError(err)
	s.Nil(cleared.IntentOverride)

	fetchedAfterClear, err := s.adapter.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedAfterClear.IntentOverride)
}

func (s *FlatFeeIntentOverrideAdapterSuite) TestNilKindIgnoresStaleOverrideColumns() {
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
	override *intentoverride.FlatFee,
	servicePeriod timeutil.ClosedPeriod,
	fullServicePeriod timeutil.ClosedPeriod,
	billingPeriod timeutil.ClosedPeriod,
	taxCodeID string,
) {
	s.T().Helper()

	s.Require().NotNil(override)
	s.Equal(intentoverride.KindEdit, override.Kind)
	s.Require().NotNil(override.Name)
	s.Equal("manual flat fee", *override.Name)
	s.True(override.Description.IsPresent())
	s.Equal("manual description", lo.FromPtr(override.Description.OrEmpty()))
	s.Require().NotNil(override.Metadata)
	s.Equal(models.Metadata{"source": "manual"}, *override.Metadata)
	s.True(override.TaxBehavior.IsPresent())
	s.Require().NotNil(override.TaxBehavior.OrEmpty())
	s.Equal(productcatalog.InclusiveTaxBehavior, *override.TaxBehavior.OrEmpty())
	s.Equal(taxCodeID, lo.FromPtr(override.TaxCodeID))
	s.True(override.FeatureKey.IsPresent())
	s.Equal("manual-feature", lo.FromPtr(override.FeatureKey.OrEmpty()))
	s.Require().NotNil(override.ServicePeriod)
	s.Equal(servicePeriod, *override.ServicePeriod)
	s.Require().NotNil(override.FullServicePeriod)
	s.Equal(fullServicePeriod, *override.FullServicePeriod)
	s.Require().NotNil(override.BillingPeriod)
	s.Equal(billingPeriod, *override.BillingPeriod)
	s.Require().NotNil(override.PaymentTerm)
	s.Equal(productcatalog.InAdvancePaymentTerm, *override.PaymentTerm)
	s.Require().NotNil(override.ProRating)
	s.True(override.ProRating.Enabled)
	s.Equal(productcatalog.ProRatingModeProratePrices, override.ProRating.Mode)
	s.Require().NotNil(override.AmountBeforeProration)
	s.Equal(float64(42), override.AmountBeforeProration.InexactFloat64())
	s.True(override.PercentageDiscounts.IsPresent())
	s.Require().NotNil(override.PercentageDiscounts.OrEmpty())
	s.Equal(models.NewPercentage(10), override.PercentageDiscounts.OrEmpty().Percentage)
}

func (s *FlatFeeIntentOverrideAdapterSuite) requireExplicitClearOverrideMatches(override *intentoverride.FlatFee) {
	s.T().Helper()

	s.Require().NotNil(override)
	s.Equal(intentoverride.KindEdit, override.Kind)
	s.Nil(override.Name)
	s.True(override.Description.IsPresent())
	s.Nil(override.Description.OrEmpty())
	s.Nil(override.Metadata)
	s.True(override.TaxBehavior.IsPresent())
	s.Nil(override.TaxBehavior.OrEmpty())
	s.Nil(override.TaxCodeID)
	s.Nil(override.ServicePeriod)
	s.Nil(override.FullServicePeriod)
	s.Nil(override.BillingPeriod)
	s.True(override.FeatureKey.IsPresent())
	s.Nil(override.FeatureKey.OrEmpty())
	s.Nil(override.PaymentTerm)
	s.Nil(override.ProRating)
	s.Nil(override.AmountBeforeProration)
	s.True(override.PercentageDiscounts.IsPresent())
	s.Nil(override.PercentageDiscounts.OrEmpty())
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
