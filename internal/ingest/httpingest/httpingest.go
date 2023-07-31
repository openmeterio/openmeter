package httpingest

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"golang.org/x/exp/slog"

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

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, params api.IngestEventsParams) {
	logger := h.getLogger()

	var event event.Event

	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		logger.ErrorCtx(r.Context(), "unable to parse event", "error", err)

		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
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

	namespace := h.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	err = h.config.Collector.Ingest(event, namespace)
	if err != nil {
		logger.ErrorCtx(r.Context(), "unable to forward event to collector", "error", err)

		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	logger.DebugCtx(r.Context(), "event forwarded to downstream collector")

	w.WriteHeader(http.StatusOK)
}

func (h Handler) getLogger() *slog.Logger {
	logger := h.config.Logger

	if logger == nil {
		logger = slog.Default()
	}

	return logger
}
