package reconciler

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ShrinkUsageBasedPatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods
}

func (p ShrinkUsageBasedPatch) Operation() PatchOperation {
	return PatchOperationShrink
}

func (p ShrinkUsageBasedPatch) UniqueReferenceID() string {
	return p.UniqueID
}

func (p ShrinkUsageBasedPatch) GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error) {
	expectedLine, err := p.Target.GetExpectedLineOrErr(input.Subscription, input.Currency)
	if err != nil {
		return nil, err
	}

	switch p.Existing.Type() {
	case billing.LineOrHierarchyTypeLine:
		existingLine, err := p.Existing.AsGenericLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		return p.getInvoicePatchesForLine(existingLine, expectedLine, input.Invoices)
	case billing.LineOrHierarchyTypeHierarchy:
		existingHierarchy, err := p.Existing.AsHierarchy()
		if err != nil {
			return nil, fmt.Errorf("getting hierarchy: %w", err)
		}

		return p.getInvoicePatchesForHierarchy(existingHierarchy, expectedLine, input.Invoices)
	default:
		return nil, fmt.Errorf("unsupported line or hierarchy type: %s", p.Existing.Type())
	}
}

func (p ShrinkUsageBasedPatch) getInvoicePatchesForLine(existingLine billing.GenericInvoiceLine, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]invoiceupdater.Patch, error) {
	if shouldSkipLinePatch(existingLine, expectedLine) {
		return nil, nil
	}

	if existingLine.GetServicePeriod().To.Equal(expectedLine.ServicePeriod.To) {
		return nil, nil
	}

	if !expectedLine.ServicePeriod.To.Before(existingLine.GetServicePeriod().To) {
		return nil, fmt.Errorf("shrink patch requires target end before existing end: existing=%s..%s target=%s..%s", existingLine.GetServicePeriod().From, existingLine.GetServicePeriod().To, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	return getPatchesForUpdateUsageBasedLine(existingLine, expectedLine, invoices)
}

func (p ShrinkUsageBasedPatch) getInvoicePatchesForHierarchy(existingHierarchy *billing.SplitLineHierarchy, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]invoiceupdater.Patch, error) {
	if !expectedLine.ServicePeriod.To.Before(existingHierarchy.Group.ServicePeriod.End) {
		return nil, fmt.Errorf("shrink patch requires target end before existing hierarchy end: existing=%s..%s target=%s..%s", existingHierarchy.Group.ServicePeriod.Start, existingHierarchy.Group.ServicePeriod.End, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	patches := make([]invoiceupdater.Patch, 0, len(existingHierarchy.Lines)+1)

	for _, child := range existingHierarchy.Lines {
		if child.Line.GetServicePeriod().To.Before(expectedLine.ServicePeriod.To) {
			continue
		}

		if child.Line.GetServicePeriod().From.After(expectedLine.ServicePeriod.To) {
			patches = append(patches, invoiceupdater.NewDeleteLinePatch(child.Line.GetLineID(), child.Line.GetInvoiceID()))
			continue
		}

		if !child.Line.GetServicePeriod().To.Equal(expectedLine.ServicePeriod.To) {
			updatedLine, err := child.Line.CloneWithoutChildren()
			if err != nil {
				return nil, fmt.Errorf("cloning child: %w", err)
			}

			updatedLine.UpdateServicePeriod(func(p *timeutil.ClosedPeriod) {
				p.To = expectedLine.ServicePeriod.To
			})

			if invoices.IsGatheringInvoice(updatedLine.GetInvoiceID()) {
				invoiceAtAccessor, ok := updatedLine.(billing.InvoiceAtAccessor)
				if !ok {
					return nil, fmt.Errorf("last child is not an invoice at accessor: %T", updatedLine)
				}
				invoiceAtAccessor.SetInvoiceAt(expectedLine.InvoiceAt)
			}

			if updatedLine.GetManagedBy() == billing.SubscriptionManagedLine {
				updatedLine.SetDeletedAt(nil)
			}

			if !invoiceupdater.IsFlatFee(updatedLine) {
				if updatedLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
					patches = append(patches, invoiceupdater.NewDeleteLinePatch(updatedLine.GetLineID(), updatedLine.GetInvoiceID()))
					continue
				}
			}

			patches = append(patches, invoiceupdater.NewUpdateLinePatch(updatedLine))
		}
	}

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.End = expectedLine.ServicePeriod.To
	patches = append(patches, invoiceupdater.NewUpdateSplitLineGroupPatch(updatedGroup))

	return patches, nil
}
