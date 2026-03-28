package reconciler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

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
	InvoicePatches                     []InvoicePatch
	Invoices                           persistedstate.Invoices
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (p *Plan) IsEmpty() bool {
	if p == nil {
		return true
	}

	return len(p.InvoicePatches) == 0
}

func (s *Service) diffItem(
	target *targetstate.StateItem,
	existing persistedstate.Item,
	patches PatchCollection,
) error {
	switch {
	case target == nil && existing == nil:
		return nil
	case target == nil && existing != nil:
		uniqueID := lo.FromPtr(existing.ChildUniqueReferenceID())
		return patches.AddDelete(uniqueID, existing)
	case target != nil && existing == nil:
		return patches.AddCreate(*target)
	}

	existingPeriod := existing.ServicePeriod()
	targetPeriod := target.GetServicePeriod()

	if decision, err := semanticProrateDecision(existing, *target); err != nil {
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

// filterInScopeLinesForInvoiceSync removes target items that should not
// participate in direct billing reconciliation. We intentionally do this before
// diff planning so non-billable targets behave as absent targets and naturally
// reconcile to delete/no-op outcomes.
//
// We also render GetExpectedLine() here and drop items that do not realize to
// an invoice line. That rendering step is only expected for direct billing
// syncs. Other provisioning backends, such as charges, can keep different
// target realization and proration behavior without going through this filter.
func filterInScopeLinesForInvoiceSync(inScopeLines []targetstate.StateItem) ([]targetstate.StateItem, error) {
	out := make([]targetstate.StateItem, 0, len(inScopeLines))

	for _, line := range inScopeLines {
		if !line.IsBillable() {
			continue
		}

		expectedLine, err := line.GetExpectedLine()
		if err != nil {
			return nil, fmt.Errorf("generating expected line[%s]: %w", line.UniqueID, err)
		}

		if expectedLine == nil {
			continue
		}

		out = append(out, line)
	}

	return out, nil
}

func (s *Service) Plan(ctx context.Context, input PlanInput) (*Plan, error) {
	inScopeLines, err := filterInScopeLinesForInvoiceSync(input.Target.Items)
	if err != nil {
		return nil, fmt.Errorf("filtering in-scope lines for invoice sync: %w", err)
	}
	persisted := input.Persisted

	if len(inScopeLines) == 0 && len(persisted.ByUniqueID) == 0 {
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

	patchCollections, err := newPatchCollectionRouter(len(deletedLines)+len(inScopeLineUniqueIDs), input.Persisted.Invoices)
	if err != nil {
		return nil, fmt.Errorf("creating collection by type: %w", err)
	}

	// TODO: Once we have charges wired in we need a helper function to determine the default routing for new lines depending on the
	// settlement type set on the subscription and feature flags in the config of subscription sync.
	defaultCollection := patchCollections.ResolveDefaultCollection()

	for _, id := range deletedLines {
		line, ok := persisted.ByUniqueID[id]
		if !ok {
			return nil, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}

		if err := s.diffItem(nil, line, patchCollections.GetCollectionFor(line)); err != nil {
			return nil, fmt.Errorf("diffing deleted line[%s]: %w", id, err)
		}
	}

	for _, id := range inScopeLineUniqueIDs {
		targetLine := inScopeLinesByUniqueID[id]
		existingLine, ok := persisted.ByUniqueID[id]
		if !ok {
			// The line is not in the persisted state, so we need to fall back to the default collection esentially
			// forcing it to be created using the specified collection. This allows us to transition from invocing based
			// upcoming lines to charges based provisioning in a graceful manner.

			if err := s.diffItem(&targetLine, nil, defaultCollection); err != nil {
				return nil, fmt.Errorf("diffing new line[%s]: %w", id, err)
			}
			continue
		}

		if err := s.diffItem(&targetLine, existingLine, patchCollections.GetCollectionFor(existingLine)); err != nil {
			return nil, fmt.Errorf("diffing existing line[%s]: %w", id, err)
		}
	}

	return &Plan{
		InvoicePatches:                     patchCollections.CollectInvoicePatches(),
		Invoices:                           input.Persisted.Invoices,
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

	patches := input.Plan.InvoicePatches
	invoicePatches := make([]invoiceupdater.Patch, 0, len(patches))

	for _, patch := range patches {
		newInvoicePatches, err := patch.GetInvoicePatches()
		if err != nil {
			return fmt.Errorf("getting invoice patches for patch[%s/%s]: %w", patch.Operation(), patch.UniqueReferenceID(), err)
		}

		invoicePatches = append(invoicePatches, newInvoicePatches...)
	}

	invoiceUpdater := invoiceupdater.New(s.billingService, s.logger)

	if input.DryRun {
		invoiceUpdater.LogPatches(invoicePatches, input.Plan.Invoices)
		return nil
	}

	if err := invoiceUpdater.ApplyPatches(ctx, input.Customer, invoicePatches); err != nil {
		return fmt.Errorf("updating invoices: %w", err)
	}

	return nil
}
