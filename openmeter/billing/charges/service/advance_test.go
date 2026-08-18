package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestAdvanceCharges(t *testing.T) {
	suite.Run(t, new(AdvanceChargesTestSuite))
}

type AdvanceChargesTestSuite struct {
	BaseSuite
}

func (s *AdvanceChargesTestSuite) SetupSuite() {
	s.UseRealRecognizer = true
	s.BaseSuite.SetupSuite()
}

func (s *AdvanceChargesTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *AdvanceChargesTestSuite) TestAdvanceChargesReturnsEmptyForAlreadyActiveCreditCharges() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-advance-usage-only")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From)

	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(100),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "flat-fee",
			}),
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(100),
				}),
				name:              "usage-based",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based",
				featureKey:        apiRequestsTotal.Feature.Key,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 2)

	// Create auto-advances credit-then-invoice flat fee charges that start now.
	flatFeeCharge, err := createdCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(flatFeeCharge.Status))

	// Create auto-advances credit-only usage-based charges: the returned charge is already active.
	usageBasedCharge, err := createdCharges[1].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(usageBasedCharge.Status))
	s.NotNil(usageBasedCharge.State.AdvanceAfter)
	s.True(servicePeriod.To.Equal(*usageBasedCharge.State.AdvanceAfter))

	// AdvanceCharges is a noop: both charges are already active and not yet past the service period.
	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Empty(advancedCharges)

	fetchedFlatFee := s.mustGetChargeByID(lo.Must(createdCharges[0].GetChargeID()))
	s.Equal(meta.ChargeTypeFlatFee, fetchedFlatFee.Type())
	fetchedFlatFeeCharge, err := fetchedFlatFee.AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatFeeCharge.Status, fetchedFlatFeeCharge.Status)

	// DB state matches what Create returned.
	fetchedUsageBased := s.mustGetChargeByID(usageBasedCharge.GetChargeID())
	usageBasedFromDB, err := fetchedUsageBased.AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(usageBasedCharge.Status, usageBasedFromDB.Status)
	s.NotNil(usageBasedFromDB.State.AdvanceAfter)
	s.True(servicePeriod.To.Equal(*usageBasedFromDB.State.AdvanceAfter))
}

func (s *AdvanceChargesTestSuite) TestAdvanceChargesActivatesCreditThenInvoiceFlatFeeAtServicePeriodStart() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-advance-empty")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-04-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From.Add(-time.Second))

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(100),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee-only",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "flat-fee-only",
			}),
		},
	})
	s.NoError(err)

	clock.SetTime(servicePeriod.From)

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 1)

	flatFeeCharge, err := advancedCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(flatFeeCharge.Status))
}

func (s *AdvanceChargesTestSuite) TestAdvanceChargesActivatesCreditThenInvoiceUsageBasedChargesAtServicePeriodStart() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-advance-credit-then-invoice")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-05-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-06-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From)

	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(100),
				}),
				name:              "usage-based-cti",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based-cti",
				featureKey:        apiRequestsTotal.Feature.Key,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 1)

	usageBasedChargeID, err := createdCharges[0].GetChargeID()
	s.NoError(err)

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advancedCharges, 1)
	s.Equal(meta.ChargeTypeUsageBased, advancedCharges[0].Type())

	advancedCharge, err := advancedCharges[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(advancedCharge.Status))
	s.NotNil(advancedCharge.State.AdvanceAfter)
	s.True(servicePeriod.To.Equal(*advancedCharge.State.AdvanceAfter))

	fetchedCharge := s.mustGetChargeByID(usageBasedChargeID)
	usageBasedCharge, err := fetchedCharge.AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(usageBasedCharge.Status))
	s.NotNil(usageBasedCharge.State.AdvanceAfter)
	s.True(servicePeriod.To.Equal(*usageBasedCharge.State.AdvanceAfter))
}

// usageBasedIntentServicePeriod is the fixed service period newUsageBasedIntent bakes
// into every intent it builds.
var usageBasedIntentServicePeriod = timeutil.ClosedPeriod{
	From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
}

