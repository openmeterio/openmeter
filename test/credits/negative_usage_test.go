package credits

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestUsageBasedChargeNegativeUsage(t *testing.T) {
	suite.Run(t, new(usageBasedChargeNegativeUsageSuite))
}

type usageBasedChargeNegativeUsageSuite struct {
	BaseSuite
}

type usageBasedChargeNegativeUsageFixture struct {
	customer      customer.CustomerID
	chargeID      meta.ChargeID
	featureKey    string
	servicePeriod timeutil.ClosedPeriod
}

func (s *usageBasedChargeNegativeUsageSuite) TestGatheringPreviewClampsNegativeUsage() {
	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	clock.SetTime(setupAt)

	ctx := s.T().Context()
	fixture := s.setupUsageBasedCharge()

	// Given negative SUM usage on a charge-backed gathering line.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -5, fixture.servicePeriod.From.Add(time.Hour))
	clock.SetTime(fixture.servicePeriod.To.Add(time.Second))

	// When the gathering invoice is expanded as a live standard-invoice preview.
	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespace:        fixture.customer.Namespace,
		CustomerID:       &filter.FilterULID{FilterString: filter.FilterString{Eq: &fixture.customer.ID}},
		ExtendedStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusGathering},
		Expand: billing.InvoiceExpands{}.
			With(billing.InvoiceExpandLines).
			With(billing.InvoiceExpandCalculateGatheringInvoiceWithLiveData),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices.Items, 1)

	// Then the preview retains its zero-priced line and reports the rating warning.
	preview, err := invoices.Items[0].AsStandardInvoice()
	s.Require().NoError(err)
	s.Require().Len(preview.Lines.OrEmpty(), 1)
	s.Zero(preview.Lines.OrEmpty()[0].Totals.Total.InexactFloat64())
	s.requireNegativeUsageWarning(preview, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
	s.requireNoPersistedChargeRatingWarning(fixture.chargeID)
}

