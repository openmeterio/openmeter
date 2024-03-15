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

package flushhandler

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/sink/models"
)

type DrainCompleteFunc func()

type FlushEventHandlers struct {
	handlers        []FlushEventHandler
	onDrainComplete []DrainCompleteFunc
}

func NewFlushEventHandlers() *FlushEventHandlers {
	return &FlushEventHandlers{}
}

func (f *FlushEventHandlers) AddHandler(handler FlushEventHandler) {
	f.handlers = append(f.handlers, handler)
}

func (f *FlushEventHandlers) OnDrainComplete(fn DrainCompleteFunc) {
	f.onDrainComplete = append(f.onDrainComplete, fn)
}

func (f *FlushEventHandlers) OnFlushSuccess(ctx context.Context, events []models.SinkMessage) error {
	var finalError error

	for _, handler := range f.handlers {
		if err := handler.OnFlushSuccess(ctx, events); err != nil {
			finalError = errors.Join(finalError, err)
		}
	}

	return finalError
}

func (f *FlushEventHandlers) Start(ctx context.Context) error {
	for _, handler := range f.handlers {
		if err := handler.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (f *FlushEventHandlers) WaitForDrain(ctx context.Context) error {
	for _, handler := range f.handlers {
		if err := handler.WaitForDrain(ctx); err != nil {
			return err
		}
	}

	for _, fn := range f.onDrainComplete {
		fn()
	}

	return nil
}
