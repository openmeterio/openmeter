package reconciler

import (
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/streaming"
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

	updateInput := invoiceupdater.NewUpdateLinePatchInput{
		Line:      existingLine.GetLineID(),
		InvoiceID: existingLine.GetInvoiceID(),
	}
	if !existingLine.GetServicePeriod().Equal(expectedLine.ServicePeriod) {
		updateInput.ServicePeriod = mo.Some(expectedLine.ServicePeriod)

		isGatheringInvoice, err := invoices.IsGatheringInvoice(existingLine.GetInvoiceID())
		if err != nil {
			return nil, fmt.Errorf("getting invoice type for line[%s]: %w", existingLine.GetLineID().ID, err)
		}

		if isGatheringInvoice {
			updateInput.InvoiceAt = mo.Some(expectedLine.InvoiceAt)
		}
	}

	if !invoiceupdater.IsFlatFee(existingLine) {
		targetPeriod := existingLine.GetServicePeriod()
		if period, ok := updateInput.ServicePeriod.Get(); ok {
			targetPeriod = period
		}

		if targetPeriod.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
			return lo.ToPtr(invoiceupdater.NewDeleteLinePatch(existingLine.GetLineID(), existingLine.GetInvoiceID())), nil
		}
	}

	if existingLine.GetDeletedAt() != nil {
		updateInput.DeletedAt = mo.Some[*time.Time](nil)
	}

	if updateInput.IsNoop() {
		return nil, nil
	}

	patch, err := invoiceupdater.NewUpdateLinePatch(updateInput)
	if err != nil {
		return nil, fmt.Errorf("creating update line patch: %w", err)
	}

	return lo.ToPtr(patch), nil
}
