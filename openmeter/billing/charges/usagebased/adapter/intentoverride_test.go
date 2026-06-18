package adapter

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/intentoverride"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestUsageBasedIntentOverrideAdapter(t *testing.T) {
	suite.Run(t, new(UsageBasedIntentOverrideAdapterSuite))
}

type UsageBasedIntentOverrideAdapterSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *entdb.Client
	adapter  usagebased.Adapter

	taxCodeEnv *taxcodetestutils.TestEnv
}

func (s *UsageBasedIntentOverrideAdapterSuite) SetupSuite() {
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

func (s *UsageBasedIntentOverrideAdapterSuite) TearDownSuite() {
	s.dbClient.Close()
	s.testDB.EntDriver.Close()
	s.testDB.PGDriver.Close()
}

func (s *UsageBasedIntentOverrideAdapterSuite) TestUpdateAndReadIntentOverride() {
	ctx := s.T().Context()
	namespace := "usagebased-intentoverride-adapter"
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
	overridePrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromFloat(0.2),
	})
	overrideDiscounts := productcatalog.Discounts{
		Percentage: lo.ToPtr(productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(10),
		}),
	}

	charge.IntentOverride = &intentoverride.UsageBased{
		OverrideBase: intentoverride.OverrideBase{
			Kind:        intentoverride.KindEdit,
			Name:        lo.ToPtr("manual usage based"),
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
		FeatureKey: lo.ToPtr("feature-override"),
		Price:      overridePrice,
		Discounts:  &overrideDiscounts,
	}

	updated, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	s.Require().NoError(err)
	s.requireOverrideMatches(updated.IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideTaxCodeID, overridePrice, overrideDiscounts)

	fetched, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.requireOverrideMatches(fetched.IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideTaxCodeID, overridePrice, overrideDiscounts)

	fetchedByIDs, err := s.adapter.GetByIDs(ctx, usagebased.GetByIDsInput{
		Namespace: namespace,
		IDs:       []string{charge.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedByIDs, 1)
	s.requireOverrideMatches(fetchedByIDs[0].IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideTaxCodeID, overridePrice, overrideDiscounts)

	fetched.ChargeBase.IntentOverride = &intentoverride.UsageBased{
		OverrideBase: intentoverride.OverrideBase{
			Kind:        intentoverride.KindEdit,
			Description: mo.Some((*string)(nil)),
			TaxBehavior: mo.Some((*productcatalog.TaxBehavior)(nil)),
		},
		Discounts: &productcatalog.Discounts{},
	}
	clearedValues, err := s.adapter.UpdateCharge(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.requireExplicitClearOverrideMatches(clearedValues.IntentOverride)

	fetchedClearedValues, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.requireExplicitClearOverrideMatches(fetchedClearedValues.IntentOverride)

	rawClearedValues, err := s.dbClient.ChargeUsageBased.Get(ctx, charge.ID)
	s.Require().NoError(err)
	s.Require().NotNil(rawClearedValues.OverrideDescription)
	s.Empty(*rawClearedValues.OverrideDescription)
	s.Require().NotNil(rawClearedValues.OverrideTaxBehavior)
	s.Empty(*rawClearedValues.OverrideTaxBehavior)
	s.Nil(rawClearedValues.OverrideFeatureKey)
	s.Nil(rawClearedValues.OverridePrice)
	s.Require().NotNil(rawClearedValues.OverrideDiscounts)
	s.True(rawClearedValues.OverrideDiscounts.IsEmpty())

	fetchedClearedValues.ChargeBase.IntentOverride = nil
	cleared, err := s.adapter.UpdateCharge(ctx, fetchedClearedValues.ChargeBase)
	s.Require().NoError(err)
	s.Nil(cleared.IntentOverride)

	fetchedAfterClear, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedAfterClear.IntentOverride)
}

func (s *UsageBasedIntentOverrideAdapterSuite) TestNilKindIgnoresStaleOverrideColumns() {
	ctx := s.T().Context()
	namespace := "usagebased-intentoverride-stale"
	charge := s.createCharge(namespace)

	_, err := s.dbClient.ChargeUsageBased.UpdateOneID(charge.ID).
		SetOverrideName("stale manual name").
		SetOverrideFeatureKey("stale-feature").
		Save(ctx)
	s.Require().NoError(err)

	fetched, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetched.IntentOverride)
}

func (s *UsageBasedIntentOverrideAdapterSuite) requireOverrideMatches(
	override *intentoverride.UsageBased,
	servicePeriod timeutil.ClosedPeriod,
	fullServicePeriod timeutil.ClosedPeriod,
	billingPeriod timeutil.ClosedPeriod,
	taxCodeID string,
	price *productcatalog.Price,
	discounts productcatalog.Discounts,
) {
	s.T().Helper()

	s.Require().NotNil(override)
	s.Equal(intentoverride.KindEdit, override.Kind)
	s.Require().NotNil(override.Name)
	s.Equal("manual usage based", *override.Name)
	s.True(override.Description.IsPresent())
	s.Equal("manual description", lo.FromPtr(override.Description.OrEmpty()))
	s.Require().NotNil(override.Metadata)
	s.Equal(models.Metadata{"source": "manual"}, *override.Metadata)
	s.True(override.TaxBehavior.IsPresent())
	s.Require().NotNil(override.TaxBehavior.OrEmpty())
	s.Equal(productcatalog.InclusiveTaxBehavior, *override.TaxBehavior.OrEmpty())
	s.Equal(taxCodeID, lo.FromPtr(override.TaxCodeID))
	s.Require().NotNil(override.FeatureKey)
	s.Equal("feature-override", *override.FeatureKey)
	s.Require().NotNil(override.ServicePeriod)
	s.Equal(servicePeriod, *override.ServicePeriod)
	s.Require().NotNil(override.FullServicePeriod)
	s.Equal(fullServicePeriod, *override.FullServicePeriod)
	s.Require().NotNil(override.BillingPeriod)
	s.Equal(billingPeriod, *override.BillingPeriod)
	s.Require().NotNil(override.Price)
	s.True(override.Price.Equal(price))
	s.Require().NotNil(override.Discounts)
	s.True(override.Discounts.Equal(discounts))
}

func (s *UsageBasedIntentOverrideAdapterSuite) requireExplicitClearOverrideMatches(override *intentoverride.UsageBased) {
	s.T().Helper()

	s.Require().NotNil(override)
	s.Equal(intentoverride.KindEdit, override.Kind)
	s.True(override.Description.IsPresent())
	s.Nil(override.Description.OrEmpty())
	s.True(override.TaxBehavior.IsPresent())
	s.Nil(override.TaxBehavior.OrEmpty())
	s.Nil(override.FeatureKey)
	s.Nil(override.Price)
	s.Require().NotNil(override.Discounts)
	s.True(override.Discounts.IsEmpty())
}

func (s *UsageBasedIntentOverrideAdapterSuite) createCharge(namespace string) usagebased.Charge {
	s.T().Helper()

	customerID := s.createCustomer(namespace)
	taxCodeID := s.taxCodeEnv.CreateTaxCode(s.T(), namespace).ID
	featureID := ulid.Make().String()
	featureKey := namespace + "-feature"
	s.createFeature(namespace, featureID, featureKey)
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	createdCharges, err := s.adapter.CreateCharges(s.T().Context(), usagebased.CreateChargesInput{
		Namespace: namespace,
		Intents: []usagebased.CreateIntent{
			{
				Intent: usagebased.Intent{
					Intent: chargesmeta.Intent{
						Name:       "usage-based-charge",
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
					InvoiceAt:      servicePeriod.To,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					FeatureKey:     featureKey,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(0.1),
					}),
				},
				FeatureID:    featureID,
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
	})
	s.Require().NoError(err)
	s.Require().Len(createdCharges, 1)
	s.Nil(createdCharges[0].IntentOverride)

	return createdCharges[0]
}

func (s *UsageBasedIntentOverrideAdapterSuite) createCustomer(namespace string) string {
	s.T().Helper()

	customer, err := s.dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("test-customer").
		Save(s.T().Context())
	s.Require().NoError(err)

	return customer.ID
}

func (s *UsageBasedIntentOverrideAdapterSuite) createFeature(namespace, featureID, featureKey string) {
	s.T().Helper()

	_, err := s.dbClient.Feature.Create().
		SetNamespace(namespace).
		SetID(featureID).
		SetName("test-feature").
		SetKey(featureKey).
		Save(s.T().Context())
	s.Require().NoError(err)
}
