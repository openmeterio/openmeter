package reconciler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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
	NewSubscriptionItems               []targetstate.SubscriptionItemWithPeriods
	LinesToDelete                      []billing.LineOrHierarchy
	LinesToUpsert                      []LineUpsert
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (p *Plan) IsEmpty() bool {
	if p == nil {
		return true
	}

	return len(p.NewSubscriptionItems) == 0 && len(p.LinesToDelete) == 0 && len(p.LinesToUpsert) == 0
}

type LineUpsert struct {
	Target   targetstate.SubscriptionItemWithPeriods
	Existing billing.LineOrHierarchy
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

	return &Plan{
		NewSubscriptionItems: lo.Map(newLines, func(id string, _ int) targetstate.SubscriptionItemWithPeriods {
			return inScopeLinesByUniqueID[id]
		}),
		LinesToDelete:                      linesToDelete,
		LinesToUpsert:                      linesToUpsert,
		SubscriptionMaxGenerationTimeLimit: input.Target.MaxGenerationTimeLimit,
	}, nil
}
