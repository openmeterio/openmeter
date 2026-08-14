package reconciler

import (
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

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

	lines := group.Lines()

	patches := make([]invoiceupdater.Patch, 0, 1+len(lines))

	for _, line := range lines {
		if line.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
			return nil
		}
	}

	if group.Group.DeletedAt == nil {
		patches = append(patches, invoiceupdater.NewDeleteSplitLineGroupPatch(group.Group.NamespacedID))
	}

	for _, line := range lines {
		if line.GetDeletedAt() != nil {
			continue
		}

		patches = append(patches, invoiceupdater.NewDeleteLinePatch(line.GetID(), line.GetInvoiceID().ID))
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

	if !expectedLine.ServicePeriod.To.Before(existingHierarchy.Group.ServicePeriod.To) {
		return fmt.Errorf("shrink patch requires target end before existing hierarchy end: existing=%s..%s target=%s..%s", existingHierarchy.Group.ServicePeriod.From, existingHierarchy.Group.ServicePeriod.To, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	lines := existingHierarchy.Lines()

	patches := make([]invoiceupdater.Patch, 0, len(lines)+1)

	for _, child := range lines {
		if child.Line.GetServicePeriod().To.Before(expectedLine.ServicePeriod.To) {
			continue
		}

		if child.Line.GetServicePeriod().From.After(expectedLine.ServicePeriod.To) {
			patches = append(patches, invoiceupdater.NewDeleteLinePatch(child.Line.GetLineID(), child.Line.GetInvoiceID()))
			continue
		}

		if !child.Line.GetServicePeriod().To.Equal(expectedLine.ServicePeriod.To) {
			updatedPeriod := child.Line.GetServicePeriod()
			updatedPeriod.To = expectedLine.ServicePeriod.To
			updateInput := invoiceupdater.NewUpdateLinePatchInput{
				Line:          child.Line.GetLineID(),
				InvoiceID:     child.Line.GetInvoiceID(),
				ServicePeriod: mo.Some(updatedPeriod),
			}

			if child.Invoice.AsInvoice().Type() == billing.InvoiceTypeGathering {
				updateInput.InvoiceAt = mo.Some(expectedLine.InvoiceAt)
			}

			if child.Line.GetManagedBy() == billing.SubscriptionManagedLine && child.Line.GetDeletedAt() != nil {
				updateInput.DeletedAt = mo.Some[*time.Time](nil)
			}

			if updatedPeriod.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
				patches = append(patches, invoiceupdater.NewDeleteLinePatch(child.Line.GetLineID(), child.Line.GetInvoiceID()))
				continue
			}

			patch, err := invoiceupdater.NewUpdateLinePatch(updateInput)
			if err != nil {
				return fmt.Errorf("creating update line patch: %w", err)
			}

			patches = append(patches, patch)
		}
	}

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.To = expectedLine.ServicePeriod.To
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

	if existingHierarchy.Group.ServicePeriod.To.Equal(expectedLine.ServicePeriod.To) {
		return nil
	}

	if !expectedLine.ServicePeriod.To.After(existingHierarchy.Group.ServicePeriod.To) {
		return fmt.Errorf("[line] extend patch requires target end after existing end: existing=%s..%s target=%s..%s", existingHierarchy.Group.ServicePeriod.From, existingHierarchy.Group.ServicePeriod.To, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	patches := make([]invoiceupdater.Patch, 0, 2)

	if len(existingHierarchy.Lines) > 0 {
		lines := slices.Clone(existingHierarchy.Lines)
		slices.SortFunc(lines, func(i, j billing.LineWithInvoiceHeader) int {
			return timeutil.Compare(i.Line.GetServicePeriod().To, j.Line.GetServicePeriod().To)
		})

		lastChild := lines[len(lines)-1]
		updatedPeriod := lastChild.Line.GetServicePeriod()
		updatedPeriod.To = expectedLine.ServicePeriod.To
		updateInput := invoiceupdater.NewUpdateLinePatchInput{
			Line:          lastChild.Line.GetLineID(),
			InvoiceID:     lastChild.Line.GetInvoiceID(),
			ServicePeriod: mo.Some(updatedPeriod),
		}

		if lastChild.Line.GetManagedBy() == billing.SubscriptionManagedLine && lastChild.Line.GetDeletedAt() != nil {
			updateInput.DeletedAt = mo.Some[*time.Time](nil)
		}

		if lastChild.Invoice.AsInvoice().Type() == billing.InvoiceTypeGathering {
			updateInput.InvoiceAt = mo.Some(expectedLine.InvoiceAt)
		}

		patch, err := invoiceupdater.NewUpdateLinePatch(updateInput)
		if err != nil {
			return fmt.Errorf("creating update line patch: %w", err)
		}

		patches = append(patches, patch)
	}

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.To = expectedLine.ServicePeriod.To
	patches = append(patches, invoiceupdater.NewUpdateSplitLineGroupPatch(updatedGroup))

	return c.addPatches(target.UniqueID, PatchOperationExtend, patches...)
}

func (c *lineHierarchyPatchCollection) AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error {
	return fmt.Errorf("unsupported operation %s for line hierarchy invoice patches", PatchOperationProrate)
}
