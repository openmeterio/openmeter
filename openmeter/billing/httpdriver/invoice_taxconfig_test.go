package httpdriver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *ProfileTaxConfigTestSuite) TestInvoiceListResolvesTaxCodesOnce() {
	ctx := s.T().Context()

	// given: a customer with a standard invoice and a gathering invoice, on a profile whose
	// default references a tax code
	taxCode := s.TaxCodeForStripeCode(ctx, s.namespace, "txcd_10000000")
	s.seedLegacyProfile(taxCode.ID)
	cust := s.CreateTestCustomer(s.namespace, "test")

	now := time.Now().Truncate(time.Microsecond).UTC()

	_, err := s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: currencyx.FiatCode(currency.USD),
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

	// when: the invoices are listed
	s.resolver.BatchResolveCalls = 0

	recorder := httptest.NewRecorder()
	s.handler.ListInvoices().With(api.ListInvoicesParams{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	s.Require().Equal(http.StatusOK, recorder.Code, recorder.Body.String())

	// then: both invoices show the default's stripe code, from one tax code lookup
	var resp api.InvoicePaginatedResponse
	s.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &resp))
	s.Require().Len(resp.Items, 2)
	s.ElementsMatch([]api.InvoiceStatus{api.InvoiceStatusGathering, api.InvoiceStatusDraft}, lo.Map(resp.Items, func(item api.Invoice, _ int) api.InvoiceStatus {
		return item.Status
	}))

	for _, item := range resp.Items {
		s.Require().NotNil(item.Workflow.Workflow.Invoicing)
		s.Equal(&api.TaxConfig{
			Behavior:  lo.ToPtr(api.TaxBehaviorInclusive),
			Stripe:    &api.StripeTaxConfig{Code: "txcd_10000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}, item.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	}

	s.Equal(1, s.resolver.BatchResolveCalls)
}
