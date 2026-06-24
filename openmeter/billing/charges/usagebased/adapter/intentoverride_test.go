package adapter

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
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

func TestUsageBasedIntentOverrideAdapter(t *testing.T) {
	suite.Run(t, new(UsageBasedIntentOverrideAdapterSuite))
}

type UsageBasedIntentOverrideAdapterSuite struct {
	suite.Suite

	testDB   *testutils.TestDB
	dbClient *entdb.Client
	adapter  usagebased.Adapter
	meta     chargesmeta.Adapter

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
	s.meta = metaAdapter
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
	overrideInvoiceAt := time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC)
	overridePrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromFloat(0.2),
	})
	overrideDiscounts := productcatalog.Discounts{
		Percentage: lo.ToPtr(productcatalog.PercentageDiscount{
			Percentage: models.NewPercentage(10),
		}),
	}

	charge.Intent.IntentDeletedAt = lo.ToPtr(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	charge.IntentOverride = &usagebased.IntentOverride{
		Name:        "manual usage based",
		Description: lo.ToPtr("manual description"),
		Metadata: models.Metadata{
			"source": "manual",
		},
		TaxBehavior:       lo.ToPtr(productcatalog.InclusiveTaxBehavior),
		TaxCodeID:         &overrideTaxCodeID,
		ServicePeriod:     overrideServicePeriod,
		FullServicePeriod: overrideFullServicePeriod,
		BillingPeriod:     overrideBillingPeriod,
		InvoiceAt:         overrideInvoiceAt,
		FeatureKey:        "feature-override",
		Price:             *overridePrice,
		Discounts:         overrideDiscounts,
	}

	_, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	s.Require().ErrorContains(err, "override does not exist")

	chargeWithoutOverride := charge.ChargeBase
	chargeWithoutOverride.IntentOverride = nil
	updated, err := s.adapter.UpdateCharge(ctx, chargeWithoutOverride)
	s.Require().NoError(err)
	s.NotNil(updated.Intent.IntentDeletedAt)
	fetchedBeforeOverrideCreate, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedBeforeOverrideCreate.IntentOverride)
	s.NotNil(fetchedBeforeOverrideCreate.DeletedAt)

	updated.IntentOverride = charge.IntentOverride
	updated, err = s.adapter.CreateChargeOverride(ctx, updated)
	s.Require().NoError(err)
	s.Nil(updated.DeletedAt)
	s.requireOverrideMatches(updated.IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideInvoiceAt, overrideTaxCodeID, overridePrice, overrideDiscounts)

	_, err = s.adapter.CreateChargeOverride(ctx, updated)
	s.Require().Error(err)

	overrideInvoiceAt = time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)
	updated.IntentOverride.InvoiceAt = overrideInvoiceAt
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	s.requireOverrideMatches(updated.IntentOverride, overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideInvoiceAt, overrideTaxCodeID, overridePrice, overrideDiscounts)

	updated.IntentOverride.Description = nil
	updated.IntentOverride.Metadata = nil
	updated.IntentOverride.TaxBehavior = nil
	updated.IntentOverride.TaxCodeID = nil
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	s.Require().NotNil(updated.IntentOverride)
	s.Nil(updated.IntentOverride.Description)
	s.Nil(updated.IntentOverride.Metadata)
	s.Nil(updated.IntentOverride.TaxBehavior)
	s.Nil(updated.IntentOverride.TaxCodeID)

	fetched, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Require().NotNil(fetched.IntentOverride)
	s.Nil(fetched.IntentOverride.Description)
	s.Nil(fetched.IntentOverride.Metadata)
	s.Nil(fetched.IntentOverride.TaxBehavior)
	s.Nil(fetched.IntentOverride.TaxCodeID)

	fetchedByIDs, err := s.adapter.GetByIDs(ctx, usagebased.GetByIDsInput{
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

	cleared, err := s.adapter.DeleteChargeOverride(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.Nil(cleared.IntentOverride)
	s.NotNil(cleared.DeletedAt)

	fetchedAfterClear, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedAfterClear.IntentOverride)
	s.NotNil(fetchedAfterClear.DeletedAt)
}

func (s *UsageBasedIntentOverrideAdapterSuite) TestDeleteChargeWithIntentOverrideDeletesOverrideIntent() {
	ctx := s.T().Context()
	namespace := "usagebased-intentoverride-delete"
	charge := s.createCharge(namespace)
	deletedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(deletedAt)
	defer clock.UnFreeze()

	charge.IntentOverride = &usagebased.IntentOverride{
		Name:              "manual usage based",
		ServicePeriod:     charge.Intent.ServicePeriod,
		FullServicePeriod: charge.Intent.FullServicePeriod,
		BillingPeriod:     charge.Intent.BillingPeriod,
		InvoiceAt:         charge.Intent.InvoiceAt,
		FeatureKey:        charge.Intent.FeatureKey,
		Price:             charge.Intent.Price,
		Discounts:         charge.Intent.Discounts,
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

	s.Require().NoError(s.adapter.DeleteCharge(ctx, usagebased.Charge{ChargeBase: updated}))

	fetched, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Equal(usagebased.StatusDeleted, fetched.Status)
	s.Nil(fetched.Intent.IntentDeletedAt)
	s.Require().NotNil(fetched.IntentOverride)
	s.Require().NotNil(fetched.IntentOverride.IntentDeletedAt)
	s.Require().NotNil(fetched.DeletedAt)
	s.Equal(deletedAt, *fetched.IntentOverride.IntentDeletedAt)
	s.Equal(deletedAt, *fetched.DeletedAt)
}

func (s *UsageBasedIntentOverrideAdapterSuite) TestOverrideNotPresentIgnoresStaleOverrideColumns() {
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
	override *usagebased.IntentOverride,
	servicePeriod timeutil.ClosedPeriod,
	fullServicePeriod timeutil.ClosedPeriod,
	billingPeriod timeutil.ClosedPeriod,
	invoiceAt time.Time,
	taxCodeID string,
	price *productcatalog.Price,
	discounts productcatalog.Discounts,
) {
	s.T().Helper()

	s.Require().NotNil(override)
	s.Equal("manual usage based", override.Name)
	s.Equal("manual description", lo.FromPtr(override.Description))
	s.Equal(models.Metadata{"source": "manual"}, override.Metadata)
	s.Require().NotNil(override.TaxBehavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *override.TaxBehavior)
	s.Equal(taxCodeID, lo.FromPtr(override.TaxCodeID))
	s.Equal("feature-override", override.FeatureKey)
	s.Equal(servicePeriod, override.ServicePeriod)
	s.Equal(fullServicePeriod, override.FullServicePeriod)
	s.Equal(billingPeriod, override.BillingPeriod)
	s.Equal(invoiceAt, override.InvoiceAt)
	s.Equal(lo.FromPtr(price), override.Price)
	s.Equal(discounts, override.Discounts)
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
	s.Require().NoError(s.meta.RegisterCharges(s.T().Context(), chargesmeta.RegisterChargesInput{
		Namespace: namespace,
		Type:      chargesmeta.ChargeTypeUsageBased,
		Charges: []chargesmeta.IDWithUniqueReferenceID{
			{
				ID:                createdCharges[0].ID,
				UniqueReferenceID: createdCharges[0].Intent.UniqueReferenceID,
			},
		},
	}))
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
