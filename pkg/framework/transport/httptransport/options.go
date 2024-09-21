// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httptransport

import "context"

func (h *handler[Request, Response]) apply(options []HandlerOption) {
	var opts handlerOptions

	opts.apply(options)

	h.errorHandler = opts.resolveErrorHandler()
	h.operationNameFunc = opts.operationNameFunc
	h.errorEncoders = opts.errorEncoders
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

func WithErrorEncoder(errorEncoder ErrorEncoder) HandlerOption {
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

type handlerOptions struct {
	errorHandler ErrorHandler
	// errorEncoder is responible for outputting the resulting error
	errorEncoders []ErrorEncoder

	operationNameFunc func(ctx context.Context) string
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
