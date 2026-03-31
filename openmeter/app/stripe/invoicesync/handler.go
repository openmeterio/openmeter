package invoicesync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// postTxEvent holds an event that must be published after the transaction commits.
// Using a concrete type instead of closures avoids capturing the transaction context.
type postTxEvent struct {
	event marshaler.Event // nil means nothing to publish
}

// LockFunc acquires an advisory lock for the given invoice within the current transaction context.
// This ensures only one sync plan executes at a time per invoice.
// It should block until the lock is available, or return ErrSyncPlanLocked if the lock is
// held by another worker and a timeout occurs.
type LockFunc func(ctx context.Context, namespace, invoiceID string) error

// ErrSyncPlanLocked is returned by LockFunc when the lock is held by another worker.
var ErrSyncPlanLocked = fmt.Errorf("sync plan is locked by another worker")

// HandlerConfig configures the sync plan handler.
type HandlerConfig struct {
	Adapter                Adapter
	AppService             app.Service
	BillingService         billing.Service
	StripeAppService       StripeAppServiceForSync
	SecretService          secret.Service
	StripeAppClientFactory stripeclient.StripeAppClientFactory
	Publisher              eventbus.Publisher
	LockFunc               LockFunc
	Logger                 *slog.Logger
}

// StripeAppServiceForSync is the subset of the Stripe app service needed for sync plan execution.
type StripeAppServiceForSync interface {
	GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error)
}

// Handler handles ExecuteSyncPlanEvent events.
type Handler struct {
	adapter                Adapter
	appService             app.Service
	billingService         billing.Service
	stripeAppService       StripeAppServiceForSync
	secretService          secret.Service
	stripeAppClientFactory stripeclient.StripeAppClientFactory
	publisher              eventbus.Publisher
	lockFunc               LockFunc
	logger                 *slog.Logger
}

// NewHandler creates a new sync plan handler.
func NewHandler(config HandlerConfig) (*Handler, error) {
	if config.Adapter == nil {
		return nil, fmt.Errorf("adapter is required")
	}
	if config.AppService == nil {
		return nil, fmt.Errorf("app service is required")
	}
	if config.BillingService == nil {
		return nil, fmt.Errorf("billing service is required")
	}
	if config.StripeAppService == nil {
		return nil, fmt.Errorf("stripe app service is required")
	}
	if config.SecretService == nil {
		return nil, fmt.Errorf("secret service is required")
	}
	if config.StripeAppClientFactory == nil {
		return nil, fmt.Errorf("stripe app client factory is required")
	}
	if config.Publisher == nil {
		return nil, fmt.Errorf("publisher is required")
	}
	if config.LockFunc == nil {
		return nil, fmt.Errorf("lock function is required")
	}
	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &Handler{
		adapter:                config.Adapter,
		appService:             config.AppService,
		billingService:         config.BillingService,
		stripeAppService:       config.StripeAppService,
		secretService:          config.SecretService,
		stripeAppClientFactory: config.StripeAppClientFactory,
		publisher:              config.Publisher,
		lockFunc:               config.LockFunc,
		logger:                 config.Logger,
	}, nil
}

