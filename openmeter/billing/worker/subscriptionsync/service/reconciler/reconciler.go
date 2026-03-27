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
	Patches                            []Patch
	SubscriptionMaxGenerationTimeLimit time.Time
}

func (p *Plan) IsEmpty() bool {
	if p == nil {
		return true
	}

	return len(p.Patches) == 0
}

type diffItemResult struct {
	Patch   Patch
	Changed bool
}

func (s *Service) diffItem(
	target *targetstate.SubscriptionItemWithPeriods,
	existing persistedstate.Entity,
) (Patch, error) {
	switch {
	case target == nil && existing == nil:
		return nil, nil
	case target == nil && existing != nil:
		return s.NewDeletePatch(existing)
	case target != nil && existing == nil && target.IsBillable():
		return s.NewCreatePatch(NewCreatePatchInput{
			UniqueID: target.UniqueID,
			Target:   *target,
		})
	case target != nil && existing == nil && expectedLine == nil:
		// If the target is not nil, but the expected line is nil, we should not create a patch (most probably
		// because the line is ignored or empty service period)
		return nil, nil
	case target != nil && existing != nil && expectedLine == nil:
		return s.NewDeletePatch(existing)
	}

	existingPeriod := existing.GetServicePeriod()
	targetPeriod := expectedLine.ServicePeriod

	if shouldProrateDecision, err := semanticProrateDecision(existing, *expectedLine); err != nil {
		return nil, err
	} else if shouldProrateDecision {
		// Flat fee lines do not produce usage-based shrink/extend patches. Any period
		// change for a flat fee line is reconciled through ProratePatch so that the
		// service period and per-unit amount are updated together.
		return ProratePatch{
			UniqueID:       target.UniqueID,
			Existing:       existing,
			Target:         *target,
			OriginalPeriod: existingPeriod,
			TargetPeriod:   targetPeriod,
			OriginalAmount: decision.OriginalAmount,
			TargetAmount:   decision.TargetAmount,
		}, nil
	}

	switch {
	case targetPeriod.To.Before(existingPeriod.To):
		return s.NewLineShrinkUsageBasedPatch(NewLineShrinkUsageBasedPatchInput{
			Existing: existing,
			Target:   *target,
		})
	case targetPeriod.To.After(existingPeriod.To):
		return s.NewLineExtendUsageBasedPatch(NewLineExtendUsageBasedPatchInput{
			Existing: existing,
			Target:   *target,
		})
	default:
		return nil, nil
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
	// Keep patch planning order stable between runs for logging, debugging and
	// testability. Downstream application does not depend on this ordering.
	slices.Sort(deletedLines)
	slices.Sort(inScopeLineUniqueIDs)

	patches := make([]Patch, 0, len(deletedLines)+len(inScopeLineUniqueIDs))

	for _, id := range deletedLines {
		line, ok := persisted.ByUniqueID[id]
		if !ok {
			return nil, fmt.Errorf("existing line[%s] not found in the existing lines", id)
		}

		patch, err := s.diffItem(nil, nil, line)
		if err != nil {
			return nil, fmt.Errorf("diffing deleted line[%s]: %w", id, err)
		}
		if patch != nil {
			patches = append(patches, patch)
		}
	}

	for _, id := range inScopeLineUniqueIDs {
		targetLine := inScopeLinesByUniqueID[id]
		expectedLine, err := targetLine.GetExpectedLine(input.Subscription, input.Currency)
		if err != nil {
			return nil, fmt.Errorf("generating expected line[%s]: %w", id, err)
		}

		existingLine, ok := persisted.ByUniqueID[id]
		if !ok {
			patch, err := s.diffItem(&targetLine, expectedLine, nil)
			if err != nil {
				return nil, fmt.Errorf("diffing new line[%s]: %w", id, err)
			}
			if patch != nil {
				patches = append(patches, patch)
			}
			continue
		}

		patch, err := s.diffItem(&targetLine, expectedLine, existingLine)
		if err != nil {
			return nil, fmt.Errorf("diffing existing line[%s]: %w", id, err)
		}
		if patch != nil {
			patches = append(patches, patch)
		}
	}

	filteredPatches := lo.Filter(patches, func(p Patch, _ int) bool {
		return p != nil
	})

	if err := s.validatePatches(filteredPatches); err != nil {
		return nil, fmt.Errorf("validating patches: %w", err)
	}

	return &Plan{
		Patches:                            filteredPatches,
		SubscriptionMaxGenerationTimeLimit: input.Target.MaxGenerationTimeLimit,
	}, nil
}

func (s *Service) validatePatches(patches []Patch) error {
	for _, patch := range patches {
		if patch == nil {
			return fmt.Errorf("patch is nil")
		}

		// Let's mandate that all patches are implementing either InvoicePatch or ChargePatch.
		_, isInvoicePatch := patch.(InvoicePatch)
		_, isChargePatch := patch.(ChargePatch)

		// TODO: let's decide later if we want to support mixed invoice and charge patches.

		if !isInvoicePatch && !isChargePatch {
			return fmt.Errorf("patch is not an invoice or charge patch: %T", patch)
		}
	}
	return nil
}

func (s *Service) Apply(ctx context.Context, input ApplyInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	if input.Plan == nil || input.Plan.IsEmpty() {
		return nil
	}

	// TODO: Let's validate that patches are either invoice or charge patches.

	invoicePatches := make([]invoiceupdater.Patch, 0, len(input.Plan.Patches))

	for _, patch := range input.Plan.Patches {
		invoicePatch, ok := patch.(InvoicePatch)
		if !ok {
			continue
		}

		newInvoicePatches, err := invoicePatch.GetInvoicePatches(GetInvoicePatchesInput{
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
