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

	createInvoice := func(subjectKey string) billing.StandardInvoice {
		cust := s.CreateTestCustomer(ns, subjectKey)

		return s.CreateDraftInvoice(s.T(), ctx, DraftInvoiceInput{Namespace: ns, Customer: cust})
	}

	// given: two invoices written before a sync checkpoint.
	createInvoice("unchanged")
	changed := createInvoice("changed")

	// when: one of them is updated at the checkpoint.
	clock.FreezeTime(checkpoint)
	defer clock.UnFreeze()

	_, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      changed.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.StandardInvoice) error {
			invoice.Description = lo.ToPtr("changed")

			return nil
		},
	})
	s.Require().NoError(err)

	// then: only the updated invoice matches. Its created_at is before the
	// checkpoint, so this can only match on updated_at. The v3 route lists only
	// standard invoices, so list the way it does.
	result, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Page:         pagination.NewPage(1, 10),
		Namespace:    ns,
		OnlyStandard: true,
		UpdatedAt:    &filter.FilterTime{Gte: &checkpoint},
	})
	s.Require().NoError(err)
	s.Equal([]string{changed.ID}, lo.Map(result.Items, func(inv billing.Invoice, _ int) string {
		generic, err := inv.AsGenericInvoice()
		s.Require().NoError(err)

		return generic.GetID()
	}))
}
