package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
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
	s.BaseSuite.SetupSuite()
}

func (s *AdvanceChargesTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *AdvanceChargesTestSuite) TestAdvanceChargesReturnsEmptyForAlreadyActiveCreditCharges() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-advance-usage-only")

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

func (s *AdvanceChargesTestSuite) TestAdvanceChargesSkipsInvoiceOnlyFlatFeeCharges() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-advance-empty")

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-03-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-04-01T00:00:00Z", time.UTC).AsTime(),
	}

	_, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.InvoiceOnlySettlementMode,
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

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: cust.GetID(),
	})
	s.NoError(err)
	s.Empty(advancedCharges)
}

func (s *AdvanceChargesTestSuite) TestAdvanceChargesActivatesCreditThenInvoiceUsageBasedChargesAtServicePeriodStart() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-advance-credit-then-invoice")

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
