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

package errorsx

import (
	"context"
	"errors"
	"log/slog"
)

// Handler handles an error.
type Handler interface {
	Handle(err error)
	HandleContext(ctx context.Context, err error)
}

// AppHandler contains some custom logic to handle an error.
type AppHandler struct {
	Handler Handler
}

func NewAppHandler(handler Handler) AppHandler {
	return AppHandler{Handler: handler}
}

func (h AppHandler) Handle(err error) {
	// ignore context cancellation errors: they generally occur due to the client canceling the request
	if errors.Is(err, context.Canceled) {
		return
	}

	h.Handler.Handle(err)
}

func (h AppHandler) HandleContext(ctx context.Context, err error) {
	// ignore context cancellation errors: they generally occur due to the client canceling the request
	if errors.Is(err, context.Canceled) {
		return
	}

	h.Handler.HandleContext(ctx, err)
}

// SlogHandler is a Handler that logs errors using slog.
type SlogHandler struct {
	Logger *slog.Logger
}

func NewSlogHandler(logger *slog.Logger) SlogHandler {
	return SlogHandler{Logger: logger}
}

func (h SlogHandler) Handle(err error) {
	h.Logger.Error(err.Error())
}

func (h SlogHandler) HandleContext(ctx context.Context, err error) {
	h.Logger.ErrorContext(ctx, err.Error())
}

// NopHandler ignores all errors.
type NopHandler struct{}

func (h NopHandler) Handle(err error) {
}

func (h NopHandler) HandleContext(ctx context.Context, err error) {
}

// ContextHandler always accepts a context.
type ContextHandler struct {
	Handler Handler
}

func NewContextHandler(handler Handler) ContextHandler {
	return ContextHandler{Handler: handler}
}

func (h ContextHandler) Handle(ctx context.Context, err error) {
	h.Handler.HandleContext(ctx, err)
}

func (h ContextHandler) HandleContext(ctx context.Context, err error) {
	h.Handler.HandleContext(ctx, err)
}
