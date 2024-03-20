package http

import "context"

func (h *handler[Request, Response]) apply(options []HandlerOption) {
	var opts handlerOptions

	opts.apply(options)

	h.errorHandler = opts.resolveErrorHandler()
	h.operationNameFunc = opts.operationNameFunc
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

type handlerOptions struct {
	errorHandler      ErrorHandler
	operationNameFunc func(ctx context.Context) string
}

func (h *handlerOptions) apply(options []HandlerOption) {
	for _, o := range options {
		o.apply(h)
	}
}

func (o handlerOptions) resolveErrorHandler() ErrorHandler {
	if o.errorHandler == nil {
		return nil
	}

	return o.errorHandler
}
