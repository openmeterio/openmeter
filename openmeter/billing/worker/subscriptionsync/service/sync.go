package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

type InvoiceByID map[string]billing.StandardInvoice

func (i InvoiceByID) IsGatheringInvoice(invoiceID string) bool {
	invoice, ok := i[invoiceID]
	if !ok {
		// If the invoice is not found, we assume that it is gathering, just to be safe
		return true
	}

	return invoice.Status == billing.StandardInvoiceStatusGathering
}

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
			// Let's fetch the invoices for the customer
			invoices, err := s.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
				Namespaces: []string{subs.Subscription.Namespace},
				Customers:  []string{customerID.ID},
			})
			if err != nil {
				return fmt.Errorf("listing invoices: %w", err)
			}

			invoiceByID := lo.SliceToMap(invoices.Items, func(i billing.StandardInvoice) (string, billing.StandardInvoice) {
				return i.ID, i
			})

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

			patches, err := s.getPatchesFromPlan(linesDiff, subs, currency, invoiceByID)
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
	NewSubscriptionItems               []subscriptionItemWithPeriods
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
	Target   subscriptionItemWithPeriods
	Existing billing.LineOrHierarchy
}

// calculateSyncPlan calculates the sync plan for the subscription, it returns the lines to delete, the lines to upsert and the new subscription items.
func (s *Service) compareSubscriptionWithExistingLines(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) (*subscriptionSyncPlan, error) {
	span := tracex.Start[*subscriptionSyncPlan](ctx, s.tracer, "billing.worker.subscription.sync.compareSubscriptionWithExistingLines")

	return span.Wrap(func(ctx context.Context) (*subscriptionSyncPlan, error) {
		// Let's see what's in scope for the subscription
		// TODO: afaik this is already sorted, let's doublecheck that
		slices.SortFunc(subs.Phases, func(i, j subscription.SubscriptionPhaseView) int {
			return timeutil.Compare(i.SubscriptionPhase.ActiveFrom, j.SubscriptionPhase.ActiveFrom)
		})

		upcomingLinesResult, err := s.collectUpcomingLines(ctx, subs, asOf)
		if err != nil {
			return nil, fmt.Errorf("collecting upcoming lines: %w", err)
		}

		inScopeLines := upcomingLinesResult.Lines

		// Let's load the existing lines for the subscription
		existingLines, err := s.billingService.GetLinesForSubscription(ctx, billing.GetLinesForSubscriptionInput{
			Namespace:      subs.Subscription.Namespace,
			SubscriptionID: subs.Subscription.ID,
			CustomerID:     subs.Subscription.CustomerId,
		})
		if err != nil {
			return nil, fmt.Errorf("getting existing lines: %w", err)
		}

		if len(inScopeLines) == 0 && len(existingLines) == 0 {
			// The subscription has no invoicable items, no present lines exist, so there's nothing to do
			return &subscriptionSyncPlan{
				SubscriptionMaxGenerationTimeLimit: upcomingLinesResult.SubscriptionMaxGenerationTimeLimit,
			}, nil
		}

		existingLinesByUniqueID, unique := slicesx.UniqueGroupBy(
			lo.Filter(existingLines, func(l billing.LineOrHierarchy, _ int) bool {
				return l.ChildUniqueReferenceID() != nil
			}),
			func(l billing.LineOrHierarchy) string {
				return *l.ChildUniqueReferenceID()
			})
		if !unique {
			return nil, fmt.Errorf("duplicate unique ids in the existing lines")
		}

		// let's correct the period start (+invoiceAt) for any upcoming lines if needed
		inScopeLines, err = s.correctPeriodStartForUpcomingLines(ctx, subs.Subscription.ID, inScopeLines, existingLinesByUniqueID)
		if err != nil {
			return nil, fmt.Errorf("correcting period start for upcoming lines: %w", err)
		}

		inScopeLinesByUniqueID, unique := slicesx.UniqueGroupBy(inScopeLines, func(i subscriptionItemWithPeriods) string {
			return i.UniqueID
		})
		if !unique {
			return nil, fmt.Errorf("duplicate unique ids in the upcoming lines")
		}

		existingLineUniqueIDs := lo.Keys(existingLinesByUniqueID)
		inScopeLineUniqueIDs := lo.Keys(inScopeLinesByUniqueID)
		// Let's execute the synchronization
		deletedLines, newLines := lo.Difference(existingLineUniqueIDs, inScopeLineUniqueIDs)
		lineIDsToUpsert := lo.Intersect(existingLineUniqueIDs, inScopeLineUniqueIDs)

		linesToDelete, err := slicesx.MapWithErr(deletedLines, func(id string) (billing.LineOrHierarchy, error) {
			line, ok := existingLinesByUniqueID[id]
			if !ok {
				return billing.LineOrHierarchy{}, fmt.Errorf("existing line[%s] not found in the existing lines", id)
			}

			return line, nil
		})
		if err != nil {
			return nil, fmt.Errorf("mapping deleted lines: %w", err)
		}

		linesToUpsert, err := slicesx.MapWithErr(lineIDsToUpsert, func(id string) (subscriptionSyncPlanLineUpsert, error) {
			existingLine, ok := existingLinesByUniqueID[id]
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
			NewSubscriptionItems: lo.Map(newLines, func(id string, _ int) subscriptionItemWithPeriods {
				return inScopeLinesByUniqueID[id]
			}),
			LinesToDelete:                      linesToDelete,
			LinesToUpsert:                      linesToUpsert,
			SubscriptionMaxGenerationTimeLimit: upcomingLinesResult.SubscriptionMaxGenerationTimeLimit,
		}, nil
	})
}

// correctPeriodStartForUpcomingLines corrects the period start for the upcoming lines, it will adjust the period start for the lines.
//
// The adjustment only happens if the line is subscription managed and has billing.subscription.sync.ignore annotation. This esentially
// allows for reanchoring if the period calculation changes.
func (s *Service) correctPeriodStartForUpcomingLines(ctx context.Context, subscriptionID string, inScopeLines []subscriptionItemWithPeriods, existingLinesByUniqueID map[string]billing.LineOrHierarchy) ([]subscriptionItemWithPeriods, error) {
	for idx, line := range inScopeLines {
		if line.PeriodIndex == 0 {
			// This is the first period, so we don't need to correct the period start
			continue
		}

		// We are not correcting periods for lines that are already ignored
		if existingCurrentLine, ok := existingLinesByUniqueID[line.UniqueID]; ok {
			syncIgnore, err := s.lineOrHierarchyHasAnnotation(existingCurrentLine, billing.AnnotationSubscriptionSyncIgnore)
			if err != nil {
				return nil, fmt.Errorf("checking if line has subscription sync ignore annotation: %w", err)
			}

			if syncIgnore {
				continue
			}
		}

		previousPeriodUniqueID := strings.Join([]string{
			subscriptionID,
			line.PhaseKey,
			line.Spec.ItemKey,
			fmt.Sprintf("v[%d]", line.ItemVersion),
			fmt.Sprintf("period[%d]", line.PeriodIndex-1),
		}, "/")

		existingPreviousLine, ok := existingLinesByUniqueID[previousPeriodUniqueID]
		if !ok {
			// This is a new line, so we don't need to correct the period start
			continue
		}

		existingPreviousLineSyncIgnoreAnnotation, err := s.lineOrHierarchyHasAnnotation(existingPreviousLine, billing.AnnotationSubscriptionSyncIgnore)
		if err != nil {
			return nil, fmt.Errorf("checking if previous line has subscription sync ignore annotation: %w", err)
		}

		if !existingPreviousLineSyncIgnoreAnnotation {
			continue
		}

		// If the previous line does not have the AnnotationSubscriptionSyncForceContinuousLines annotations, we don't need to perform the period correction
		existingPreviousLineSyncForceContinuousLinesAnnotation, err := s.lineOrHierarchyHasAnnotation(existingPreviousLine, billing.AnnotationSubscriptionSyncForceContinuousLines)
		if err != nil {
			return nil, fmt.Errorf("checking if previous line has subscription sync force continuous lines annotation: %w", err)
		}

		if !existingPreviousLineSyncForceContinuousLinesAnnotation {
			continue
		}

		previousServicePeriod := existingPreviousLine.ServicePeriod()

		// If the lines are continuous we are fine
		if line.ServicePeriod.Start.Equal(previousServicePeriod.End) {
			continue
		}

		if !line.ServicePeriod.Start.Equal(line.FullServicePeriod.Start) {
			// These should match otherwise any pro-rating logic will be invalid (we are never truncating the start of the service period so this should never happen)
			return nil, fmt.Errorf("line[%s] service period and full service period start does not match", line.UniqueID)
		}

		// We are not overriding the billing period start as that is only used to determine the invoiceAt for inAdvance items
		inScopeLines[idx].ServicePeriod.Start = previousServicePeriod.End
		inScopeLines[idx].FullServicePeriod.Start = previousServicePeriod.End

		if line.FullServicePeriod.Start.Equal(line.BillingPeriod.Start) {
			// If the billing period is not truncated, we can update the line's billing period start too
			inScopeLines[idx].BillingPeriod.Start = previousServicePeriod.End
		}
	}

	return inScopeLines, nil
}

func (s *Service) lineOrHierarchyHasAnnotation(lineOrHierarchy billing.LineOrHierarchy, annotation string) (bool, error) {
	switch lineOrHierarchy.Type() {
	case billing.LineOrHierarchyTypeLine:
		previousLine, err := lineOrHierarchy.AsStandardLine()
		if err != nil {
			return false, fmt.Errorf("getting previous line: %w", err)
		}

		return s.lineHasAnnotation(previousLine, annotation), nil
	case billing.LineOrHierarchyTypeHierarchy:
		hierarchy, err := lineOrHierarchy.AsHierarchy()
		if err != nil {
			return false, fmt.Errorf("getting previous hierarchy: %w", err)
		}

		return s.hierarchyHasAnnotation(hierarchy, annotation), nil
	default:
		return false, nil
	}
}

func (s *Service) lineHasAnnotation(line *billing.StandardLine, annotation string) bool {
	if line.ManagedBy != billing.SubscriptionManagedLine {
		// We only correct the period start for subscription managed lines, for manual edits
		// we should not apply this logic, as the user might have created a setup where the period start
		// is no longer valid.
		return false
	}

	return line.Annotations.GetBool(annotation)
}

func (s *Service) hierarchyHasAnnotation(hierarchy *billing.SplitLineHierarchy, annotation string) bool {
	servicePeriod := hierarchy.Group.ServicePeriod

	// The correction can only happen if the last line the progressively billed group is in scope for the period correction
	for _, line := range hierarchy.Lines {
		if line.Line.Period.End.Equal(servicePeriod.End) {
			return s.lineHasAnnotation(line.Line, annotation)
		}
	}

	return false
}

func (s *Service) getPatchesFromPlan(p *subscriptionSyncPlan, subs subscription.SubscriptionView, currency currencyx.Calculator, invoiceByID InvoiceByID) ([]linePatch, error) {
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
		expectedLine, err := s.lineFromSubscritionRateCard(subs, line.Target, currency)
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

		updatePatches, err := s.getPatchesForExistingLineOrHierarchy(line.Existing, expectedLine, invoiceByID)
		if err != nil {
			return nil, fmt.Errorf("updating line[%s]: %w", line.Target.UniqueID, err)
		}

		patches = append(patches, updatePatches...)
	}

	return patches, nil
}

