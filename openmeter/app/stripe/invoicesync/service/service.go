package invoicesyncservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ invoicesync.Service = (*Service)(nil)

type Config struct {
	Adapter   invoicesync.Adapter
	Publisher eventbus.Publisher
	Logger    *slog.Logger
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.Publisher == nil {
		return errors.New("publisher cannot be null")
	}

	if c.Logger == nil {
		return errors.New("logger cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:   config.Adapter,
		publisher: config.Publisher,
		logger:    config.Logger,
	}, nil
}

type Service struct {
	adapter   invoicesync.Adapter
	publisher eventbus.Publisher
	logger    *slog.Logger
}

func (s *Service) CreateDraftSyncPlan(ctx context.Context, input invoicesync.CreateSyncPlanInput) error {
	if err := s.cancelAllActivePlans(ctx, input.Invoice.Namespace, input.Invoice.ID); err != nil {
		return err
	}

	sessionID, ops, err := invoicesync.GenerateDraftSyncPlan(input.GeneratorInput)
	if err != nil {
		return fmt.Errorf("generating draft sync plan: %w", err)
	}

	return s.createAndPublish(ctx, input, invoicesync.SyncPlanPhaseDraft, sessionID, ops)
}

func (s *Service) CreateIssuingSyncPlan(ctx context.Context, input invoicesync.CreateSyncPlanInput) error {
	if err := s.cancelAllActivePlans(ctx, input.Invoice.Namespace, input.Invoice.ID); err != nil {
		return err
	}

	sessionID, ops, err := invoicesync.GenerateIssuingSyncPlan(input.GeneratorInput)
	if err != nil {
		return fmt.Errorf("generating issuing sync plan: %w", err)
	}

	return s.createAndPublish(ctx, input, invoicesync.SyncPlanPhaseIssuing, sessionID, ops)
}

func (s *Service) CreateDeleteSyncPlan(ctx context.Context, input invoicesync.CreateSyncPlanInput) error {
	if err := s.cancelAllActivePlans(ctx, input.Invoice.Namespace, input.Invoice.ID); err != nil {
		return err
	}

	sessionID, ops, err := invoicesync.GenerateDeleteSyncPlan(input.GeneratorInput)
	if err != nil {
		return fmt.Errorf("generating delete sync plan: %w", err)
	}

	// No-op if the invoice has no Stripe external ID.
	if len(ops) == 0 {
		return nil
	}

	return s.createAndPublish(ctx, input, invoicesync.SyncPlanPhaseDelete, sessionID, ops)
}

func (s *Service) CancelActivePlan(ctx context.Context, namespace, invoiceID string, phase invoicesync.SyncPlanPhase) error {
	existing, err := s.adapter.GetActiveSyncPlanByInvoice(ctx, namespace, invoiceID, phase)
	if err != nil {
		return fmt.Errorf("checking for existing sync plan: %w", err)
	}

	if existing == nil {
		return nil
	}

	if err := s.adapter.FailPlan(ctx, existing.ID, "superseded by new sync plan"); err != nil {
		return fmt.Errorf("canceling existing sync plan: %w", err)
	}

	return nil
}

// cancelAllActivePlans cancels all active plans for an invoice regardless of phase.
// This ensures only one plan runs at a time per invoice, preventing duplicate Stripe API calls
// when a newer plan (e.g., issuing) supersedes an older one (e.g., draft).
func (s *Service) cancelAllActivePlans(ctx context.Context, namespace, invoiceID string) error {
	plans, err := s.adapter.GetActiveSyncPlansByInvoice(ctx, namespace, invoiceID)
	if err != nil {
		return fmt.Errorf("checking for active sync plans: %w", err)
	}

	for _, plan := range plans {
		if err := s.adapter.FailPlan(ctx, plan.ID, "superseded by new sync plan"); err != nil {
			return fmt.Errorf("canceling sync plan %s: %w", plan.ID, err)
		}
	}

	return nil
}

// createAndPublish persists a sync plan and defers event publishing until after the
// outermost transaction commits, ensuring the plan is visible in the DB when the
// worker processes the event.
func (s *Service) createAndPublish(ctx context.Context, input invoicesync.CreateSyncPlanInput, phase invoicesync.SyncPlanPhase, sessionID string, ops []invoicesync.SyncOperation) error {
	plan, err := s.adapter.CreateSyncPlan(ctx, invoicesync.SyncPlan{
		Namespace:  input.Invoice.Namespace,
		InvoiceID:  input.Invoice.ID,
		AppID:      input.GeneratorInput.AppID,
		SessionID:  sessionID,
		Phase:      phase,
		Operations: ops,
	})
	if err != nil {
		return fmt.Errorf("creating sync plan: %w", err)
	}

	event := invoicesync.ExecuteSyncPlanEvent{
		PlanID:     plan.ID,
		InvoiceID:  input.Invoice.ID,
		Namespace:  input.Invoice.Namespace,
		CustomerID: input.Invoice.Customer.CustomerID,
	}

	// Defer publish until after the outermost transaction commits.
	// If we're not inside a transaction, OnCommit executes immediately.
	transaction.OnCommit(ctx, func(ctx context.Context) {
		if err := s.publisher.Publish(ctx, event); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish sync plan event; plan will be picked up by next sync",
				"plan_id", plan.ID,
				"invoice_id", input.Invoice.ID,
				"phase", phase,
				"error", err,
			)
		}
	})

	return nil
}
