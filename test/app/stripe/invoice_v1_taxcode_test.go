package appstripe

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// remapStripeTaxCode points the Tax Code's Stripe mapping at another code, the way a legacy
// mapping change (made before mappings became immutable) would.
func (s *StripeInvoiceTestSuite) remapStripeTaxCode(ctx context.Context, tc taxcode.TaxCode, stripeCode string) {
	s.T().Helper()

	_, err := s.TaxCodeEnv.Service.UpdateTaxCode(ctx, taxcode.UpdateTaxCodeInput{
		NamespacedID: tc.NamespacedID,
		Name:         tc.Name,
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: stripeCode},
		},
		// The test drives the update directly, not through the API, so the immutability
		// guard doesn't apply.
		AllowAppMappingOverride: true,
	})
	s.Require().NoError(err)
}

// renderInvoiceLinesToV1 reads the invoice back and maps it to v1 like the invoice handlers do,
// returning its lines by name.
func (s *StripeInvoiceTestSuite) renderInvoiceLinesToV1(ctx context.Context, invoiceID billing.InvoiceID) map[string]apiv1models.InvoiceLine {
	s.T().Helper()

	reread, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoiceID,
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)

	taxCodes, err := billingconvert.ResolveInvoiceTaxCodes(ctx, s.TaxCodeResolver, billingconvert.ResolveInvoiceTaxCodesInput{
		Namespace:     reread.Namespace,
		Workflows:     []billing.WorkflowConfig{reread.Workflow.Config},
		StandardLines: reread.Lines.OrEmpty(),
	})
	s.Require().NoError(err)

	apiInvoice, err := billinghttpdriver.MapStandardInvoiceToAPI(reread, taxCodes)
	s.Require().NoError(err)
	s.Require().NotNil(apiInvoice.Lines)

	return lo.SliceToMap(*apiInvoice.Lines, func(line apiv1models.InvoiceLine) (string, apiv1models.InvoiceLine) {
		return line.Name, line
	})
}

// requireV1LineStripeCode requires the line's tax config, its rate card's and its detailed
// lines' to show the same Stripe code (empty meaning no stripe object at all) and to keep the
// Tax Code ID and the behavior.
func (s *StripeInvoiceTestSuite) requireV1LineStripeCode(line apiv1models.InvoiceLine, taxCodeID string, stripeCode string) {
	s.T().Helper()

	var wantStripe *apiv1models.StripeTaxConfig
	if stripeCode != "" {
		wantStripe = &apiv1models.StripeTaxConfig{Code: stripeCode}
	}

	want := &apiv1models.TaxConfig{
		Behavior:  lo.ToPtr(apiv1models.TaxBehaviorExclusive),
		Stripe:    wantStripe,
		TaxCodeId: lo.ToPtr(taxCodeID),
	}

	s.Equal(want, line.TaxConfig, "line tax config")
	s.Require().NotNil(line.RateCard)
	s.Equal(want, line.RateCard.TaxConfig, "rate card tax config")

	s.Require().NotNil(line.Children)
	s.Require().NotEmpty(*line.Children)

	for _, child := range *line.Children {
		s.Equal(want, child.TaxConfig, "detailed line tax config")
	}
}

