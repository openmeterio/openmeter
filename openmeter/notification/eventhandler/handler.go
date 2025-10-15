package eventhandler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ notification.EventHandler = (*Handler)(nil)

type Handler struct {
	repo    notification.Repository
	webhook webhook.Handler

	logger *slog.Logger
	tracer trace.Tracer

	reconcileInterval time.Duration

	stopCh      chan struct{}
	stopChClose func()
}

func (h *Handler) Start() error {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("notification event handler panicked",
					"error", err,
					"code.stacktrace", string(debug.Stack()))

				h.stopChClose()
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ticker := time.NewTicker(h.reconcileInterval)
		defer ticker.Stop()

		for {
			select {
			case <-h.stopCh:
				h.logger.Debug("close event received: stopping reconciler")
				return
			case <-ticker.C:
				if err := h.Reconcile(ctx); err != nil {
					h.logger.ErrorContext(ctx, "failed to reconcile event(s)", "error", err)
				}
			}
		}
	}()

	return nil
}

func (h *Handler) Close() error {
	h.stopChClose()

	return nil
}

func New(config Config) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.ReconcileInterval == 0 {
		config.ReconcileInterval = notification.DefaultReconcileInterval
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
	}, nil
}
