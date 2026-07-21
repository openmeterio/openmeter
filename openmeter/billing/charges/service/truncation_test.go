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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestChargeTimestampTruncation(t *testing.T) {
	suite.Run(t, new(ChargeTimestampTruncationTestSuite))
}

type ChargeTimestampTruncationTestSuite struct {
	BaseSuite
}

func (s *ChargeTimestampTruncationTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *ChargeTimestampTruncationTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *ChargeTimestampTruncationTestSuite) TestCreateTruncatesFlatFeeIntentAndProrationInputs() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-truncation-flatfee")
	defaultTaxCodes := s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")

	clock.FreezeTime(datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime())
	defer clock.UnFreeze()

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					CustomerID:        cust.ID,
					Currency:          currenciestestutils.NewFiatCurrency(s.T(), "USD"),
					UniqueReferenceID: lo.ToPtr("flat-fee-truncation"),
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaultTaxCodes.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat-fee-truncation",
						ServicePeriod:     timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:01.600Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:03.400Z", time.UTC).AsTime()},
						FullServicePeriod: timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00.400Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:03.400Z", time.UTC).AsTime()},
						BillingPeriod:     timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:01.600Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:03.400Z", time.UTC).AsTime()},
					},
					InvoiceAt:   datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:01.600Z", time.UTC).AsTime(),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
					ProRating: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
					AmountBeforeProration: alpacadecimal.NewFromInt(90),
				},
				SettlementMode: productcatalog.CreditOnlySettlementMode,
			}),
		},
	})
	s.NoError(err)
	s.Len(created, 1)

	createdFlatFee, err := created[0].AsFlatFeeCharge()
	s.NoError(err)

	expectedServicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:01Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:03Z", time.UTC).AsTime(),
	}
	expectedFullServicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:03Z", time.UTC).AsTime(),
	}
	expectedInvoiceAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:01Z", time.UTC).AsTime()

	s.Equal(expectedServicePeriod, createdFlatFee.Intent.GetEffectiveServicePeriod())
	s.Equal(expectedFullServicePeriod, createdFlatFee.Intent.GetEffectiveIntent().FullServicePeriod)
	s.Equal(expectedServicePeriod, createdFlatFee.Intent.GetEffectiveIntent().BillingPeriod)
	s.True(expectedInvoiceAt.Equal(createdFlatFee.Intent.GetEffectiveInvoiceAt()))
	s.True(alpacadecimal.NewFromInt(60).Equal(createdFlatFee.State.AmountAfterProration))
}

func (s *ChargeTimestampTruncationTestSuite) TestUsageBasedAdvanceTruncatesPersistedCalculationTimestamps() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-truncation-usagebased")
	defaultTaxCodes := s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00.750Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00.750Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00.900Z", time.UTC).AsTime())
	defer clock.UnFreeze()

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(usagebased.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					CustomerID:        cust.ID,
					Currency:          currenciestestutils.NewFiatCurrency(s.T(), "USD"),
					UniqueReferenceID: lo.ToPtr("usage-based-truncation"),
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaultTaxCodes.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "usage-based-truncation",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt: datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00.750Z", time.UTC).AsTime(),
					Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(100),
					})),
				},
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				FeatureKey:     apiRequestsTotal.Feature.Key,
			}),
		},
	})
	s.NoError(err)
	s.Len(created, 1)

	createdUsageBased, err := created[0].AsUsageBasedCharge()
	s.NoError(err)
	s.NotNil(createdUsageBased.State.AdvanceAfter)
	s.True(datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00Z", time.UTC).AsTime().Equal(*createdUsageBased.State.AdvanceAfter))
	s.Equal(
		timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00Z", time.UTC).AsTime(),
		},
		createdUsageBased.Intent.GetEffectiveServicePeriod(),
	)
	s.True(datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00Z", time.UTC).AsTime().Equal(createdUsageBased.Intent.GetEffectiveInvoiceAt()))

	clock.FreezeTime(datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:02:00.900Z", time.UTC).AsTime())

	advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Len(advanced, 1)

	finalCharge, err := advanced[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Nil(finalCharge.State.AdvanceAfter)
	s.Len(finalCharge.Realizations, 1)

	finalRun := finalCharge.Realizations[0]
	expectedCollectionEnd := datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00Z", time.UTC).AsTime()
	expectedStoredAtLT := datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00Z", time.UTC).AsTime()

	s.True(expectedCollectionEnd.Equal(finalRun.StoredAtLT))
	s.True(expectedStoredAtLT.Equal(finalRun.StoredAtLT))
}
