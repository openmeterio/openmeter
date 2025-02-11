package log

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
)

type PropagationStrategy string

const (
	// PropagationStrategyPanic will propagate the panic after logging it
	PropagationStrategyPanic PropagationStrategy = "panic"

	// PropagationStrategyExit will exit the program after logging the panic
	PropagationStrategyExit PropagationStrategy = "exit"

	// PropagationStrategyContinue will continue the program after logging the panic
	PropagationStrategyContinue PropagationStrategy = "continue"
)

type panicLoggerOptions struct {
	propagationStrategy PropagationStrategy
}

func WithPropagationStrategy(strategy PropagationStrategy) func(*panicLoggerOptions) {
	return func(o *panicLoggerOptions) {
		o.propagationStrategy = strategy
	}
}

// PanicLogger is a function that logs panics and then propagates the failure based on the PropagationStrategy setting
// Usage (in main):
//
//	defer log.PanicLogger(
//		log.WithPropagationStrategy(log.PropagationStrategyExit),
//	)
func PanicLogger(options ...func(*panicLoggerOptions)) {
	opts := &panicLoggerOptions{
		propagationStrategy: PropagationStrategyPanic,
	}

	for _, o := range options {
		o(opts)
	}

	if r := recover(); r != nil {
		description := fmt.Sprintf("panic: %s", r)

		slog.Error(description, "stack", string(debug.Stack()))

		switch opts.propagationStrategy {
		case PropagationStrategyExit:
			os.Exit(1)
		case PropagationStrategyContinue:
			return
		case PropagationStrategyPanic:
			fallthrough
		default:
			// Let's propagate the panic as we don't know how the system should recover from it
			panic(r)
		}
	}
}
