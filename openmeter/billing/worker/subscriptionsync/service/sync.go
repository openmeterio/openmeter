package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	SubscriptionSyncComponentName billing.ComponentName = "subscription-sync"
)

func (s *Service) invoicePendingLines(ctx context.Context, customer customer.CustomerID) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.invoicePendingLines", trace.WithAttributes(
		attribute.String("customer_id", customer.ID),
	))

	return span.Wrap(func(ctx context.Context) error {
		_, err := s.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer:                   customer,
			ProgressiveBillingOverride: lo.ToPtr(false),
		})
		if err != nil {
			if errors.Is(err, billing.ErrInvoiceCreateNoLines) {
				return nil
			}

			return err
		}

		return nil
	})
}

func (s *Service) HandleSubscriptionSyncEvent(ctx context.Context, event *subscription.SubscriptionSyncEvent) error {
	if event == nil {
		return nil
	}

	subsView, err := s.subscriptionService.GetView(ctx, event.Subscription.NamespacedID)
	if err != nil {
		return fmt.Errorf("getting subscription view: %w", err)
	}

	return s.SynchronizeSubscriptionAndInvoiceCustomer(ctx, subsView, time.Now())
}

func (s *Service) SynchronizeSubscriptionAndInvoiceCustomer(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscriptionAndInvoiceCustomer", trace.WithAttributes(
		attribute.String("subscription_id", subs.Subscription.ID),
		attribute.String("as_of", asOf.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) error {
		if err := s.SynchronizeSubscription(ctx, subs, asOf); err != nil {
			return fmt.Errorf("synchronize subscription: %w", err)
		}

		customerID := customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}
		// Invoice any pending lines invoicable now, so that any in advance fees are invoiced immediately.
		if err := s.invoicePendingLines(ctx, customerID); err != nil {
			return fmt.Errorf("invoice pending lines (post): %w [customer_id=%s]", err, customerID.ID)
		}

		return nil
	})
}

func (s *Service) SynchronizeSubscription(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscription", trace.WithAttributes(
		attribute.String("subscription_id", subs.Subscription.ID),
		attribute.String("as_of", asOf.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) error {
		if !subs.Spec.HasBillables() {
			if err := s.updateSyncState(ctx, updateSyncStateInput{
				SubscriptionView: subs,
			}); err != nil {
				return fmt.Errorf("updating sync state: %w", err)
			}

			s.logger.DebugContext(ctx, "subscription has no billables, skipping sync", "subscription_id", subs.Subscription.ID)
			return nil
		}

		customerID := customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}

		// TODO[later]: Right now we are getting the billing profile as a validation step, but later if we allow more collection
		// alignment settings, we should use the collection settings from here to determine the generation end (overriding asof).
		_, err := s.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customerID,
		})
		if err != nil {
			return fmt.Errorf("getting billing profile: %w", err)
		}

		currency, err := subs.Spec.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("getting currency calculator: %w", err)
		}

		return s.billingService.WithLock(ctx, customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}, func(ctx context.Context) error {
			persistedLoader := persistedstate.NewLoader(s.billingService)
			persistedInvoices, err := persistedLoader.LoadInvoicesForCustomer(ctx, customerID)
			if err != nil {
				return err
			}

			// Calculate per line patches
			linesDiff, err := s.compareSubscriptionWithExistingLines(ctx, subs, asOf)
			if err != nil {
				return err
			}

			if linesDiff == nil || linesDiff.IsEmpty() {
				generationLimit := time.Time{}
				if linesDiff != nil {
					generationLimit = linesDiff.SubscriptionMaxGenerationTimeLimit
				}

				if err := s.updateSyncState(ctx, updateSyncStateInput{
					SubscriptionView:       subs,
					MaxGenerationTimeLimit: generationLimit,
				}); err != nil {
					return fmt.Errorf("updating sync state: %w", err)
				}

				return nil
			}

			patches, err := s.getPatchesFromPlan(linesDiff, subs, currency, persistedInvoices)
			if err != nil {
				return err
			}

			// Apply the patches to the customer's invoices

			newLinePatches, err := s.getNewUpcomingLinePatches(ctx,
				subs,
				currency,
				linesDiff.NewSubscriptionItems,
			)
			if err != nil {
				return fmt.Errorf("provisioning pending lines: %w", err)
			}

			patches = append(patches, newLinePatches...)

			invoiceUpdater := NewInvoiceUpdater(s.billingService, s.logger)
			if err := invoiceUpdater.ApplyPatches(ctx, customerID, patches); err != nil {
				return fmt.Errorf("updating invoices: %w", err)
			}

			if err := s.updateSyncState(ctx, updateSyncStateInput{
				SubscriptionView:       subs,
				MaxGenerationTimeLimit: linesDiff.SubscriptionMaxGenerationTimeLimit,
			}); err != nil {
				return fmt.Errorf("updating sync state: %w", err)
			}

			return nil
		})
	})
}

