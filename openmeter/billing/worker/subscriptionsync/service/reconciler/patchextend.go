package reconciler

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	"github.com/samber/lo"
)

type NewLineExtendUsageBasedPatchInput struct {
	Existing persistedstate.Entity
	Target   targetstate.SubscriptionItemWithPeriods
}

func (i NewLineExtendUsageBasedPatchInput) Validate() error {
	var errs []error

	if i.Existing == nil {
		errs = append(errs, errors.New("existing is required"))
	}

	if err := i.Target.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target: %w", err))
	}

	return errors.Join(errs...)
}

func (s *Service) NewLineExtendUsageBasedPatch(input NewLineExtendUsageBasedPatchInput) (Patch, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("new line extend usage based patch: %w", err)
	}

	return newFromEntity(newFromEntityInput{
		Entity: input.Existing,
		NewInvoicePatch: func(lineOrHierarchy billing.LineOrHierarchy) (Patch, error) {
			return LineExtendUsageBasedPatch{
				Existing: lineOrHierarchy,
				Target:   input.Target,
			}, nil
		},
	})
}

var _ InvoicePatch = (*LineExtendUsageBasedPatch)(nil)

type LineExtendUsageBasedPatch struct {
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods
}

func (p LineExtendUsageBasedPatch) Operation() PatchOperation {
	return PatchOperationExtend
}

func (p LineExtendUsageBasedPatch) UniqueReferenceID() string {
	return lo.FromPtr(p.Existing.ChildUniqueReferenceID())
}

func (p LineExtendUsageBasedPatch) GetInvoicePatches(input GetInvoicePatchesInput) ([]invoiceupdater.Patch, error) {
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

func (p LineExtendUsageBasedPatch) getInvoicePatchesForLine(existingLine billing.GenericInvoiceLine, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]invoiceupdater.Patch, error) {
	if shouldSkipLinePatch(existingLine, expectedLine) {
		return nil, nil
	}

	if existingLine.GetServicePeriod().To.Equal(expectedLine.ServicePeriod.To) {
		return nil, nil
	}

	if !expectedLine.ServicePeriod.To.After(existingLine.GetServicePeriod().To) {
		return nil, fmt.Errorf("[line] extend patch requires target end after existing end: existing=%s..%s target=%s..%s", existingLine.GetServicePeriod().From, existingLine.GetServicePeriod().To, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	return getPatchesForUpdateUsageBasedLine(existingLine, expectedLine, invoices)
}

func (p LineExtendUsageBasedPatch) getInvoicePatchesForHierarchy(existingHierarchy *billing.SplitLineHierarchy, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]invoiceupdater.Patch, error) {
	if shouldSkipHierarchyPatch(existingHierarchy, expectedLine) {
		return nil, nil
	}

	if existingHierarchy.Group.ServicePeriod.End.Equal(expectedLine.ServicePeriod.To) {
		return nil, nil
	}

	if !expectedLine.ServicePeriod.To.After(existingHierarchy.Group.ServicePeriod.End) {
		return nil, fmt.Errorf("[line] extend patch requires target end after existing end: existing=%s..%s target=%s..%s", existingHierarchy.Group.ServicePeriod.Start, existingHierarchy.Group.ServicePeriod.End, expectedLine.ServicePeriod.From, expectedLine.ServicePeriod.To)
	}

	patches := make([]invoiceupdater.Patch, 0, 2)

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
