package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	dbchargeusagebasedcostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedcostbasis"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	featurepkg "github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestUsageBasedCostBasisCreate(t *testing.T) {
	suite.Run(t, new(UsageBasedCostBasisCreateSuite))
}

type UsageBasedCostBasisCreateSuite struct {
	BaseSuite
}

func (s *UsageBasedCostBasisCreateSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *UsageBasedCostBasisCreateSuite) TearDownTest() {
	s.setCustomCurrencyEnabled(false)
	s.BaseSuite.TearDownTest()
}

func (s *UsageBasedCostBasisCreateSuite) TestCustomCurrencyCreditThenInvoiceIsDisabledByDefault() {
	// given:
	// - a valid manual cost-basis intent for a custom-currency usage-based charge
	// - the test-only custom-currency switch left at its default value
	// when:
	// - the charge is created
	// then:
	// - creation is rejected before any charge cost basis is persisted
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("usage-based-cost-basis-disabled")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "usage-based-cost-basis-disabled")
	currency := s.createTestCustomCurrency(ctx, namespace)
	featureMeters := s.createFeatureMeters(ctx, namespace, "disabled-feature")
	intent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: s.newFiatCurrency("USD"),
		Rate:         alpacadecimal.NewFromInt(2),
	})

	_, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
		Namespace: namespace,
		Intents: []usagebased.Intent{
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "disabled", "disabled-feature", productcatalog.CreditThenInvoiceSettlementMode, &intent),
		},
		FeatureMeters: featureMeters,
	})
	s.Require().ErrorIs(err, meta.ErrCustomCurrencyNotSupported)
	s.Require().Equal(0, s.countCostBases(namespace))
}

