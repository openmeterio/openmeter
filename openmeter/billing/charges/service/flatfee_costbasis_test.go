package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	dbchargeflatfeecostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeecostbasis"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestFlatFeeCostBasisCreate(t *testing.T) {
	suite.Run(t, new(FlatFeeCostBasisCreateSuite))
}

type FlatFeeCostBasisCreateSuite struct {
	BaseSuite
}

func (s *FlatFeeCostBasisCreateSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *FlatFeeCostBasisCreateSuite) TearDownTest() {
	s.setCustomCurrencyEnabled(false)
	s.BaseSuite.TearDownTest()
}

func (s *FlatFeeCostBasisCreateSuite) TestCustomCurrencyCreditOnlyIsDisabledByDefault() {
	// given:
	// - a valid custom-currency credit-only flat fee
	// - the test-only custom-currency switch left at its default value
	// when:
	// - the charge is created
	// then:
	// - creation is rejected before the charge is persisted
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("flat-fee-cost-basis-disabled")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "flat-fee-cost-basis-disabled")
	currency := s.createTestCustomCurrency(ctx, namespace)

	_, err := s.Charges.flatFeeService.Create(ctx, flatfee.CreateInput{
		Namespace: namespace,
		Intents: []flatfee.Intent{
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "disabled", productcatalog.CreditOnlySettlementMode, nil),
		},
		FeatureMeters: feature.FeatureMeterCollection{},
	})
	s.Require().ErrorIs(err, meta.ErrCustomCurrencyNotSupported)
	s.Require().Equal(0, s.countCostBases(namespace))
}

func (s *FlatFeeCostBasisCreateSuite) TestCreatePersistsManualPinnedAndDynamicCostBasesInInputOrder() {
	// given:
	// - manual, pinned, and dynamic cost-basis intents in one custom-currency batch
	// when:
	// - custom-currency creation is enabled, the flat fees are created, and each is reloaded through the charge service
	// then:
	// - each charge references its own persisted cost basis and its intent and resolved state survive the normal read path
	ctx := s.T().Context()
	now := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("flat-fee-cost-basis-modes")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "flat-fee-cost-basis-modes")
	currency := s.createTestCustomCurrency(ctx, namespace)
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
	created, err := s.Charges.flatFeeService.Create(ctx, flatfee.CreateInput{
		Namespace: namespace,
		Intents: []flatfee.Intent{
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "manual", productcatalog.CreditThenInvoiceSettlementMode, &manualIntent),
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "pinned", productcatalog.CreditThenInvoiceSettlementMode, &pinnedIntent),
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "dynamic", productcatalog.CreditThenInvoiceSettlementMode, &dynamicIntent),
		},
		FeatureMeters: feature.FeatureMeterCollection{},
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
		s.Require().NotNil(result.GatheringLineToCreate, "charge index %d", idx)
		s.Require().Equal(currencyx.FiatCode("USD"), result.GatheringLineToCreate.Currency, "charge index %d", idx)
		s.Require().NotNil(result.Charge.State.CostBasisID, "charge index %d", idx)

		reloaded, err := s.mustGetChargeByID(result.Charge.GetChargeID()).AsFlatFeeCharge()
		s.Require().NoError(err)
		s.Require().Equal(result.Charge.Intent.GetCostBasisIntent().Kind(), reloaded.Intent.GetCostBasisIntent().Kind(), "charge index %d", idx)
		s.Require().Equal(result.Charge.State.CostBasisID, reloaded.State.CostBasisID, "charge index %d", idx)
		s.Require().Equal(result.Charge.State.ResolvedCostBasis, reloaded.State.ResolvedCostBasis, "charge index %d", idx)

		chargeEntity, err := s.DBClient.ChargeFlatFee.Get(ctx, result.Charge.ID)
		s.Require().NoError(err)
		s.Require().NotNil(chargeEntity.CostBasisID)
		s.Require().Equal(*result.Charge.State.CostBasisID, *chargeEntity.CostBasisID)
		seenCostBasisIDs[*chargeEntity.CostBasisID] = struct{}{}

		costBasisEntity, err := s.DBClient.ChargeFlatFeeCostBasis.Get(ctx, *chargeEntity.CostBasisID)
		s.Require().NoError(err)
		persisted, err := costbasis.Get(costBasisEntity)
		s.Require().NoError(err)
		s.Require().Equal(result.Charge.Intent.GetCostBasisIntent().Kind(), persisted.Intent.Kind(), "charge index %d", idx)
		s.Require().Equal(result.Charge.State.ResolvedCostBasis, persisted.State, "charge index %d", idx)
	}
	s.Require().Len(seenCostBasisIDs, 3)
}

