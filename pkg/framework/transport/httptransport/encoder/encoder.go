package encoder

import (
	"context"
	"net/http"
)

type ResponseEncoder[Response any] func(ctx context.Context, w http.ResponseWriter, r *http.Request, response Response) error

// ResponseWriteError identifies a failed write to the response connection,
// allowing the transport to distinguish disconnects from serialization failures.
type ResponseWriteError struct {
	Err error
}

func (e *ResponseWriteError) Error() string { return e.Err.Error() }
func (e *ResponseWriteError) Unwrap() error { return e.Err }

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool
