package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/render"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func init() {
	// See https://github.com/getkin/kin-openapi/issues/640
	openapi3filter.RegisterBodyDecoder("application/cloudevents+json", jsonBodyDecoder)
}

func jsonBodyDecoder(body io.Reader, header http.Header, schema *openapi3.SchemaRef, encFn openapi3filter.EncodingFn) (interface{}, error) {
	var value interface{}
	if err := json.NewDecoder(body).Decode(&value); err != nil {
		return nil, &openapi3filter.ParseError{Kind: openapi3filter.KindInvalidFormat, Cause: err}
	}
	return value, nil
}

type IngestHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, params api.IngestEventsParams)
}

type Config struct {
	NamespaceManager   *namespace.Manager
	StreamingConnector streaming.Connector
	IngestHandler      IngestHandler
	Meters             []*models.Meter
}

type Router struct {
	config Config
}

// Make sure we conform to ServerInterface
var _ api.ServerInterface = (*Router)(nil)

func NewRouter(config Config) (*Router, error) {
	return &Router{
		config: config,
	}, nil
}

// CreateNamespace handles the HTTP request for creating a new namespace.
func (a *Router) CreateNamespace(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("operation", "createNamespace")

	if a.config.NamespaceManager.IsManagementDisabled() {
		logger.Warn("namespace management is disabled")
		models.NewStatusProblem(r.Context(), errors.New("namespace management is disabled"), http.StatusForbidden).Respond(w, r)

		return
	}

	var namespace api.Namespace
	if err := render.DecodeJSON(r.Body, &namespace); err != nil {
		logger.Warn("cannot parse request body", "error", err)
		models.NewStatusProblem(r.Context(), fmt.Errorf("cannot parse request body"), http.StatusBadRequest).Respond(w, r)

		return
	}

	err := a.config.NamespaceManager.CreateNamespace(r.Context(), namespace.Namespace)
	if err != nil {
		logger.Error("connector", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (a *Router) IngestEvents(w http.ResponseWriter, r *http.Request, params api.IngestEventsParams) {
	a.config.IngestHandler.ServeHTTP(w, r, params)
}

func (a *Router) ListMeters(w http.ResponseWriter, r *http.Request, params api.ListMetersParams) {
	list := make([]render.Renderer, 0, len(a.config.Meters))
	for _, m := range a.config.Meters {
		list = append(list, m)
	}

	_ = render.RenderList(w, r, list)
}

func (a *Router) CreateMeter(w http.ResponseWriter, r *http.Request, params api.CreateMeterParams) {
	logger := slog.With("operation", "createMeter")

	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	var meter models.Meter
	if err := render.DecodeJSON(r.Body, &meter); err != nil {
		logger.Warn("cannot parse request body", "error", err)
		models.NewStatusProblem(r.Context(), fmt.Errorf("cannot parse request body"), http.StatusBadRequest).Respond(w, r)
	}

	if err := meter.Validate(); err != nil {
		logger.Warn("invalid meter", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
		return
	}

	err := a.config.StreamingConnector.CreateMeter(r.Context(), namespace, &meter)
	if err != nil {
		logger.Error("connector", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, &meter)
}

func (a *Router) DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string, params api.DeleteMeterParams) {
	logger := slog.With("operation", "deleteMeter", "id", meterIdOrSlug, "params", params)

	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}
	err := a.config.StreamingConnector.DeleteMeter(r.Context(), namespace, meterIdOrSlug)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			logger.Warn("meter not found")
			models.NewStatusProblem(r.Context(), err, http.StatusNotFound).Respond(w, r)
			return
		}

		logger.Error("connector", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *Router) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string, params api.GetMeterParams) {
	logger := slog.With("operation", "getMeter", "id", meterIdOrSlug, "params", params)

	for _, meter := range a.config.Meters {
		if meter.ID == meterIdOrSlug || meter.Slug == meterIdOrSlug {
			_ = render.Render(w, r, meter)
			return
		}
	}

	logger.Warn("meter not found")
	models.NewStatusProblem(r.Context(), fmt.Errorf("meter is not found with ID or slug %s", meterIdOrSlug), http.StatusNotFound).Respond(w, r)
}

type GetMeterValuesResponse struct {
	WindowSize *models.WindowSize   `json:"windowSize"`
	Data       []*models.MeterValue `json:"data"`
}

func (rd *GetMeterValuesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func ValidateGetMeterValuesParams(params api.GetMeterValuesParams) error {
	if params.From != nil && params.To != nil && params.From.After(*params.To) {
		return errors.New("from must be before to")
	}

	if params.WindowSize != nil {
		windowDuration := params.WindowSize.Duration()
		if params.From != nil && params.From.Truncate(windowDuration) != *params.From {
			return errors.New("from must be aligned to window size")
		}
		if params.To != nil && params.To.Truncate(windowDuration) != *params.To {
			return errors.New("to must be aligned to window size")
		}
	}

	return nil
}

func (a *Router) GetMeterValues(w http.ResponseWriter, r *http.Request, meterIdOrSlug string, params api.GetMeterValuesParams) {
	logger := slog.With("operation", "getMeterValues", "id", meterIdOrSlug, "params", params)

	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	// Set defaults if meter is found in static config and params are not set
	for _, meter := range a.config.Meters {
		if meter.ID == meterIdOrSlug || meter.Slug == meterIdOrSlug {
			if params.Aggregation == nil {
				params.Aggregation = &meter.Aggregation
			}

			if params.WindowSize == nil {
				params.WindowSize = &meter.WindowSize
			}
		}
	}

	// Validate parameters
	if err := ValidateGetMeterValuesParams(params); err != nil {
		logger.Warn("invalid parameters", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
		return
	}

	// TODO: if we change OpenAPI type to array of strings it doesn't parse correctly
	var groupBy *[]string
	if params.GroupBy != nil {
		tmp := strings.Split(*params.GroupBy, ",")
		groupBy = &tmp
	}

	values, windowSize, err := a.config.StreamingConnector.QueryMeter(
		r.Context(),
		namespace,
		meterIdOrSlug,
		&streaming.QueryParams{
			From:        params.From,
			To:          params.To,
			Subject:     params.Subject,
			GroupBy:     groupBy,
			Aggregation: params.Aggregation,
			WindowSize:  params.WindowSize,
		},
	)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			logger.Warn("meter not found", "error", err)
			models.NewStatusProblem(r.Context(), err, http.StatusNotFound).Respond(w, r)
			return
		}

		logger.Error("connector", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	resp := &GetMeterValuesResponse{
		WindowSize: windowSize,
		Data:       values,
	}

	_ = render.Render(w, r, resp)
}
