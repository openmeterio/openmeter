package errorsx

import (
	"context"
	"errors"
	"log/slog"
)

var (
	_ Handler = (*SlogHandler)(nil)
	_ Handler = (*NopHandler)(nil)
)

// Handler handles an error.
type Handler interface {
	Handle(err error)
	HandleContext(ctx context.Context, err error)
}

// SlogHandler is a Handler that logs errors using slog.
type SlogHandler struct {
	Logger *slog.Logger
}

func NewSlogHandler(logger *slog.Logger) SlogHandler {
	return SlogHandler{Logger: logger}
}

func (h SlogHandler) Handle(err error) {
	// Context canceled errors are logged as warnings.
	if errors.Is(err, context.Canceled) {
		h.Logger.Warn(err.Error())
		return
	}

	// Warn errors are logged as warnings.
	if wErr, ok := ErrorAs[*warnError](err); ok {
		h.Logger.Warn(wErr.Error())
		return
	}

	// All other errors are logged as errors.
	h.Logger.Error(err.Error())
}

func (h SlogHandler) HandleContext(ctx context.Context, err error) {
	// Context canceled errors are logged as warnings.
	if errors.Is(err, context.Canceled) {
		h.Logger.WarnContext(ctx, err.Error())
		return
	}

	// Warn errors are logged as warnings.
	if wErr, ok := ErrorAs[*warnError](err); ok {
		h.Logger.WarnContext(ctx, wErr.Error())
		return
	}

	// All other errors are logged as errors.
	h.Logger.ErrorContext(ctx, err.Error())
}

// NopHandler ignores all errors.
type NopHandler struct{}

func NewNopHandler() NopHandler {
	return NopHandler{}
}

func (h NopHandler) Handle(err error) {
}

func (h NopHandler) HandleContext(ctx context.Context, err error) {
}
