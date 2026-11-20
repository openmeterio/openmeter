package httpdriver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *ProfileTaxConfigTestSuite) TestInvoiceListResolvesTaxCodesOnce() {
	ctx := s.T().Context()

	// given: a customer with a standard invoice and a gathering invoice, on a profile whose
	// default references a tax code
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), s.namespace, app.AppTypeStripe, "txcd_10000000")
	s.seedLegacyProfile(taxCode.ID)
	cust := s.CreateTestCustomer(s.namespace, "test")

	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Customer:     cust.GetID(),
		Currency:     currencyx.FiatCode(currency.USD),
		Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Namespace:     s.namespace,
				Period:        timeutil.ClosedPeriod{From: now, To: now.Add(24 * time.Hour)},
				InvoiceAt:     now,
				ManagedBy:     billing.ManuallyManagedLine,
				Name:          "due now",
				PerUnitAmount: alpacadecimal.NewFromFloat(100),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
			billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
				Namespace:     s.namespace,
				Period:        timeutil.ClosedPeriod{From: now.Add(24 * time.Hour), To: now.Add(48 * time.Hour)},
				InvoiceAt:     now.Add(24 * time.Hour),
				ManagedBy:     billing.ManuallyManagedLine,
				Name:          "due later",
				PerUnitAmount: alpacadecimal.NewFromFloat(100),
				PaymentTerm:   productcatalog.InAdvancePaymentTerm,
			}),
		}),
	})
	s.Require().NoError(err)

	_, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     &now,
	})
	s.Require().NoError(err)

	// A draft line with its own tax code, so the listing also has to resolve line references.
	lineTaxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), s.namespace, app.AppTypeStripe, "txcd_20000000")
	drafts, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespace:  s.namespace,
		CustomerID: &filter.FilterULID{FilterString: filter.FilterString{In: &[]string{cust.ID}}},
	})
	s.Require().NoError(err)
	var draft billing.StandardInvoice
	for _, item := range drafts.Items {
		if item.Type() != billing.InvoiceTypeStandard {
			continue
		}

		draft, err = item.AsStandardInvoice()
		s.Require().NoError(err)

		break
	}

	s.Require().NotNil(draft.ID, "expected a standard invoice")

	_, err = s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      draft.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceSystem,
		EditFn: func(invoice *billing.StandardInvoice) error {
			lines, err := invoice.Lines.MapWithErr(func(line *billing.StandardLine) (*billing.StandardLine, error) {
				line.TaxConfig = &productcatalog.TaxCodeConfig{TaxCodeID: lineTaxCode.ID}

				return line, nil
			})
			if err != nil {
				return err
			}

			invoice.Lines = lines

			return nil
		},
	})
	s.Require().NoError(err)

	// when: the invoices are listed
	s.resolver.BatchResolveCalls = 0

	recorder := httptest.NewRecorder()
	s.handler.ListInvoices().With(apiv1models.ListInvoicesParams{
		Expand: &[]apiv1models.InvoiceExpand{apiv1models.InvoiceExpandLines},
	}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	s.Require().Equal(http.StatusOK, recorder.Code, recorder.Body.String())

	// then: both invoices show the default's stripe code, the draft line shows its own tax
	// code's stripe code, all from one tax code lookup
	var resp apiv1models.InvoicePaginatedResponse
	s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
	s.Require().Len(resp.Items, 2)
	s.ElementsMatch([]apiv1models.InvoiceStatus{apiv1models.InvoiceStatusGathering, apiv1models.InvoiceStatusDraft}, lo.Map(resp.Items, func(item apiv1models.Invoice, _ int) apiv1models.InvoiceStatus {
		return item.Status
	}))

	for _, item := range resp.Items {
		s.Require().NotNil(item.Workflow.Workflow.Invoicing)
		s.Equal(&apiv1models.TaxConfig{
			Behavior:  lo.ToPtr(apiv1models.TaxBehaviorInclusive),
			Stripe:    &apiv1models.StripeTaxConfig{Code: "txcd_10000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}, item.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	}

	for _, item := range resp.Items {
		if item.Status != apiv1models.InvoiceStatusDraft {
			continue
		}

		s.Require().NotNil(item.Lines)
		s.Require().NotEmpty(*item.Lines)
		for _, line := range *item.Lines {
			s.Require().NotNil(line.TaxConfig)
			s.Equal(&apiv1models.TaxConfig{
				Behavior:  lo.ToPtr(apiv1models.TaxBehaviorInclusive),
				Stripe:    &apiv1models.StripeTaxConfig{Code: "txcd_20000000"},
				TaxCodeId: lo.ToPtr(lineTaxCode.ID),
			}, line.TaxConfig)
		}
	}

	s.Equal(1, s.resolver.BatchResolveCalls)
}
