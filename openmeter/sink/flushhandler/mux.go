package flushhandler

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/sink/models"
)

type DrainCompleteFunc func()

type FlushEventHandlers struct {
	handlers        []FlushEventHandler
	onDrainComplete []DrainCompleteFunc
}

func NewFlushEventHandlers() *FlushEventHandlers {
	return &FlushEventHandlers{}
}

func (f *FlushEventHandlers) AddHandler(handler FlushEventHandler) {
	f.handlers = append(f.handlers, handler)
}

func (f *FlushEventHandlers) OnDrainComplete(fn DrainCompleteFunc) {
	f.onDrainComplete = append(f.onDrainComplete, fn)
}

func (f *FlushEventHandlers) OnFlushSuccess(ctx context.Context, events []models.SinkMessage) error {
	var finalError error

	for _, handler := range f.handlers {
		if err := handler.OnFlushSuccess(ctx, events); err != nil {
			finalError = errors.Join(finalError, err)
		}
	}

	return finalError
}

func (f *FlushEventHandlers) Start(ctx context.Context) error {
	for _, handler := range f.handlers {
		if err := handler.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (f *FlushEventHandlers) WaitForDrain(ctx context.Context) error {
	for _, handler := range f.handlers {
		if err := handler.WaitForDrain(ctx); err != nil {
			return err
		}
	}

	for _, fn := range f.onDrainComplete {
		fn()
	}

	return nil
}
