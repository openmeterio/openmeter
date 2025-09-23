package encoder

import (
	"context"
	"net/http"
)

type ResponseEncoder[Response any] func(ctx context.Context, w http.ResponseWriter, r *http.Request, response Response) error

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool
