package reconciler

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *Service) Apply(ctx context.Context, input ApplyInput) error {
	patches, err := s.getPatchesFromPlan(input.Plan, input.Subscription, input.Currency, input.Invoices)
	if err != nil {
		return err
	}

	newLinePatches, err := s.getNewUpcomingLinePatches(ctx, input.Subscription, input.Currency, input.Plan.NewSubscriptionItems)
	if err != nil {
		return fmt.Errorf("provisioning pending lines: %w", err)
	}

	patches = append(patches, newLinePatches...)

	invoiceUpdater := NewInvoiceUpdater(s.billingService, s.logger)
	if err := invoiceUpdater.ApplyPatches(ctx, input.Customer, patches); err != nil {
		return fmt.Errorf("updating invoices: %w", err)
	}

	return nil
}

func (s *Service) getPatchesFromPlan(p *Plan, subs subscription.SubscriptionView, currency currencyx.Calculator, invoices persistedstate.Invoices) ([]Patch, error) {
	patches := make([]Patch, 0, len(p.LinesToDelete)+len(p.LinesToUpsert))

	for _, line := range p.LinesToDelete {
		deletePatches, err := GetDeletePatchesForLine(line)
		if err != nil {
			return nil, fmt.Errorf("getting delete patches for line: %w", err)
		}

		patches = append(patches, deletePatches...)
	}

	for _, line := range p.LinesToUpsert {
		expectedLine, err := targetstate.LineFromSubscriptionRateCard(subs, line.Target, currency)
		if err != nil {
			return nil, fmt.Errorf("generating expected line[%s]: %w", line.Target.UniqueID, err)
		}

		if expectedLine == nil {
			deletePatches, err := GetDeletePatchesForLine(line.Existing)
			if err != nil {
				return nil, fmt.Errorf("getting delete patches for line: %w", err)
			}

			patches = append(patches, deletePatches...)
			continue
		}

		updatePatches, err := s.getPatchesForExistingLineOrHierarchy(line.Existing, *expectedLine, invoices)
		if err != nil {
			return nil, fmt.Errorf("updating line[%s]: %w", line.Target.UniqueID, err)
		}

		patches = append(patches, updatePatches...)
	}

	return patches, nil
}

func (s *Service) getNewUpcomingLinePatches(ctx context.Context, subs subscription.SubscriptionView, currency currencyx.Calculator, subsItems []targetstate.SubscriptionItemWithPeriods) ([]Patch, error) {
	newLines, err := slicesx.MapWithErr(subsItems, func(subsItem targetstate.SubscriptionItemWithPeriods) (*billing.GatheringLine, error) {
		line, err := targetstate.LineFromSubscriptionRateCard(subs, subsItem, currency)
		if err != nil {
			return nil, fmt.Errorf("generating line from subscription item [%s]: %w", subsItem.SubscriptionItem.ID, err)
		}

		return line, nil
	})
	if err != nil {
		return nil, fmt.Errorf("creating new lines: %w", err)
	}

	lines := lo.Filter(newLines, func(l *billing.GatheringLine, _ int) bool {
		return l != nil
	})

	return lo.Map(lines, func(l *billing.GatheringLine, _ int) Patch {
		return NewCreateLinePatch(*l)
	}), nil
}

func (s *Service) getPatchesForExistingLineOrHierarchy(existingLine billing.LineOrHierarchy, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]Patch, error) {
	switch existingLine.Type() {
	case billing.LineOrHierarchyTypeLine:
		line, err := existingLine.AsGenericLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		return s.getPatchesForExistingLine(line, expectedLine, invoices)
	case billing.LineOrHierarchyTypeHierarchy:
		group, err := existingLine.AsHierarchy()
		if err != nil {
			return nil, fmt.Errorf("getting hierarchy: %w", err)
		}

		return s.getPatchesForExistingHierarchy(group, expectedLine, invoices)
	default:
		return nil, fmt.Errorf("unsupported line or hierarchy type: %s", existingLine.Type())
	}
}

func (s *Service) getPatchesForExistingLine(existingLine billing.GenericInvoiceLine, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]Patch, error) {
	if expectedLine.Annotations.GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return nil, nil
	}

	if existingLine.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return nil, nil
	}

	if existingLine.GetManagedBy() != billing.SubscriptionManagedLine {
		return nil, nil
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

	if !IsFlatFee(targetLine) {
		if targetLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
			return []Patch{
				NewDeleteLinePatch(existingLine.GetLineID(), existingLine.GetInvoiceID()),
			}, nil
		}
	}

	if targetLine.GetDeletedAt() != nil {
		targetLine.SetDeletedAt(nil)
		wasChange = true
	}

	if IsFlatFee(targetLine) {
		if !IsFlatFee(expectedLine) {
			return nil, errors.New("cannot merge flat fee line with usage based line")
		}

		perUnitAmountExisting, err := GetFlatFeePerUnitAmount(existingLine)
		if err != nil {
			return nil, fmt.Errorf("getting flat fee per unit amount: %w", err)
		}

		perUnitAmountExpected, err := GetFlatFeePerUnitAmount(expectedLine)
		if err != nil {
			return nil, fmt.Errorf("getting flat fee per unit amount: %w", err)
		}

		if !perUnitAmountExisting.Equal(perUnitAmountExpected) {
			if err := SetFlatFeePerUnitAmount(targetLine, perUnitAmountExpected); err != nil {
				return nil, fmt.Errorf("setting flat fee per unit amount: %w", err)
			}
			wasChange = true
		}
	}

	if !wasChange {
		return nil, nil
	}

	return []Patch{
		NewUpdateLinePatch(targetLine),
	}, nil
}

func (s *Service) getPatchesForExistingHierarchy(existingHierarchy *billing.SplitLineHierarchy, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]Patch, error) {
	if existingHierarchy.Group.ServicePeriod.End.Equal(expectedLine.ServicePeriod.To) {
		return nil, nil
	}

	patches := []Patch{}

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
			patches = append(patches, NewUpdateLinePatch(lastChild))
		}

		updatedGroup := existingHierarchy.Group.ToUpdate()
		updatedGroup.ServicePeriod.End = expectedLine.ServicePeriod.To
		patches = append(patches, NewUpdateSplitLineGroupPatch(updatedGroup))

		return patches, nil
	}

	for _, child := range existingHierarchy.Lines {
		if child.Line.GetServicePeriod().To.Before(expectedLine.ServicePeriod.To) {
			continue
		}

		if child.Line.GetServicePeriod().From.After(expectedLine.ServicePeriod.To) {
			patches = append(patches, NewDeleteLinePatch(child.Line.GetLineID(), child.Line.GetInvoiceID()))
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

			if !IsFlatFee(updatedLine) {
				if updatedLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
					patches = append(patches, NewDeleteLinePatch(updatedLine.GetLineID(), updatedLine.GetInvoiceID()))
					continue
				}
			}

			patches = append(patches, NewUpdateLinePatch(updatedLine))
		}
	}

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.End = expectedLine.ServicePeriod.To
	patches = append(patches, NewUpdateSplitLineGroupPatch(updatedGroup))

	return patches, nil
}
