package reconciler

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func shouldSkipLinePatch(existingLine billing.GenericInvoiceLine, expectedLine billing.GatheringLine) bool {
	if expectedLine.Annotations.GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return true
	}

	if existingLine.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return true
	}

	if existingLine.GetManagedBy() != billing.SubscriptionManagedLine {
		return true
	}

	return false
}

func shouldSkipHierarchyPatch(existingHierarchy *billing.SplitLineHierarchy, expectedLine billing.GatheringLine) bool {
	if expectedLine.Annotations.GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return true
	}

	for _, line := range existingHierarchy.Lines {
		if line.Line.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
			return true
		}
	}

	return false
}

func getPatchesForUpdateUsageBasedLine(existingLine billing.GenericInvoiceLine, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) (*invoiceupdater.Patch, error) {
	if shouldSkipLinePatch(existingLine, expectedLine) {
		return nil, nil
	}

	if invoiceupdater.IsFlatFee(existingLine) {
		return nil, fmt.Errorf("usage based patch cannot be applied to flat fee line[%s]", existingLine.GetLineID().ID)
	}

	targetLine, err := existingLine.CloneWithoutChildren()
	if err != nil {
		return nil, fmt.Errorf("cloning line: %w", err)
	}

	wasChange := false
	if !targetLine.GetServicePeriod().Equal(expectedLine.ServicePeriod) {
		wasChange = true

		targetLine.UpdateServicePeriod(func(p *timeutil.ClosedPeriod) {
			*p = expectedLine.ServicePeriod
		})

		isGatheringInvoice, err := invoices.IsGatheringInvoice(targetLine.GetInvoiceID())
		if err != nil {
			return nil, fmt.Errorf("getting invoice type for line[%s]: %w", targetLine.GetLineID().ID, err)
		}

		if isGatheringInvoice {
			invoiceAtAccessor, ok := targetLine.(billing.InvoiceAtAccessor)
			if !ok {
				return nil, fmt.Errorf("target line is not an invoice at accessor: %T", targetLine)
			}

			invoiceAtAccessor.SetInvoiceAt(expectedLine.InvoiceAt)
		}
	}

	if !invoiceupdater.IsFlatFee(targetLine) {
		if targetLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
			return lo.ToPtr(invoiceupdater.NewDeleteLinePatch(existingLine.GetLineID(), existingLine.GetInvoiceID())), nil
		}
	}

	if targetLine.GetDeletedAt() != nil {
		targetLine.SetDeletedAt(nil)
		wasChange = true
	}

	if !wasChange {
		return nil, nil
	}

	return lo.ToPtr(invoiceupdater.NewUpdateLinePatch(targetLine)), nil
}
