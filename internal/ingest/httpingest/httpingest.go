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
		logger.ErrorCtx(r.Context(), "unable to forward event to sink", "error", err)

		_ = render.Render(w, r, api.ErrInternalServerError(err))

		return
	}

	logger.InfoCtx(r.Context(), "event forwarded to downstream sink")

	w.WriteHeader(http.StatusOK)
}

func (h Handler) getLogger() *slog.Logger {
	logger := h.Logger

	if logger == nil {
		logger = slog.Default()
	}

	return logger
}
