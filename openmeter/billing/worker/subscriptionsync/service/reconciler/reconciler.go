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
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
	// SemanticPatches is the target shape for reconciliation. The legacy line buckets remain
	// in place temporarily while Apply still consumes the older plan format.
	SemanticPatches                    []SemanticPatch
	NewSubscriptionItems               []targetstate.SubscriptionItemWithPeriods
	LinesToDelete                      []billing.LineOrHierarchy
	LinesToUpsert                      []LineUpsert
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (p *Plan) IsEmpty() bool {
	if p == nil {
		return true
	}

	return len(p.SemanticPatches) == 0 && len(p.NewSubscriptionItems) == 0 && len(p.LinesToDelete) == 0 && len(p.LinesToUpsert) == 0
}

type LineUpsert struct {
	Target   targetstate.SubscriptionItemWithPeriods
	Existing billing.LineOrHierarchy
}

type SemanticPatchOperation string

const (
	SemanticPatchOperationCreate  SemanticPatchOperation = "create"
	SemanticPatchOperationDelete  SemanticPatchOperation = "delete"
	SemanticPatchOperationShrink  SemanticPatchOperation = "shrink"
	SemanticPatchOperationExtend  SemanticPatchOperation = "extend"
	SemanticPatchOperationProrate SemanticPatchOperation = "prorate"
)

type SemanticPatch interface {
	semanticPatch()
	Operation() SemanticPatchOperation
	UniqueReferenceID() string
}

type CreatePatch struct {
	UniqueID string
	Target   targetstate.SubscriptionItemWithPeriods
}

func (CreatePatch) semanticPatch() {}

func (p CreatePatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationCreate
}

func (p CreatePatch) UniqueReferenceID() string {
	return p.UniqueID
}

type DeletePatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
}

func (DeletePatch) semanticPatch() {}

func (p DeletePatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationDelete
}

func (p DeletePatch) UniqueReferenceID() string {
	return p.UniqueID
}

type ShrinkPatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods
}

func (ShrinkPatch) semanticPatch() {}

func (p ShrinkPatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationShrink
}

func (p ShrinkPatch) UniqueReferenceID() string {
	return p.UniqueID
}

type ExtendPatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods
}

func (ExtendPatch) semanticPatch() {}

func (p ExtendPatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationExtend
}

func (p ExtendPatch) UniqueReferenceID() string {
	return p.UniqueID
}

type ProratePatch struct {
	UniqueID string
	Existing billing.LineOrHierarchy
	Target   targetstate.SubscriptionItemWithPeriods

	OriginalPeriod timeutil.ClosedPeriod
	TargetPeriod   timeutil.ClosedPeriod

	OriginalAmount alpacadecimal.Decimal
	TargetAmount   alpacadecimal.Decimal
}

func (ProratePatch) semanticPatch() {}

func (p ProratePatch) Operation() SemanticPatchOperation {
	return SemanticPatchOperationProrate
}

func (p ProratePatch) UniqueReferenceID() string {
	return p.UniqueID
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

	linesToUpsert, err := slicesx.MapWithErr(lineIDsToUpsert, func(id string) (LineUpsert, error) {
		existingLine, ok := persisted.ByUniqueID[id]
		if !ok {
			return LineUpsert{}, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}
		return LineUpsert{
			Target:   inScopeLinesByUniqueID[id],
			Existing: existingLine,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("mapping upsert lines: %w", err)
	}

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
		NewSubscriptionItems: lo.Map(newLines, func(id string, _ int) targetstate.SubscriptionItemWithPeriods {
			return inScopeLinesByUniqueID[id]
		}),
		LinesToDelete:                      linesToDelete,
		LinesToUpsert:                      linesToUpsert,
		SubscriptionMaxGenerationTimeLimit: input.Target.MaxGenerationTimeLimit,
	}, nil
}