// Handle processes a sync plan execution event.
func (h *Handler) Handle(ctx context.Context, event *ExecuteSyncPlanEvent) error {
	if event == nil {
		return nil
	}

	logger := h.logger.With(
		"plan_id", event.PlanID,
		"invoice_id", event.InvoiceID,
		"namespace", event.Namespace,
	)

	// Quick check before acquiring lock — skip obviously terminal plans
	plan, err := h.adapter.GetSyncPlan(ctx, event.PlanID)
	if err != nil {
		return fmt.Errorf("getting sync plan: %w", err)
	}
	if plan == nil {
		logger.WarnContext(ctx, "sync plan not found, skipping")
		return nil
	}
	if plan.Status == PlanStatusCompleted || plan.Status == PlanStatusFailed {
		logger.InfoContext(ctx, "sync plan already in terminal state", "status", plan.Status)
		return nil
	}
	if plan.AppID == "" {
		return fmt.Errorf("sync plan %s has no app ID", plan.ID)
	}

	// Execute within a transaction with an advisory lock to prevent parallel execution
	// of multiple plans for the same invoice (e.g., draft and issuing plans).
	// Returns a postTxEvent carrying any event that must be published after commit.
	result, err := transaction.Run(ctx, h.adapter, func(ctx context.Context) (postTxEvent, error) {
		// Acquire advisory lock scoped to this invoice
		if err := h.lockFunc(ctx, event.Namespace, event.InvoiceID); err != nil {
			if errors.Is(err, ErrSyncPlanLocked) {
				logger.InfoContext(ctx, "invoice sync locked by another plan, skipping")
				return postTxEvent{}, nil
			}
			return postTxEvent{}, fmt.Errorf("acquiring sync plan lock: %w", err)
		}

		// Re-fetch plan inside lock — another worker may have advanced it
		plan, err = h.adapter.GetSyncPlan(ctx, event.PlanID)
		if err != nil {
			return postTxEvent{}, fmt.Errorf("getting sync plan under lock: %w", err)
		}
		if plan == nil || plan.Status == PlanStatusCompleted || plan.Status == PlanStatusFailed {
			return postTxEvent{}, nil
		}

		// Check if a newer plan exists for this invoice — if so, this plan has been
		// superseded and should not continue executing (even if cancelAllActivePlans
		// already marked it as failed, the events may already be in-flight).
		superseded, err := h.isSuperseded(ctx, plan)
		if err != nil {
			return postTxEvent{}, fmt.Errorf("checking if plan is superseded: %w", err)
		}
		if superseded {
			logger.InfoContext(ctx, "plan superseded by a newer plan, canceling")
			if err := h.adapter.FailPlan(ctx, plan.ID, "superseded by newer plan"); err != nil {
				logger.ErrorContext(ctx, "failed to cancel superseded plan", "error", err)
			}
			return postTxEvent{}, nil
		}

		// Create Stripe client
		stripeClient, err := h.createStripeClient(ctx, event.Namespace, plan)
		if err != nil {
			return postTxEvent{}, fmt.Errorf("creating stripe client: %w", err)
		}

		// Execute next operation
		executor := &Executor{
			Adapter: h.adapter,
			Logger:  logger,
		}

		execResult, err := executor.ExecuteNextOperation(ctx, stripeClient, plan)
		if err != nil {
			return postTxEvent{}, err
		}

		// Write back external IDs immediately so the invoice always reflects Stripe state.
		// This ensures canceled plans don't leave stale state — the next plan will see
		// correct external IDs and generate update ops instead of duplicate creates.
		//
		// On failure this rolls back the transaction (including CompleteOperation), causing
		// Kafka to redeliver. The idempotency key ensures no duplicate Stripe API calls.
		if execResult.InvoicingExternalID != nil || len(execResult.LineExternalIDs) > 0 || len(execResult.LineDiscountExternalIDs) > 0 {
			if err := h.billingService.SyncExternalIDs(ctx, billing.SyncExternalIDsInput{
				Invoice: billing.InvoiceID{
					Namespace: event.Namespace,
					ID:        event.InvoiceID,
				},
				InvoicingExternalID:     execResult.InvoicingExternalID,
				LineExternalIDs:         execResult.LineExternalIDs,
				LineDiscountExternalIDs: execResult.LineDiscountExternalIDs,
			}); err != nil {
				return postTxEvent{}, fmt.Errorf("syncing external IDs: %w", err)
			}
		}

		if !execResult.Done {
			return postTxEvent{
				event: ExecuteSyncPlanEvent{
					PlanID:     event.PlanID,
					InvoiceID:  event.InvoiceID,
					Namespace:  event.Namespace,
					CustomerID: event.CustomerID,
				},
			}, nil
		}

		if execResult.Failed {
			logger.ErrorContext(ctx, "sync plan failed", "error", execResult.FailError)
			return postTxEvent{}, h.handlePlanFailure(ctx, event, plan.Phase, execResult.FailError)
		}

		// Refresh plan to get completed operation responses
		plan, err = h.adapter.GetSyncPlan(ctx, event.PlanID)
		if err != nil {
			return postTxEvent{}, fmt.Errorf("refreshing sync plan: %w", err)
		}

		return h.handlePlanCompletion(ctx, event, plan)
	})
	if err != nil {
		return err
	}

	// Publish after commit so DB state is visible, using the original (non-tx) context.
	if result.event != nil {
		return h.publisher.Publish(ctx, result.event)
	}

	return nil
}