func (s *UsageBasedCostBasisCreateSuite) TestCreatePersistsManualPinnedAndDynamicCostBasesInInputOrder() {
	// given:
	// - manual, pinned, and dynamic cost-basis intents in one custom-currency batch
	// when:
	// - custom-currency creation is enabled, the charges are created, and each is reloaded through the charge service
	// then:
	// - each charge references its own persisted cost basis and its intent and resolved state survive the normal read path
	ctx := s.T().Context()
	now := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("usage-based-cost-basis-modes")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "usage-based-cost-basis-modes")
	currency := s.createTestCustomCurrency(ctx, namespace)
	featureMeters := s.createFeatureMeters(ctx, namespace, "modes-feature")
	fiatCurrency := s.newFiatCurrency("USD")
	pinnedCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: currency.ID,
		FiatCode:   currencyx.Code("USD"),
		Rate:       alpacadecimal.NewFromFloat(0.25),
	})
	s.Require().NoError(err)

	manualIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	pinnedIntent := costbasis.NewIntent(costbasis.PinnedIntent{
		FiatCurrency:        fiatCurrency,
		CurrencyCostBasisID: pinnedCostBasis.ID,
	})
	dynamicIntent := costbasis.NewIntent(costbasis.DynamicIntent{
		FiatCurrency: fiatCurrency,
	})

	s.setCustomCurrencyEnabled(true)
	created, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
		Namespace: namespace,
		Intents: []usagebased.Intent{
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "manual", "modes-feature", productcatalog.CreditThenInvoiceSettlementMode, &manualIntent),
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "pinned", "modes-feature", productcatalog.CreditThenInvoiceSettlementMode, &pinnedIntent),
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "dynamic", "modes-feature", productcatalog.CreditThenInvoiceSettlementMode, &dynamicIntent),
		},
		FeatureMeters: featureMeters,
	})
	s.Require().NoError(err)
	s.Require().Len(created, 3)

	s.Require().Equal(costbasis.ModeManual, created[0].Charge.Intent.GetCostBasisIntent().Kind())
	s.Require().NotNil(created[0].Charge.State.ResolvedCostBasis)
	s.Require().Equal(float64(0.5), created[0].Charge.State.ResolvedCostBasis.CostBasis.InexactFloat64())
	s.Require().Nil(created[0].Charge.State.ResolvedCostBasis.CostBasisID)
	s.Require().Equal(now, created[0].Charge.State.ResolvedCostBasis.ResolvedAt)

	s.Require().Equal(costbasis.ModePinned, created[1].Charge.Intent.GetCostBasisIntent().Kind())
	s.Require().NotNil(created[1].Charge.State.ResolvedCostBasis)
	s.Require().Equal(float64(0.25), created[1].Charge.State.ResolvedCostBasis.CostBasis.InexactFloat64())
	s.Require().Equal(pinnedCostBasis.ID, *created[1].Charge.State.ResolvedCostBasis.CostBasisID)
	s.Require().Equal(now, created[1].Charge.State.ResolvedCostBasis.ResolvedAt)

	s.Require().Equal(costbasis.ModeDynamic, created[2].Charge.Intent.GetCostBasisIntent().Kind())
	s.Require().Nil(created[2].Charge.State.ResolvedCostBasis)

	seenCostBasisIDs := map[string]struct{}{}
	for idx, result := range created {
		s.Require().NotNil(result.Charge.State.CostBasisID, "charge index %d", idx)

		reloaded, err := s.mustGetChargeByID(result.Charge.GetChargeID()).AsUsageBasedCharge()
		s.Require().NoError(err)
		s.Require().Equal(result.Charge.Intent.GetCostBasisIntent().Kind(), reloaded.Intent.GetCostBasisIntent().Kind(), "charge index %d", idx)
		s.Require().Equal(result.Charge.State.CostBasisID, reloaded.State.CostBasisID, "charge index %d", idx)
		s.Require().Equal(result.Charge.State.ResolvedCostBasis, reloaded.State.ResolvedCostBasis, "charge index %d", idx)

		chargeEntity, err := s.DBClient.ChargeUsageBased.Get(ctx, result.Charge.ID)
		s.Require().NoError(err)
		s.Require().NotNil(chargeEntity.CostBasisID)
		s.Require().Equal(*result.Charge.State.CostBasisID, *chargeEntity.CostBasisID)
		seenCostBasisIDs[*chargeEntity.CostBasisID] = struct{}{}

		costBasisEntity, err := s.DBClient.ChargeUsageBasedCostBasis.Get(ctx, *chargeEntity.CostBasisID)
		s.Require().NoError(err)
		persisted, err := costbasis.Get(costBasisEntity)
		s.Require().NoError(err)
		s.Require().Equal(result.Charge.Intent.GetCostBasisIntent().Kind(), persisted.Intent.Kind(), "charge index %d", idx)
		s.Require().Equal(result.Charge.State.ResolvedCostBasis, persisted.State, "charge index %d", idx)
	}
	s.Require().Len(seenCostBasisIDs, 3)
}

