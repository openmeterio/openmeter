package reconciler

import (
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func expandCreatePatch(input ExpandInput, target targetstate.SubscriptionItemWithPeriods) ([]invoiceupdater.Patch, error) {
	line, err := targetstate.LineFromSubscriptionRateCard(input.Subscription, target, input.Currency)
	if err != nil {
		return nil, fmt.Errorf("generating line from subscription item [%s]: %w", target.SubscriptionItem.ID, err)
	}

	if line == nil {
		return nil, nil
	}

	return []invoiceupdater.Patch{invoiceupdater.NewCreateLinePatch(*line)}, nil
}

func expandShrinkPatch(input ExpandInput, existing billing.LineOrHierarchy, target targetstate.SubscriptionItemWithPeriods) ([]invoiceupdater.Patch, error) {
	return expandExistingPatch(input, existing, target, PatchOperationShrink)
}

func expandExtendPatch(input ExpandInput, existing billing.LineOrHierarchy, target targetstate.SubscriptionItemWithPeriods) ([]invoiceupdater.Patch, error) {
	return expandExistingPatch(input, existing, target, PatchOperationExtend)
}

func expandExistingPatch(input ExpandInput, existing billing.LineOrHierarchy, target targetstate.SubscriptionItemWithPeriods, operation PatchOperation) ([]invoiceupdater.Patch, error) {
	expectedLine, err := materializeExpectedLine(input, target, operation)
	if err != nil {
		return nil, err
	}

	return expandExistingLineOrHierarchy(existing, *expectedLine, input.Invoices, operation)
}

func materializeExpectedLine(input ExpandInput, target targetstate.SubscriptionItemWithPeriods, operation PatchOperation) (*billing.GatheringLine, error) {
	expectedLine, err := targetstate.LineFromSubscriptionRateCard(input.Subscription, target, input.Currency)
	if err != nil {
		return nil, fmt.Errorf("generating expected line[%s]: %w", target.UniqueID, err)
	}

	if expectedLine == nil {
		return nil, fmt.Errorf("%s patch[%s] cannot be expanded to a nil target line", operation, target.UniqueID)
	}

	return expectedLine, nil
}

func expandExistingLineOrHierarchy(existingLine billing.LineOrHierarchy, expectedLine billing.GatheringLine, invoices persistedstate.Invoices, operation PatchOperation) ([]invoiceupdater.Patch, error) {
	switch existingLine.Type() {
	case billing.LineOrHierarchyTypeLine:
		line, err := existingLine.AsGenericLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		return expandExistingLine(line, expectedLine, invoices, operation)
	case billing.LineOrHierarchyTypeHierarchy:
		group, err := existingLine.AsHierarchy()
		if err != nil {
			return nil, fmt.Errorf("getting hierarchy: %w", err)
		}

		return expandExistingHierarchy(group, expectedLine, invoices, operation)
	default:
		return nil, fmt.Errorf("unsupported line or hierarchy type: %s", existingLine.Type())
	}
}

func expandExistingLine(existingLine billing.GenericInvoiceLine, expectedLine billing.GatheringLine, invoices persistedstate.Invoices, operation PatchOperation) ([]invoiceupdater.Patch, error) {
	if shouldSkipExistingLinePatch(existingLine, expectedLine) {
		return nil, nil
	}

	if err := validateLineExpansionOperation(existingLine.GetServicePeriod(), expectedLine.ServicePeriod, operation); err != nil {
		return nil, err
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

		if invoices.IsGatheringInvoice(targetLine.GetInvoiceID()) {
			invoiceAtAccessor, ok := targetLine.(billing.InvoiceAtAccessor)
			if !ok {
				return nil, fmt.Errorf("target line is not an invoice at accessor: %T", targetLine)
			}

			invoiceAtAccessor.SetInvoiceAt(expectedLine.InvoiceAt)
		}
	}

	if !invoiceupdater.IsFlatFee(targetLine) {
		if targetLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
			return []invoiceupdater.Patch{
				invoiceupdater.NewDeleteLinePatch(existingLine.GetLineID(), existingLine.GetInvoiceID()),
			}, nil
		}
	}

	if targetLine.GetDeletedAt() != nil {
		targetLine.SetDeletedAt(nil)
		wasChange = true
	}

	if invoiceupdater.IsFlatFee(targetLine) {
		return nil, fmt.Errorf("%s patch cannot be applied to flat fee line[%s]", operation, existingLine.GetLineID().ID)
	}

	if !wasChange {
		return nil, nil
	}

	return []invoiceupdater.Patch{
		invoiceupdater.NewUpdateLinePatch(targetLine),
	}, nil
}

