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
	overrideDiscounts := billing.Discounts{
		Percentage: &billing.PercentageDiscount{
			PercentageDiscount: productcatalog.PercentageDiscount{
				Percentage: models.NewPercentage(10),
			},
			CorrelationID: ulid.Make().String(),
		},
	}

	s.Require().NoError(charge.Intent.Mutate(chargesmeta.ChangeTargetBase, func(fields *usagebased.IntentMutableFields) {
		fields.IntentDeletedAt = lo.ToPtr(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	}))
	override := usagebased.IntentMutableFields{
		IntentMutableFields: chargesmeta.IntentMutableFields{
			Name:        "manual usage based",
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
		InvoiceAt:  overrideInvoiceAt,
		FeatureKey: "feature-override",
		Price:      *overridePrice,
		Discounts:  overrideDiscounts,
	}

	chargeWithUnsavedOverride := charge.ChargeBase
	chargeWithUnsavedOverride.Intent = usagebased.NewOverridableIntent(chargeWithUnsavedOverride.Intent.GetBaseIntent(), &override)
	_, err := s.adapter.UpdateCharge(ctx, chargeWithUnsavedOverride)
	s.Require().ErrorContains(err, "override does not exist")

	updated, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	s.Require().NoError(err)
	s.NotNil(updated.Intent.GetBaseIntent().IntentDeletedAt)
	fetchedBeforeOverrideCreate, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedBeforeOverrideCreate.Intent.GetOverrideLayerMutableFields())
	s.NotNil(fetchedBeforeOverrideCreate.DeletedAt)

	updated, err = s.adapter.CreateChargeOverride(ctx, updated, override)
	s.Require().NoError(err)
	s.Nil(updated.DeletedAt)
	s.requireOverrideMatches(updated.Intent.GetOverrideLayerMutableFields(), overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideInvoiceAt, overrideTaxCodeID, overridePrice, overrideDiscounts)

	_, err = s.adapter.CreateChargeOverride(ctx, updated, override)
	s.Require().Error(err)

	overrideInvoiceAt = time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)
	override = *updated.Intent.GetOverrideLayerMutableFields()
	override.InvoiceAt = overrideInvoiceAt
	updated.Intent = usagebased.NewOverridableIntent(updated.Intent.GetBaseIntent(), &override)
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	s.requireOverrideMatches(updated.Intent.GetOverrideLayerMutableFields(), overrideServicePeriod, overrideFullServicePeriod, overrideBillingPeriod, overrideInvoiceAt, overrideTaxCodeID, overridePrice, overrideDiscounts)

	override = *updated.Intent.GetOverrideLayerMutableFields()
	override.Description = nil
	override.Metadata = nil
	override.TaxConfig.Behavior = nil
	override.TaxConfig.TaxCodeID = overrideTaxCodeID
	updated.Intent = usagebased.NewOverridableIntent(updated.Intent.GetBaseIntent(), &override)
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	s.Require().NotNil(updated.Intent.GetOverrideLayerMutableFields())
	s.Nil(updated.Intent.GetOverrideLayerMutableFields().Description)
	s.Nil(updated.Intent.GetOverrideLayerMutableFields().Metadata)
	s.Nil(updated.Intent.GetOverrideLayerMutableFields().TaxConfig.Behavior)
	s.Equal(overrideTaxCodeID, updated.Intent.GetOverrideLayerMutableFields().TaxConfig.TaxCodeID)

	fetched, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Require().NotNil(fetched.Intent.GetOverrideLayerMutableFields())
	s.Nil(fetched.Intent.GetOverrideLayerMutableFields().Description)
	s.Nil(fetched.Intent.GetOverrideLayerMutableFields().Metadata)
	s.Nil(fetched.Intent.GetOverrideLayerMutableFields().TaxConfig.Behavior)
	s.Equal(overrideTaxCodeID, fetched.Intent.GetOverrideLayerMutableFields().TaxConfig.TaxCodeID)

	fetchedByIDs, err := s.adapter.GetByIDs(ctx, usagebased.GetByIDsInput{
		Namespace: namespace,
		IDs:       []string{charge.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(fetchedByIDs, 1)
	s.Require().NotNil(fetchedByIDs[0].Intent.GetOverrideLayerMutableFields())
	s.Nil(fetchedByIDs[0].Intent.GetOverrideLayerMutableFields().Description)
	s.Nil(fetchedByIDs[0].Intent.GetOverrideLayerMutableFields().Metadata)
	s.Nil(fetchedByIDs[0].Intent.GetOverrideLayerMutableFields().TaxConfig.Behavior)
	s.Equal(overrideTaxCodeID, fetchedByIDs[0].Intent.GetOverrideLayerMutableFields().TaxConfig.TaxCodeID)

	cleared, err := s.adapter.DeleteChargeOverride(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.Nil(cleared.Intent.GetOverrideLayerMutableFields())
	s.NotNil(cleared.DeletedAt)

	fetchedAfterClear, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Nil(fetchedAfterClear.Intent.GetOverrideLayerMutableFields())
	s.NotNil(fetchedAfterClear.DeletedAt)
}

func (s *UsageBasedIntentOverrideAdapterSuite) TestDeleteChargeWithIntentOverrideDeletesOverrideIntent() {
	ctx := s.T().Context()
	namespace := "usagebased-intentoverride-delete"
	charge := s.createCharge(namespace)
	deletedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(deletedAt)
	defer clock.UnFreeze()

	baseIntent := charge.Intent.GetBaseIntent()
	override := usagebased.IntentMutableFields{
		IntentMutableFields: chargesmeta.IntentMutableFields{
			Name:              "manual usage based",
			TaxConfig:         baseIntent.TaxConfig,
			ServicePeriod:     baseIntent.ServicePeriod,
			FullServicePeriod: baseIntent.FullServicePeriod,
			BillingPeriod:     baseIntent.BillingPeriod,
		},
		InvoiceAt:  baseIntent.InvoiceAt,
		FeatureKey: baseIntent.FeatureKey,
		Price:      baseIntent.Price,
		Discounts:  baseIntent.Discounts,
	}

	chargeWithUnsavedOverride := charge.ChargeBase
	chargeWithUnsavedOverride.Intent = usagebased.NewOverridableIntent(baseIntent, &override)
	_, err := s.adapter.UpdateCharge(ctx, chargeWithUnsavedOverride)
	s.Require().ErrorContains(err, "override does not exist")

	updated := charge.ChargeBase
	updated, err = s.adapter.UpdateCharge(ctx, updated)
	s.Require().NoError(err)
	updated, err = s.adapter.CreateChargeOverride(ctx, updated, override)
	s.Require().NoError(err)
	s.Require().NotNil(updated.Intent.GetOverrideLayerMutableFields())
	s.Nil(updated.Intent.GetBaseIntent().IntentDeletedAt)
	s.Nil(updated.Intent.GetOverrideLayerMutableFields().IntentDeletedAt)
	s.Nil(updated.DeletedAt)

	s.Require().NoError(s.adapter.DeleteCharge(ctx, usagebased.Charge{ChargeBase: updated}))

	fetched, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Equal(usagebased.StatusDeleted, fetched.Status)
	s.Nil(fetched.Intent.GetBaseIntent().IntentDeletedAt)
	s.Require().NotNil(fetched.Intent.GetOverrideLayerMutableFields())
	s.Require().NotNil(fetched.Intent.GetOverrideLayerMutableFields().IntentDeletedAt)
	s.Require().NotNil(fetched.DeletedAt)
	s.Equal(deletedAt, *fetched.Intent.GetOverrideLayerMutableFields().IntentDeletedAt)
	s.Equal(deletedAt, *fetched.DeletedAt)
}

// newTestUnitConfig builds a deterministic unit config for round-trip assertions.
func newTestUnitConfig(factor int64, displayUnit string) *productcatalog.UnitConfig {
	return &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(factor),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
		Precision:        0,
		DisplayUnit:      lo.ToPtr(displayUnit),
	}
}

// TestUnitConfigRoundTrip verifies unit_config persists through the charge write
// sites. unit_config is a mutable field in IntentMutableFields (alongside price),
// so it round-trips through both the base layer (create/update/clear) and the
// override layer (create/update/clear).
func (s *UsageBasedIntentOverrideAdapterSuite) TestUnitConfigRoundTrip() {
	ctx := s.T().Context()
	namespace := "usagebased-unitconfig-adapter"
	charge := s.createCharge(namespace)

	// base create→read: createCharge persisted a base unit_config (divide 1000)
	fetched, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{ChargeID: charge.GetChargeID()})
	s.Require().NoError(err)
	s.Require().NotNil(fetched.Intent.GetBaseIntent().UnitConfig)
	s.True(fetched.Intent.GetBaseIntent().UnitConfig.Equal(newTestUnitConfig(1000, "K")))

	// base update→read: the base unit_config is mutable (lives alongside price)
	s.Require().NoError(fetched.Intent.Mutate(chargesmeta.ChangeTargetBase, func(f *usagebased.IntentMutableFields) {
		f.UnitConfig = newTestUnitConfig(1000000, "M")
	}))
	updated, err := s.adapter.UpdateCharge(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.True(updated.Intent.GetBaseIntent().UnitConfig.Equal(newTestUnitConfig(1000000, "M")))

	fetched, err = s.adapter.GetByID(ctx, usagebased.GetByIDInput{ChargeID: charge.GetChargeID()})
	s.Require().NoError(err)
	s.True(fetched.Intent.GetBaseIntent().UnitConfig.Equal(newTestUnitConfig(1000000, "M")))

	// base clear→read
	s.Require().NoError(fetched.Intent.Mutate(chargesmeta.ChangeTargetBase, func(f *usagebased.IntentMutableFields) {
		f.UnitConfig = nil
	}))
	updated, err = s.adapter.UpdateCharge(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.Nil(updated.Intent.GetBaseIntent().UnitConfig)

	fetched, err = s.adapter.GetByID(ctx, usagebased.GetByIDInput{ChargeID: charge.GetChargeID()})
	s.Require().NoError(err)
	s.Nil(fetched.Intent.GetBaseIntent().UnitConfig)

	// override create→read: the override layer carries its own unit_config snapshot
	baseIntent := fetched.Intent.GetBaseIntent()
	override := usagebased.IntentMutableFields{
		IntentMutableFields: chargesmeta.IntentMutableFields{
			Name:              "override with unit config",
			TaxConfig:         baseIntent.TaxConfig,
			ServicePeriod:     baseIntent.ServicePeriod,
			FullServicePeriod: baseIntent.FullServicePeriod,
			BillingPeriod:     baseIntent.BillingPeriod,
		},
		InvoiceAt:  baseIntent.InvoiceAt,
		FeatureKey: baseIntent.FeatureKey,
		Price:      baseIntent.Price,
		Discounts:  baseIntent.Discounts,
		UnitConfig: newTestUnitConfig(1000, "K"),
	}
	withOverride, err := s.adapter.CreateChargeOverride(ctx, fetched.ChargeBase, override)
	s.Require().NoError(err)
	s.Require().NotNil(withOverride.Intent.GetOverrideLayerMutableFields())
	s.True(withOverride.Intent.GetOverrideLayerMutableFields().UnitConfig.Equal(newTestUnitConfig(1000, "K")))

	fetched, err = s.adapter.GetByID(ctx, usagebased.GetByIDInput{ChargeID: charge.GetChargeID()})
	s.Require().NoError(err)
	s.Require().NotNil(fetched.Intent.GetOverrideLayerMutableFields())
	s.True(fetched.Intent.GetOverrideLayerMutableFields().UnitConfig.Equal(newTestUnitConfig(1000, "K")))

	// override update→read
	override = *fetched.Intent.GetOverrideLayerMutableFields()
	override.UnitConfig = newTestUnitConfig(1000000, "M")
	fetched.Intent = usagebased.NewOverridableIntent(fetched.Intent.GetBaseIntent(), &override)
	updated, err = s.adapter.UpdateCharge(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.True(updated.Intent.GetOverrideLayerMutableFields().UnitConfig.Equal(newTestUnitConfig(1000000, "M")))

	fetched, err = s.adapter.GetByID(ctx, usagebased.GetByIDInput{ChargeID: charge.GetChargeID()})
	s.Require().NoError(err)
	s.True(fetched.Intent.GetOverrideLayerMutableFields().UnitConfig.Equal(newTestUnitConfig(1000000, "M")))

	// override clear→read
	override = *fetched.Intent.GetOverrideLayerMutableFields()
	override.UnitConfig = nil
	fetched.Intent = usagebased.NewOverridableIntent(fetched.Intent.GetBaseIntent(), &override)
	updated, err = s.adapter.UpdateCharge(ctx, fetched.ChargeBase)
	s.Require().NoError(err)
	s.Nil(updated.Intent.GetOverrideLayerMutableFields().UnitConfig)

	fetched, err = s.adapter.GetByID(ctx, usagebased.GetByIDInput{ChargeID: charge.GetChargeID()})
	s.Require().NoError(err)
	s.Require().NotNil(fetched.Intent.GetOverrideLayerMutableFields())
	s.Nil(fetched.Intent.GetOverrideLayerMutableFields().UnitConfig)
}

func (s *UsageBasedIntentOverrideAdapterSuite) requireOverrideMatches(
	override *usagebased.IntentMutableFields,
	servicePeriod timeutil.ClosedPeriod,
	fullServicePeriod timeutil.ClosedPeriod,
	billingPeriod timeutil.ClosedPeriod,
	invoiceAt time.Time,
	taxCodeID string,
	price *productcatalog.Price,
	discounts billing.Discounts,
) {
	s.T().Helper()

	s.Require().NotNil(override)
	s.Equal("manual usage based", override.Name)
	s.Equal("manual description", lo.FromPtr(override.Description))
	s.Equal(models.Metadata{"source": "manual"}, override.Metadata)
	s.Require().NotNil(override.TaxConfig.Behavior)
	s.Equal(productcatalog.InclusiveTaxBehavior, *override.TaxConfig.Behavior)
	s.Equal(taxCodeID, override.TaxConfig.TaxCodeID)
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
						ManagedBy:  billing.SubscriptionManagedLine,
						CustomerID: customerID,
						Currency:   currencyx.Code("USD"),
					},
					IntentMutableFields: usagebased.IntentMutableFields{
						IntentMutableFields: chargesmeta.IntentMutableFields{
							Name: "usage-based-charge",
							TaxConfig: productcatalog.TaxCodeConfig{
								TaxCodeID: taxCodeID,
							},
							ServicePeriod:     servicePeriod,
							FullServicePeriod: servicePeriod,
							BillingPeriod:     servicePeriod,
						},
						InvoiceAt:  servicePeriod.To,
						FeatureKey: featureKey,
						Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(0.1),
						}),
						UnitConfig: newTestUnitConfig(1000, "K"),
					},
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				}.AsOverridableIntent(),
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
				UniqueReferenceID: createdCharges[0].Intent.GetUniqueReferenceID(),
			},
		},
	}))
	s.Nil(createdCharges[0].Intent.GetOverrideLayerMutableFields())

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