type collectUpcomingLinesResult struct {
	Lines                              []subscriptionItemWithPeriods
	SubscriptionMaxGenerationTimeLimit time.Time
}

// collectUpcomingLines collects the upcoming lines for the subscription, if it does not return any lines the subscription doesn't
// have any invoicable items.
//
// AsOf is a guideline for the end of generation, but the actual end of generation can be different based on the collection (as we
// always yield at least one line if an invoicable line exists).
//
// This approach allows us to not to have to poll all the subscriptions periodically, but we can act when an invoice is created or when
// a subscription is updated.
func (s *Service) collectUpcomingLines(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) (collectUpcomingLinesResult, error) {
	span := tracex.Start[collectUpcomingLinesResult](ctx, s.tracer, "billing.worker.subscription.sync.collectUpcomingLines")

	return span.Wrap(func(ctx context.Context) (collectUpcomingLinesResult, error) {
		inScopeLines := make([]subscriptionItemWithPeriods, 0, 128)

		maxGenerationTimeLimit := time.Time{}

		for _, phase := range subs.Phases {
			iterator, err := NewPhaseIterator(s.logger, s.tracer, subs, phase.SubscriptionPhase.Key)
			if err != nil {
				return collectUpcomingLinesResult{}, fmt.Errorf("creating phase iterator: %w", err)
			}

			if !iterator.HasInvoicableItems() {
				continue
			}

			// Lets figure out until when we need to generate lines
			generationLimit := asOf

			// we need to generate exactly until the end of the current billing cycle
			currBillingPeriod, err := subs.Spec.GetAlignedBillingPeriodAt(asOf)
			if err != nil {
				// Due to logic constraints, we cannot generate these lines before the subscription actually starts
				switch {
				case subscription.IsValidationIssueWithCode(err, subscription.ErrCodeSubscriptionBillingPeriodQueriedBeforeSubscriptionStart):
					s.logger.InfoContext(ctx, "asOf is before subscription start, advancing generation time to subscription start", "subscription_id", subs.Subscription.ID, "as_of", asOf, "subscription_start", subs.Spec.ActiveFrom)

					// We advance until subscription start to generate the first set of lines (if later we cancel or stg else, sync will handle that)
					generationLimit = subs.Subscription.ActiveFrom
				default:
					return collectUpcomingLinesResult{}, fmt.Errorf("getting aligned billing period: %w", err)
				}
			}

			// As its intended to be used as a limit we'll take it as end inclusice start exclusive (instead of normal start inclusive end exclusive)
			if !currBillingPeriod.From.IsZero() && !generationLimit.Equal(currBillingPeriod.From) {
				generationLimit = currBillingPeriod.To
			}

			if phaseStart := iterator.PhaseStart(); phaseStart.After(generationLimit) {
				// We need to have invoicable items, so we need to advance the limit here at least to phaseStart to see
				// if we can have any invoicable items.

				generationLimit = iterator.GetMinimumBillableTime()

				if generationLimit.IsZero() {
					// This should not happen, but if it does, we should skip this phase
					continue
				}
			}

			items, err := iterator.Generate(ctx, generationLimit)
			if err != nil {
				return collectUpcomingLinesResult{}, fmt.Errorf("generating items: %w", err)
			}

			if maxGenerationTimeLimit.Before(generationLimit) {
				maxGenerationTimeLimit = generationLimit
			}

			inScopeLines = append(inScopeLines, items...)

			if phaseEnd := iterator.PhaseEnd(); phaseEnd != nil && !phaseEnd.Before(asOf) {
				// we are done with the generation, as the phase end is after the asOf, and we have invoicable items
				break
			}
		}

		return collectUpcomingLinesResult{
			Lines:                              inScopeLines,
			SubscriptionMaxGenerationTimeLimit: maxGenerationTimeLimit,
		}, nil
	})
}