func expandExistingHierarchy(existingHierarchy *billing.SplitLineHierarchy, expectedLine billing.GatheringLine, invoices persistedstate.Invoices, operation PatchOperation) ([]invoiceupdater.Patch, error) {
	if err := validateHierarchyExpansionOperation(existingHierarchy.Group.ServicePeriod.ToClosedPeriod(), expectedLine.ServicePeriod, operation); err != nil {
		return nil, err
	}

	if existingHierarchy.Group.ServicePeriod.End.Equal(expectedLine.ServicePeriod.To) {
		return nil, nil
	}

	patches := []invoiceupdater.Patch{}

	if existingHierarchy.Group.ServicePeriod.End.Before(expectedLine.ServicePeriod.To) {
		if len(existingHierarchy.Lines) > 0 {
			lines := existingHierarchy.Lines
			slices.SortFunc(lines, func(i, j billing.LineWithInvoiceHeader) int {
				return timeutil.Compare(i.Line.GetServicePeriod().To, j.Line.GetServicePeriod().To)
			})

			lastChild, err := lines[len(lines)-1].Line.CloneWithoutChildren()
			if err != nil {
				return nil, fmt.Errorf("cloning last child: %w", err)
			}

			if lastChild.GetManagedBy() == billing.SubscriptionManagedLine {
				lastChild.SetDeletedAt(nil)
			}

			lastChild.UpdateServicePeriod(func(p *timeutil.ClosedPeriod) {
				p.To = expectedLine.ServicePeriod.To
			})

			if invoices.IsGatheringInvoice(lastChild.GetInvoiceID()) {
				invoiceAtAccessor, ok := lastChild.(billing.InvoiceAtAccessor)
				if !ok {
					return nil, fmt.Errorf("last child is not an invoice at accessor: %T", lastChild)
				}

				invoiceAtAccessor.SetInvoiceAt(expectedLine.InvoiceAt)
			}

			patches = append(patches, invoiceupdater.NewUpdateLinePatch(lastChild))
		}

		updatedGroup := existingHierarchy.Group.ToUpdate()
		updatedGroup.ServicePeriod.End = expectedLine.ServicePeriod.To
		patches = append(patches, invoiceupdater.NewUpdateSplitLineGroupPatch(updatedGroup))

		return patches, nil
	}

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

func validateLineExpansionOperation(existingPeriod timeutil.ClosedPeriod, expectedPeriod timeutil.ClosedPeriod, operation PatchOperation) error {
	switch operation {
	case PatchOperationShrink:
		if !expectedPeriod.To.Before(existingPeriod.To) {
			return fmt.Errorf("shrink patch requires target end before existing end: existing=%s..%s target=%s..%s", existingPeriod.From, existingPeriod.To, expectedPeriod.From, expectedPeriod.To)
		}
	case PatchOperationExtend:
		if !expectedPeriod.To.After(existingPeriod.To) {
			return fmt.Errorf("extend patch requires target end after existing end: existing=%s..%s target=%s..%s", existingPeriod.From, existingPeriod.To, expectedPeriod.From, expectedPeriod.To)
		}
	default:
		return fmt.Errorf("unsupported line expansion operation: %s", operation)
	}

	return nil
}

func validateHierarchyExpansionOperation(existingPeriod timeutil.ClosedPeriod, expectedPeriod timeutil.ClosedPeriod, operation PatchOperation) error {
	switch operation {
	case PatchOperationShrink:
		if !expectedPeriod.To.Before(existingPeriod.To) {
			return fmt.Errorf("shrink patch requires target end before existing hierarchy end: existing=%s..%s target=%s..%s", existingPeriod.From, existingPeriod.To, expectedPeriod.From, expectedPeriod.To)
		}
	case PatchOperationExtend:
		if !expectedPeriod.To.After(existingPeriod.To) {
			return fmt.Errorf("extend patch requires target end after existing hierarchy end: existing=%s..%s target=%s..%s", existingPeriod.From, existingPeriod.To, expectedPeriod.From, expectedPeriod.To)
		}
	default:
		return fmt.Errorf("unsupported hierarchy expansion operation: %s", operation)
	}

	return nil
}

func shouldSkipExistingLinePatch(existingLine billing.GenericInvoiceLine, expectedLine billing.GatheringLine) bool {
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
