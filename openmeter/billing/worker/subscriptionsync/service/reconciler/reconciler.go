package reconciler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type Reconciler interface {
	Plan(ctx context.Context, input PlanInput) (*Plan, error)
	Apply(ctx context.Context, input ApplyInput) error
}

type Config struct {
	BillingService billing.Service
	Logger         *slog.Logger
}

func (c Config) Validate() error {
	if c.BillingService == nil {
		return fmt.Errorf("billing service is required")
	}
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	return nil
}

type Service struct {
	billingService billing.Service
	logger         *slog.Logger
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		billingService: config.BillingService,
		logger:         config.Logger,
	}, nil
}

type PlanInput struct {
	Subscription subscription.SubscriptionView
	Currency     currencyx.Calculator
	Target       targetstate.State
	Persisted    persistedstate.State
}

type ApplyInput struct {
	Customer     customer.CustomerID
	Subscription subscription.SubscriptionView
	Currency     currencyx.Calculator
	Invoices     persistedstate.Invoices
	Plan         *Plan
}

type Plan struct {
	SemanticPatches                    []SemanticPatch
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (p *Plan) IsEmpty() bool {
	if p == nil {
		return true
	}

	return len(p.SemanticPatches) == 0
}

func (s *Service) diffItem(
	target *targetstate.SubscriptionItemWithPeriods,
	expectedLine *billing.GatheringLine, // TODO[later]: let's merge this with target as they are the same thing's different calculation stages
	existing *billing.LineOrHierarchy,
) (SemanticPatch, bool, error) {
	switch {
	case target == nil && existing == nil:
		return nil, false, nil
	case target == nil && existing != nil:
		uniqueID := lo.FromPtr(existing.ChildUniqueReferenceID())
		return DeletePatch{
			UniqueID: uniqueID,
			Existing: *existing,
		}, true, nil
	case target != nil && existing == nil:
		return CreatePatch{
			UniqueID: target.UniqueID,
			Target:   *target,
		}, true, nil
	case target != nil && existing != nil && expectedLine == nil:
		return DeletePatch{
			UniqueID: target.UniqueID,
			Existing: *existing,
		}, true, nil
	}

	existingPeriod := existing.ServicePeriod()
	targetPeriod := expectedLine.ServicePeriod

	if decision, err := semanticProrateDecision(*existing, *expectedLine); err != nil {
		return nil, false, err
	} else if decision.ShouldProrate {
		return ProratePatch{
			UniqueID:       target.UniqueID,
			Existing:       *existing,
			Target:         *target,
			OriginalPeriod: existingPeriod,
			TargetPeriod:   targetPeriod,
			OriginalAmount: decision.OriginalAmount,
			TargetAmount:   decision.TargetAmount,
		}, true, nil
	}

	switch {
	case targetPeriod.To.Before(existingPeriod.To):
		return ShrinkPatch{
			UniqueID: target.UniqueID,
			Existing: *existing,
			Target:   *target,
		}, true, nil
	case targetPeriod.To.After(existingPeriod.To):
		return ExtendPatch{
			UniqueID: target.UniqueID,
			Existing: *existing,
			Target:   *target,
		}, true, nil
	default:
		return nil, false, nil
	}
}

type ProrateDecision struct {
	ShouldProrate  bool
	OriginalAmount alpacadecimal.Decimal
	TargetAmount   alpacadecimal.Decimal
}

func semanticProrateDecision(existing billing.LineOrHierarchy, expectedLine billing.GatheringLine) (ProrateDecision, error) {
	if !IsFlatFee(expectedLine) {
		return ProrateDecision{}, nil
	}

	// expectedLine is materialized through targetstate.LineFromSubscriptionRateCard, which
	// applies the existing subscription-sync proration rules when deriving the flat-fee amount.
	targetAmount, err := GetFlatFeePerUnitAmount(expectedLine)
	if err != nil {
		return ProrateDecision{}, fmt.Errorf("getting expected flat fee amount: %w", err)
	}

	switch existing.Type() {
	case billing.LineOrHierarchyTypeLine:
		existingLine, err := existing.AsGenericLine()
		if err != nil {
			return ProrateDecision{}, fmt.Errorf("getting existing line: %w", err)
		}

		if !IsFlatFee(existingLine) {
			return ProrateDecision{}, nil
		}

		existingAmount, err := GetFlatFeePerUnitAmount(existingLine)
		if err != nil {
			return ProrateDecision{}, fmt.Errorf("getting existing flat fee amount: %w", err)
		}

		return ProrateDecision{
			ShouldProrate:  !existingAmount.Equal(targetAmount) || !existingLine.GetServicePeriod().Equal(expectedLine.ServicePeriod),
			OriginalAmount: existingAmount,
			TargetAmount:   targetAmount,
		}, nil
	case billing.LineOrHierarchyTypeHierarchy:
		return ProrateDecision{}, errors.New("flat fee lines cannot be reconciled against a split line hierarchy")
	default:
		return ProrateDecision{}, fmt.Errorf("unsupported line or hierarchy type: %s", existing.Type())
	}
}

func (s *Service) Plan(ctx context.Context, input PlanInput) (*Plan, error) {
	inScopeLines := input.Target.Items
	persisted := input.Persisted

	if len(inScopeLines) == 0 && len(persisted.Lines) == 0 {
		return &Plan{
			SubscriptionMaxGenerationTimeLimit: input.Target.MaxGenerationTimeLimit,
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
	deletedLines, _ := lo.Difference(existingLineUniqueIDs, inScopeLineUniqueIDs)

	semanticPatches := make([]SemanticPatch, 0, len(deletedLines)+len(inScopeLineUniqueIDs))

	for _, id := range deletedLines {
		line, ok := persisted.ByUniqueID[id]
		if !ok {
			return nil, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}

		patch, changed, err := s.diffItem(nil, nil, &line)
		if err != nil {
			return nil, fmt.Errorf("diffing deleted line[%s]: %w", id, err)
		}
		if changed {
			semanticPatches = append(semanticPatches, patch)
		}
	}

	for _, id := range inScopeLineUniqueIDs {
		targetLine := inScopeLinesByUniqueID[id]
		// TODO: make this a member of the targetstate.SubscriptionItemWithPeriods
		expectedLine, err := targetstate.LineFromSubscriptionRateCard(input.Subscription, targetLine, input.Currency)
		if err != nil {
			return nil, fmt.Errorf("generating expected line[%s]: %w", id, err)
		}

		existingLine, ok := persisted.ByUniqueID[id]
		if !ok {
			patch, changed, err := s.diffItem(&targetLine, expectedLine, nil)
			if err != nil {
				return nil, fmt.Errorf("diffing new line[%s]: %w", id, err)
			}
			if changed {
				semanticPatches = append(semanticPatches, patch)
			}
			continue
		}

		patch, changed, err := s.diffItem(&targetLine, expectedLine, &existingLine)
		if err != nil {
			return nil, fmt.Errorf("diffing existing line[%s]: %w", id, err)
		}
		if changed {
			semanticPatches = append(semanticPatches, patch)
		}
	}

	return &Plan{
		SemanticPatches: lo.Filter(semanticPatches, func(p SemanticPatch, _ int) bool {
			return p != nil
		}),
		SubscriptionMaxGenerationTimeLimit: input.Target.MaxGenerationTimeLimit,
	}, nil
}