func (s *Service) lineFromSubscritionRateCard(subs subscription.SubscriptionView, item subscriptionItemWithPeriods, currency currencyx.Calculator) (*billing.StandardLine, error) {
	line := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   subs.Subscription.Namespace,
				Name:        item.Spec.RateCard.AsMeta().Name,
				Description: item.Spec.RateCard.AsMeta().Description,
			}),
			ManagedBy:              billing.SubscriptionManagedLine,
			Currency:               subs.Spec.Currency,
			ChildUniqueReferenceID: &item.UniqueID,
			TaxConfig:              item.Spec.RateCard.AsMeta().TaxConfig,
			Period:                 item.ServicePeriod,
			InvoiceAt:              item.GetInvoiceAt(),
			RateCardDiscounts:      s.discountsToBillingDiscounts(item.Spec.RateCard.AsMeta().Discounts),

			Subscription: &billing.SubscriptionReference{
				SubscriptionID: subs.Subscription.ID,
				PhaseID:        item.PhaseID,
				ItemID:         item.SubscriptionItem.ID,
				BillingPeriod: timeutil.ClosedPeriod{
					From: item.BillingPeriod.Start,
					To:   item.BillingPeriod.End,
				},
			},
		},
	}

	// If we don't know the full service period for in-arrears items, we should wait with generating a line
	if price := item.SubscriptionItem.RateCard.AsMeta().Price; price != nil && price.GetPaymentTerm() == productcatalog.InArrearsPaymentTerm {
		if item.FullServicePeriod.Duration() == time.Duration(0) {
			return nil, nil
		}
	}

	switch item.SubscriptionItem.RateCard.AsMeta().Price.Type() {
	case productcatalog.FlatPriceType:
		price, err := item.SubscriptionItem.RateCard.AsMeta().Price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat: %w", err)
		}

		// TODO[OM-1040]: We should support rounding errors in prorating calculations (such as 1/3 of a dollar is $0.33, 3*$0.33 is $0.99, if we bill
		// $1.00 in three equal pieces we should charge the customer $0.01 as the last split)
		perUnitAmount := currency.RoundToPrecision(price.Amount)
		if !item.ServicePeriod.IsEmpty() && s.shouldProrate(item, subs) {
			perUnitAmount = currency.RoundToPrecision(price.Amount.Mul(item.PeriodPercentage()))
		}

		if perUnitAmount.IsZero() {
			// We don't need to bill the customer for zero amount items (zero amount items are not allowed on the lines
			// either, so we can safely return here)
			return nil, nil
		}

		line.UsageBased = &billing.UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      perUnitAmount,
				PaymentTerm: price.PaymentTerm,
			}),
			FeatureKey: lo.FromPtr(item.SubscriptionItem.RateCard.AsMeta().FeatureKey),
		}

	default:
		if item.SubscriptionItem.RateCard.AsMeta().Price == nil {
			return nil, fmt.Errorf("price must be defined for usage based price")
		}

		line.UsageBased = &billing.UsageBasedLine{
			Price:      item.SubscriptionItem.RateCard.AsMeta().Price,
			FeatureKey: *item.SubscriptionItem.RateCard.AsMeta().FeatureKey,
		}
	}

	return line, nil
}

