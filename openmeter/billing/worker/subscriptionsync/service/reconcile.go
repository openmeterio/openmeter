package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type subscriptionReconciliationPlan struct {
	NewSubscriptionItems               []targetstate.SubscriptionItemWithPeriods
	LinesToDelete                      []billing.LineOrHierarchy
	LinesToUpsert                      []subscriptionReconciliationLineUpsert
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (s *subscriptionReconciliationPlan) IsEmpty() bool {
	if s == nil {
		return true
	}

	return len(s.NewSubscriptionItems) == 0 && len(s.LinesToDelete) == 0 && len(s.LinesToUpsert) == 0
}

type subscriptionReconciliationLineUpsert struct {
	Target   targetstate.SubscriptionItemWithPeriods
	Existing billing.LineOrHierarchy
}

func (s *Service) buildSyncPlan(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) (*subscriptionReconciliationPlan, error) {
	span := tracex.Start[*subscriptionReconciliationPlan](ctx, s.tracer, "billing.worker.subscription.sync.buildSyncPlan")

	return span.Wrap(func(ctx context.Context) (*subscriptionReconciliationPlan, error) {
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

		return s.reconcileSubscription(target, persisted)
	})
}

func (s *Service) reconcileSubscription(target targetstate.State, persisted persistedstate.State) (*subscriptionReconciliationPlan, error) {
	inScopeLines := target.Items

	if len(inScopeLines) == 0 && len(persisted.Lines) == 0 {
		return &subscriptionReconciliationPlan{
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

	linesToUpsert, err := slicesx.MapWithErr(lineIDsToUpsert, func(id string) (subscriptionReconciliationLineUpsert, error) {
		existingLine, ok := persisted.ByUniqueID[id]
		if !ok {
			return subscriptionReconciliationLineUpsert{}, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}

		return subscriptionReconciliationLineUpsert{
			Target:   inScopeLinesByUniqueID[id],
			Existing: existingLine,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("mapping upsert lines: %w", err)
	}

	return &subscriptionReconciliationPlan{
		NewSubscriptionItems: lo.Map(newLines, func(id string, _ int) targetstate.SubscriptionItemWithPeriods {
			return inScopeLinesByUniqueID[id]
		}),
		LinesToDelete:                      linesToDelete,
		LinesToUpsert:                      linesToUpsert,
		SubscriptionMaxGenerationTimeLimit: target.MaxGenerationTimeLimit,
	}, nil
}
