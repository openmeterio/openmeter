package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// These suites drive the REAL charges path end-to-end: a usage-based charge whose
// intent carries a unit_config is created, usage is seeded, and the charge is
// invoiced — exercising rating.GenerateDetailedLines via RateableIntent.GetUnitConfig.
// The enabled/disabled split proves the unitConfig.enabled flag actually gates the
// converted amount through the production wiring (not just the in-memory mutator).

func TestUsageBasedUnitConfigRatingEnabled(t *testing.T) {
	suite.Run(t, new(unitConfigRatingEnabledSuite))
}

type unitConfigRatingEnabledSuite struct {
	BaseSuite
}

func (s *unitConfigRatingEnabledSuite) SetupSuite() {
	s.UnitConfigEnabled = true
	s.BaseSuite.SetupSuite()
}

func (s *unitConfigRatingEnabledSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *unitConfigRatingEnabledSuite) TestRatesConvertedQuantity() {
	// flag on: 7400 raw / 1000, ceiling => 8 billed units * $1 = $8.
	s.runUnitConfigChargesScenario(8)
}

func TestUsageBasedUnitConfigRatingDisabled(t *testing.T) {
	suite.Run(t, new(unitConfigRatingDisabledSuite))
}

type unitConfigRatingDisabledSuite struct {
	BaseSuite
}

func (s *unitConfigRatingDisabledSuite) SetupSuite() {
	// UnitConfigEnabled defaults to false: the intent still carries a unit_config,
	// but the mutator is not registered, so rating must bill the raw quantity.
	s.BaseSuite.SetupSuite()
}

func (s *unitConfigRatingDisabledSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *unitConfigRatingDisabledSuite) TestRatesRawQuantityWhenFlagOff() {
	// flag off: unit_config ignored, 7400 raw units * $1 = $7400 (parity with today).
	s.runUnitConfigChargesScenario(7400)
}

// runUnitConfigChargesScenario creates a usage-based charge carrying a divide-by-1000
// ceiling unit_config, seeds 7400 raw units, invoices mid-period, and asserts the
// rated line amount matches expectedAmount.
func (s *BaseSuite) runUnitConfigChargesScenario(expectedAmount float64) {
	s.T().Helper()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-unit-config-rating")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	invoiceAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T00:00:00Z", time.UTC).AsTime()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()
	defer s.UsageBasedTestHandler.Reset()

	// Cap credit-only accrual at 0 so the full amount is invoiced (no credits).
	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)

	// Meter is in raw units, bill in thousands: divide by 1000, round up.
	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:          cust.GetID(),
				currency:          USD,
				servicePeriod:     servicePeriod,
				settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)}),
				unitConfig:        unitConfig,
				name:              "usage-based-unit-config",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based-unit-config",
				featureKey:        meterSlug,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(res, 1)

	// 7400 raw units. Flag on: ceil(7400/1000) = 8 billed units.
	s.MockStreamingConnector.AddSimpleEvent(
		meterSlug,
		7400,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)
	clock.FreezeTime(invoiceAt)

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(invoiceAt),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

	stdLine := invoices[0].Lines.OrEmpty()[0]

	// The raw metered quantity is always 7400; unit_config only changes the priced
	// amount in this ticket. (The displayed UsageBased.Quantity staying in raw units
	// until the charges line-mapper also converts is a separate, later scope item.)
	s.Require().NotNil(stdLine.UsageBased.MeteredQuantity)
	s.Equal(float64(7400), lo.FromPtr(stdLine.UsageBased.MeteredQuantity).InexactFloat64())

	s.RequireTotals(billingtest.ExpectedTotals{
		Amount: expectedAmount,
		Total:  expectedAmount,
	}, stdLine.Totals)
}
