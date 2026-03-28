package reconciler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
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
	Subscription subscription.Subscription
	Currency     currencyx.Calculator
	Target       targetstate.State
	Persisted    persistedstate.State
}

type ApplyInput struct {
	DryRun       bool
	Customer     customer.CustomerID
	Subscription subscription.Subscription
	Currency     currencyx.Calculator
	Invoices     persistedstate.Invoices
	Plan         *Plan
}

func (i ApplyInput) Validate() error {
	var errs []error
	if i.Plan == nil {
		errs = append(errs, fmt.Errorf("plan is required"))
	}
	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := i.Subscription.NamespacedID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("subscription namespaced id: %w", err))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	return errors.Join(errs...)
}

type Plan struct {
	PatchCollection                    InvoicePatchCollection
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (p *Plan) IsEmpty() bool {
	if p == nil {
		return true
	}

	if p.PatchCollection == nil {
		return true
	}

	return p.PatchCollection.IsEmpty()
}

func (s *Service) diffItem(
	target *targetstate.StateItem,
	expectedLine *billing.GatheringLine, // TODO[later]: let's merge this with target as they are the same thing's different calculation stages
	existing persistedstate.Item,
	patches PatchCollection,
) error {
	switch {
	case target == nil && existing == nil:
		return nil
	case target == nil && existing != nil:
		uniqueID := lo.FromPtr(existing.ChildUniqueReferenceID())
		return patches.AddDelete(uniqueID, existing)
	case target != nil && existing == nil && expectedLine != nil:
		patches.AddCreate(*target)
		return nil
	case target != nil && existing == nil && expectedLine == nil:
		// If the target is not nil, but the expected line is nil, we should not create a patch (most probably
		// because the line is ignored or empty service period)
		return nil
	case target != nil && existing != nil && expectedLine == nil:
		return patches.AddDelete(target.UniqueID, existing)
	}

	existingPeriod := existing.ServicePeriod()
	targetPeriod := expectedLine.ServicePeriod

	if decision, err := semanticProrateDecision(existing, *expectedLine); err != nil {
		return err
	} else if decision.ShouldProrate {
		// Flat fee lines do not produce usage-based shrink/extend patches. Any period
		// change for a flat fee line is reconciled through ProratePatch so that the
		// service period and per-unit amount are updated together.
		return patches.AddProrate(existing, *target, existingPeriod, targetPeriod, decision.OriginalAmount, decision.TargetAmount)
	}

	switch {
	case targetPeriod.To.Before(existingPeriod.To):
		return patches.AddShrink(target.UniqueID, existing, *target)
	case targetPeriod.To.After(existingPeriod.To):
		return patches.AddExtend(existing, *target)
	default:
		return nil
	}
}

type ProrateDecision struct {
	ShouldProrate  bool
	OriginalAmount alpacadecimal.Decimal
	TargetAmount   alpacadecimal.Decimal
}

func semanticProrateDecision(existing persistedstate.Item, expectedLine billing.GatheringLine) (ProrateDecision, error) {
	if !invoiceupdater.IsFlatFee(expectedLine) {
		return ProrateDecision{}, nil
	}

	// expectedLine is materialized through targetstate.LineFromSubscriptionRateCard, which
	// applies the existing subscription-sync proration rules when deriving the flat-fee amount.
	targetAmount, err := invoiceupdater.GetFlatFeePerUnitAmount(expectedLine)
	if err != nil {
		return ProrateDecision{}, fmt.Errorf("getting expected flat fee amount: %w", err)
	}

	switch existing.Type() {
	case billing.LineOrHierarchyTypeLine:
		lineGetter, ok := existing.(persistedstate.LineGetter)
		if !ok {
			return ProrateDecision{}, fmt.Errorf("persisted item does not implement line getter: %T", existing)
		}

		existingLine := lineGetter.GetLine()

		if !invoiceupdater.IsFlatFee(existingLine) {
			return ProrateDecision{}, nil
		}

		existingAmount, err := invoiceupdater.GetFlatFeePerUnitAmount(existingLine)
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

func persistedItemAsLineOrHierarchy(item persistedstate.Item) (billing.LineOrHierarchy, error) {
	switch item.Type() {
	case billing.LineOrHierarchyTypeLine:
		lineGetter, ok := item.(persistedstate.LineGetter)
		if !ok {
			return billing.LineOrHierarchy{}, fmt.Errorf("persisted item does not implement line getter: %T", item)
		}

		line := lineGetter.GetLine()
		invoiceLine := line.AsInvoiceLine()
		switch invoiceLine.Type() {
		case billing.InvoiceLineTypeStandard:
			standardLine, err := invoiceLine.AsStandardLine()
			if err != nil {
				return billing.LineOrHierarchy{}, fmt.Errorf("getting standard line: %w", err)
			}

			return billing.NewLineOrHierarchy(&standardLine), nil
		case billing.InvoiceLineTypeGathering:
			gatheringLine, err := invoiceLine.AsGatheringLine()
			if err != nil {
				return billing.LineOrHierarchy{}, fmt.Errorf("getting gathering line: %w", err)
			}

			return billing.NewLineOrHierarchy(gatheringLine), nil
		default:
			return billing.LineOrHierarchy{}, fmt.Errorf("unsupported invoice line type: %s", invoiceLine.Type())
		}
	case billing.LineOrHierarchyTypeHierarchy:
		hierarchyGetter, ok := item.(persistedstate.SplitLineHierarchyGetter)
		if !ok {
			return billing.LineOrHierarchy{}, fmt.Errorf("persisted item does not implement split line hierarchy getter: %T", item)
		}

		return billing.NewLineOrHierarchy(hierarchyGetter.GetSplitLineHierarchy()), nil
	default:
		return billing.LineOrHierarchy{}, fmt.Errorf("unsupported item type: %s", item.Type())
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

	inScopeLinesByUniqueID, unique := slicesx.UniqueGroupBy(inScopeLines, func(i targetstate.StateItem) string {
		return i.UniqueID
	})
	if !unique {
		return nil, fmt.Errorf("duplicate unique ids in the upcoming lines")
	}

	existingLineUniqueIDs := lo.Keys(persisted.ByUniqueID)
	inScopeLineUniqueIDs := lo.Keys(inScopeLinesByUniqueID)
	deletedLines, _ := lo.Difference(existingLineUniqueIDs, inScopeLineUniqueIDs)
	// Keep patch planning order stable between runs for logging, debugging and
	// testability. Downstream application does not depend on this ordering.
	slices.Sort(deletedLines)
	slices.Sort(inScopeLineUniqueIDs)

	patches := newInvoicePatchCollection(len(deletedLines) + len(inScopeLineUniqueIDs))

	for _, id := range deletedLines {
		line, ok := persisted.ByUniqueID[id]
		if !ok {
			return nil, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}

		if err := s.diffItem(nil, nil, line, patches); err != nil {
			return nil, fmt.Errorf("diffing deleted line[%s]: %w", id, err)
		}
	}

	for _, id := range inScopeLineUniqueIDs {
		targetLine := inScopeLinesByUniqueID[id]
		expectedLine, err := targetLine.GetExpectedLine()
		if err != nil {
			return nil, fmt.Errorf("generating expected line[%s]: %w", id, err)
		}

		existingLine, ok := persisted.ByUniqueID[id]
		if !ok {
			if err := s.diffItem(&targetLine, expectedLine, nil, patches); err != nil {
				return nil, fmt.Errorf("diffing new line[%s]: %w", id, err)
			}
			continue
		}

		if err := s.diffItem(&targetLine, expectedLine, existingLine, patches); err != nil {
			return nil, fmt.Errorf("diffing existing line[%s]: %w", id, err)
		}
	}

	return &Plan{
		PatchCollection:                    filterInvoicePatchCollection(patches),
		SubscriptionMaxGenerationTimeLimit: input.Target.MaxGenerationTimeLimit,
	}, nil
}

func (s *Service) Apply(ctx context.Context, input ApplyInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	if input.Plan == nil || input.Plan.IsEmpty() {
		return nil
	}

	patches := input.Plan.PatchCollection.Patches()
	invoicePatches := make([]invoiceupdater.Patch, 0, len(patches))

	for _, patch := range patches {
		newInvoicePatches, err := patch.GetInvoicePatches(GetInvoicePatchesInput{
			Subscription: input.Subscription,
			Currency:     input.Currency,
			Invoices:     input.Invoices,
		})
		if err != nil {
			return fmt.Errorf("getting invoice patches for patch[%s/%s]: %w", patch.Operation(), patch.UniqueReferenceID(), err)
		}

		invoicePatches = append(invoicePatches, newInvoicePatches...)
	}

	invoiceUpdater := invoiceupdater.New(s.billingService, s.logger)

	if input.DryRun {
		invoiceUpdater.LogPatches(invoicePatches, input.Invoices.ByID)
		return nil
	}

	if err := invoiceUpdater.ApplyPatches(ctx, input.Customer, invoicePatches); err != nil {
		return fmt.Errorf("updating invoices: %w", err)
	}

	return nil
}

func filterInvoicePatchCollection(collection *invoicePatchCollection) InvoicePatchCollection {
	if collection == nil {
		return nil
	}

	filtered := make([]InvoicePatch, 0, len(collection.patches))
	for _, patch := range collection.patches {
		if patch != nil {
			filtered = append(filtered, patch)
		}
	}

	return &invoicePatchCollection{
		patches: filtered,
	}
}
