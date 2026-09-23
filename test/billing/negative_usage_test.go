package billing

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestNegativeUsage(t *testing.T) {
	suite.Run(t, new(negativeUsageSuite))
}

type negativeUsageSuite struct {
	BaseSuite
}

type negativeUsageFixture struct {
	customer      customer.CustomerID
	featureKey    string
	servicePeriod timeutil.ClosedPeriod
}

func (s *negativeUsageSuite) TestLegacyFinalCollectionClampsNegativeUsage() {
	ctx := s.T().Context()
	fixture := s.setupLegacyUsageLine()

	// Given negative SUM usage over the service period.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -2, fixture.servicePeriod.From.Add(time.Hour))
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -3, fixture.servicePeriod.From.Add(2*time.Hour))

	// When the completed line is collected into a standard invoice.
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

	// Then the negative usage is retained for audit, but pricing and totals use zero.
	line := invoices[0].Lines.OrEmpty()[0]
	s.requireUsageQuantities(line, -5, 0, 0, 0)
	s.Zero(line.Totals.Amount.InexactFloat64())
	s.Zero(line.Totals.Total.InexactFloat64())
	s.requireNegativeUsageWarning(invoices[0], billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
}

func (s *negativeUsageSuite) TestLegacyProgressiveBillingClampsNegativeUsageAndPrePeriodUsage() {
	ctx := s.T().Context()
	fixture := s.setupLegacyUsageLine(WithProgressiveBilling())
	midPeriod := fixture.servicePeriod.From.Add(24 * time.Hour)

	// Given negative SUM usage before the progressive billing boundary.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -2, fixture.servicePeriod.From.Add(time.Hour))
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -3, fixture.servicePeriod.From.Add(2*time.Hour))

	// When the line is progressively billed.
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
		AsOf:     lo.ToPtr(midPeriod),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

	// Then the partial line is rated at zero and carries the warning.
	line := invoices[0].Lines.OrEmpty()[0]
	s.requireUsageQuantities(line, -5, 0, 0, 0)
	s.Zero(line.Totals.Amount.InexactFloat64())
	s.Zero(line.Totals.Total.InexactFloat64())
	s.requireNegativeUsageWarning(invoices[0], billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")

	// When later usage makes the final interval positive.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, 8, midPeriod.Add(time.Hour))
	invoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
		AsOf:     lo.ToPtr(fixture.servicePeriod.To),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.Require().Len(invoices[0].Lines.OrEmpty(), 1)

	// Then the final interval bills its positive usage while the negative pre-period usage is clamped.
	line = invoices[0].Lines.OrEmpty()[0]
	s.requireUsageQuantities(line, 8, 8, -5, 0)
	s.Equal(float64(8), line.Totals.Amount.InexactFloat64())
	s.Equal(float64(8), line.Totals.Total.InexactFloat64())
	s.requireNegativeUsageWarning(invoices[0], billing.WarnNegativePreLinePeriodMeteredQuantityClamped.Code, "8", "-5")
}

func (s *negativeUsageSuite) setupLegacyUsageLine(profileOptions ...BillingProfileProvisionOption) negativeUsageFixture {
	s.T().Helper()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("billing-negative-usage")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	s.T().Cleanup(feature.Cleanup)

	profileOptions = append(profileOptions, WithManualApproval())
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), profileOptions...)

	servicePeriod := timeutil.ClosedPeriod{
		From: lo.Must(time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")),
		To:   lo.Must(time.Parse(time.RFC3339, "2026-01-03T00:00:00Z")),
	}

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{{
			GatheringLineBase: billing.GatheringLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name: "negative usage",
				}),
				ServicePeriod: servicePeriod,
				InvoiceAt:     servicePeriod.To,
				ManagedBy:     billing.ManuallyManagedLine,
				FeatureKey:    feature.Feature.Key,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				})),
			},
		}}),
	})
	s.Require().NoError(err)

	return negativeUsageFixture{
		customer:      cust.GetID(),
		featureKey:    feature.Feature.Key,
		servicePeriod: servicePeriod,
	}
}

func (s *negativeUsageSuite) requireUsageQuantities(line *billing.StandardLine, metered, rated, meteredPrePeriod, ratedPrePeriod float64) {
	s.T().Helper()
	s.Require().NotNil(line.UsageBased)
	s.Require().NotNil(line.UsageBased.MeteredQuantity)
	s.Require().NotNil(line.UsageBased.Quantity)
	s.Require().NotNil(line.UsageBased.MeteredPreLinePeriodQuantity)
	s.Require().NotNil(line.UsageBased.PreLinePeriodQuantity)
	s.Equal(metered, line.UsageBased.MeteredQuantity.InexactFloat64())
	s.Equal(rated, line.UsageBased.Quantity.InexactFloat64())
	s.Equal(meteredPrePeriod, line.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())
	s.Equal(ratedPrePeriod, line.UsageBased.PreLinePeriodQuantity.InexactFloat64())
}

func (s *negativeUsageSuite) requireNegativeUsageWarning(invoice billing.StandardInvoice, code, originalQuantity, originalPrePeriodQuantity string) {
	s.T().Helper()

	issue, found := lo.Find(invoice.ValidationIssues, func(issue billing.ValidationIssue) bool {
		return issue.Code == code
	})
	s.Require().True(found, "expected validation warning %q", code)
	s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
	s.Equal(originalQuantity, issue.Attributes["original_metered_quantity"])
	s.Equal(originalPrePeriodQuantity, issue.Attributes["original_pre_line_metered_quantity"])
}
