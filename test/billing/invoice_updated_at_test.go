package billing

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s *InvoicingTestSuite) TestListInvoicesUpdatedAt() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("list-invoices-updated-at")
	checkpoint := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	clock.FreezeTime(checkpoint.Add(-time.Hour))
	defer clock.UnFreeze()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	ids := func(result billing.ListInvoicesResponse) []string {
		return lo.Map(result.Items, func(inv billing.Invoice, _ int) string {
			generic, err := inv.AsGenericInvoice()
			s.Require().NoError(err)

			return generic.GetID()
		})
	}

	createInvoice := func(subjectKey string) string {
		cust := s.CreateTestCustomer(ns, subjectKey)

		return s.CreateDraftInvoice(s.T(), ctx, DraftInvoiceInput{Namespace: ns, Customer: cust}).ID
	}

	// The v3 route lists only standard invoices, so list the way it does.
	list := func(updatedAt filter.FilterTime) billing.ListInvoicesResponse {
		result, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Page:         pagination.NewPage(1, 1),
			Namespace:    ns,
			OnlyStandard: true,
			UpdatedAt:    &updatedAt,
		})
		s.Require().NoError(err)

		return result
	}

	// given: an invoice written before a sync checkpoint and one written at it.
	before := createInvoice("before-checkpoint")

	clock.FreezeTime(checkpoint)
	defer clock.UnFreeze()
	atCheckpoint := createInvoice("at-checkpoint")

	// when/then: gte includes the boundary, gt excludes it, and lt selects the
	// earlier invoice; page totals count only the filtered rows.
	gte := list(filter.FilterTime{Gte: &checkpoint})
	s.Equal([]string{atCheckpoint}, ids(gte))
	s.Equal(1, gte.TotalCount)

	s.Empty(list(filter.FilterTime{Gt: &checkpoint}).Items)

	lt := list(filter.FilterTime{Lt: &checkpoint})
	s.Equal([]string{before}, ids(lt))
	s.Equal(1, lt.TotalCount)
}
