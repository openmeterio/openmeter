package httptransport

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/operation"
)

type HandlerWithArgs[Request any, Response any, ArgType any] interface {
	With(ArgType) Handler[Request, Response]
	Chain(outer operation.Middleware[Request, Response], others ...operation.Middleware[Request, Response]) HandlerWithArgs[Request, Response, ArgType]
}

type RequestDecoderWithArgs[Request any, ArgType any] func(ctx context.Context, r *http.Request, arg ArgType) (Request, error)

func NewHandlerWithArgs[Request any, Response any, ArgType any](
	requestDecoder RequestDecoderWithArgs[Request, ArgType],
	op operation.Operation[Request, Response],
	responseEncoder ResponseEncoder[Response],

	options ...HandlerOption) HandlerWithArgs[Request, Response, ArgType] {

	return handlerWithArgs[Request, Response, ArgType]{
		handler:        newHandler(nil, op, responseEncoder, options...),
		requestDecoder: requestDecoder,
	}

}

type handlerWithArgs[Request any, Response any, ArgType any] struct {
	handler handler[Request, Response]

	requestDecoder RequestDecoderWithArgs[Request, ArgType]
}

func (h handlerWithArgs[Request, Response, ArgType]) With(arg ArgType) Handler[Request, Response] {
	// We are relying here on using non-pointer receivers and that handler is not a pointer
	// if the receiver is changed we need an explicit clone here
	res := h.handler
	res.decodeRequest = func(ctx context.Context, r *http.Request) (Request, error) {
		return h.requestDecoder(ctx, r, arg)
	}
	return res
}

func (h handlerWithArgs[Request, Response, ArgType]) Chain(outer operation.Middleware[Request, Response], others ...operation.Middleware[Request, Response]) HandlerWithArgs[Request, Response, ArgType] {
	// We are relying here on using non-pointer receivers and that handler is not a pointer
	// if the receiver is changed we need an explicit clone here
	h.handler.operation = operation.Chain(outer, others...)(h.handler.operation)
	return h
}
