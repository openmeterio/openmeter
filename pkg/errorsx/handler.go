package errorsx

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

var (
	_ httptransport.ErrorHandler = Handler(nil)
	_ Handler                    = (*SlogHandler)(nil)
	_ Handler                    = (*NopHandler)(nil)
)

// Handler handles an error.
// Same interface as httptransport.ErrorHandler.
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
	h.log(context.Background(), err, true)
}

func (h SlogHandler) HandleContext(ctx context.Context, err error) {
	// Only downgrade cancellations of the caller's context. Independent dependency
	// cancellations remain failures while the caller is still waiting for a response.
	h.log(ctx, err, errors.Is(ctx.Err(), context.Canceled))
}

func (h SlogHandler) log(ctx context.Context, err error, warnOnCancellation bool) {
	level := slog.LevelError
	messageErr := err
	if (warnOnCancellation && contextx.IsCanceledError(err)) || isAPIError(err) {
		level = slog.LevelWarn
	} else if wErr, ok := lo.ErrorsAs[*warnError](err); ok {
		level = slog.LevelWarn
		messageErr = wErr
	}

	if !h.Logger.Enabled(ctx, level) {
		return
	}

	// Skip runtime.Callers, log, and Handle or HandleContext to report the call site.
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	record := slog.NewRecord(time.Now(), level, messageErr.Error(), pcs[0])
	record.Add(diagnosticAttributes(err)...)
	_ = h.Logger.Handler().Handle(ctx, record)
}

// NopHandler ignores all errors. Useful for testing.
type NopHandler struct{}

func NewNopHandler() NopHandler {
	return NopHandler{}
}

func (h NopHandler) Handle(err error) {
}

func (h NopHandler) HandleContext(ctx context.Context, err error) {
}
