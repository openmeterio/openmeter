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
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
	ChargesService charges.Service
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
	chargesService charges.Service
	logger         *slog.Logger

	invoiceUpdater *invoiceupdater.Updater
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		billingService: config.BillingService,
		logger:         config.Logger,
		invoiceUpdater: invoiceupdater.New(config.BillingService, config.Logger),
		chargesService: config.ChargesService,
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
	ChargePatches                      charges.ApplyPatchesInput
	Invoices                           persistedstate.Invoices
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (p *Plan) IsEmpty() bool {
	if p == nil {
		return true
	}

	return len(p.InvoicePatches) == 0 && p.ChargePatches.IsEmpty()
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

	// Charge-backed targets do not use invoice-style semantic proration. The charge
	// stack materializes and prorates the charge state itself, so reconciliation only
	// needs to detect create/delete/period-shape changes here.
	//
	// In case of charges based sync, the flatfee charge is responsible for handling the omission
	// of empty invoice lines.
	if patches.GetBackendType() == BackendTypeInvoicing {
		if decision, err := semanticProrateDecision(existing, *target); err != nil {
			return err
		} else if decision.ShouldProrate {
			// Flat fee lines do not produce usage-based shrink/extend patches. Any period
			// change for a flat fee line is reconciled through ProratePatch so that the
			// service period and per-unit amount are updated together.
			return patches.AddProrate(existing, *target, existingPeriod, targetPeriod, decision.OriginalAmount, decision.TargetAmount)
		}
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

// filterInScopeLines removes target items that should not participate in
// reconciliation. We intentionally do this before diff planning so
// non-billable targets behave as absent targets and naturally reconcile to
// delete/no-op outcomes.
//
// The router decides which backend a target would use if it had to be created.
// Only invoicing-backed targets are gated on GetExpectedLine(): if a target
// does not realize to an invoice line, it should not be diffed as an invoice
// artifact. Charge-backed targets are filtered by billability only.
func filterInScopeLines(inScopeLines []targetstate.StateItem, patchCollections *patchCollectionRouter) ([]targetstate.StateItem, error) {
	out := make([]targetstate.StateItem, 0, len(inScopeLines))

	for _, line := range inScopeLines {
		if !line.IsBillable() {
			continue
		}

		defaultCollection, err := patchCollections.ResolveDefaultCollection(line)
		if err != nil {
			return nil, fmt.Errorf("resolving default patch collection for line[%s]: %w", line.UniqueID, err)
		}

		if defaultCollection.GetBackendType() == BackendTypeCharges {
			out = append(out, line)
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
	if input.Subscription.SettlementMode == productcatalog.CreditOnlySettlementMode && s.chargesService == nil {
		return nil, fmt.Errorf("credit only settlement mode is not supported without charges service enabled")
	}

	patchCollections, err := newPatchCollectionRouter(len(input.Target.Items)+len(input.Persisted.ByUniqueID), input.Persisted.Invoices)
	if err != nil {
		return nil, fmt.Errorf("creating collection by type: %w", err)
	}

	if patchCollections == nil {
		return nil, fmt.Errorf("patchCollectionRouter is nil")
	}

	inScopeLines, err := filterInScopeLines(input.Target.Items, patchCollections)
	if err != nil {
		return nil, fmt.Errorf("filtering in-scope lines: %w", err)
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

	for _, id := range deletedLines {
		line, ok := persisted.ByUniqueID[id]
		if !ok {
			return nil, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}

		patchCollection, err := patchCollections.GetCollectionFor(line)
		if err != nil {
			return nil, fmt.Errorf("getting patch collection for deleted line[%s]: %w", id, err)
		}

		if err := s.diffItem(nil, line, patchCollection); err != nil {
			return nil, fmt.Errorf("diffing deleted line[%s]: %w", id, err)
		}
	}

	for _, id := range inScopeLineUniqueIDs {
		targetLine := inScopeLinesByUniqueID[id]
		existingLine, ok := persisted.ByUniqueID[id]
		if !ok {
			// The line is not in the persisted state, so we need to fall back to the default collection essentially
			// forcing it to be created using the specified collection. This allows us to transition from invoicing based
			// upcoming lines to charges based provisioning in a graceful manner.
			defaultCollection, err := patchCollections.ResolveDefaultCollection(targetLine)
			if err != nil {
				return nil, fmt.Errorf("resolving default patch collection for new line[%s]: %w", id, err)
			}

			if err := s.diffItem(&targetLine, nil, defaultCollection); err != nil {
				return nil, fmt.Errorf("diffing new line[%s]: %w", id, err)
			}
			continue
		}

		patchCollection, err := patchCollections.GetCollectionFor(existingLine)
		if err != nil {
			return nil, fmt.Errorf("getting patch collection for existing line[%s]: %w", id, err)
		}

		if err := s.diffItem(&targetLine, existingLine, patchCollection); err != nil {
			return nil, fmt.Errorf("diffing existing line[%s]: %w", id, err)
		}
	}

	chargePatches, err := patchCollections.CollectChargePatches()
	if err != nil {
		return nil, fmt.Errorf("collecting charge patches: %w", err)
	}

	return &Plan{
		InvoicePatches:                     patchCollections.CollectInvoicePatches(),
		ChargePatches:                      chargePatches,
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

	if input.DryRun {
		s.invoiceUpdater.LogPatches(invoicePatches, input.Plan.Invoices)
		logChargesPatches(ctx, s.logger, input.Plan.ChargePatches)
		return nil
	}

	if err := s.invoiceUpdater.ApplyPatches(ctx, input.Customer, invoicePatches); err != nil {
		return fmt.Errorf("updating invoices: %w", err)
	}

	if !input.Plan.ChargePatches.IsEmpty() {
		if s.chargesService != nil {
			// Let's finalize the input's global fields
			chargePatches := input.Plan.ChargePatches
			chargePatches.CustomerID = input.Customer

			if err := s.chargesService.ApplyPatches(ctx, chargePatches); err != nil {
				return fmt.Errorf("updating charges: %w", err)
			}
		} else {
			return fmt.Errorf("charges service is required when there are charge patches")
		}
	}

	return nil
}
