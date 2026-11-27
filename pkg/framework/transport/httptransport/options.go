package httptransport

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

func (h *handler[Request, Response]) apply(options []HandlerOption) {
	var opts handlerOptions

	opts.apply(options)

	h.errorHandler = opts.resolveErrorHandler()
	h.operationNameFunc = opts.operationNameFunc
	h.errorEncoders = opts.errorEncoders
	h.primaryDatabase = opts.primaryDatabase
}

type HandlerOption interface {
	apply(o *handlerOptions)
}

type optionFunc func(o *handlerOptions)

func (fn optionFunc) apply(o *handlerOptions) {
	fn(o)
}

func WithErrorHandler(errorHandler ErrorHandler) HandlerOption {
	return optionFunc(func(o *handlerOptions) {
		o.errorHandler = errorHandler
	})
}

func WithErrorEncoder(errorEncoder encoder.ErrorEncoder) HandlerOption {
	return optionFunc(func(o *handlerOptions) {
		o.errorEncoders = append(o.errorEncoders, errorEncoder)
	})
}

func WithOperationName(name string) HandlerOption {
	return optionFunc(func(o *handlerOptions) {
		o.operationNameFunc = func(ctx context.Context) string {
			return name
		}
	})
}

func WithOperationNameFunc(fn func(ctx context.Context) string) HandlerOption {
	return optionFunc(func(o *handlerOptions) {
		o.operationNameFunc = fn
	})
}

// WithPrimaryDatabase keeps a GET or HEAD operation out of the read-only scope.
// Use it when the operation writes (lazily saved snapshots, get-or-create), or when
// it must not be served from a read replica because callers read it right after a write.
func WithPrimaryDatabase() HandlerOption {
	return optionFunc(func(o *handlerOptions) {
		o.primaryDatabase = true
	})
}

type handlerOptions struct {
	errorHandler ErrorHandler
	// errorEncoder is responible for outputting the resulting error
	errorEncoders []encoder.ErrorEncoder

	operationNameFunc func(ctx context.Context) string

	primaryDatabase bool
}

func (h *handlerOptions) apply(options []HandlerOption) {
	for _, o := range options {
		o.apply(h)
	}
}

type dummyErrorHandler struct{}

func (dummyErrorHandler) HandleContext(ctx context.Context, err error) {}

func (o handlerOptions) resolveErrorHandler() ErrorHandler {
	if o.errorHandler == nil {
		return dummyErrorHandler{}
	}

	return o.errorHandler
}

func AppendOptions(base []HandlerOption, items ...HandlerOption) []HandlerOption {
	return append(base, items...)
}
