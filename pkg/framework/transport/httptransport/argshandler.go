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

	options ...HandlerOption,
) HandlerWithArgs[Request, Response, ArgType] {
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
