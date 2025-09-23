package httptransport

import (
	"context"
	"errors"
	"net/http"

	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

var defaultHandlerOptions = []HandlerOption{
	WithErrorEncoder(commonhttp.GenericErrorEncoder()),
}

type Handler[Request any, Response any] interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Chain(outer operation.Middleware[Request, Response], others ...operation.Middleware[Request, Response]) Handler[Request, Response]
}

// NewHandler returns a new HTTP handler that wraps the given [operation.Operation].
func NewHandler[Request any, Response any](
	requestDecoder RequestDecoder[Request],
	op operation.Operation[Request, Response],
	responseEncoder encoder.ResponseEncoder[Response],

	options ...HandlerOption,
) Handler[Request, Response] {
	return newHandler(requestDecoder, op, responseEncoder, options...)
}

func newHandler[Request any, Response any](
	requestDecoder RequestDecoder[Request],
	op operation.Operation[Request, Response],
	responseEncoder encoder.ResponseEncoder[Response],

	options ...HandlerOption,
) handler[Request, Response] {
	h := handler[Request, Response]{
		operation: op,

		decodeRequest:  requestDecoder,
		encodeResponse: responseEncoder,
	}

	options = append(options, defaultHandlerOptions...)

	h.apply(options)

	return h
}

type handler[Request any, Response any] struct {
	operation         operation.Operation[Request, Response]
	operationNameFunc func(ctx context.Context) string

	decodeRequest  RequestDecoder[Request]
	encodeResponse encoder.ResponseEncoder[Response]
	errorEncoders  []encoder.ErrorEncoder

	errorHandler ErrorHandler
}

type RequestDecoder[Request any] func(ctx context.Context, r *http.Request) (Request, error)

// ErrorHandler receives a transport error to be processed for diagnostic purposes.
// Usually this means logging the error.
type ErrorHandler interface {
	HandleContext(ctx context.Context, err error)
}

func (h handler[Request, Response]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO: rewrite this as a generic hook
	if h.operationNameFunc != nil {
		ctx = contextx.WithAttr(ctx, string(semconv.HTTPRouteKey), h.operationNameFunc(ctx))
	}

	request, err := h.decodeRequest(ctx, r)
	if err != nil {
		// Might be a client error (can be encoded, non-terminal)
		// Might be a server error (terminal)

		handled := h.encodeError(ctx, err, w, r)
		if !handled {
			h.errorHandler.HandleContext(ctx, err)
		}

		return
	}

	response, err := h.operation(ctx, request)
	if err != nil {
		// Might be a client error (can be encoded, non-terminal)
		// Might be a server error (terminal)

		handled := h.encodeError(ctx, err, w, r)
		if !handled {
			h.errorHandler.HandleContext(ctx, err)
		}

		return
	}

	if err := h.encodeResponse(ctx, w, r, response); err != nil {
		// Always a server error (terminal)?

		h.errorHandler.HandleContext(ctx, err)
		return
	}
}

func (h handler[Request, Response]) encodeError(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
	for _, errorEncoder := range h.errorEncoders {
		if errorEncoder(ctx, err, w, r) {
			return true
		}
	}

	if encoder, ok := err.(SelfEncodingError); ok {
		if encoder.EncodeError(ctx, w) {
			return true
		}
	}

	models.NewStatusProblem(ctx, errors.New("internal server error"), http.StatusInternalServerError).Respond(w)

	return false
}

func (h handler[Request, Response]) Chain(outer operation.Middleware[Request, Response], others ...operation.Middleware[Request, Response]) Handler[Request, Response] {
	h.operation = operation.Chain(outer, others...)(h.operation)
	return h
}

type SelfEncodingError interface {
	EncodeError(ctx context.Context, w http.ResponseWriter) bool
}