func (s *usageBasedChargeNegativeUsageSuite) TestFinalCollectionClampsNegativeUsage() {
	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	clock.SetTime(setupAt)

	ctx := s.T().Context()
	fixture := s.setupUsageBasedCharge()

	// Given negative SUM usage over the service period.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -2, fixture.servicePeriod.From.Add(time.Hour))
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -3, fixture.servicePeriod.From.Add(2*time.Hour))

	// When the completed charge-backed line is collected and rated.
	clock.SetTime(fixture.servicePeriod.To.Add(time.Second))
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
		AsOf:     lo.ToPtr(fixture.servicePeriod.To),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.requirePersistedChargeWarning(fixture.chargeID, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
	s.requireNegativeUsageWarning(invoices[0], billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")

	clock.SetTime(invoices[0].DefaultCollectionAtForStandardInvoice())
	invoice, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)

	// Then the negative usage is retained for audit, but pricing and totals use zero.
	line := invoice.Lines.OrEmpty()[0]
	s.requireNegativeUsageWarning(invoice, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
	s.requirePersistedChargeWarning(fixture.chargeID, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
	s.requireUsageQuantities(line, -5, 0, 0, 0)
	s.Zero(line.Totals.Amount.InexactFloat64())
	s.Zero(line.Totals.Total.InexactFloat64())
}

func (s *usageBasedChargeNegativeUsageSuite) TestProgressiveBillingReconcilesRecoveredUsage() {
	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	clock.SetTime(setupAt)

	ctx := s.T().Context()
	fixture := s.setupUsageBasedCharge(billingtest.WithProgressiveBilling())
	midPeriod := fixture.servicePeriod.From.Add(24 * time.Hour)

	// Given negative SUM usage before the progressive billing boundary.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -2, fixture.servicePeriod.From.Add(time.Hour))
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -3, fixture.servicePeriod.From.Add(2*time.Hour))

	// When the charge-backed line is progressively billed and rated.
	clock.SetTime(midPeriod)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
		AsOf:     lo.ToPtr(midPeriod),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.requirePersistedChargeWarning(fixture.chargeID, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")

	clock.SetTime(invoices[0].DefaultCollectionAtForStandardInvoice())
	invoice, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)

	// Then the partial line is rated at zero and carries the warning.
	line := invoice.Lines.OrEmpty()[0]
	s.requireNegativeUsageWarning(invoice, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
	s.requirePersistedChargeWarning(fixture.chargeID, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
	s.requireUsageQuantities(line, -5, 0, 0, 0)
	s.Zero(line.Totals.Amount.InexactFloat64())
	s.Zero(line.Totals.Total.InexactFloat64())

	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.Require().NoError(err)

	// When later usage makes the final interval positive.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, 8, midPeriod.Add(time.Hour))
	clock.SetTime(fixture.servicePeriod.To.Add(time.Second))
	invoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
		AsOf:     lo.ToPtr(fixture.servicePeriod.To),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	s.requireNoPersistedChargeRatingWarning(fixture.chargeID)

	clock.SetTime(invoices[0].DefaultCollectionAtForStandardInvoice())
	invoice, err = s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)

	// Then the final interval reconciles to the positive cumulative usage.
	line = invoice.Lines.OrEmpty()[0]
	s.requireUsageQuantities(line, 8, 3, -5, 0)
	s.Equal(float64(3), line.Totals.Amount.InexactFloat64())
	s.Equal(float64(3), line.Totals.Total.InexactFloat64())
	s.Empty(invoice.ValidationIssues)
	s.requireNoPersistedChargeRatingWarning(fixture.chargeID)
}

func (s *usageBasedChargeNegativeUsageSuite) TestProgressiveBillingKeepsLaterInvoicesClearAfterRecovery() {
	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	clock.SetTime(setupAt)

	ctx := s.T().Context()
	fixture := s.setupUsageBasedCharge(billingtest.WithProgressiveBilling())
	firstBoundary := fixture.servicePeriod.From.Add(16 * time.Hour)
	secondBoundary := fixture.servicePeriod.From.Add(32 * time.Hour)

	// Given a first partial run with negative usage.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, -5, fixture.servicePeriod.From.Add(time.Hour))
	clock.SetTime(firstBoundary)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
		AsOf:     lo.ToPtr(firstBoundary),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	clock.SetTime(invoices[0].DefaultCollectionAtForStandardInvoice())
	firstInvoice, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)
	s.requirePersistedChargeWarning(fixture.chargeID, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
	_, err = s.BillingService.ApproveInvoice(ctx, firstInvoice.GetInvoiceID())
	s.Require().NoError(err)

	// When the next run has positive cumulative usage and a negative raw pre-line quantity.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, 8, firstBoundary.Add(time.Hour))
	clock.SetTime(secondBoundary)
	invoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
		AsOf:     lo.ToPtr(secondBoundary),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	clock.SetTime(invoices[0].DefaultCollectionAtForStandardInvoice())
	secondInvoice, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)
	s.requireNoPersistedChargeRatingWarning(fixture.chargeID)
	s.Empty(secondInvoice.ValidationIssues)
	_, err = s.BillingService.ApproveInvoice(ctx, secondInvoice.GetInvoiceID())
	s.Require().NoError(err)

	// Then a later clean run does not restore the charge warning or change the first invoice.
	s.MockStreamingConnector.AddSimpleEvent(fixture.featureKey, 2, secondBoundary.Add(time.Hour))
	clock.SetTime(fixture.servicePeriod.To.Add(time.Second))
	invoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: fixture.customer,
		AsOf:     lo.ToPtr(fixture.servicePeriod.To),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)
	clock.SetTime(invoices[0].DefaultCollectionAtForStandardInvoice())
	finalInvoice, err := s.BillingService.AdvanceInvoice(ctx, invoices[0].GetInvoiceID())
	s.Require().NoError(err)
	s.Require().Len(finalInvoice.Lines.OrEmpty(), 1)
	s.Equal(float64(2), finalInvoice.Lines.OrEmpty()[0].Totals.Total.InexactFloat64())
	s.Empty(finalInvoice.ValidationIssues)

	s.requireNoPersistedChargeRatingWarning(fixture.chargeID)
	s.requireNegativeUsageWarning(firstInvoice, billing.WarnNegativeMeteredQuantityClamped.Code, "-5", "0")
	s.Empty(secondInvoice.ValidationIssues)
}