type updateSyncStateInput struct {
	SubscriptionView       subscription.SubscriptionView
	MaxGenerationTimeLimit time.Time
}

func (s *Service) updateSyncState(ctx context.Context, in updateSyncStateInput) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.updateSyncState", trace.WithAttributes(
		attribute.String("subscription_id", in.SubscriptionView.Subscription.ID),
		attribute.String("max_generation_time_limit", in.MaxGenerationTimeLimit.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) error {
		if !in.SubscriptionView.Spec.HasBillables() {
			return s.subscriptionSyncAdapter.UpsertSyncState(ctx, subscriptionsync.UpsertSyncStateInput{
				SubscriptionID: models.NamespacedID{
					ID:        in.SubscriptionView.Subscription.ID,
					Namespace: in.SubscriptionView.Subscription.Namespace,
				},
				HasBillables: false,
				SyncedAt:     clock.Now().UTC(),
			})
		}

		nextSyncAfter := in.MaxGenerationTimeLimit

		if in.MaxGenerationTimeLimit.IsZero() {
			// Fallback: we cannot determine the next sync after, so we'll just mandate the sync
			if nextSyncAfter.IsZero() {
				s.logger.WarnContext(ctx, "cannot determine the next sync after, syncing immediately", "subscription_id", in.SubscriptionView.Subscription.ID)
				nextSyncAfter = clock.Now().UTC()
			}
		}

		return s.subscriptionSyncAdapter.UpsertSyncState(ctx, subscriptionsync.UpsertSyncStateInput{
			SubscriptionID: models.NamespacedID{
				ID:        in.SubscriptionView.Subscription.ID,
				Namespace: in.SubscriptionView.Subscription.Namespace,
			},
			HasBillables:  true,
			NextSyncAfter: lo.ToPtr(nextSyncAfter),
			SyncedAt:      clock.Now().UTC(),
		})
	})
}

