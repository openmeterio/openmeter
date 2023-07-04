package httpingest

import (
	"context"
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

	Logger  *slog.Logger
	Context context.Context
}

// Collector is a receiver of events that handles sending those events to some downstream broker.
type Collector interface {
	Receive(ev event.Event) error
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.getLogger()
	h.Context = r.Context()

	contentType := r.Header.Get("Content-Type")

	if contentType == "application/cloudevents-batch+json" {
		err := h.processBatchRequest(w, r)
		if err != nil {
			logger.ErrorCtx(h.Context, "unable to process batch request", "error", err)
			_ = render.Render(w, r, api.ErrInternalServerError(err))
			return
		}
	} else {
		err := h.processSingleRequest(w, r)
		if err != nil {
			logger.ErrorCtx(h.Context, "unable to process single request", "error", err)
			_ = render.Render(w, r, api.ErrInternalServerError(err))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h Handler) processBatchRequest(w http.ResponseWriter, r *http.Request) error {
	var events []event.Event

	err := json.NewDecoder(r.Body).Decode(&events)
	if err != nil {
		return err
	}

	for _, event := range events {
		err = h.processEvent(event)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h Handler) processSingleRequest(w http.ResponseWriter, r *http.Request) error {
	var event event.Event

	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		return err
	}

	err = h.processEvent(event)
	if err != nil {
		return err
	}

	return nil
}

func (h Handler) processEvent(event event.Event) error {
	logger := h.getLogger()

	logger = logger.With(
		slog.String("event_id", event.ID()),
		slog.String("event_subject", event.Subject()),
		slog.String("event_source", event.Source()),
	)

	if event.Time().IsZero() {
		logger.DebugCtx(h.Context, "event does not have a timestamp")
		event.SetTime(time.Now().UTC())
	}

	err := h.Collector.Receive(event)
	if err != nil {
		return err
	}

	logger.InfoCtx(h.Context, "event forwarded to downstream collector")
	return nil
}

func (h Handler) getLogger() *slog.Logger {
	logger := h.Logger

	if logger == nil {
		logger = slog.Default()
	}

	return logger
}
