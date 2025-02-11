package paniclogger

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

// PanicLogger is a function that logs panics and re-panics, should be deferred in the main function
// Usage (in main):
//
//	defer paniclogger.PanicLogger()
func PanicLogger() {
	if r := recover(); r != nil {
		description := fmt.Sprintf("panic: %s", r)

		slog.Error(description, "stack", string(debug.Stack()))

		// Let's propagate the panic as we don't know how the system should recover from it
		panic(r)
	}
}
