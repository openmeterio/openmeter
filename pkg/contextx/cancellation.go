package contextx

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsCanceledError recognizes cancellation causes in Go and gRPC error trees.
// Callers must also check their request context before treating the error as a
// client disconnect: a dependency can cancel its own context independently.
func IsCanceledError(err error) bool {
	if errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
		return true
	}

	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, cause := range joined.Unwrap() {
			if IsCanceledError(cause) {
				return true
			}
		}
	} else if cause := errors.Unwrap(err); cause != nil {
		return IsCanceledError(cause)
	}

	return false
}
