package invoiceupdater

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestUpdateImmutableInvoiceChecksOnlyPresentFields(t *testing.T) {
	t.Parallel()

	t.Run("flat fee amount takes precedence over service period drift", func(t *testing.T) {
		t.Parallel()

		line := newUpdateLinePatchTestStandardLine()
		line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(10),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})
		invoice := newUpdateLinePatchTestImmutableInvoice(line)
		service := &immutableInvoiceBillingServiceStub{invoice: invoice}
		updater := &Updater{billingService: service}
		updatedPeriod := line.Period
		updatedPeriod.To = updatedPeriod.To.AddDate(0, 0, 1)
		patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
			Line:                 line.GetLineID(),
			InvoiceID:            line.InvoiceID,
			ServicePeriod:        mo.Some(updatedPeriod),
			FlatFeePerUnitAmount: mo.Some(alpacadecimal.NewFromInt(25)),
		})
		require.NoError(t, err)
		update, err := patch.AsUpdateLinePatch()
		require.NoError(t, err)

		err = updater.updateImmutableInvoice(t.Context(), invoice, invoicePatches{
			updatedLines: []PatchLineUpdate{update},
		})
		require.NoError(t, err)
		require.Len(t, service.upsertedIssues, 1)
		require.Contains(t, service.upsertedIssues[0].Message, "new per unit amount: 25")
		require.NotContains(t, service.upsertedIssues[0].Message, "service period")
	})

	t.Run("deleted at clear preserves existing immutable policy", func(t *testing.T) {
		t.Parallel()

		line := newUpdateLinePatchTestStandardLine()
		line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(10),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})
		deletedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		line.DeletedAt = &deletedAt
		invoice := newUpdateLinePatchTestImmutableInvoice(line)
		service := &immutableInvoiceBillingServiceStub{invoice: invoice}
		updater := &Updater{billingService: service}
		patch, err := NewUpdateLinePatch(NewUpdateLinePatchInput{
			Line:      line.GetLineID(),
			InvoiceID: line.InvoiceID,
			DeletedAt: mo.Some[*time.Time](nil),
		})
		require.NoError(t, err)
		update, err := patch.AsUpdateLinePatch()
		require.NoError(t, err)

		err = updater.updateImmutableInvoice(t.Context(), invoice, invoicePatches{
			updatedLines: []PatchLineUpdate{update},
		})
		require.NoError(t, err)
		require.Empty(t, service.upsertedIssues)
	})
}

type immutableInvoiceBillingServiceStub struct {
	billing.Service

	invoice        billing.StandardInvoice
	upsertedIssues billing.ValidationIssues
}

func (s *immutableInvoiceBillingServiceStub) GetStandardInvoiceById(_ context.Context, _ billing.GetStandardInvoiceByIdInput) (billing.StandardInvoice, error) {
	return s.invoice, nil
}

func (s *immutableInvoiceBillingServiceStub) UpsertValidationIssues(_ context.Context, input billing.UpsertValidationIssuesInput) error {
	s.upsertedIssues = input.Issues

	return nil
}

func newUpdateLinePatchTestImmutableInvoice(line *billing.StandardLine) billing.StandardInvoice {
	return billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "invoice-1",
		},
		Lines: billing.NewStandardInvoiceLines([]*billing.StandardLine{line}),
	}
}
