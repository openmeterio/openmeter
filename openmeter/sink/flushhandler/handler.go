package flushhandler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/sink/models"
)

const (
	defaultFlushChanSize   = 1000
	defaultCallbackTimeout = 30 * time.Second
)

type FlushEventHandlerOptions struct {
	Name     string
	Callback FlushCallback

	Logger      *slog.Logger
	MetricMeter metric.Meter

	DrainTimeout    time.Duration
	CallbackTimeout time.Duration
}

type flushEventHandler struct {
	name      string
	events    chan []models.SinkMessage
	drainDone chan struct{}

	callback FlushCallback

	callbackTimeout time.Duration
	drainTimeout    time.Duration

	metrics *metrics

	logger *slog.Logger

	isShutdown atomic.Bool
	mu         sync.Mutex
}

func NewFlushEventHandler(opts FlushEventHandlerOptions) (FlushEventHandler, error) {
	// validate options
	if opts.Name == "" {
		return nil, errors.New("name is required")
	}

	if opts.Callback == nil {
		return nil, errors.New("callback is required")
	}

	if opts.Logger == nil {
		return nil, errors.New("logger is required")
	}

	if opts.MetricMeter == nil {
		return nil, errors.New("metric meter is required")
	}

	if opts.CallbackTimeout == 0 {
		opts.CallbackTimeout = defaultCallbackTimeout
	}

	if opts.DrainTimeout == 0 {
		opts.DrainTimeout = defaultCallbackTimeout
	}

	// construct underlying object
	metrics, err := newMetrics(opts.Name, opts.MetricMeter)
	if err != nil {
		return nil, err
	}

	return &flushEventHandler{
		callback:        opts.Callback,
		callbackTimeout: opts.CallbackTimeout,
		drainTimeout:    opts.DrainTimeout,
		name:            opts.Name,
		events:          make(chan []models.SinkMessage, defaultFlushChanSize),
		drainDone:       make(chan struct{}),
		metrics:         metrics,
		logger:          opts.Logger,
	}, nil
}

func (f *flushEventHandler) Start(ctx context.Context) error {
	go f.start(ctx)
	return nil
}

func (f *flushEventHandler) start(ctx context.Context) {
	defer close(f.drainDone)

	if f.isShutdown.Load() {
		f.logger.Error("failed to start flush event handler as it is already shut down")
		return
	}

	shouldRun := true
	for shouldRun {
		select {
		case event := <-f.events:
			if err := f.invokeCallbackWithTimeout(event); err != nil {
				f.logger.Error("failed to invoke callback", "error", err)
			}
		case <-ctx.Done():
			shouldRun = false
			f.isShutdown.Store(true)
		}
	}

	// let's drain the queue using a new context, as the parent context is already canceled
	drainContext, cancel := context.WithTimeout(context.Background(), f.drainTimeout)
	defer cancel()

	for event := range f.events {
		if err := f.invokeCallback(drainContext, event); err != nil {
			f.logger.Error("failed to invoke callback", "error", err)
		}
	}
}

func (f *flushEventHandler) invokeCallbackWithTimeout(events []models.SinkMessage) error {
	// We are using a background context here, as if the parent context is canceled, we still want to
	// allow the callbacks to call external systems. In exchange we are limiting the work with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), f.callbackTimeout)
	defer cancel()

	return f.invokeCallback(ctx, events)
}

func (f *flushEventHandler) invokeCallback(ctx context.Context, events []models.SinkMessage) error {
	startTime := time.Now()

	if err := f.callback(ctx, events); err != nil {
		f.metrics.eventsFailed.Add(ctx, 1)
		return err
	}

	f.metrics.eventProcessingTime.Record(ctx, time.Since(startTime).Milliseconds())
	f.metrics.eventsProcessed.Add(ctx, 1)

	return nil
}

func (f *flushEventHandler) OnFlushSuccess(ctx context.Context, event []models.SinkMessage) error {
	if f.isShutdown.Load() {
		return errors.New("handler is shutting down")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	select {
	case f.events <- event:
		f.metrics.eventsReceived.Add(ctx, 1)
	case <-ctx.Done():
		f.metrics.eventsFailed.Add(ctx, 1)
		return fmt.Errorf("context canceled handler: %s", f.name)
	default:
		f.logger.Error("flush handler: work queue full, callback might be hanging", "event", event, "name", f.name)
		f.metrics.eventChannelFull.Add(ctx, 1)
		select {
		case f.events <- event:
			f.metrics.eventsReceived.Add(ctx, 1)
		case <-ctx.Done():
			f.metrics.eventsFailed.Add(ctx, 1)
			return fmt.Errorf("context canceled handler: %s", f.name)
		}
	}

	return nil
}

func (f *flushEventHandler) WaitForDrain(ctx context.Context) error {
	select {
	case <-f.drainDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context canceled while wainting for drain in handler %s", f.name)
	}
}