// handlePlanCompletion writes results back to the invoice and triggers advancement.
// Returns a postTxEvent if an event must be published after the transaction commits.
func (h *Handler) handlePlanCompletion(ctx context.Context, event *ExecuteSyncPlanEvent, plan *SyncPlan) (postTxEvent, error) {
	invoiceID := billing.InvoiceID{
		Namespace: event.Namespace,
		ID:        event.InvoiceID,
	}
	now := clock.Now().UTC().Format(time.RFC3339)

	switch plan.Phase {
	case SyncPlanPhaseDraft:
		upsertResult, err := BuildUpsertResultFromPlan(plan)
		if err != nil {
			return postTxEvent{}, fmt.Errorf("building upsert result: %w", err)
		}

		_, err = h.billingService.SyncDraftInvoice(ctx, billing.SyncDraftStandardInvoiceInput{
			InvoiceID:            invoiceID,
			UpsertInvoiceResults: upsertResult,
			AdditionalMetadata: map[string]string{
				MetadataKeyDraftSyncPlanID:      plan.ID,
				MetadataKeyDraftSyncCompletedAt: now,
			},
		})
		if err != nil {
			// If the invoice already moved past the expected state (e.g., a newer issuing plan
			// advanced it), this plan's completion is stale — the Stripe operations already
			// succeeded, so we can safely skip the state machine callback.
			if billing.IsInvoiceStateMismatchError(err) {
				h.logger.InfoContext(ctx, "skipping stale draft sync completion, invoice already advanced",
					"plan_id", plan.ID,
					"error", err,
				)
				return postTxEvent{}, nil
			}
			return postTxEvent{}, fmt.Errorf("syncing draft invoice: %w", err)
		}

	case SyncPlanPhaseIssuing:
		finalizeResult, err := BuildFinalizeResultFromPlan(plan)
		if err != nil {
			return postTxEvent{}, fmt.Errorf("building finalize result: %w", err)
		}

		_, err = h.billingService.SyncIssuingInvoice(ctx, billing.SyncIssuingStandardInvoiceInput{
			InvoiceID:             invoiceID,
			FinalizeInvoiceResult: finalizeResult,
			AdditionalMetadata: map[string]string{
				MetadataKeyIssuingSyncPlanID:      plan.ID,
				MetadataKeyIssuingSyncCompletedAt: now,
			},
		})
		if err != nil {
			if billing.IsInvoiceStateMismatchError(err) {
				h.logger.InfoContext(ctx, "skipping stale issuing sync completion, invoice already advanced",
					"plan_id", plan.ID,
					"error", err,
				)
				return postTxEvent{}, nil
			}
			return postTxEvent{}, fmt.Errorf("syncing issuing invoice: %w", err)
		}

	case SyncPlanPhaseDelete:
		// For delete, we just need to advance the invoice after the transaction commits.
		return postTxEvent{event: billing.AdvanceStandardInvoiceEvent{
			Invoice:    invoiceID,
			CustomerID: event.CustomerID,
		}}, nil

	default:
		return postTxEvent{}, fmt.Errorf("unknown sync plan phase %q for plan %s", plan.Phase, plan.ID)
	}

	return postTxEvent{}, nil
}

// handlePlanFailure triggers the invoice into a sync-failed state, surfacing the Stripe error
// as a validation issue on the invoice so it's visible to API consumers.
func (h *Handler) handlePlanFailure(ctx context.Context, event *ExecuteSyncPlanEvent, phase SyncPlanPhase, failError string) error {
	return h.billingService.FailSyncInvoice(ctx, billing.FailSyncInvoiceInput{
		Invoice: billing.InvoiceID{
			Namespace: event.Namespace,
			ID:        event.InvoiceID,
		},
		AppType:   app.AppTypeStripe,
		Operation: phaseToOperation(phase),
		Err:       errors.New(failError),
	})
}

// phaseToOperation maps a sync plan phase to the corresponding billing operation.
func phaseToOperation(phase SyncPlanPhase) billing.StandardInvoiceOperation {
	switch phase {
	case SyncPlanPhaseIssuing:
		return billing.StandardInvoiceOpFinalize
	case SyncPlanPhaseDelete:
		return billing.StandardInvoiceOpDelete
	default:
		return billing.StandardInvoiceOpSync
	}
}

// isSuperseded checks if a newer plan exists for the same invoice, meaning this plan's
// events are stale (e.g., a draft plan still in-flight after an issuing plan was created).
func (h *Handler) isSuperseded(ctx context.Context, plan *SyncPlan) (bool, error) {
	plans, err := h.adapter.GetActiveSyncPlansByInvoice(ctx, plan.Namespace, plan.InvoiceID)
	if err != nil {
		return false, err
	}
	for _, other := range plans {
		if other.ID != plan.ID && other.CreatedAt.After(plan.CreatedAt) {
			return true, nil
		}
	}
	return false, nil
}

// createStripeClient creates a Stripe API client for the invoice's app.
// Caller must ensure plan.AppID is non-empty (checked in Handle before this is called).
func (h *Handler) createStripeClient(ctx context.Context, namespace string, plan *SyncPlan) (stripeclient.StripeAppClient, error) {
	appID := plan.AppID

	appData, err := h.stripeAppService.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: app.AppID{
			Namespace: namespace,
			ID:        appID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting stripe app data: %w", err)
	}

	apiKeySecret, err := h.secretService.GetAppSecret(ctx, secretentity.NewSecretID(
		app.AppID{
			Namespace: namespace,
			ID:        appID,
		},
		appData.APIKey.ID,
		appstripeentity.APIKeySecretKey,
	))
	if err != nil {
		return nil, fmt.Errorf("getting stripe api key: %w", err)
	}

	return h.stripeAppClientFactory(stripeclient.StripeAppClientConfig{
		AppID: app.AppID{
			Namespace: namespace,
			ID:        appID,
		},
		AppService: h.appService,
		APIKey:     apiKeySecret.Value,
		Logger:     h.logger.With("app_id", appID),
	})
}
