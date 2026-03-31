package invoicesync

import (
	"context"
	"encoding/json"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// Adapter provides persistence for sync plans and their operations.
type Adapter interface {
	entutils.TxCreator

	// CreateSyncPlan persists a new sync plan with all its operations.
	CreateSyncPlan(ctx context.Context, plan SyncPlan) (SyncPlan, error)

	// GetSyncPlan retrieves a sync plan by ID, including its operations ordered by sequence.
	GetSyncPlan(ctx context.Context, planID string) (*SyncPlan, error)

	// GetActiveSyncPlanByInvoice retrieves the most recent non-completed plan for an invoice and phase.
	GetActiveSyncPlanByInvoice(ctx context.Context, namespace, invoiceID string, phase SyncPlanPhase) (*SyncPlan, error)

	// GetActiveSyncPlansByInvoice retrieves all non-completed plans for an invoice (any phase).
	GetActiveSyncPlansByInvoice(ctx context.Context, namespace, invoiceID string) ([]SyncPlan, error)

	// GetNextPendingOperation returns the next pending operation in sequence order, or nil if all are done.
	GetNextPendingOperation(ctx context.Context, planID string) (*SyncOperation, error)

	// CompleteOperation marks an operation as completed and stores the Stripe response.
	CompleteOperation(ctx context.Context, opID string, stripeResponse json.RawMessage) error

	// FailOperation marks an operation as failed with the given error message.
	FailOperation(ctx context.Context, opID string, errMsg string) error

	// UpdatePlanStatus updates the overall plan status.
	UpdatePlanStatus(ctx context.Context, planID string, status PlanStatus, errMsg *string) error

	// CompletePlan marks the plan as completed.
	CompletePlan(ctx context.Context, planID string) error

	// FailPlan marks the plan as failed and cancels remaining pending operations.
	FailPlan(ctx context.Context, planID string, errMsg string) error
}
