package service

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/filter"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type assertGatheringPreviewInput struct {
	Namespace  string
	CustomerID string

	ExpectedInvoiceTotals billingtest.ExpectedTotals
	ExpectedLines         int
	ExpectedLineTotals    billingtest.ExpectedTotals
	ExpectedDetailedLines int

	AssertLine func(*billing.StandardLine)
}

func (s *BaseSuite) assertGatheringPreview(input assertGatheringPreviewInput) billing.StandardInvoice {
	s.T().Helper()

	invoices, err := s.BillingService.ListInvoices(s.T().Context(), billing.ListInvoicesInput{
		Namespaces:       []string{input.Namespace},
		CustomerID:       &filter.FilterULID{FilterString: filter.FilterString{Eq: &input.CustomerID}},
		ExtendedStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusGathering},
		Expand: billing.InvoiceExpands{}.
			With(billing.InvoiceExpandLines).
			With(billing.InvoiceExpandCalculateGatheringInvoiceWithLiveData),
	})
	s.NoError(err)
	s.Require().Len(invoices.Items, 1)

	previewInvoice, err := invoices.Items[0].AsStandardInvoice()
	s.NoError(err)
	s.RequireTotals(input.ExpectedInvoiceTotals, previewInvoice.Totals)

	s.Require().True(previewInvoice.Lines.IsPresent())
	s.Require().Len(previewInvoice.Lines.OrEmpty(), input.ExpectedLines)
	if input.ExpectedLines == 0 {
		return previewInvoice
	}

	previewLine := previewInvoice.Lines.OrEmpty()[0]
	s.Len(previewLine.DetailedLines, input.ExpectedDetailedLines)
	s.RequireTotals(input.ExpectedLineTotals, previewLine.Totals)

	if input.AssertLine != nil {
		input.AssertLine(previewLine)
	}

	return previewInvoice
}
