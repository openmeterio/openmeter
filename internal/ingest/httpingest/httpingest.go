package httpingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/ingest"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Handler receives an event in CloudEvents format and forwards it to a {Collector}.
type Handler struct {
	config HandlerConfig
}

type HandlerConfig struct {
	Collector        ingest.Collector
	NamespaceManager *namespace.Manager
	Logger           *slog.Logger
}

func NewHandler(config HandlerConfig) (*Handler, error) {
	if config.Collector == nil {
		return nil, errors.New("collector is required")
	}
	if config.NamespaceManager == nil {
		return nil, errors.New("namespace manager is required")
	}

	handler := Handler{
		config: config,
	}

	return &handler, nil
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, namespace string) {
	logger := h.getLogger()

	contentType := r.Header.Get("Content-Type")

	var err error
	switch contentType {
	case "application/cloudevents+json":
		err = h.processSingleRequest(w, r, namespace)
	case "application/cloudevents-batch+json":
		err = h.processBatchRequest(w, r, namespace)
	default:
		// this should never happen
		err = errors.New("invalid content type: " + contentType)
	}

	if err != nil {
		logger.ErrorContext(r.Context(), "unable to process request", "error", err)

		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) processBatchRequest(w http.ResponseWriter, r *http.Request, namespace string) error {
	var events api.IngestEventsApplicationCloudeventsBatchPlusJSONBody

	err := json.NewDecoder(r.Body).Decode(&events)
	if err != nil {
		return fmt.Errorf("parsing event: %w", err)
	}

	for _, event := range events {
		err = h.processEvent(r.Context(), event, namespace)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h Handler) processSingleRequest(w http.ResponseWriter, r *http.Request, namespace string) error {
	var event api.IngestEventsApplicationCloudeventsPlusJSONRequestBody

	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		return fmt.Errorf("parsing event: %w", err)
	}

	err = h.processEvent(r.Context(), event, namespace)
	if err != nil {
		return err
	}

	return nil
}

func (h Handler) processEvent(ctx context.Context, event event.Event, namespace string) error {
	logger := h.getLogger()

	logger = logger.With(
		slog.String("event_id", event.ID()),
		slog.String("event_subject", event.Subject()),
		slog.String("event_source", event.Source()),
	)

	if event.Time().IsZero() {
		logger.DebugContext(ctx, "event does not have a timestamp")
		event.SetTime(time.Now().UTC())
	}

	err := h.config.Collector.Ingest(ctx, namespace, event)
	if err != nil {
		// TODO: attach context to error and log at a higher level
		logger.ErrorContext(ctx, "unable to forward event to collector", "error", err)

		return fmt.Errorf("forwarding event to collector: %w", err)
	}

	logger.DebugContext(ctx, "event forwarded to downstream collector")

	return nil
}

func (h Handler) getLogger() *slog.Logger {
	logger := h.config.Logger

	if logger == nil {
		logger = slog.Default()
	}

	return logger
}
