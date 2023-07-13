// Copyright © 2023 Tailfin Cloud Inc.
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

package httpingest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/go-chi/render"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/api"
)

// Handler receives an event in CloudEvents format and forwards it to a {Collector}.
type Handler struct {
	Collector Collector

	Logger *slog.Logger
}

// Collector is a receiver of events that handles sending those events to some downstream broker.
type Collector interface {
	Receive(ev event.Event) error
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.getLogger()

	var event event.Event

	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		logger.ErrorCtx(r.Context(), "unable to parse event", "error", err)

		_ = render.Render(w, r, api.ErrInternalServerError(err))

		return
	}

	logger = logger.With(
		slog.String("event_id", event.ID()),
		slog.String("event_subject", event.Subject()),
		slog.String("event_source", event.Source()),
	)

	if event.Time().IsZero() {
		logger.DebugCtx(r.Context(), "event does not have a timestamp")

		event.SetTime(time.Now().UTC())
	}

	err = h.Collector.Receive(event)
	if err != nil {
		logger.ErrorCtx(r.Context(), "unable to forward event to collector", "error", err)

		_ = render.Render(w, r, api.ErrInternalServerError(err))

		return
	}

	logger.InfoCtx(r.Context(), "event forwarded to downstream collector")

	w.WriteHeader(http.StatusOK)
}

func (h Handler) getLogger() *slog.Logger {
	logger := h.Logger

	if logger == nil {
		logger = slog.Default()
	}

	return logger
}