func (s *Service) discountsToBillingDiscounts(discounts productcatalog.Discounts) billing.Discounts {
	out := billing.Discounts{}

	if discounts.Usage != nil {
		out.Usage = &billing.UsageDiscount{
			UsageDiscount: *discounts.Usage,
		}
	}

	if discounts.Percentage != nil {
		out.Percentage = &billing.PercentageDiscount{
			PercentageDiscount: *discounts.Percentage,
		}
	}

	return out
}

func (s *Service) shouldProrate(item subscriptionItemWithPeriods, subView subscription.SubscriptionView) bool {
	if !subView.Subscription.ProRatingConfig.Enabled {
		return false
	}

	// We only prorate flat prices
	if item.Spec.RateCard.AsMeta().Price.Type() != productcatalog.FlatPriceType {
		return false
	}

	// We do not prorate due to the subscription ending
	if subView.Subscription.ActiveTo != nil && !subView.Subscription.ActiveTo.After(item.ServicePeriod.End) {
		return false
	}

	// We're just gonna prorate all flat prices based on subscription settings
	switch subView.Subscription.ProRatingConfig.Mode {
	case productcatalog.ProRatingModeProratePrices:
		return true
	default:
		return false
	}
}

func (s *Service) getNewUpcomingLinePatches(ctx context.Context, subs subscription.SubscriptionView, currency currencyx.Calculator, subsItems []subscriptionItemWithPeriods) ([]linePatch, error) {
	newLines, err := slicesx.MapWithErr(subsItems, func(subsItem subscriptionItemWithPeriods) (*billing.StandardLine, error) {
		line, err := s.lineFromSubscritionRateCard(subs, subsItem, currency)
		if err != nil {
			return nil, fmt.Errorf("generating line from subscription item [%s]: %w", subsItem.SubscriptionItem.ID, err)
		}

		return line, nil
	})
	if err != nil {
		return nil, fmt.Errorf("creating new lines: %w", err)
	}

	lines := lo.Filter(newLines, func(l *billing.StandardLine, _ int) bool {
		return l != nil
	})

	return lo.Map(lines, func(l *billing.StandardLine, _ int) linePatch {
		return newCreateLinePatch(*l)
	}), nil
}

