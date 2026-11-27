package httptransport

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

var defaultHandlerOptions = []HandlerOption{
	WithErrorEncoder(commonhttp.GenericErrorEncoder()),
}

// tracer reads the globally configured TracerProvider (set during telemetry init).
// Used to start an application-level span named after the handler operation, as a
// child of the otelhttp server span.
var tracer = otel.Tracer("github.com/openmeterio/openmeter/pkg/framework/transport/httptransport")

// operationSpansEnabled is a global toggle for the per-operation child span. It is off
// by default — the span is added to every operation request, so enabling it across the
// whole API surface meaningfully increases trace-span volume. Set once at startup via
// EnableOperationSpans.
var operationSpansEnabled atomic.Bool

// EnableOperationSpans globally enables or disables the application-level per-operation
// child span. Intended to be called once during startup from telemetry configuration.
func EnableOperationSpans(enabled bool) {
	operationSpansEnabled.Store(enabled)
}

// ReadOnlyMarker marks the context of a read-only request, see transaction.WithWriteObserver.
// The operation name is empty for handlers that do not declare one.
type ReadOnlyMarker func(ctx context.Context, operation string) context.Context

// readOnlyMarker is nil until SetReadOnlyMarker is called, which leaves
// every request unmarked.
var readOnlyMarker atomic.Pointer[ReadOnlyMarker]

// SetReadOnlyMarker makes every GET and HEAD operation run inside a read-only
// scope, unless the handler opts out with WithPrimaryDatabase. Intended to be called
// once during startup; passing nil turns the marking off again.
func SetReadOnlyMarker(marker ReadOnlyMarker) {
	if marker == nil {
		readOnlyMarker.Store(nil)

		return
	}

	readOnlyMarker.Store(&marker)
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

	primaryDatabase bool
}

type RequestDecoder[Request any] func(ctx context.Context, r *http.Request) (Request, error)

// ErrorHandler receives a transport error to be processed for diagnostic purposes.
// Usually this means logging the error.
type ErrorHandler interface {
	HandleContext(ctx context.Context, err error)
}

func (h handler[Request, Response]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var operationName string

	// TODO: rewrite this as a generic hook
	if h.operationNameFunc != nil {
		name := h.operationNameFunc(ctx)
		operationName = name
		ctx = contextx.WithAttr(ctx, string(semconv.HTTPRouteKey), name)

		// When enabled globally (EnableOperationSpans), start an application-level span
		// named after the operation, as a child of the otelhttp server span. The server
		// span stays route-named; this one carries the operation identity (e.g.
		// "query-entitlement-access") and exposes the handler-vs-middleware timing split.
		// Off by default: it adds a span to every operation request API-wide, so it is
		// gated by a single startup toggle rather than enabled unconditionally.
		if operationSpansEnabled.Load() {
			var span trace.Span
			ctx, span = tracer.Start(ctx, name)
			defer span.End()
		}
	}

	// Marked before decoding, because decoders read from the database too.
	if (r.Method == http.MethodGet || r.Method == http.MethodHead) && !h.primaryDatabase {
		if marker := readOnlyMarker.Load(); marker != nil {
			ctx = (*marker)(ctx, operationName)
		}
	}

	request, err := h.decodeRequest(ctx, r)
	if err != nil {
		// Might be a client error (can be encoded, non-terminal)
		// Might be a server error (terminal)
		h.handleError(ctx, err, w, r)

		return
	}

	response, err := h.operation(ctx, request)
	if err != nil {
		// Might be a client error (can be encoded, non-terminal)
		// Might be a server error (terminal)
		h.handleError(ctx, err, w, r)

		return
	}

	if err := h.encodeResponse(ctx, w, r, response); err != nil {
		// Headers may already be committed. Preserve them, but report a failed write
		// to a disconnected caller as cancellation rather than a server failure.
		var writeErr *encoder.ResponseWriteError
		var networkErr net.Error
		// A server write deadline can also cancel the request context.
		// Keep that timeout as a failure even though the connection is now closed.
		if errors.Is(r.Context().Err(), context.Canceled) && errors.As(err, &writeErr) &&
			(!errors.As(err, &networkErr) || !networkErr.Timeout()) {
			err = errors.Join(context.Canceled, err)
		}

		h.errorHandler.HandleContext(ctx, err)

		return
	}
}

func (h handler[Request, Response]) handleError(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) {
	// Internal contexts can be canceled while the HTTP request remains active, so only
	// classify the error as a client disconnect when the request context is canceled too.
	if contextx.IsCanceledError(err) && errors.Is(r.Context().Err(), context.Canceled) {
		w.WriteHeader(models.StatusClientClosedRequest)
		h.errorHandler.HandleContext(ctx, err)

		return
	}

	if !h.encodeError(ctx, err, w, r) {
		h.errorHandler.HandleContext(ctx, err)
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