func (s *FlatFeeCostBasisCreateSuite) TestSetResolvedDynamicCostBasisIsRetrySafe() {
	// given:
	// - an unresolved dynamic cost basis attached to a flat-fee charge
	// - a first persisted resolution
	// when:
	// - a stale retry attempts to persist a different resolution
	// then:
	// - the original persisted resolution is returned and remains unchanged
	ctx := s.T().Context()
	now := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("flat-fee-cost-basis-retry")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "flat-fee-cost-basis-retry")
	currency := s.createTestCustomCurrency(ctx, namespace)
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
	created, err := s.Charges.flatFeeService.Create(ctx, flatfee.CreateInput{
		Namespace: namespace,
		Intents: []flatfee.Intent{
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "dynamic-retry", productcatalog.CreditThenInvoiceSettlementMode, &dynamicIntent),
		},
		FeatureMeters: feature.FeatureMeterCollection{},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	s.Require().NotNil(created[0].Charge.State.CostBasisID)

	firstState := costbasis.State{
		CostBasis:   currencyCostBasis.Rate,
		CostBasisID: lo.ToPtr(currencyCostBasis.ID),
		ResolvedAt:  now,
	}
	first, err := s.FlatFeeAdapter.SetResolvedCostBasis(ctx, costbasis.SetResolvedCostBasisInput{
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
	retried, err := s.FlatFeeAdapter.SetResolvedCostBasis(ctx, costbasis.SetResolvedCostBasisInput{
		NamespacedID: first.NamespacedID,
		State:        retryState,
	})
	s.Require().NoError(err)
	s.Require().Equal(&firstState, retried.State)

	reloaded, err := s.mustGetChargeByID(created[0].Charge.GetChargeID()).AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Require().Equal(created[0].Charge.State.CostBasisID, reloaded.State.CostBasisID)
	s.Require().Equal(&firstState, reloaded.State.ResolvedCostBasis)
}

func (s *FlatFeeCostBasisCreateSuite) TestResolveDynamicCostBasisUsesServicePeriodFrom() {
	// given:
	// - two successive currency cost bases
	// when:
	// - dynamic resolution uses a service-period start in the first interval
	// then:
	// - the first value is selected and ResolvedAt equals the service-period start
	ctx := s.T().Context()
	firstEffectiveAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	secondEffectiveAt := firstEffectiveAt.Add(10 * 24 * time.Hour)
	servicePeriodFrom := firstEffectiveAt.Add(5 * 24 * time.Hour)
	clock.FreezeTime(firstEffectiveAt)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("flat-fee-cost-basis-resolution-times")
	currency := s.createTestCustomCurrency(ctx, namespace)
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

	resolver, err := costbasis.NewResolver(costbasis.ResolverConfig{Currencies: s.CurrencyService})
	s.Require().NoError(err)
	resolved, err := resolver.ResolveDynamicState(ctx, costbasis.ResolveDynamicStateInput{
		CurrencyID: currency.NamespacedID,
		Intent: costbasis.NewIntent(costbasis.DynamicIntent{
			FiatCurrency: s.newFiatCurrency("USD"),
		}),
		ServicePeriodFrom: servicePeriodFrom,
	})
	s.Require().NoError(err)
	s.Require().Equal(first.ID, *resolved.CostBasisID)
	s.Require().Equal(first.Rate.InexactFloat64(), resolved.CostBasis.InexactFloat64())
	s.Require().Equal(servicePeriodFrom, resolved.ResolvedAt)
}

func (s *FlatFeeCostBasisCreateSuite) TestDynamicCostBasisResolvesWhenChargeBecomesActive() {
	// given:
	// - an unresolved dynamic cost basis with two successive currency rates
	// - activation delayed until after the second rate becomes effective
	// when:
	// - the charge advances from created to active
	// then:
	// - it persists the rate effective at service-period start and records that as ResolvedAt
	ctx := s.T().Context()
	servicePeriodFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	secondEffectiveAt := servicePeriodFrom.Add(10 * 24 * time.Hour)
	activationAt := secondEffectiveAt.Add(24 * time.Hour)
	clock.FreezeTime(servicePeriodFrom)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("flat-fee-cost-basis-active")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "flat-fee-cost-basis-active")
	currency := s.createTestCustomCurrency(ctx, namespace)
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
	created, err := s.Charges.flatFeeService.Create(ctx, flatfee.CreateInput{
		Namespace: namespace,
		Intents: []flatfee.Intent{
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "dynamic-active", productcatalog.CreditThenInvoiceSettlementMode, &dynamicIntent),
		},
		FeatureMeters: feature.FeatureMeterCollection{},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	s.Require().Equal(flatfee.StatusCreated, created[0].Charge.Status)
	s.Require().NotNil(created[0].Charge.State.CostBasisID)
	s.Require().Nil(created[0].Charge.State.ResolvedCostBasis)

	advanced, err := s.Charges.flatFeeService.AdvanceCharge(ctx, flatfee.AdvanceChargeInput{
		ChargeID: created[0].Charge.GetChargeID(),
	})
	s.Require().NoError(err)
	s.Require().NotNil(advanced)
	s.Require().Equal(flatfee.StatusActive, advanced.Status)
	s.Require().Equal(created[0].Charge.State.CostBasisID, advanced.State.CostBasisID)
	s.Require().NotNil(advanced.State.ResolvedCostBasis)
	s.Require().Equal(first.ID, *advanced.State.ResolvedCostBasis.CostBasisID)
	s.Require().Equal(first.Rate.InexactFloat64(), advanced.State.ResolvedCostBasis.CostBasis.InexactFloat64())
	s.Require().Equal(servicePeriodFrom, advanced.State.ResolvedCostBasis.ResolvedAt)

	reloaded, err := s.mustGetChargeByID(created[0].Charge.GetChargeID()).AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Require().Equal(advanced.State.CostBasisID, reloaded.State.CostBasisID)
	s.Require().Equal(advanced.State.ResolvedCostBasis, reloaded.State.ResolvedCostBasis)
}

func (s *FlatFeeCostBasisCreateSuite) TestPinnedCostBasisMustMatchCurrencyAndFiatCurrency() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("flat-fee-cost-basis-pinned-mismatch")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "flat-fee-cost-basis-pinned-mismatch")
	currency := s.createTestCustomCurrency(ctx, namespace)
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
			// - the flat fee is created
			// then:
			// - resolution rejects the mismatch before charge cost-basis persistence
			intent := costbasis.NewIntent(costbasis.PinnedIntent{
				FiatCurrency:        s.newFiatCurrency("USD"),
				CurrencyCostBasisID: test.costBasisID,
			})

			_, err := s.Charges.flatFeeService.Create(ctx, flatfee.CreateInput{
				Namespace: namespace,
				Intents: []flatfee.Intent{
					s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "mismatch-"+test.name, productcatalog.CreditThenInvoiceSettlementMode, &intent),
				},
				FeatureMeters: feature.FeatureMeterCollection{},
			})
			s.Require().ErrorContains(err, test.errorText)
		})
	}
	s.Require().Equal(0, s.countCostBases(namespace))
}

