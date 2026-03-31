package invoicesync

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

// Service manages the lifecycle of invoice sync plans (create, cancel).
// Plan execution is handled separately by the Handler in the billing worker.
type Service interface {
	// CreateDraftSyncPlan generates and persists a draft sync plan, canceling any active one,
	// then publishes an execution event.
	CreateDraftSyncPlan(ctx context.Context, input CreateSyncPlanInput) error

	// CreateIssuingSyncPlan generates and persists an issuing sync plan, canceling any active one,
	// then publishes an execution event.
	CreateIssuingSyncPlan(ctx context.Context, input CreateSyncPlanInput) error

	// CreateDeleteSyncPlan generates and persists a delete sync plan and publishes an execution event.
	// If the invoice has no Stripe external ID, this is a no-op.
	CreateDeleteSyncPlan(ctx context.Context, input CreateSyncPlanInput) error

	// CancelActivePlan cancels any active (pending/executing) plan for the given invoice and phase.
	CancelActivePlan(ctx context.Context, namespace, invoiceID string, phase SyncPlanPhase) error
}

// CreateSyncPlanInput contains the data needed to generate and persist a sync plan.
// The caller is responsible for building the PlanGeneratorInput (fetching Stripe customer data,
// existing line items, etc.) since that requires Stripe API access which lives in the app layer.
//
// Invoice is used by the service for namespace/ID lookups and event publishing.
// GeneratorInput.Invoice must be the same instance — it is accessed by the planner for
// line items, workflow config, and external IDs.
type CreateSyncPlanInput struct {
	Invoice        billing.StandardInvoice
	GeneratorInput PlanGeneratorInput
}
