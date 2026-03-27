package reconciler

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// Flat fee lines do not produce usage-based shrink/extend patches. Any period
// change for a flat fee line is reconciled through ProratePatch so that the
// service period and per-unit amount are updated together.
type ProratePatch struct {
	Existing billing.LineOrHierarchy
	Target   targetstate.StateItem

	OriginalPeriod timeutil.ClosedPeriod
	TargetPeriod   timeutil.ClosedPeriod

	OriginalAmount alpacadecimal.Decimal
	TargetAmount   alpacadecimal.Decimal
}

func (p ProratePatch) Operation() PatchOperation {
	return PatchOperationProrate
}

func (p ProratePatch) UniqueReferenceID() string {
	return p.Target.UniqueID
}

func (p ProratePatch) GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error) {
	expectedLine, err := p.Target.GetExpectedLineOrErr()
	if err != nil {
		return nil, err
	}

	if p.Existing.Type() != billing.LineOrHierarchyTypeLine {
		return nil, fmt.Errorf("prorate patch cannot be applied to non-line line or hierarchy type: %s", p.Existing.Type())
	}

	existingLine, err := p.Existing.AsGenericLine()
	if err != nil {
		return nil, fmt.Errorf("getting line: %w", err)
	}

	if shouldSkipLinePatch(existingLine, expectedLine) {
		return nil, nil
	}

	if !invoiceupdater.IsFlatFee(existingLine) {
		return nil, fmt.Errorf("prorate patch cannot be applied to non-flat fee line[%s]", existingLine.GetLineID().ID)
	}

	if !invoiceupdater.IsFlatFee(expectedLine) {
		return nil, errors.New("cannot merge flat fee line with usage based line")
	}

	targetLine, err := existingLine.CloneWithoutChildren()
	if err != nil {
		return nil, fmt.Errorf("cloning line: %w", err)
	}

	wasChange := false
	if !targetLine.GetServicePeriod().Equal(expectedLine.ServicePeriod) {
		wasChange = true

		targetLine.UpdateServicePeriod(func(period *timeutil.ClosedPeriod) {
			*period = expectedLine.ServicePeriod
		})

		if input.Invoices.IsGatheringInvoice(targetLine.GetInvoiceID()) {
			invoiceAtAccessor, ok := targetLine.(billing.InvoiceAtAccessor)
			if !ok {
				return nil, fmt.Errorf("target line is not an invoice at accessor: %T", targetLine)
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
		return nil, fmt.Errorf("getting flat fee per unit amount: %w", err)
	}

	perUnitAmountExpected, err := invoiceupdater.GetFlatFeePerUnitAmount(expectedLine)
	if err != nil {
		return nil, fmt.Errorf("getting flat fee per unit amount: %w", err)
	}

	if !perUnitAmountExisting.Equal(perUnitAmountExpected) {
		if err := invoiceupdater.SetFlatFeePerUnitAmount(targetLine, perUnitAmountExpected); err != nil {
			return nil, fmt.Errorf("setting flat fee per unit amount: %w", err)
		}
		wasChange = true
	}

	if !wasChange {
		return nil, nil
	}

	return []invoiceupdater.Patch{
		invoiceupdater.NewUpdateLinePatch(targetLine),
	}, nil
}
