package eventhandler

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

type Config struct {
	Repository        notification.Repository
	Webhook           webhook.Handler
	Logger            *slog.Logger
	ReconcileInterval time.Duration
}

func (c *Config) Validate() error {
	if c.Repository == nil {
		return fmt.Errorf("repository is required")
	}

	if c.Webhook == nil {
		return fmt.Errorf("webhook is required")
	}

	return nil
}

var _ notification.EventHandler = (*Handler)(nil)

type Handler struct {
	repo    notification.Repository
	webhook webhook.Handler
	logger  *slog.Logger

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

	if config.Logger == nil {
		config.Logger = slog.Default()
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
		stopCh:            stopCh,
		stopChClose:       stopChClose,
	}, nil
}
