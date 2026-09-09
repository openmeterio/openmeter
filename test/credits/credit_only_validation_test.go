package credits

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestCreditOnlyValidationSuite(t *testing.T) {
	suite.Run(t, new(CreditOnlyValidationSuite))
}

// CreditOnlyValidationSuite covers charge advancement when a credit-only charge's
// dependencies disappear after creation. Credit-only charges never create gathering
// lines, so AdvanceCharges is their only exposure to missing meters.
type CreditOnlyValidationSuite struct {
	BaseSuite
}

func (s *CreditOnlyValidationSuite) TestUsageBasedCreditOnlyAdvanceMissingMeterIsRetryable() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credit-only-usagebased-missing-meter-advance")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageChargeID meta.ChargeID
		meters        []meter.Meter
	)

	s.Run("Given an active credit-only usage charge with usage", func() {
		// given:
		// - a credit-only usage charge is created before its service period
		// when:
		// - the service period starts and the charge is advanced
		// then:
		// - the charge is active and waits for the end of its service period
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditOnlySettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "API requests",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "credit-only-missing-meter-api-requests",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.Require().NoError(err)
		s.Require().Len(created, 1)

		usageChargeID, err = created[0].GetChargeID()
		s.Require().NoError(err)
		s.RequireUsageBasedChargeStatus(usageChargeID, usagebased.StatusCreated)

		clock.FreezeTime(servicePeriod.From)
		defer clock.UnFreeze()
		_, err = s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})
		s.Require().NoError(err)

		usageCharge := s.RequireUsageBasedChargeStatus(usageChargeID, usagebased.StatusActive)
		s.Require().NotNil(usageCharge.State.AdvanceAfter)
		s.True(servicePeriod.To.Equal(*usageCharge.State.AdvanceAfter))

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			10,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		listed, err := s.MeterAdapter.ListMeters(ctx, meter.ListMetersParams{Namespace: ns})
		s.Require().NoError(err)
		meters = listed.Items
	})

	s.Run("When the meter is gone at the charge's due time", func() {
		// given:
		// - the charge is due for its final realization
		// when:
		// - the meter backing the feature is removed and the customer is advanced
		// then:
		// - advancement fails with a validation error and the charge is left untouched for a retry
		s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, nil))
		clock.FreezeTime(servicePeriod.To)
		defer clock.UnFreeze()

		_, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})
		s.Require().Error(err)
		s.True(billing.IsValidationIssueOnly(err), "expected a validation-only error, got: %v", err)

		issues, systemErr := billing.ToValidationIssues(err)
		s.Require().NoError(systemErr)
		s.Require().Len(issues, 1)
		s.Equal(billing.ErrInvoiceLineFeatureHasNoMeters.Code, issues[0].Code)
		s.Equal(
			fmt.Sprintf("feature[%s]: %s", apiRequestsTotal.Feature.Key, billing.ErrInvoiceLineFeatureHasNoMeters.Message),
			issues[0].Message,
		)

		usageCharge := s.RequireUsageBasedChargeStatus(usageChargeID, usagebased.StatusActive)
		s.Require().NotNil(usageCharge.State.AdvanceAfter)
		s.True(servicePeriod.To.Equal(*usageCharge.State.AdvanceAfter))
		s.Nil(usageCharge.State.CurrentRealizationRunID)
		s.Empty(usageCharge.Realizations)
		s.Empty(usageCharge.ValidationIssues)
	})

	s.Run("Then restoring the meter lets the charge realize normally", func() {
		// given:
		// - the meter is registered again
		// when:
		// - the customer is advanced again
		// then:
		// - the same advance succeeds and the charge starts its final realization
		clock.FreezeTime(servicePeriod.To)
		defer clock.UnFreeze()
		
		s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, meters))

		advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})
		s.Require().NoError(err)
		s.Require().Len(advanced, 1)

		usageCharge := s.RequireUsageBasedChargeStatus(usageChargeID, usagebased.StatusActiveRealizationWaitingForCollection)
		s.Empty(usageCharge.ValidationIssues)
		s.Len(usageCharge.Realizations, 1)
	})
}

func (s *CreditOnlyValidationSuite) TestFlatFeeCreditOnlyAdvancesWhenSiblingUsageMeterIsMissing() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credit-only-flatfee-sibling-missing-meter")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()

	usageServicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	flatFeeServicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-15T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(usageServicePeriod.From)
	defer clock.UnFreeze()

	// given:
	// - a credit-only usage charge is active and not due until the end of its service period
	// - a credit-only in-advance flat fee for the same customer becomes due first
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  usageServicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              "API requests",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "credit-only-sibling-api-requests",
				FeatureKey:        apiRequestsTotal.Feature.Key,
			}),
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  flatFeeServicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Name:              "monthly platform fee",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "credit-only-sibling-platform-fee",
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 2)

	usageCharge, err := created[0].AsUsageBasedCharge()
	s.Require().NoError(err)
	s.Equal(usagebased.StatusActive, usageCharge.Status)
	s.Require().NotNil(usageCharge.State.AdvanceAfter)
	s.True(usageServicePeriod.To.Equal(*usageCharge.State.AdvanceAfter))

	flatFeeCharge, err := created[1].AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusCreated, flatFeeCharge.Status)

	// when:
	// - the usage meter disappears before the flat fee is advanced
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, nil))
	clock.FreezeTime(flatFeeServicePeriod.From)

	advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})

	// then:
	// - the flat fee settles against credit without resolving meters for the not-due usage charge
	s.Require().NoError(err)
	s.Require().Len(advanced, 1)
	advancedFlatFee, err := advanced[0].AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusFinal, advancedFlatFee.Status)
	s.RequireFlatFeeChargeStatus(flatFeeCharge.GetChargeID(), flatfee.StatusFinal)

	reloadedUsageCharge := s.RequireUsageBasedChargeStatus(usageCharge.GetChargeID(), usagebased.StatusActive)
	s.Require().NotNil(reloadedUsageCharge.State.AdvanceAfter)
	s.True(usageServicePeriod.To.Equal(*reloadedUsageCharge.State.AdvanceAfter))
	s.Empty(reloadedUsageCharge.ValidationIssues)
}

func (s *CreditOnlyValidationSuite) TestFlatFeeCreditOnlyIgnoresMissingMeterOnOwnFeature() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credit-only-flatfee-own-missing-meter")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	defer apiRequestsTotal.Cleanup()

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	// given:
	// - a credit-only in-advance flat fee references a metered feature and is not yet due
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				Name:              "feature-backed platform fee",
				ManagedBy:         billing.SubscriptionManagedLine,
				UniqueReferenceID: "credit-only-own-feature-platform-fee",
				FeatureKey:        apiRequestsTotal.Feature.Key,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)

	flatFeeCharge, err := created[0].AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusCreated, flatFeeCharge.Status)

	// when:
	// - the feature's meter disappears before the fee is due
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, nil))
	clock.FreezeTime(servicePeriod.From)

	advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{Customer: cust.GetID()})

	// then:
	// - a flat fee never depends on the meter, so it settles against credit as usual
	s.Require().NoError(err)
	s.Require().Len(advanced, 1)
	advancedFlatFee, err := advanced[0].AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusFinal, advancedFlatFee.Status)
	s.Empty(s.RequireFlatFeeChargeStatus(flatFeeCharge.GetChargeID(), flatfee.StatusFinal).ValidationIssues)
}