func (s *usageBasedChargeNegativeUsageSuite) setupUsageBasedCharge(profileOptions ...billingtest.BillingProfileProvisionOption) usageBasedChargeNegativeUsageFixture {
	s.T().Helper()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-negative-usage")
	s.ProvisionDefaultTaxCodes(ctx, ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	s.T().Cleanup(feature.Cleanup)

	profileOptions = append(profileOptions,
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID(), profileOptions...)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-01-03T00:00:00Z", time.UTC).AsTime(),
	}

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.NewCreateChargeIntents(
			s.CreateMockChargeIntent(CreateMockChargeIntentInput{
				Customer:       cust.GetID(),
				Currency:       USD,
				ServicePeriod:  servicePeriod,
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
				Name:              "negative usage",
				ManagedBy:         billing.ManuallyManagedLine,
				UniqueReferenceID: "negative-usage",
				FeatureKey:        feature.Feature.Key,
			}),
		),
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)
	chargeID, err := created[0].GetChargeID()
	s.Require().NoError(err)

	return usageBasedChargeNegativeUsageFixture{
		customer:      cust.GetID(),
		chargeID:      chargeID,
		featureKey:    feature.Feature.Key,
		servicePeriod: servicePeriod,
	}
}

func (s *usageBasedChargeNegativeUsageSuite) requireUsageQuantities(line *billing.StandardLine, metered, rated, meteredPrePeriod, ratedPrePeriod float64) {
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

func (s *usageBasedChargeNegativeUsageSuite) requireNegativeUsageWarning(invoice billing.StandardInvoice, code, originalQuantity, originalPrePeriodQuantity string) {
	s.T().Helper()

	issue, found := lo.Find(invoice.ValidationIssues, func(issue billing.ValidationIssue) bool {
		return issue.Code == code
	})
	s.Require().True(found, "expected validation warning %q in %#v", code, invoice.ValidationIssues)
	s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
	s.Equal(originalQuantity, issue.Attributes["original_metered_quantity"])
	s.Equal(originalPrePeriodQuantity, issue.Attributes["original_pre_line_metered_quantity"])
}

func (s *usageBasedChargeNegativeUsageSuite) requirePersistedChargeWarning(chargeID meta.ChargeID, code, originalQuantity, originalPrePeriodQuantity string) {
	s.T().Helper()

	charge, err := s.MustGetChargeByID(chargeID).AsUsageBasedCharge()
	s.Require().NoError(err)
	ratingIssues := lo.Filter(charge.ValidationIssues, func(issue billing.ValidationIssue, _ int) bool {
		return issue.Component == billing.ValidationComponentBillingRating
	})
	s.Require().Len(ratingIssues, 1)
	issue := ratingIssues[0]
	s.Equal(code, issue.Code)
	s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
	s.Equal(originalQuantity, issue.Attributes["original_metered_quantity"])
	s.Equal(originalPrePeriodQuantity, issue.Attributes["original_pre_line_metered_quantity"])
}

func (s *usageBasedChargeNegativeUsageSuite) requireNoPersistedChargeRatingWarning(chargeID meta.ChargeID) {
	s.T().Helper()

	charge, err := s.MustGetChargeByID(chargeID).AsUsageBasedCharge()
	s.Require().NoError(err)
	s.False(charge.ValidationIssues.HasComponent(billing.ValidationComponentBillingRating))
}