type subscriptionSyncPlan struct {
	NewSubscriptionItems               []targetstate.SubscriptionItemWithPeriods
	LinesToDelete                      []billing.LineOrHierarchy
	LinesToUpsert                      []subscriptionSyncPlanLineUpsert
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (s *subscriptionSyncPlan) IsEmpty() bool {
	if s == nil {
		return true
	}

	return len(s.NewSubscriptionItems) == 0 && len(s.LinesToDelete) == 0 && len(s.LinesToUpsert) == 0
}

type subscriptionSyncPlanLineUpsert struct {
	Target   targetstate.SubscriptionItemWithPeriods
	Existing billing.LineOrHierarchy
}

// calculateSyncPlan calculates the sync plan for the subscription, it returns the lines to delete, the lines to upsert and the new subscription items.
func (s *Service) compareSubscriptionWithExistingLines(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) (*subscriptionSyncPlan, error) {
	span := tracex.Start[*subscriptionSyncPlan](ctx, s.tracer, "billing.worker.subscription.sync.compareSubscriptionWithExistingLines")

	return span.Wrap(func(ctx context.Context) (*subscriptionSyncPlan, error) {
		persistedLoader := persistedstate.NewLoader(s.billingService)
		persisted, err := persistedLoader.LoadForSubscription(ctx, subs)
		if err != nil {
			return nil, err
		}

		targetBuilder := targetstate.NewBuilder(s.logger, s.tracer)
		target, err := targetBuilder.Build(ctx, subs, asOf, persisted)
		if err != nil {
			return nil, err
		}

		inScopeLines := target.Items

		if len(inScopeLines) == 0 && len(persisted.Lines) == 0 {
			return &subscriptionSyncPlan{
				SubscriptionMaxGenerationTimeLimit: target.MaxGenerationTimeLimit,
			}, nil
		}

		inScopeLinesByUniqueID, unique := slicesx.UniqueGroupBy(inScopeLines, func(i targetstate.SubscriptionItemWithPeriods) string {
			return i.UniqueID
		})
		if !unique {
			return nil, fmt.Errorf("duplicate unique ids in the upcoming lines")
		}

		existingLineUniqueIDs := lo.Keys(persisted.ByUniqueID)
		inScopeLineUniqueIDs := lo.Keys(inScopeLinesByUniqueID)
		// Let's execute the synchronization
		deletedLines, newLines := lo.Difference(existingLineUniqueIDs, inScopeLineUniqueIDs)
		lineIDsToUpsert := lo.Intersect(existingLineUniqueIDs, inScopeLineUniqueIDs)

		linesToDelete, err := slicesx.MapWithErr(deletedLines, func(id string) (billing.LineOrHierarchy, error) {
			line, ok := persisted.ByUniqueID[id]
			if !ok {
				return billing.LineOrHierarchy{}, fmt.Errorf("existing line[%s] not found in the existing lines", id)
			}

			return line, nil
		})
		if err != nil {
			return nil, fmt.Errorf("mapping deleted lines: %w", err)
		}

		linesToUpsert, err := slicesx.MapWithErr(lineIDsToUpsert, func(id string) (subscriptionSyncPlanLineUpsert, error) {
			existingLine, ok := persisted.ByUniqueID[id]
			if !ok {
				return subscriptionSyncPlanLineUpsert{}, fmt.Errorf("existing line[%s] not found in the existing lines", id)
			}

			return subscriptionSyncPlanLineUpsert{
				Target:   inScopeLinesByUniqueID[id],
				Existing: existingLine,
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("mapping upsert lines: %w", err)
		}

		return &subscriptionSyncPlan{
			NewSubscriptionItems: lo.Map(newLines, func(id string, _ int) targetstate.SubscriptionItemWithPeriods {
				return inScopeLinesByUniqueID[id]
			}),
			LinesToDelete:                      linesToDelete,
			LinesToUpsert:                      linesToUpsert,
			SubscriptionMaxGenerationTimeLimit: target.MaxGenerationTimeLimit,
		}, nil
	})
}

func (s *Service) lineHasAnnotation(managedBy billing.InvoiceLineManagedBy, annotations models.Annotations, annotation string) bool {
	if managedBy != billing.SubscriptionManagedLine {
		// We only correct the period start for subscription managed lines, for manual edits
		// we should not apply this logic, as the user might have created a setup where the period start
		// is no longer valid.
		return false
	}

	return annotations.GetBool(annotation)
}

func (s *Service) hierarchyHasAnnotation(hierarchy *billing.SplitLineHierarchy, annotation string) (bool, error) {
	servicePeriod := hierarchy.Group.ServicePeriod

	// The correction can only happen if the last line the progressively billed group is in scope for the period correction
	for _, child := range hierarchy.Lines {
		if child.Line.GetServicePeriod().To.Equal(servicePeriod.End) && child.Line.GetDeletedAt() == nil {
			return s.lineHasAnnotation(child.Line.GetManagedBy(), child.Line.GetAnnotations(), annotation), nil
		}
	}

	return false, nil
}

func (s *Service) getPatchesFromPlan(p *subscriptionSyncPlan, subs subscription.SubscriptionView, currency currencyx.Calculator, invoices persistedstate.Invoices) ([]linePatch, error) {
	patches := make([]linePatch, 0, len(p.LinesToDelete)+len(p.LinesToUpsert))

	// Let's update the existing lines
	for _, line := range p.LinesToDelete {
		deletePatches, err := s.getDeletePatchesForLine(line)
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

		// The line have 0 amount, so we are not going to bill it. This can happen if we are quickly changing subscriptions
		// immediately.
		if expectedLine == nil {
			deletePatches, err := s.getDeletePatchesForLine(line.Existing)
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

func (s *Service) getNewUpcomingLinePatches(ctx context.Context, subs subscription.SubscriptionView, currency currencyx.Calculator, subsItems []targetstate.SubscriptionItemWithPeriods) ([]linePatch, error) {
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

	return lo.Map(lines, func(l *billing.GatheringLine, _ int) linePatch {
		return newCreateLinePatch(*l)
	}), nil
}

func (s *Service) getPatchesForExistingLineOrHierarchy(existingLine billing.LineOrHierarchy, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]linePatch, error) {
	// TODO/WARNING[later]: This logic should be fine with everything that can be billed progressively, however the following use-cases
	// will behave strangely:
	//
	// - An in advance flat fee cannot be prorated, as that would require void/credit logic.
	// - If a volume based item's tiered are changed, then the old volume based item will be billed at that rate, and the
	//   new volume based item's calculation will start from the new tiered rates beginning. (e.g. if we have a 1 month long tiered
	//   price, then we change the tiered price in the middle of the month, the old tiered price will be billed for the first half
	//   and the new tiered price will be billed for the second half, meaning that the customer will be billed for less sum traffic) [OM]
	// - If a meter is unique counted, then the unique count will be reset at the change's time

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

func (s *Service) getPatchesForExistingLine(existingLine billing.GenericInvoiceLine, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]linePatch, error) {
	// Lines can be manually marked as ignored in syncing, which is used for cases where we're doing backwards incompatible changes
	if expectedLine.Annotations.GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return nil, nil
	}

	if existingLine.GetAnnotations().GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return nil, nil
	}

	// Manual edits prevent resyncronization so that we preserve the user intent
	if existingLine.GetManagedBy() != billing.SubscriptionManagedLine {
		return nil, nil
	}

	// We assume that only the period can change, maybe some pricing data due to prorating (for flat lines)

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
				// If the period has changed we need to adjust the invoice at to the expected line's invoice at for
				// gathering invoices (for standard it doesn't matter)
				return nil, fmt.Errorf("target line is not an invoice at accessor: %T", targetLine)
			}

			invoiceAtAccessor.SetInvoiceAt(expectedLine.InvoiceAt)
		}
	}

	if !isFlatFee(targetLine) {
		// UBP Empty lines are not allowed, let's delete them instead
		if targetLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
			return []linePatch{
				newDeleteLinePatch(existingLine.GetLineID(), existingLine.GetInvoiceID()),
			}, nil
		}
	}

	if targetLine.GetDeletedAt() != nil {
		targetLine.SetDeletedAt(nil)
		wasChange = true
	}

	// Let's handle the flat fee prorating (e.g. syncronizing the amount maybe in retrospect)
	if isFlatFee(targetLine) {
		if !isFlatFee(expectedLine) {
			return nil, errors.New("cannot merge flat fee line with usage based line")
		}

		perUnitAmountExisting, err := getFlatFeePerUnitAmount(existingLine)
		if err != nil {
			return nil, fmt.Errorf("getting flat fee per unit amount: %w", err)
		}

		perUnitAmountExpected, err := getFlatFeePerUnitAmount(expectedLine)
		if err != nil {
			return nil, fmt.Errorf("getting flat fee per unit amount: %w", err)
		}

		if !perUnitAmountExisting.Equal(perUnitAmountExpected) {
			if err := setFlatFeePerUnitAmount(targetLine, perUnitAmountExpected); err != nil {
				return nil, fmt.Errorf("setting flat fee per unit amount: %w", err)
			}
			wasChange = true
		}
	}

	if !wasChange {
		return nil, nil
	}

	return []linePatch{
		newUpdateLinePatch(targetLine),
	}, nil
}

