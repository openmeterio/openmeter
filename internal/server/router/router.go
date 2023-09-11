package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func init() {
	// See https://github.com/getkin/kin-openapi/issues/640
	openapi3filter.RegisterBodyDecoder("application/cloudevents+json", jsonBodyDecoder)
	openapi3filter.RegisterBodyDecoder("application/cloudevents-batch+json", jsonBodyDecoder)
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

	queryParams := &streaming.QueryParams{
		From:           params.From,
		To:             params.To,
		Aggregation:    params.Aggregation,
		WindowSize:     params.WindowSize,
		GroupBySubject: true,
	}

	// Moved here to preserve backward compatibility
	// We now allow aggregating data without grouping by window size,
	// but this endpoint currently returns data groupped by window size even if none is passed.
	if params.WindowSize == nil {
		windowSize := models.WindowSizeMinute
		params.WindowSize = &windowSize
	}

	if params.Subject != nil {
		queryParams.Subject = append(queryParams.Subject, *params.Subject)
	}

	// TODO: if we change OpenAPI type to array of strings it doesn't parse correctly
	if params.GroupBy != nil {
		queryParams.GroupBy = strings.Split(*params.GroupBy, ",")
	}

	result, err := a.config.StreamingConnector.QueryMeter(r.Context(), namespace, meterIdOrSlug, queryParams)
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
		WindowSize: result.WindowSize,
		Data:       result.Values,
	}

	_ = render.Render(w, r, resp)
}

// QueryMeter queries the values stored for a meter.
func (a *Router) QueryMeter(w http.ResponseWriter, r *http.Request, meterIDOrSlug string, params api.QueryMeterParams) {
	logger := slog.With("operation", "queryMeter", "id", meterIDOrSlug, "params", params)

	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	// Set defaults if meter is found in static config and params are not set
	for _, meter := range a.config.Meters {
		if meter.ID == meterIDOrSlug || meter.Slug == meterIDOrSlug {
			if params.Aggregation == nil {
				params.Aggregation = &meter.Aggregation
			}
		}
	}

	// Validate parameters
	if err := validateQueryMeterParams(params); err != nil {
		logger.Warn("invalid parameters", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
		return
	}

	queryParams := &streaming.QueryParams{
		From:        params.From,
		To:          params.To,
		Aggregation: params.Aggregation,
		WindowSize:  params.WindowSize,
	}

	if params.Subject != nil {
		queryParams.Subject = *params.Subject
	}

	if params.GroupBy != nil {
		queryParams.GroupBy = *params.GroupBy
	}

	result, err := a.config.StreamingConnector.QueryMeter(r.Context(), namespace, meterIDOrSlug, queryParams)
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

	resp := &QueryMeterResponse{
		WindowSize: result.WindowSize,
		From:       params.From,
		To:         params.To,
		Data: slicesx.Map(result.Values, func(val *models.MeterValue) models.MeterQueryRow {
			row := models.MeterQueryRow{
				Value:   val.Value,
				GroupBy: val.GroupBy,
			}

			if val.Subject != "" {
				row.Subject = &val.Subject
			}

			if !val.WindowStart.IsZero() {
				row.WindowStart = &val.WindowStart
			}

			if !val.WindowEnd.IsZero() {
				row.WindowEnd = &val.WindowEnd
			}

			return row
		}),
	}

	_ = render.Render(w, r, resp)
}

func validateQueryMeterParams(params api.QueryMeterParams) error {
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

// QueryMeterResponse is returned by the QueryMeter endpoint.
type QueryMeterResponse struct {
	WindowSize *models.WindowSize     `json:"windowSize,omitempty"`
	From       *time.Time             `json:"from,omitempty"`
	To         *time.Time             `json:"to,omitempty"`
	Data       []models.MeterQueryRow `json:"data"`
}

// Render implements the chi renderer interface.
func (resp QueryMeterResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// ListMeterSubjects lists the subjects of a meter.
func (a *Router) ListMeterSubjects(w http.ResponseWriter, r *http.Request, meterIDOrSlug string, params api.ListMeterSubjectsParams) {
	logger := slog.With("operation", "listMeterSubjects", "id", meterIDOrSlug, "params", params)

	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	subjects, err := a.config.StreamingConnector.ListMeterSubjects(r.Context(), namespace, meterIDOrSlug)
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

	render.JSON(w, r, subjects)
}
