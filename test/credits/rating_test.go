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
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
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
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)

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
		Intents: charges.NewCreateChargeIntents(
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
		),
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
		s.NotNil(charge.Intent.GetUniqueReferenceID())
		realtimeByReference[*charge.Intent.GetUniqueReferenceID()] = charge.Expands.RealtimeUsage.Total
		bookedByReference[*charge.Intent.GetUniqueReferenceID()] = charge.Realizations.Sum().Total
	}

	s.True(alpacadecimal.NewFromInt(7).Equal(realtimeByReference[standardRateReference]))
	s.True(alpacadecimal.NewFromInt(14).Equal(realtimeByReference[doubleRateReference]))
	s.True(alpacadecimal.Zero.Equal(bookedByReference[standardRateReference]))
	s.True(alpacadecimal.Zero.Equal(bookedByReference[doubleRateReference]))

	// A negative live snapshot remains readable and surfaces its warning on each expanded charge.
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotal.Feature.Key, -12, secondUsageAt.Add(time.Minute))

	result, err = s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Page:        pagination.NewPage(1, 20),
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
		ChargeTypes: []meta.ChargeType{meta.ChargeTypeUsageBased},
		Expands: meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandRealtimeUsage,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(result.Items, 2)
	for _, item := range result.Items {
		charge, err := item.AsUsageBasedCharge()
		s.Require().NoError(err)
		s.Require().NotNil(charge.Expands.RealtimeUsage)
		s.Zero(charge.Expands.RealtimeUsage.Total.InexactFloat64())
		s.Require().Len(charge.ValidationIssues, 1)
		s.Equal(billing.WarnNegativeMeteredQuantityClamped.Code, charge.ValidationIssues[0].Code)
		s.Equal("-5", charge.ValidationIssues[0].Attributes["original_metered_quantity"])
	}

	unexpanded, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Page:        pagination.NewPage(1, 20),
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
		ChargeTypes: []meta.ChargeType{meta.ChargeTypeUsageBased},
	})
	s.Require().NoError(err)
	s.Require().Len(unexpanded.Items, 2)
	for _, item := range unexpanded.Items {
		charge, err := item.AsUsageBasedCharge()
		s.Require().NoError(err)
		s.Empty(charge.ValidationIssues)
	}
}

func (s *RatingTestSuite) TestListChargesRealtimeUsageAsOf() {
	// given: a usage-based charge with one event stored on time and one stored late.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("credits-rating-list-charges-as-of")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)

	cust := s.CreateLedgerBackedCustomer(ns, "rating-list-charges-as-of")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-04-01T00:00:00Z", time.UTC).AsTime(),
	}
	closeAt := servicePeriod.From.Add(6 * time.Hour)

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.NewCreateChargeIntents(
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:          cust.GetID(),
				Currency:          USD,
				ServicePeriod:     servicePeriod,
				SettlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				Price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				Name:              "usage-based-as-of",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "usage-based-as-of",
				FeatureKey:        apiRequestsTotal.Feature.Key,
			}),
		),
	})
	s.Require().NoError(err)

	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotal.Feature.Key, 2, servicePeriod.From.Add(2*time.Hour))
	// Happened before the close but was ingested after it, so a close read taken at
	// closeAt never saw it.
	s.MockStreamingConnector.AddSimpleEvent(apiRequestsTotal.Feature.Key, 5, servicePeriod.From.Add(3*time.Hour), streamingtestutils.WithStoredAt(closeAt.Add(time.Hour)))

	clock.FreezeTime(closeAt.Add(6 * time.Hour))

	listRealtime := func(asOf *time.Time) (charges.Charge, error) {
		result, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
			Page:              pagination.NewPage(1, 20),
			Namespace:         ns,
			CustomerIDs:       []string{cust.ID},
			ChargeTypes:       []meta.ChargeType{meta.ChargeTypeUsageBased},
			Expands:           meta.Expands{meta.ExpandRealizations, meta.ExpandRealtimeUsage},
			RealtimeUsageAsOf: asOf,
		})
		if err != nil {
			return charges.Charge{}, err
		}

		s.Require().Len(result.Items, 1)

		return result.Items[0], nil
	}

	// when: the close is re-read after the late event arrived.
	closeRead, err := listRealtime(&closeAt)
	s.Require().NoError(err)
	liveRead, err := listRealtime(nil)
	s.Require().NoError(err)

	// then: the as-of read matches what a live read returned at the close; the live
	// read includes the late event.
	closeCharge, err := closeRead.AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Require().NotNil(closeCharge.Expands.RealtimeUsage)
	s.Equal(2.0, closeCharge.Expands.RealtimeUsage.Total.InexactFloat64())

	liveCharge, err := liveRead.AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Require().NotNil(liveCharge.Expands.RealtimeUsage)
	s.Equal(7.0, liveCharge.Expands.RealtimeUsage.Total.InexactFloat64())

	// when: the cutoff is in the future, or the realtime expand is missing.
	future := clock.Now().Add(time.Minute)
	_, futureErr := listRealtime(&future)
	_, noExpandErr := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Page:              pagination.NewPage(1, 20),
		Namespace:         ns,
		RealtimeUsageAsOf: &closeAt,
	})

	// then: both are rejected as invalid input instead of silently reading live usage.
	s.True(models.IsGenericValidationError(futureErr), futureErr)
	s.True(models.IsGenericValidationError(noExpandErr), noExpandErr)
}