func (s *UsageBasedCostBasisCreateSuite) TestSetResolvedDynamicCostBasisIsRetrySafe() {
	// given:
	// - an unresolved dynamic cost basis attached to a usage-based charge
	// - a first persisted resolution
	// when:
	// - a stale retry attempts to persist a different resolution
	// then:
	// - the original persisted resolution is returned and remains unchanged
	ctx := s.T().Context()
	now := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("usage-based-cost-basis-retry")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "usage-based-cost-basis-retry")
	currency := s.createTestCustomCurrency(ctx, namespace)
	featureMeters := s.createFeatureMeters(ctx, namespace, "retry-feature")
	currencyCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: currency.ID,
		FiatCode:   currencyx.Code("USD"),
		Rate:       alpacadecimal.NewFromFloat(0.25),
	})
	s.Require().NoError(err)

	dynamicIntent := costbasis.NewIntent(costbasis.DynamicIntent{
		FiatCurrency: s.newFiatCurrency("USD"),
	})
	s.setCustomCurrencyEnabled(true)
	created, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
		Namespace: namespace,
		Intents: []usagebased.Intent{
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "dynamic-retry", "retry-feature", productcatalog.CreditThenInvoiceSettlementMode, &dynamicIntent),
		},
		FeatureMeters: featureMeters,
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	s.Require().NotNil(created[0].Charge.State.CostBasisID)

	firstState := costbasis.State{
		CostBasis:   currencyCostBasis.Rate,
		CostBasisID: lo.ToPtr(currencyCostBasis.ID),
		ResolvedAt:  now,
	}
	first, err := s.UsageBasedAdapter.SetResolvedCostBasis(ctx, costbasis.SetResolvedCostBasisInput{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        *created[0].Charge.State.CostBasisID,
		},
		State: firstState,
	})
	s.Require().NoError(err)
	s.Require().Equal(&firstState, first.State)

	retryState := costbasis.State{
		CostBasis:   alpacadecimal.NewFromInt(9),
		CostBasisID: lo.ToPtr(currencyCostBasis.ID),
		ResolvedAt:  now.Add(time.Hour),
	}
	retried, err := s.UsageBasedAdapter.SetResolvedCostBasis(ctx, costbasis.SetResolvedCostBasisInput{
		NamespacedID: first.NamespacedID,
		State:        retryState,
	})
	s.Require().NoError(err)
	s.Require().Equal(&firstState, retried.State)

	reloaded, err := s.mustGetChargeByID(created[0].Charge.GetChargeID()).AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Require().Equal(created[0].Charge.State.CostBasisID, reloaded.State.CostBasisID)
	s.Require().Equal(&firstState, reloaded.State.ResolvedCostBasis)
}

func (s *UsageBasedCostBasisCreateSuite) TestDynamicCostBasisResolvesWhenChargeBecomesActive() {
	// given:
	// - an unresolved dynamic cost basis with two successive currency rates
	// - activation delayed until after the second rate becomes effective
	// when:
	// - the charge advances from created to active
	// then:
	// - it idempotently persists the rate effective at service-period start and records that as ResolvedAt
	ctx := s.T().Context()
	servicePeriodFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	secondEffectiveAt := servicePeriodFrom.Add(10 * 24 * time.Hour)
	activationAt := secondEffectiveAt.Add(24 * time.Hour)
	clock.FreezeTime(servicePeriodFrom)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("usage-based-cost-basis-active")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "usage-based-cost-basis-active")
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	currency := s.createTestCustomCurrency(ctx, namespace)
	featureMeters := s.createFeatureMeters(ctx, namespace, "active-feature")
	first, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: currency.ID,
		FiatCode:   currencyx.Code("USD"),
		Rate:       alpacadecimal.NewFromFloat(0.25),
	})
	s.Require().NoError(err)
	_, err = s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:     namespace,
		CurrencyID:    currency.ID,
		FiatCode:      currencyx.Code("USD"),
		Rate:          alpacadecimal.NewFromFloat(0.5),
		EffectiveFrom: lo.ToPtr(secondEffectiveAt),
	})
	s.Require().NoError(err)

	dynamicIntent := costbasis.NewIntent(costbasis.DynamicIntent{
		FiatCurrency: s.newFiatCurrency("USD"),
	})
	s.setCustomCurrencyEnabled(true)
	clock.FreezeTime(activationAt)
	created, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
		Namespace: namespace,
		Intents: []usagebased.Intent{
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "dynamic-active", "active-feature", productcatalog.CreditThenInvoiceSettlementMode, &dynamicIntent),
		},
		FeatureMeters: featureMeters,
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	s.Require().Equal(usagebased.StatusCreated, created[0].Charge.Status)
	s.Require().NotNil(created[0].Charge.State.CostBasisID)
	s.Require().Nil(created[0].Charge.State.ResolvedCostBasis)

	advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customer.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(advanced, 1)

	reloaded, err := s.mustGetChargeByID(created[0].Charge.GetChargeID()).AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Require().Equal(usagebased.StatusActive, reloaded.Status)
	s.Require().Equal(created[0].Charge.State.CostBasisID, reloaded.State.CostBasisID)
	s.Require().NotNil(reloaded.State.ResolvedCostBasis)
	s.Require().Equal(first.ID, *reloaded.State.ResolvedCostBasis.CostBasisID)
	s.Require().Equal(first.Rate.InexactFloat64(), reloaded.State.ResolvedCostBasis.CostBasis.InexactFloat64())
	s.Require().Equal(servicePeriodFrom, reloaded.State.ResolvedCostBasis.ResolvedAt)
}

