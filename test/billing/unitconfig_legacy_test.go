package billing

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// This suite is the legacy/standard-line mirror of the charges-path
// TestRatesConvertedQuantity (openmeter/billing/charges/service/unitconfig_rating_test.go).
// It drives a plain (non-charge) usage-based gathering line carrying a unit_config through
// CreatePendingInvoiceLines -> InvoicePendingLines, exercising the legacy
// lineengine.BuildStandardInvoiceLines path plus billing rating (StandardLine.GetUnitConfig
// -> the UnitConfig mutator), proving the legacy path converts and snapshots identically to
// the charges path. The flag-off guard on this path is covered at the rating layer
// (rating/service/rate/unitconfig_rating_test.go, forbidunitconfig_test.go).

func TestLegacyUnitConfigRating(t *testing.T) {
	suite.Run(t, new(legacyUnitConfigRatingSuite))
}

type legacyUnitConfigRatingSuite struct {
	BaseSuite
}

func (s *legacyUnitConfigRatingSuite) TestRatesConvertedQuantity() {
	// 7400 raw / 1000, ceiling => 8 billed units * $1 = $8.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("billing-legacy-unit-config")

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)

	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(),
		WithProgressiveBilling(),
		WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")),
		To:   lo.Must(time.Parse(time.RFC3339, "2026-01-16T00:00:00Z")),
	}
	invoiceAt := servicePeriod.To

	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []billing.GatheringLine{
			{
				GatheringLineBase: billing.GatheringLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Name: "legacy-usage-based-unit-config",
					}),
					ServicePeriod: servicePeriod,
					InvoiceAt:     invoiceAt,
					ManagedBy:     billing.ManuallyManagedLine,
					FeatureKey:    feature.Feature.Key,
					Price: lo.FromPtr(productcatalog.NewPriceFrom(
						productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(1)},
					)),
					UnitConfig: unitConfig,
				},
			},
		},
	})
	s.Require().NoError(err)

	// 7400 raw units within the service period. ceil(7400/1000) = 8 billed units.
	s.MockStreamingConnector.AddSimpleEvent(
		feature.Feature.Key,
		7400,
		lo.Must(time.Parse(time.RFC3339, "2026-01-15T00:00:00Z")),
	)

	// Collect after the line's invoice-at so the completed-period line is billable.
	clock.FreezeTime(invoiceAt.Add(time.Hour))

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

	stdLine := invoices[0].Lines.OrEmpty()[0]

	// MeteredQuantity is the raw audit value (7400); the customer-facing billable Quantity
	// is the converted ceil(7400/1000) = 8, matching the priced amount. This is the exact
	// assertion the charges path makes, proving the legacy path converts identically.
	s.Require().NotNil(stdLine.UsageBased.MeteredQuantity)
	s.Equal(float64(7400), lo.FromPtr(stdLine.UsageBased.MeteredQuantity).InexactFloat64())

	s.Require().NotNil(stdLine.UsageBased.Quantity)
	s.Equal(float64(8), lo.FromPtr(stdLine.UsageBased.Quantity).InexactFloat64())

	// The config that produced the conversion is snapshotted onto the line at billing time
	// and exposed via GetUnitConfig, completing the audit trail on the legacy path.
	s.Require().NotNil(stdLine.UsageBased.UnitConfig)
	s.True(unitConfig.Equal(stdLine.UsageBased.UnitConfig),
		"applied unit_config snapshot must match the config used at rating time")
	s.Require().NotNil(stdLine.GetUnitConfig())
	s.True(unitConfig.Equal(stdLine.GetUnitConfig()))

	s.Equal(float64(8), stdLine.Totals.Amount.InexactFloat64())
	s.Equal(float64(8), stdLine.Totals.Total.InexactFloat64())
}