func (s *Service) getPatchesForExistingHierarchy(existingHierarchy *billing.SplitLineHierarchy, expectedLine billing.GatheringLine, invoices persistedstate.Invoices) ([]linePatch, error) {
	// Parts of the line has been already invoiced using progressive invoicing, so we need to examine the children

	// Nothing to do here, as split lines are UBP lines and thus we don't need the flat fee corrections
	// TODO[later]: When we implement progressive billing based pro-rating, we need to support adjusting flat fee
	// segments here.

	if existingHierarchy.Group.ServicePeriod.End.Equal(expectedLine.ServicePeriod.To) {
		// The line is already in the expected state, so we can safely return here
		return nil, nil
	}

	patches := []linePatch{}

	// Case #1: The line is being expanded (e.g. continue subscription)
	if existingHierarchy.Group.ServicePeriod.End.Before(expectedLine.ServicePeriod.To) {
		// Expansion of the line (e.g. continue subscription)

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
				// We are not supporting period changes for children, and we need to maintain the consistency so
				// even for overridden lines we need to update the period

				// We however allow deletions, so we are only un-deleting the line here if it was deleted by the sync engine
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
			patches = append(patches, newUpdateLinePatch(lastChild))
		}

		// We have already updated the last child, so we need to update at least the periods regardless of managed_by to keep the consistency
		updatedGroup := existingHierarchy.Group.ToUpdate()
		updatedGroup.ServicePeriod.End = expectedLine.ServicePeriod.To
		patches = append(patches, newUpdateSplitLineGroupPatch(updatedGroup))

		return patches, nil
	}

	// Shrink of the line (e.g. subscription cancled, subscription item edit)

	for _, child := range existingHierarchy.Lines {
		if child.Line.GetServicePeriod().To.Before(expectedLine.ServicePeriod.To) {
			// The child is not affected by the period shrink, so we can skip it
			continue
		}

		if child.Line.GetServicePeriod().From.After(expectedLine.ServicePeriod.To) {
			// The child is after the period shrink, so we need to delete it as it became invalid
			patches = append(patches, newDeleteLinePatch(child.Line.GetLineID(), child.Line.GetInvoiceID()))
			continue
		}

		// The child is partially affected by the period shrink, so we need to adjust the period
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

			if !isFlatFee(updatedLine) {
				// UBP Empty lines are not allowed, let's delete them instead
				if updatedLine.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
					patches = append(patches, newDeleteLinePatch(updatedLine.GetLineID(), updatedLine.GetInvoiceID()))
					continue
				}
			}

			patches = append(patches, newUpdateLinePatch(updatedLine))
		}
	}

	// Let's make sure that the group's end is updated to the expected period end, so that
	// we can reliably detect last child

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.End = expectedLine.ServicePeriod.To
	patches = append(patches, newUpdateSplitLineGroupPatch(updatedGroup))

	return patches, nil
}
