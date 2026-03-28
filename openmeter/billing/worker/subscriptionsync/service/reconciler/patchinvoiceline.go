package reconciler

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var (
	_ InvoicePatchCollection = (*lineInvoicePatchCollection)(nil)
	_ PatchCollection        = (*lineInvoicePatchCollection)(nil)
)

type lineInvoicePatchCollection struct {
	invoicePatchCollectionBase

	invoices persistedstate.Invoices
}

func newLineInvoicePatchCollection(invoices persistedstate.Invoices, preallocatedCapacity int) (*lineInvoicePatchCollection, error) {
	if invoices == nil {
		return nil, fmt.Errorf("invoices is required")
	}

	return &lineInvoicePatchCollection{
		invoicePatchCollectionBase: newInvoicePatchCollectionBase(preallocatedCapacity),
		invoices:                   invoices,
	}, nil
}

func (c *lineInvoicePatchCollection) AddCreate(target targetstate.StateItem) error {
	line, err := target.GetExpectedLine()
	if err != nil {
		return fmt.Errorf("generating line from subscription item [%s]: %w", target.SubscriptionItem.ID, err)
	}

	if line == nil {
		return nil
	}

	return c.addPatches(target.UniqueID, PatchOperationCreate, invoiceupdater.NewCreateLinePatch(*line))
}

func (c *lineInvoicePatchCollection) AddDelete(uniqueID string, existing persistedstate.Item) error {
	existingLine, err := persistedstate.ItemAsLine(existing)
	if err != nil {
		return err
	}

	if existingLine.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return nil
	}

	if existingLine.GetDeletedAt() != nil {
		return nil
	}

	return c.addPatches(uniqueID, PatchOperationDelete, invoiceupdater.NewDeleteLinePatch(existingLine.GetLineID(), existingLine.GetInvoiceID()))
}

func (c *lineInvoicePatchCollection) AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error {
	existingLine, err := persistedstate.ItemAsLine(existing)
	if err != nil {
		return err
	}

	expectedLine, err := target.GetExpectedLineOrErr()
	if err != nil {
		return err
	}

	if shouldSkipLinePatch(existingLine, expectedLine) {
		return nil
	}

	if existingLine.GetServicePeriod().To.Equal(expectedLine.ServicePeriod.To) {
		return nil
	}

	if !expectedLine.ServicePeriod.To.Before(existingLine.GetServicePeriod().To) {
		return fmt.Errorf("shrink patch requires target end before existing end: existing=%s..%s target=%s..%s", existingLine.GetServicePeriod().From, existingLine.GetServicePeriod().To, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	patches, err := getPatchesForUpdateUsageBasedLine(existingLine, expectedLine, c.invoices)
	if err != nil {
		return err
	}

	if patches == nil {
		return nil
	}

	return c.addPatches(uniqueID, PatchOperationShrink, *patches)
}

func (c *lineInvoicePatchCollection) AddExtend(existing persistedstate.Item, target targetstate.StateItem) error {
	existingLine, err := persistedstate.ItemAsLine(existing)
	if err != nil {
		return err
	}

	expectedLine, err := target.GetExpectedLineOrErr()
	if err != nil {
		return err
	}

	if shouldSkipLinePatch(existingLine, expectedLine) {
		return nil
	}

	if existingLine.GetServicePeriod().To.Equal(expectedLine.ServicePeriod.To) {
		return nil
	}

	if !expectedLine.ServicePeriod.To.After(existingLine.GetServicePeriod().To) {
		return fmt.Errorf("[line] extend patch requires target end after existing end: existing=%s..%s target=%s..%s", existingLine.GetServicePeriod().From, existingLine.GetServicePeriod().To, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	patches, err := getPatchesForUpdateUsageBasedLine(existingLine, expectedLine, c.invoices)
	if err != nil {
		return err
	}

	if patches == nil {
		return nil
	}

	return c.addPatches(target.UniqueID, PatchOperationExtend, *patches)
}

func (c *lineInvoicePatchCollection) AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error {
	existingLine, err := persistedstate.ItemAsLine(existing)
	if err != nil {
		return err
	}

	expectedLine, err := target.GetExpectedLineOrErr()
	if err != nil {
		return err
	}

	if shouldSkipLinePatch(existingLine, expectedLine) {
		return nil
	}

	if !invoiceupdater.IsFlatFee(existingLine) {
		return fmt.Errorf("prorate patch cannot be applied to non-flat fee line[%s]", existingLine.GetLineID().ID)
	}

	if !invoiceupdater.IsFlatFee(expectedLine) {
		return fmt.Errorf("cannot merge flat fee line with usage based line")
	}

	targetLine, err := existingLine.CloneWithoutChildren()
	if err != nil {
		return fmt.Errorf("cloning line: %w", err)
	}

	wasChange := false
	if !targetLine.GetServicePeriod().Equal(expectedLine.ServicePeriod) {
		wasChange = true

		targetLine.UpdateServicePeriod(func(period *timeutil.ClosedPeriod) {
			*period = expectedLine.ServicePeriod
		})

		isGatheringInvoice, err := c.invoices.IsGatheringInvoice(targetLine.GetInvoiceID())
		if err != nil {
			return fmt.Errorf("getting invoice type for line[%s]: %w", targetLine.GetLineID().ID, err)
		}

		if isGatheringInvoice {
			invoiceAtAccessor, ok := targetLine.(billing.InvoiceAtAccessor)
			if !ok {
				return fmt.Errorf("target line is not an invoice at accessor: %T", targetLine)
			}

			invoiceAtAccessor.SetInvoiceAt(expectedLine.InvoiceAt)
		}
	}

	if targetLine.GetDeletedAt() != nil {
		targetLine.SetDeletedAt(nil)
		wasChange = true
	}

	perUnitAmountExisting, err := invoiceupdater.GetFlatFeePerUnitAmount(existingLine)
	if err != nil {
		return fmt.Errorf("getting flat fee per unit amount: %w", err)
	}

	perUnitAmountExpected, err := invoiceupdater.GetFlatFeePerUnitAmount(expectedLine)
	if err != nil {
		return fmt.Errorf("getting flat fee per unit amount: %w", err)
	}

	if !perUnitAmountExisting.Equal(perUnitAmountExpected) {
		if err := invoiceupdater.SetFlatFeePerUnitAmount(targetLine, perUnitAmountExpected); err != nil {
			return fmt.Errorf("setting flat fee per unit amount: %w", err)
		}
		wasChange = true
	}

	if !wasChange {
		return nil
	}

	return c.addPatches(target.UniqueID, PatchOperationProrate, invoiceupdater.NewUpdateLinePatch(targetLine))
}
