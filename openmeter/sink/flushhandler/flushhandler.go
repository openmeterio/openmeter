package flushhandler

import "github.com/openmeterio/openmeter/internal/sink/flushhandler"

type (
	FlushEventHandler = flushhandler.FlushEventHandler
	FlushCallback     = flushhandler.FlushCallback
)

// FlushHandlers
type (
	DrainCompleteFunc  = flushhandler.DrainCompleteFunc
	FlushEventHandlers = flushhandler.FlushEventHandlers
)

func NewFlushEventHandlers() *FlushEventHandlers {
	return flushhandler.NewFlushEventHandlers()
}

// FlushHandler
type (
	FlushEventHandlerOptions = flushhandler.FlushEventHandlerOptions
)

func NewFlushEventHandler(opts FlushEventHandlerOptions) (FlushEventHandler, error) {
	return flushhandler.NewFlushEventHandler(opts)
}