func (s *FlatFeeCostBasisCreateSuite) TestCreateRollsBackCostBasesWhenChargeCreationFails() {
	// given:
	// - two cost-basis-backed intents with the same charge unique reference
	// when:
	// - the charge constraint rejects the bulk create after cost bases were inserted
	// then:
	// - the surrounding transaction removes all newly inserted charge cost bases
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("flat-fee-cost-basis-rollback")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "flat-fee-cost-basis-rollback")
	currency := s.createTestCustomCurrency(ctx, namespace)
	intent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: s.newFiatCurrency("USD"),
		Rate:         alpacadecimal.NewFromInt(2),
	})

	s.setCustomCurrencyEnabled(true)
	_, err := s.Charges.flatFeeService.Create(ctx, flatfee.CreateInput{
		Namespace: namespace,
		Intents: []flatfee.Intent{
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "duplicate", productcatalog.CreditThenInvoiceSettlementMode, &intent),
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "duplicate", productcatalog.CreditThenInvoiceSettlementMode, &intent),
		},
		FeatureMeters: feature.FeatureMeterCollection{},
	})
	s.Require().Error(err)
	s.Require().Equal(0, s.countCostBases(namespace))
}

func (s *FlatFeeCostBasisCreateSuite) TestCreateWithoutCostBasisLeavesChargeReferenceEmpty() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("flat-fee-without-cost-basis")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "flat-fee-without-cost-basis")
	currency := s.createTestCustomCurrency(ctx, namespace)
	s.setCustomCurrencyEnabled(true)

	created, err := s.Charges.flatFeeService.Create(ctx, flatfee.CreateInput{
		Namespace: namespace,
		Intents: []flatfee.Intent{
			s.newFlatFeeIntent(customer.ID, currency, defaults.InvoicingTaxCodeID, "credit-only", productcatalog.CreditOnlySettlementMode, nil),
		},
		FeatureMeters: feature.FeatureMeterCollection{},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	s.Require().Nil(created[0].Charge.Intent.GetCostBasisIntent())
	s.Require().Nil(created[0].Charge.State.CostBasisID)
	s.Require().Nil(created[0].Charge.State.ResolvedCostBasis)

	chargeEntity, err := s.DBClient.ChargeFlatFee.Get(ctx, created[0].Charge.ID)
	s.Require().NoError(err)
	s.Require().Nil(chargeEntity.CostBasisID)
	s.Require().Equal(0, s.countCostBases(namespace))
}

func (s *FlatFeeCostBasisCreateSuite) setCustomCurrencyEnabled(enabled bool) {
	s.T().Helper()
	enabler, ok := s.Charges.flatFeeService.(customCurrencyEnabler)
	s.Require().True(ok)
	s.Require().NoError(enabler.SetEnableCustomCurrency(s.T(), enabled))
}

func (s *FlatFeeCostBasisCreateSuite) countCostBases(namespace string) int {
	s.T().Helper()

	count, err := s.DBClient.ChargeFlatFeeCostBasis.Query().
		Where(dbchargeflatfeecostbasis.Namespace(namespace)).
		Count(s.T().Context())
	s.Require().NoError(err)

	return count
}

func (s *FlatFeeCostBasisCreateSuite) newFiatCurrency(code currencyx.Code) *currencyx.FiatCurrency {
	s.T().Helper()

	fiatCurrency, err := currencyx.NewFiatCurrency(code)
	s.Require().NoError(err)

	return fiatCurrency
}

func (s *FlatFeeCostBasisCreateSuite) newFlatFeeIntent(
	customerID string,
	currency currencies.Currency,
	taxCodeID string,
	uniqueReferenceID string,
	settlementMode productcatalog.SettlementMode,
	costBasis *costbasis.Intent,
) flatfee.Intent {
	s.T().Helper()

	period := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}

	return flatfee.Intent{
		Intent: meta.Intent{
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        customerID,
			Currency:          currency,
			TaxConfig:         productcatalog.TaxCodeConfig{TaxCodeID: taxCodeID},
			UniqueReferenceID: lo.ToPtr(uniqueReferenceID),
		},
		IntentMutableFields: flatfee.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              uniqueReferenceID,
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			InvoiceAt:             period.From,
			PaymentTerm:           productcatalog.InAdvancePaymentTerm,
			AmountBeforeProration: alpacadecimal.NewFromInt(10),
		},
		SettlementMode: settlementMode,
		CostBasis:      costBasis,
	}
}