// v1 shows the Stripe code a line was billed with on Stripe invoices, never the Tax Code's live
// mapping: a stamped line keeps its stamp when the mapping changes, and a line without a stamp
// shows none (the provider default), though its Tax Code maps to a code now. Backfilling the
// stamp of old lines is covered by the migration test.
func (s *StripeInvoiceTestSuite) TestV1ShowsTheBilledStripeCodeOnStripeInvoices() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("stripe-v1-billed-code")
	period := timeutil.ClosedPeriod{
		From: lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z")),
		To:   lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:14Z")),
	}

	clock.FreezeTime(period.To.Add(time.Minute))
	defer clock.UnFreeze()

	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)

	stripeApp, err := s.Fixture.setupApp(ctx, namespace)
	s.Require().NoError(err)

	s.ProvisionBillingProfile(ctx, namespace, stripeApp.GetID(), billingtest.WithManualApproval())

	// given: a draft invoice synced to Stripe with three lines: two on Tax Codes that map to
	// Stripe and one on a Tax Code without a Stripe mapping
	cust := s.CreateTestCustomer(namespace, "test")
	_, err = s.Fixture.setupAppCustomerData(ctx, stripeApp, cust)
	s.Require().NoError(err)

	s.StripeAppClient.
		On("GetCustomer", defaultStripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: defaultStripeCustomerID,
			DefaultPaymentMethod: &stripeclient.StripePaymentMethod{
				ID: "pm_taxcode_test",
				BillingAddress: &models.Address{
					Country: lo.ToPtr(models.CountryCode("US")),
				},
			},
		}, nil)
	defer s.StripeAppClient.Restore()

	stampedTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")
	unstampedTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_30000000")
	providerDefault := s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), namespace)

	gatheringLines := lo.MapToSlice(map[string]string{
		"Stamped fee":          stampedTaxCode.ID,
		"Unstamped fee":        unstampedTaxCode.ID,
		"Provider default fee": providerDefault.ID,
	}, func(name string, taxCodeID string) billing.GatheringLine {
		line := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
			Namespace:     namespace,
			Period:        period,
			InvoiceAt:     period.To,
			ManagedBy:     billing.ManuallyManagedLine,
			Name:          name,
			PerUnitAmount: alpacadecimal.NewFromInt(100),
			Currency:      currencyx.FiatCode(currency.USD),
			PaymentTerm:   productcatalog.InArrearsPaymentTerm,
		})
		line.TaxConfig = exclusiveTaxConfig(taxCodeID)

		return line
	})

	_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Customer:     cust.GetID(),
		Currency:     currencyx.FiatCode(currency.USD),
		Lines:        billing.NewCreatePendingInvoiceLines(gatheringLines),
	})
	s.Require().NoError(err)

	s.stubDraftSyncUpsert("in_v1_billed_code")

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)

	invoiceID := invoices[0].GetInvoiceID()
	linesByName := lo.SliceToMap(invoices[0].Lines.OrEmpty(), func(line *billing.StandardLine) (string, *billing.StandardLine) {
		return line.Name, line
	})
	s.Require().Len(linesByName, 3)

	// given: the second line has no stamp, like a line from before stamping that was not backfilled
	s.Require().NoError(s.DBClient.BillingInvoiceLine.
		UpdateOneID(linesByName["Unstamped fee"].ID).
		ClearInvoicingAppTaxCode().
		Exec(ctx))

	// when: every mapping changes, or appears, after the invoices were billed
	s.remapStripeTaxCode(ctx, stampedTaxCode, "txcd_20000000")
	s.remapStripeTaxCode(ctx, unstampedTaxCode, "txcd_40000000")
	s.remapStripeTaxCode(ctx, providerDefault, "txcd_60000000")

	apiLines := s.renderInvoiceLinesToV1(ctx, invoiceID)

	// then: the stamped line shows the code it was billed with
	s.requireV1LineStripeCode(apiLines["Stamped fee"], stampedTaxCode.ID, "txcd_10000000")

	// then: the line without a stamp shows none, though the Tax Code maps to one now
	s.requireV1LineStripeCode(apiLines["Unstamped fee"], unstampedTaxCode.ID, "")

	// then: the line stamped with the provider default shows none
	s.requireV1LineStripeCode(apiLines["Provider default fee"], providerDefault.ID, "")
}

// A gathering invoice has no billed code yet, so its v1 preview shows the Tax Code's current
// Stripe mapping, also on a Stripe profile.
func (s *StripeInvoiceTestSuite) TestV1GatheringPreviewShowsTheLiveStripeCode() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("stripe-v1-gathering-code")
	period := timeutil.ClosedPeriod{
		From: lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z")),
		To:   lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:14Z")),
	}

	clock.FreezeTime(period.To.Add(time.Minute))
	defer clock.UnFreeze()

	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)

	stripeApp, err := s.Fixture.setupApp(ctx, namespace)
	s.Require().NoError(err)

	s.ProvisionBillingProfile(ctx, namespace, stripeApp.GetID(), billingtest.WithManualApproval())

	// given: a gathering invoice on a Stripe profile with a line on a mapped Tax Code
	cust := s.CreateTestCustomer(namespace, "test")

	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	line := billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     namespace,
		Period:        period,
		InvoiceAt:     period.To,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          "Gathering fee",
		PerUnitAmount: alpacadecimal.NewFromInt(100),
		Currency:      currencyx.FiatCode(currency.USD),
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
	})
	line.TaxConfig = exclusiveTaxConfig(taxCode.ID)

	res, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Customer:     cust.GetID(),
		Currency:     currencyx.FiatCode(currency.USD),
		Lines:        billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{line}),
	})
	s.Require().NoError(err)

	// when: the preview is rendered like the v1 invoice handlers do
	invoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
		Invoice: res.Invoice.GetInvoiceID(),
		Expand: billing.InvoiceExpands{
			billing.InvoiceExpandLines,
			billing.InvoiceExpandCalculateGatheringInvoiceWithLiveData,
		},
	})
	s.Require().NoError(err)

	preview, err := invoice.AsStandardInvoice()
	s.Require().NoError(err)
	s.Require().Equal(billing.StandardInvoiceStatusGathering, preview.Status)
	s.Require().Equal(app.AppTypeStripe, preview.Workflow.InvoicingAppType())

	taxCodes, err := billingconvert.ResolveInvoiceTaxCodes(ctx, s.TaxCodeResolver, billingconvert.ResolveInvoiceTaxCodesInput{
		Namespace:     preview.Namespace,
		Workflows:     []billing.WorkflowConfig{preview.Workflow.Config},
		StandardLines: preview.Lines.OrEmpty(),
	})
	s.Require().NoError(err)

	apiInvoice, err := billinghttpdriver.MapStandardInvoiceToAPI(preview, taxCodes)
	s.Require().NoError(err)
	s.Require().NotNil(apiInvoice.Lines)
	s.Require().Len(*apiInvoice.Lines, 1)

	// then: the line, its rate card and its detailed lines show the live mapping
	s.requireV1LineStripeCode((*apiInvoice.Lines)[0], taxCode.ID, "txcd_10000000")
}

func exclusiveTaxConfig(taxCodeID string) *productcatalog.TaxCodeConfig {
	return &productcatalog.TaxCodeConfig{
		Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		TaxCodeID: taxCodeID,
	}
}