func (s *UsageBasedCostBasisCreateSuite) TestPinnedCostBasisMustMatchCurrencyAndFiatCurrency() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("usage-based-cost-basis-pinned-mismatch")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "usage-based-cost-basis-pinned-mismatch")
	currency := s.createTestCustomCurrency(ctx, namespace)
	featureMeters := s.createFeatureMeters(ctx, namespace, "mismatch-feature")
	otherCurrency, err := s.CurrencyService.CreateCurrency(ctx, currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "OTHER",
			Name:               "Other",
			Symbol:             "O",
			Precision:          3,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	s.Require().NoError(err)

	otherCurrencyCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: otherCurrency.ID,
		FiatCode:   currencyx.Code("USD"),
		Rate:       alpacadecimal.NewFromInt(1),
	})
	s.Require().NoError(err)
	eurCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: currency.ID,
		FiatCode:   currencyx.Code("EUR"),
		Rate:       alpacadecimal.NewFromInt(1),
	})
	s.Require().NoError(err)

	tests := []struct {
		name        string
		costBasisID string
		errorText   string
	}{
		{name: "currency", costBasisID: otherCurrencyCostBasis.ID, errorText: "currency cost basis currency mismatch"},
		{name: "fiat currency", costBasisID: eurCostBasis.ID, errorText: "currency cost basis fiat currency mismatch"},
	}

	s.setCustomCurrencyEnabled(true)
	for _, test := range tests {
		s.Run(test.name, func() {
			// given:
			// - a pinned cost basis that disagrees with the charge currency or requested fiat currency
			// when:
			// - the usage-based charge is created
			// then:
			// - resolution rejects the mismatch before charge cost-basis persistence
			intent := costbasis.NewIntent(costbasis.PinnedIntent{
				FiatCurrency:        s.newFiatCurrency("USD"),
				CurrencyCostBasisID: test.costBasisID,
			})

			_, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
				Namespace: namespace,
				Intents: []usagebased.Intent{
					s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "mismatch-"+test.name, "mismatch-feature", productcatalog.CreditThenInvoiceSettlementMode, &intent),
				},
				FeatureMeters: featureMeters,
			})
			s.Require().ErrorContains(err, test.errorText)
		})
	}
	s.Require().Equal(0, s.countCostBases(namespace))
}

func (s *UsageBasedCostBasisCreateSuite) TestCreateRollsBackCostBasesWhenChargeCreationFails() {
	// given:
	// - two cost-basis-backed intents with the same charge unique reference
	// when:
	// - the charge constraint rejects the bulk create after cost bases were inserted
	// then:
	// - the surrounding transaction removes all newly inserted charge cost bases
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("usage-based-cost-basis-rollback")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "usage-based-cost-basis-rollback")
	currency := s.createTestCustomCurrency(ctx, namespace)
	featureMeters := s.createFeatureMeters(ctx, namespace, "rollback-feature")
	intent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: s.newFiatCurrency("USD"),
		Rate:         alpacadecimal.NewFromInt(2),
	})

	s.setCustomCurrencyEnabled(true)
	_, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
		Namespace: namespace,
		Intents: []usagebased.Intent{
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "duplicate", "rollback-feature", productcatalog.CreditThenInvoiceSettlementMode, &intent),
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "duplicate", "rollback-feature", productcatalog.CreditThenInvoiceSettlementMode, &intent),
		},
		FeatureMeters: featureMeters,
	})
	s.Require().Error(err)
	s.Require().Equal(0, s.countCostBases(namespace))
}