func (s *Service) getPatchesForExistingLineOrHierarchy(existingLine billing.LineOrHierarchy, expectedLine *billing.StandardLine, invoiceByID InvoiceByID) ([]linePatch, error) {
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
		line, err := existingLine.AsStandardLine()
		if err != nil {
			return nil, fmt.Errorf("getting line: %w", err)
		}

		return s.getPatchesForExistingLine(line, expectedLine, invoiceByID)
	case billing.LineOrHierarchyTypeHierarchy:
		group, err := existingLine.AsHierarchy()
		if err != nil {
			return nil, fmt.Errorf("getting hierarchy: %w", err)
		}

		return s.getPatchesForExistingHierarchy(group, expectedLine, invoiceByID)
	default:
		return nil, fmt.Errorf("unsupported line or hierarchy type: %s", existingLine.Type())
	}
}

func (s *Service) getPatchesForExistingLine(existingLine *billing.StandardLine, expectedLine *billing.StandardLine, invoiceByID InvoiceByID) ([]linePatch, error) {
	// Lines can be manually marked as ignored in syncing, which is used for cases where we're doing backwards incompatible changes
	if expectedLine.Annotations.GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return nil, nil
	}

	if existingLine.Annotations.GetBool(billing.AnnotationSubscriptionSyncIgnore) {
		return nil, nil
	}

	// Manual edits prevent resyncronization so that we preserve the user intent
	if existingLine.ManagedBy != billing.SubscriptionManagedLine {
		return nil, nil
	}

	// We assume that only the period can change, maybe some pricing data due to prorating (for flat lines)

	targetLine := existingLine.CloneWithoutChildren()

	wasChange := false

	setIfDoesNotEqual(&targetLine.Period, expectedLine.Period, &wasChange)

	if invoiceByID.IsGatheringInvoice(existingLine.InvoiceID) {
		setIfDoesNotEqual(&targetLine.InvoiceAt, expectedLine.InvoiceAt, &wasChange)
	}

	if !isFlatFee(targetLine) {
		// UBP Empty lines are not allowed, let's delete them instead
		if targetLine.Period.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
			return []linePatch{
				newDeleteLinePatch(existingLine.LineID(), existingLine.InvoiceID),
			}, nil
		}
	}

	if targetLine.DeletedAt != nil {
		targetLine.DeletedAt = nil
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

func (s *Service) getPatchesForExistingHierarchy(existingHierarchy *billing.SplitLineHierarchy, expectedLine *billing.StandardLine, invoiceByID InvoiceByID) ([]linePatch, error) {
	// Parts of the line has been already invoiced using progressive invoicing, so we need to examine the children

	// Nothing to do here, as split lines are UBP lines and thus we don't need the flat fee corrections
	// TODO[later]: When we implement progressive billing based pro-rating, we need to support adjusting flat fee
	// segments here.

	if existingHierarchy.Group.ServicePeriod.End.Equal(expectedLine.Period.End) {
		// The line is already in the expected state, so we can safely return here
		return nil, nil
	}

	patches := []linePatch{}

	// Case #1: The line is being expanded (e.g. continue subscription)
	if existingHierarchy.Group.ServicePeriod.End.Before(expectedLine.Period.End) {
		// Expansion of the line (e.g. continue subscription)

		children := existingHierarchy.Lines
		if len(children) > 0 {
			slices.SortFunc(children, func(i, j billing.LineWithInvoiceHeader) int {
				return timeutil.Compare(i.Line.Period.End, j.Line.Period.End)
			})

			lastChild := children[len(children)-1].Line.CloneWithoutChildren()

			if lastChild.ManagedBy == billing.SubscriptionManagedLine {
				// We are not supporting period changes for children, and we need to maintain the consistency so
				// even for overridden lines we need to update the period

				// We however allow deletions, so we are only un-deleting the line here if it was deleted by the sync engine
				lastChild.DeletedAt = nil
			}

			lastChild.Period.End = expectedLine.Period.End

			if invoiceByID.IsGatheringInvoice(lastChild.InvoiceID) {
				lastChild.InvoiceAt = expectedLine.InvoiceAt
			}
			patches = append(patches, newUpdateLinePatch(lastChild))
		}

		// We have already updated the last child, so we need to update at least the periods regardless of managed_by to keep the consistency
		updatedGroup := existingHierarchy.Group.ToUpdate()
		updatedGroup.ServicePeriod.End = expectedLine.Period.End
		patches = append(patches, newUpdateSplitLineGroupPatch(updatedGroup))

		return patches, nil
	}

	// Shrink of the line (e.g. subscription cancled, subscription item edit)

	for _, child := range existingHierarchy.Lines {
		if child.Line.Period.End.Before(expectedLine.Period.End) {
			// The child is not affected by the period shrink, so we can skip it
			continue
		}

		if child.Line.Period.Start.After(expectedLine.Period.End) {
			// The child is after the period shrink, so we need to delete it as it became invalid
			patches = append(patches, newDeleteLinePatch(child.Line.LineID(), child.Line.InvoiceID))
			continue
		}

		// The child is partially affected by the period shrink, so we need to adjust the period
		if !child.Line.Period.End.Equal(expectedLine.Period.End) {
			updatedChild := child.Line.CloneWithoutChildren()
			updatedChild.Period.End = expectedLine.Period.End

			if invoiceByID.IsGatheringInvoice(child.Line.InvoiceID) {
				updatedChild.InvoiceAt = expectedLine.InvoiceAt
			}

			if child.Line.ManagedBy == billing.SubscriptionManagedLine {
				updatedChild.DeletedAt = nil
			}

			if !isFlatFee(updatedChild) {
				// UBP Empty lines are not allowed, let's delete them instead
				if updatedChild.Period.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
					patches = append(patches, newDeleteLinePatch(child.Line.LineID(), child.Line.InvoiceID))
					continue
				}
			}

			patches = append(patches, newUpdateLinePatch(updatedChild))
		}
	}

	// Let's make sure that the group's end is updated to the expected period end, so that
	// we can reliably detect last child

	updatedGroup := existingHierarchy.Group.ToUpdate()
	updatedGroup.ServicePeriod.End = expectedLine.Period.End
	patches = append(patches, newUpdateSplitLineGroupPatch(updatedGroup))

	return patches, nil
}
