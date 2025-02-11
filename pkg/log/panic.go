package log

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
)

type propagationStrategy string

const (
	// propagationStrategyPanic will propagate the panic after logging it
	propagationStrategyRePanic propagationStrategy = "panic"

	// propagationStrategyExit will exit the program after logging the panic
	propagationStrategyExit propagationStrategy = "exit"

	// propagationStrategyContinue will continue the program after logging the panic
	propagationStrategyContinue propagationStrategy = "continue"
)

type panicLoggerOptions struct {
	propagationStrategy propagationStrategy
}

// WithRePanic is an option to set the propagation strategy to propagate the panic after logging it
func WithRePanic(o *panicLoggerOptions) {
	o.propagationStrategy = propagationStrategyRePanic
}

// WithExit is an option to set the propagation strategy to exit the program after logging the panic
func WithExit(o *panicLoggerOptions) {
	o.propagationStrategy = propagationStrategyExit
}

// WithContinue is an option to set the propagation strategy to continue the program after logging the panic
func WithContinue(o *panicLoggerOptions) {
	o.propagationStrategy = propagationStrategyContinue
}

// PanicLogger is a function that logs panics and then propagates the failure based on the propagationStrategy setting
// Usage (in main):
//
//	defer log.PanicLogger(log.WithExit)
func PanicLogger(options ...func(*panicLoggerOptions)) {
	opts := &panicLoggerOptions{
		propagationStrategy: propagationStrategyRePanic,
	}

	for _, o := range options {
		o(opts)
	}

	if r := recover(); r != nil {
		description := fmt.Sprintf("panic: %s", r)

		slog.Error(description, "stack", string(debug.Stack()))

		switch opts.propagationStrategy {
		case propagationStrategyExit:
			os.Exit(1)
		case propagationStrategyContinue:
			return
		case propagationStrategyRePanic:
			fallthrough
		default:
			panic(r)
		}
	}
}