// TestAdvanceChargesCustomCurrencyCreditThenInvoiceRecognitionDoesNotFail proves
// the native custom currency can pass through the real recognizer. Covered
// credit-backed usage is eligible; invoice-backed overage remains deferred.
func (s *AdvanceChargesTestSuite) TestAdvanceChargesCustomCurrencyCreditThenInvoiceRecognitionDoesNotFail() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-advance-custom-cti")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	featureMeters := s.createFeatureMeters(ctx, ns, "custom-cti-feature")

	clock.FreezeTime(usageBasedIntentServicePeriod.From)
	defer clock.UnFreeze()

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: s.newFiatCurrency("USD"),
		Rate:         alpacadecimal.NewFromFloat(0.25),
	})

	_, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
		Namespace: ns,
		Intents: []usagebased.Intent{
			s.newUsageBasedIntent(cust.ID, customCurrency, defaults.InvoicingTaxCodeID, "custom-cti", "custom-cti-feature", productcatalog.CreditThenInvoiceSettlementMode, &costBasisIntent),
		},
		FeatureMeters: featureMeters,
	})
	s.Require().NoError(err)

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(advancedCharges, 1)
}

// TestAdvanceChargesCustomCurrencyCreditOnlyRecognitionDoesNotFail proves the
// native custom currency can pass through the real recognizer for credit-only
// charges too.
func (s *AdvanceChargesTestSuite) TestAdvanceChargesCustomCurrencyCreditOnlyRecognitionDoesNotFail() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-advance-custom-credit-only")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	featureMeters := s.createFeatureMeters(ctx, ns, "custom-credit-only-feature")

	clock.FreezeTime(usageBasedIntentServicePeriod.From)
	defer clock.UnFreeze()

	_, err := s.Charges.usageBasedService.Create(ctx, usagebased.CreateInput{
		Namespace: ns,
		Intents: []usagebased.Intent{
			s.newUsageBasedIntent(cust.ID, customCurrency, defaults.InvoicingTaxCodeID, "custom-credit-only", "custom-credit-only-feature", productcatalog.CreditOnlySettlementMode, nil),
		},
		FeatureMeters: featureMeters,
	})
	s.Require().NoError(err)

	clock.SetTime(usageBasedIntentServicePeriod.To.Add(time.Second))

	_, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.Require().NoError(err)
}

// TestCollectEarningsRecognitionCurrencies proves each unique native charge
// currency is recognized once, independent of settlement mode.
func TestCollectEarningsRecognitionCurrencies(t *testing.T) {
	usd := currenciestestutils.NewFiatCurrency(t, "USD")
	custom := currenciestestutils.NewCustomCurrency(t, "ACME", 2)

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: lo.Must(currencyx.NewFiatCurrency("EUR")),
		Rate:         alpacadecimal.NewFromFloat(0.25),
	})

	newUsageBasedCharge := func(currency currencies.Currency, settlementMode productcatalog.SettlementMode, costBasis *costbasis.Intent) charges.Charge {
		return charges.NewCharge(usagebased.Charge{
			ChargeBase: usagebased.ChargeBase{
				Intent: usagebased.Intent{
					Intent:         meta.Intent{Currency: currency},
					FeatureKey:     "feature",
					SettlementMode: settlementMode,
					CostBasis:      costBasis,
				}.AsOverridableIntent(),
			},
		})
	}

	result, err := collectEarningsRecognitionCurrencies(charges.Charges{
		newUsageBasedCharge(usd, productcatalog.CreditThenInvoiceSettlementMode, nil),
		newUsageBasedCharge(custom, productcatalog.CreditOnlySettlementMode, nil),
		newUsageBasedCharge(custom, productcatalog.CreditThenInvoiceSettlementMode, &costBasisIntent),
	})
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, currencyx.Code("USD"), result[0].GetCode())
	require.Equal(t, custom.Reference(), result[1].Reference())
}
