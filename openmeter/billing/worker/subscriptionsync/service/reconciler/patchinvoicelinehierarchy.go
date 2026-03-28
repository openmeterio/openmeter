package reconciler

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var (
	_ InvoicePatchCollection = (*lineHierarchyPatchCollection)(nil)
	_ PatchCollection        = (*lineHierarchyPatchCollection)(nil)
)

type lineHierarchyPatchCollection struct {
	invoicePatchCollectionBase
}

func newLineHierarchyPatchCollection(preallocatedCapacity int) *lineHierarchyPatchCollection {
	return &lineHierarchyPatchCollection{
		invoicePatchCollectionBase: newInvoicePatchCollectionBase(preallocatedCapacity),
	}
}

func (c *lineHierarchyPatchCollection) AddCreate(target targetstate.StateItem) error {
	return fmt.Errorf("unsupported operation %s for line hierarchy invoice patches", PatchOperationCreate)
}

func (c *lineHierarchyPatchCollection) AddDelete(uniqueID string, existing persistedstate.Item) error {
	group, err := persistedstate.ItemAsSplitLineHierarchy(existing)
	if err != nil {
		return err
	}
	patches := make([]invoiceupdater.Patch, 0, 1+len(group.Lines))

	for _, line := range group.Lines {
		if line.Line.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
			return nil
		}
	}

	if group.Group.DeletedAt == nil {
		patches = append(patches, invoiceupdater.NewDeleteSplitLineGroupPatch(group.Group.NamespacedID))
	}

	for _, line := range group.Lines {
		if line.Line.GetDeletedAt() != nil {
			continue
		}

		patches = append(patches, invoiceupdater.NewDeleteLinePatch(line.Line.GetLineID(), line.Invoice.GetID()))
	}

	if len(patches) == 0 {
		return nil
	}

	return c.addPatches(uniqueID, PatchOperationDelete, patches...)
}

func (c *lineHierarchyPatchCollection) AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error {
	existingHierarchy, err := persistedstate.ItemAsSplitLineHierarchy(existing)
	if err != nil {
		return err
	}
	expectedLine, err := target.GetExpectedLineOrErr()
	if err != nil {
		return err
	}

	if shouldSkipHierarchyPatch(existingHierarchy, expectedLine) {
		return nil
	}

	if !expectedLine.ServicePeriod.To.Before(existingHierarchy.Group.ServicePeriod.End) {
		return fmt.Errorf("shrink patch requires target end before existing hierarchy end: existing=%s..%s target=%s..%s", existingHierarchy.Group.ServicePeriod.Start, existingHierarchy.Group.ServicePeriod.End, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
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
				return fmt.Errorf("cloning child: %w", err)
			}

			updatedLine.UpdateServicePeriod(func(p *timeutil.ClosedPeriod) {
				p.To = expectedLine.ServicePeriod.To
			})

			if child.Invoice.AsInvoice().Type() == billing.InvoiceTypeGathering {
				invoiceAtAccessor, ok := updatedLine.(billing.InvoiceAtAccessor)
				if !ok {
					return fmt.Errorf("last child is not an invoice at accessor: %T", updatedLine)
				}
				invoiceAtAccessor.SetInvoiceAt(expectedLine.InvoiceAt)
			}

			if updatedLine.GetManagedBy() == billing.SubscriptionManagedLine {
				updatedLine.SetDeletedAt(nil)
			}

			if updatedLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
				patches = append(patches, invoiceupdater.NewDeleteLinePatch(child.Line.GetLineID(), child.Line.GetInvoiceID()))
				continue
			}

			patches = append(patches, invoiceupdater.NewUpdateLinePatch(updatedLine))
		}
	}

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.End = expectedLine.ServicePeriod.To
	patches = append(patches, invoiceupdater.NewUpdateSplitLineGroupPatch(updatedGroup))

	return c.addPatches(uniqueID, PatchOperationShrink, patches...)
}

func (c *lineHierarchyPatchCollection) AddExtend(existing persistedstate.Item, target targetstate.StateItem) error {
	existingHierarchy, err := persistedstate.ItemAsSplitLineHierarchy(existing)
	if err != nil {
		return err
	}
	expectedLine, err := target.GetExpectedLineOrErr()
	if err != nil {
		return err
	}

	if shouldSkipHierarchyPatch(existingHierarchy, expectedLine) {
		return nil
	}

	if existingHierarchy.Group.ServicePeriod.End.Equal(expectedLine.ServicePeriod.To) {
		return nil
	}

	if !expectedLine.ServicePeriod.To.After(existingHierarchy.Group.ServicePeriod.End) {
		return fmt.Errorf("[line] extend patch requires target end after existing end: existing=%s..%s target=%s..%s", existingHierarchy.Group.ServicePeriod.Start, existingHierarchy.Group.ServicePeriod.End, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	patches := make([]invoiceupdater.Patch, 0, 2)

	if len(existingHierarchy.Lines) > 0 {
		lines := existingHierarchy.Lines
		slices.SortFunc(lines, func(i, j billing.LineWithInvoiceHeader) int {
			return timeutil.Compare(i.Line.GetServicePeriod().To, j.Line.GetServicePeriod().To)
		})

		lastChild, err := lines[len(lines)-1].Line.CloneWithoutChildren()
		if err != nil {
			return fmt.Errorf("cloning last child: %w", err)
		}

		if lastChild.GetManagedBy() == billing.SubscriptionManagedLine {
			lastChild.SetDeletedAt(nil)
		}

		lastChild.UpdateServicePeriod(func(p *timeutil.ClosedPeriod) {
			p.To = expectedLine.ServicePeriod.To
		})

		if lines[len(lines)-1].Invoice.AsInvoice().Type() == billing.InvoiceTypeGathering {
			invoiceAtAccessor, ok := lastChild.(billing.InvoiceAtAccessor)
			if !ok {
				return fmt.Errorf("last child is not an invoice at accessor: %T", lastChild)
			}

			invoiceAtAccessor.SetInvoiceAt(expectedLine.InvoiceAt)
		}

		patches = append(patches, invoiceupdater.NewUpdateLinePatch(lastChild))
	}

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.End = expectedLine.ServicePeriod.To
	patches = append(patches, invoiceupdater.NewUpdateSplitLineGroupPatch(updatedGroup))

	return c.addPatches(target.UniqueID, PatchOperationExtend, patches...)
}

func (c *lineHierarchyPatchCollection) AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error {
	return fmt.Errorf("unsupported operation %s for line hierarchy invoice patches", PatchOperationProrate)
}
