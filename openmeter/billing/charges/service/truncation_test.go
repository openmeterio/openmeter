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
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

	cust := s.CreateTestCustomer(ns, "test-subject")

	clock.FreezeTime(datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime())
	defer clock.UnFreeze()

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					Name:              "flat-fee-truncation",
					ManagedBy:         billing.SubscriptionManagedLine,
					CustomerID:        cust.ID,
					Currency:          currencyx.Code("USD"),
					ServicePeriod:     timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:01.600Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:03.400Z", time.UTC).AsTime()},
					FullServicePeriod: timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00.400Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:03.400Z", time.UTC).AsTime()},
					BillingPeriod:     timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:01.600Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:03.400Z", time.UTC).AsTime()},
					UniqueReferenceID: lo.ToPtr("flat-fee-truncation"),
				},
				InvoiceAt:      datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:01.600Z", time.UTC).AsTime(),
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				PaymentTerm:    productcatalog.InAdvancePaymentTerm,
				ProRating: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
				AmountBeforeProration: alpacadecimal.NewFromInt(90),
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

	s.Equal(expectedServicePeriod, createdFlatFee.Intent.ServicePeriod)
	s.Equal(expectedFullServicePeriod, createdFlatFee.Intent.FullServicePeriod)
	s.Equal(expectedServicePeriod, createdFlatFee.Intent.BillingPeriod)
	s.True(expectedInvoiceAt.Equal(createdFlatFee.Intent.InvoiceAt))
	s.True(alpacadecimal.NewFromInt(60).Equal(createdFlatFee.State.AmountAfterProration))
}

func (s *ChargeTimestampTruncationTestSuite) TestUsageBasedAdvanceTruncatesPersistedCalculationTimestamps() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-service-truncation-usagebased")

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
					Name:              "usage-based-truncation",
					ManagedBy:         billing.SubscriptionManagedLine,
					CustomerID:        cust.ID,
					Currency:          currencyx.Code("USD"),
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
					UniqueReferenceID: lo.ToPtr("usage-based-truncation"),
				},
				InvoiceAt:      datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00.750Z", time.UTC).AsTime(),
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				FeatureKey:     apiRequestsTotal.Feature.Key,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(100),
				})),
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
		createdUsageBased.Intent.ServicePeriod,
	)
	s.True(datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00Z", time.UTC).AsTime().Equal(createdUsageBased.Intent.InvoiceAt))

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
	expectedAsOf := datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:01:00Z", time.UTC).AsTime()

	s.True(expectedCollectionEnd.Equal(finalRun.CollectionEnd))
	s.True(expectedAsOf.Equal(finalRun.AsOf))
}

func (s *ChargeTimestampTruncationTestSuite) TestTmpApplyPatchToCreateIntentTruncatesReplacementPeriods() {
	newServicePeriodTo := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:09.900Z", time.UTC).AsTime()
	newFullServicePeriodTo := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:10.900Z", time.UTC).AsTime()
	newBillingPeriodTo := datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:11.900Z", time.UTC).AsTime()

	flatFeeIntent := flatfee.Intent{
		Intent: meta.Intent{
			Name:              "flat-fee",
			ManagedBy:         billing.SubscriptionManagedLine,
			CustomerID:        "cust",
			Currency:          currencyx.Code("USD"),
			ServicePeriod:     timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:05Z", time.UTC).AsTime()},
			FullServicePeriod: timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:06Z", time.UTC).AsTime()},
			BillingPeriod:     timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:07Z", time.UTC).AsTime()},
		},
		InvoiceAt:      datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(),
		SettlementMode: productcatalog.CreditOnlySettlementMode,
		PaymentTerm:    productcatalog.InAdvancePaymentTerm,
	}

	usageBasedIntent := usagebased.Intent{
		Intent: meta.Intent{
			Name:              "usage-based",
			ManagedBy:         billing.SubscriptionManagedLine,
			CustomerID:        "cust",
			Currency:          currencyx.Code("USD"),
			ServicePeriod:     timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:05Z", time.UTC).AsTime()},
			FullServicePeriod: timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:06Z", time.UTC).AsTime()},
			BillingPeriod:     timeutil.ClosedPeriod{From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(), To: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:07Z", time.UTC).AsTime()},
		},
		InvoiceAt:      datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:07Z", time.UTC).AsTime(),
		SettlementMode: productcatalog.CreditOnlySettlementMode,
		FeatureKey:     "api_requests",
		Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		})),
	}

	testCases := []struct {
		name   string
		charge charges.Charge
		assert func(charges.ChargeIntent)
	}{
		{
			name:   "flat fee",
			charge: charges.NewCharge(flatfee.Charge{ChargeBase: flatfee.ChargeBase{Intent: flatFeeIntent}}),
			assert: func(intent charges.ChargeIntent) {
				typed, err := intent.AsFlatFeeIntent()
				s.NoError(err)
				s.True(datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:09Z", time.UTC).AsTime().Equal(typed.ServicePeriod.To))
				s.True(datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:10Z", time.UTC).AsTime().Equal(typed.FullServicePeriod.To))
				s.True(datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:11Z", time.UTC).AsTime().Equal(typed.BillingPeriod.To))
			},
		},
		{
			name:   "usage based",
			charge: charges.NewCharge(usagebased.Charge{ChargeBase: usagebased.ChargeBase{Intent: usageBasedIntent}}),
			assert: func(intent charges.ChargeIntent) {
				typed, err := intent.AsUsageBasedIntent()
				s.NoError(err)
				s.True(datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:09Z", time.UTC).AsTime().Equal(typed.ServicePeriod.To))
				s.True(datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:10Z", time.UTC).AsTime().Equal(typed.FullServicePeriod.To))
				s.True(datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:11Z", time.UTC).AsTime().Equal(typed.BillingPeriod.To))
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			intent, err := tmpApplyPatchToCreateIntent(tc.charge, newServicePeriodTo, newFullServicePeriodTo, newBillingPeriodTo)
			s.NoError(err)
			tc.assert(intent)
		})
	}
}