func (s *UsageBasedCostBasisCreateSuite) TestCreateWithoutCostBasisLeavesChargeReferenceEmpty() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("usage-based-without-cost-basis")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "usage-based-without-cost-basis")
	currency := s.createTestCustomCurrency(ctx, namespace)
	featureMeters := s.createFeatureMeters(ctx, namespace, "credit-only-feature")

	created, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
		Namespace: namespace,
		Intents: []usagebased.Intent{
			s.newUsageBasedIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "credit-only", "credit-only-feature", productcatalog.CreditOnlySettlementMode, nil),
		},
		FeatureMeters: featureMeters,
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	s.Require().Nil(created[0].Charge.Intent.GetCostBasisIntent())
	s.Require().Nil(created[0].Charge.State.CostBasisID)
	s.Require().Nil(created[0].Charge.State.ResolvedCostBasis)

	chargeEntity, err := s.DBClient.ChargeUsageBased.Get(ctx, created[0].Charge.ID)
	s.Require().NoError(err)
	s.Require().Nil(chargeEntity.CostBasisID)
	s.Require().Equal(0, s.countCostBases(namespace))
}

func (s *UsageBasedCostBasisCreateSuite) setCustomCurrencyEnabled(enabled bool) {
	s.T().Helper()
	enabler, ok := s.Charges.usageBasedService.(customCurrencyEnabler)
	s.Require().True(ok)
	s.Require().NoError(enabler.SetEnableCustomCurrency(s.T(), enabled))
}

func (s *UsageBasedCostBasisCreateSuite) countCostBases(namespace string) int {
	s.T().Helper()

	count, err := s.DBClient.ChargeUsageBasedCostBasis.Query().
		Where(dbchargeusagebasedcostbasis.Namespace(namespace)).
		Count(s.T().Context())
	s.Require().NoError(err)

	return count
}

func (s *UsageBasedCostBasisCreateSuite) createFeatureMeters(ctx context.Context, namespace, key string) featurepkg.FeatureMeterCollection {
	s.T().Helper()

	testMeter := newTestMeter(namespace, key+"-meter")
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{testMeter}))

	feature, err := s.FeatureService.CreateFeature(ctx, featurepkg.CreateFeatureInputs{
		Namespace: namespace,
		Name:      key,
		Key:       key,
		MeterID:   lo.ToPtr(testMeter.ID),
	})
	s.Require().NoError(err)

	featureMeter := featurepkg.FeatureMeter{
		Feature: feature,
		Meter:   &testMeter,
	}

	return featurepkg.FeatureMeterCollection{
		ByKey: map[string]featurepkg.FeatureMeter{feature.Key: featureMeter},
		ByID:  map[string]featurepkg.FeatureMeter{feature.ID: featureMeter},
	}
}

func (s *UsageBasedCostBasisCreateSuite) newFiatCurrency(code currencyx.Code) *currencyx.FiatCurrency {
	s.T().Helper()

	fiatCurrency, err := currencyx.NewFiatCurrency(code)
	s.Require().NoError(err)

	return fiatCurrency
}

func (s *UsageBasedCostBasisCreateSuite) newUsageBasedIntent(
	customerID string,
	currency currencies.Currency,
	taxCodeID string,
	uniqueReferenceID string,
	featureKey string,
	settlementMode productcatalog.SettlementMode,
	costBasis *costbasis.Intent,
) usagebased.Intent {
	s.T().Helper()

	period := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}

	return usagebased.Intent{
		Intent: meta.Intent{
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        customerID,
			Currency:          currency,
			TaxConfig:         productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID},
			UniqueReferenceID: lo.ToPtr(uniqueReferenceID),
		},
		IntentMutableFields: usagebased.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              uniqueReferenceID,
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			InvoiceAt: period.To,
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
		SettlementMode: settlementMode,
		FeatureKey:     featureKey,
		CostBasis:      costBasis,
	}
}
