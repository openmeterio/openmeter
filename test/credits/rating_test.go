package credits

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestRatingTestSuite(t *testing.T) {
	suite.Run(t, new(RatingTestSuite))
}

type RatingTestSuite struct {
	BaseSuite
}

func (s *RatingTestSuite) TestListChargesExpandsRealtimeUsageForMultipleUsageBasedCharges() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("credits-rating-list-charges")

	const (
		standardRateReference = "usage-based-standard-rate"
		doubleRateReference   = "usage-based-double-rate"
	)

	cust := s.CreateLedgerBackedCustomer(ns, "rating-list-charges")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-04-01T00:00:00Z", time.UTC).AsTime(),
	}
	createAt := servicePeriod.From
	firstUsageAt := servicePeriod.From.Add(2 * time.Hour)
	secondUsageAt := servicePeriod.From.Add(5 * time.Hour)
	readAt := servicePeriod.From.Add(6 * time.Hour)

	clock.SetTime(createAt)
	defer clock.ResetTime()

	createdCharges, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     servicePeriod,
				SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				Name:              standardRateReference,
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: standardRateReference,
				FeatureKey:        apiRequestsTotal.Feature.Key,
			}),
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     servicePeriod,
				SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(2)}),
				Name:              doubleRateReference,
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: doubleRateReference,
				FeatureKey:        apiRequestsTotal.Feature.Key,
			}),
		},
	})
	s.NoError(err)
	s.Len(createdCharges, 2)

	// One event is before the service period and must not affect the current totals.
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotal.Feature.Key, 100, servicePeriod.From.Add(-time.Minute))
	// Two in-period events are visible at readAt and should be summed by realtime usage expansion.
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotal.Feature.Key, 2, firstUsageAt)
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotal.Feature.Key, 5, secondUsageAt)
	clock.SetTime(readAt)

	result, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Page:        pagination.NewPage(1, 20),
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
		ChargeTypes: []meta.ChargeType{meta.ChargeTypeUsageBased},
		Expands: meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandRealtimeUsage,
		},
	})
	s.NoError(err)
	s.Len(result.Items, 2)

	realtimeByReference := map[string]alpacadecimal.Decimal{}
	bookedByReference := map[string]alpacadecimal.Decimal{}
	for _, item := range result.Items {
		charge, err := item.AsUsageBasedCharge()
		s.NoError(err)

		s.NotNil(charge.Expands.RealtimeUsage)
		s.NotNil(charge.Intent.UniqueReferenceID)
		realtimeByReference[*charge.Intent.UniqueReferenceID] = charge.Expands.RealtimeUsage.Total
		bookedByReference[*charge.Intent.UniqueReferenceID] = charge.Realizations.Sum().Total
	}

	s.True(alpacadecimal.NewFromInt(7).Equal(realtimeByReference[standardRateReference]))
	s.True(alpacadecimal.NewFromInt(14).Equal(realtimeByReference[doubleRateReference]))
	s.True(alpacadecimal.Zero.Equal(bookedByReference[standardRateReference]))
	s.True(alpacadecimal.Zero.Equal(bookedByReference[doubleRateReference]))
}
