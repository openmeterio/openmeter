package eventhandler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"cirello.io/pglock"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Repository        notification.Repository
	Webhook           webhook.Handler
	Logger            *slog.Logger
	Tracer            trace.Tracer
	ReconcileInterval time.Duration
	SendingTimeout    time.Duration
	PendingTimeout    time.Duration
	ReconcilerWorkers int
	LockClient        *pglock.Client
}

func (c *Config) Validate() error {
	var errs []error

	if c.Repository == nil {
		errs = append(errs, fmt.Errorf("repository is required"))
	}

	if c.Webhook == nil {
		errs = append(errs, fmt.Errorf("webhook is required"))
	}

	if c.Logger == nil {
		errs = append(errs, fmt.Errorf("logger is required"))
	}

	if c.Tracer == nil {
		errs = append(errs, fmt.Errorf("tracer is required"))
	}

	if c.LockClient == nil {
		errs = append(errs, fmt.Errorf("distributed lock client is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ notification.EventHandler = (*Handler)(nil)

type Handler struct {
	repo    notification.Repository
	webhook webhook.Handler

	logger *slog.Logger
	tracer trace.Tracer

	reconcileInterval time.Duration

	running     atomic.Bool
	stopCh      chan struct{}
	ctxCancel   context.CancelFunc
	stopChClose func()

	lockClient *pglock.Client

	// Delivery status timeouts
	sendingTimeout time.Duration
	pendingTimeout time.Duration

	workerPoolSize int64
}

const reconcilerLeaderLockKey = "notification.event_handler.reconcile_lock"

func (h *Handler) Start() error {
	if !h.running.CompareAndSwap(false, true) {
		return fmt.Errorf("notification event handler is already running")
	}

	defer func() {
		if err := recover(); err != nil {
			h.logger.Error("notification event handler panicked",
				"error", err,
				"code.stacktrace", string(debug.Stack()))
			_ = h.Close()
		}
	}()

	var ctx context.Context

	ctx, h.ctxCancel = context.WithCancel(context.Background())
	defer h.ctxCancel()

	for h.running.Load() {
		err := h.lockClient.Do(ctx, reconcilerLeaderLockKey, func(rCtx context.Context, _ *pglock.Lock) error {
			ticker := time.NewTicker(h.reconcileInterval)
			defer ticker.Stop()

			for {
				select {
				case <-rCtx.Done():
					return nil
				case <-h.stopCh:
					h.logger.DebugContext(rCtx, "close event received: stopping event reconciler")
					return nil
				case <-ticker.C:
					if err := h.Reconcile(rCtx); err != nil {
						h.logger.ErrorContext(rCtx, "failed to reconcile event(s)", "error", err)
					}
				}
			}
		})
		if err != nil {
			if errors.Is(err, pglock.ErrNotAcquired) {
				h.logger.DebugContext(ctx, "reconciliation skipped: lock is not acquired")
				continue
			}

			return fmt.Errorf("failed to acquire reconciliation lock: %w", err)
		}
	}

	return nil
}

func (h *Handler) Close() error {
	if h.running.CompareAndSwap(true, false) {
		h.logger.Debug("closing notification event handler")

		h.ctxCancel()
		h.stopChClose()
	}

	return nil
}

func New(config Config) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.ReconcileInterval == 0 {
		config.ReconcileInterval = notification.DefaultReconcileInterval
	}

	if config.PendingTimeout == 0 {
		config.PendingTimeout = notification.DefaultDeliveryStatePendingTimeout
	}

	if config.SendingTimeout == 0 {
		config.SendingTimeout = notification.DefaultDeliveryStateSendingTimeout
	}

	if config.ReconcilerWorkers <= 0 {
		config.ReconcilerWorkers = notification.DefaultReconcilerWorkers
	}

	stopCh := make(chan struct{})
	stopChClose := sync.OnceFunc(func() {
		close(stopCh)
	})

	return &Handler{
		repo:              config.Repository,
		webhook:           config.Webhook,
		reconcileInterval: config.ReconcileInterval,
		logger:            config.Logger,
		tracer:            config.Tracer,
		stopCh:            stopCh,
		stopChClose:       stopChClose,
		lockClient:        config.LockClient,
		sendingTimeout:    config.SendingTimeout,
		pendingTimeout:    config.PendingTimeout,
		workerPoolSize:    int64(config.ReconcilerWorkers),
	}, nil
}
