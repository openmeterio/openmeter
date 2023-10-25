package router

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/meter"
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
	Meters             meter.Repository
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

func (a *Router) ListEvents(w http.ResponseWriter, r *http.Request, params api.ListEventsParams) {
	logger := slog.With("operation", "queryEvents")

	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	limit := 100
	if params.Limit != nil {
		limit = *params.Limit
	}
	if limit < 1 {
		err := errors.New("limit must be greater than or equal to 1")
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
		return
	}
	if limit > 100 {
		err := errors.New("limit must be less than or equal to 100")
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
		return
	}

	queryParams := streaming.ListEventsParams{
		Limit: limit,
	}

	events, err := a.config.StreamingConnector.ListEvents(r.Context(), namespace, queryParams)
	if err != nil {
		if _, ok := err.(*models.NamespaceNotFoundError); ok {
			logger.Warn("namespace not found", "error", err)
			models.NewStatusProblem(r.Context(), err, http.StatusNotFound).Respond(w, r)
			return
		}

		logger.Error("query events", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return

	}

	render.JSON(w, r, events)
}

func (a *Router) ListMeters(w http.ResponseWriter, r *http.Request, params api.ListMetersParams) {
	logger := slog.With("operation", "listMeters")

	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	meters, err := a.config.Meters.ListMeters(r.Context(), namespace)
	if err != nil {
		logger.Error("listing meters", "error", err)

		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)

		return
	}

	// TODO: remove once meter model pointer is removed
	list := slicesx.Map[models.Meter, render.Renderer](meters, func(meter models.Meter) render.Renderer {
		return &meter
	})

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
			logger.Warn("meter not found", "error", err)
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
	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	logger := slog.With("operation", "getMeter", "id", meterIdOrSlug, "namespace", namespace)

	meter, err := a.config.Meters.GetMeterByIDOrSlug(r.Context(), namespace, meterIdOrSlug)

	// TODO: remove once meter model pointer is removed
	if e := (&models.MeterNotFoundError{}); errors.As(err, &e) {
		logger.Debug("meter not found")

		// TODO: add meter id or slug as detail
		models.NewStatusProblem(r.Context(), errors.New("meter not found"), http.StatusNotFound).Respond(w, r)

		return
	} else if err != nil {
		logger.Error("getting meter", slog.Any("error", err))

		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)

		return
	}

	// TODO: remove once meter model pointer is removed
	_ = render.Render(w, r, &meter)
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
	meter, err := a.config.Meters.GetMeterByIDOrSlug(r.Context(), namespace, meterIdOrSlug)
	if err != nil { // TODO: proper error handling
		if params.Aggregation == nil {
			params.Aggregation = &meter.Aggregation
		}

		if params.WindowSize == nil {
			params.WindowSize = &meter.WindowSize
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
	meter, err := a.config.Meters.GetMeterByIDOrSlug(r.Context(), namespace, meterIDOrSlug)
	if err != nil { // TODO: proper error handling
		if params.Aggregation == nil {
			params.Aggregation = &meter.Aggregation
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
				Value:       val.Value,
				WindowStart: val.WindowStart,
				WindowEnd:   val.WindowEnd,
				GroupBy:     val.GroupBy,
			}

			if val.Subject != "" {
				row.Subject = &val.Subject
			}

			return row
		}),
	}

	// Parse media type
	accept := r.Header.Get("Accept")
	if accept == "" {
		accept = "application/json"
	}
	mediatype, _, err := mime.ParseMediaType(accept)
	// Browser can send back media type Go marks as invalid
	// If that happens, default to JSON
	if err != nil {
		logger.Debug("invalid media type, default to json", "error", err)
		mediatype = "application/json"
	}

	if mediatype == "text/csv" {
		resp.RenderCSV(w, r, queryParams.GroupBy, meterIDOrSlug)
	} else {
		_ = render.Render(w, r, resp)
	}
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

// RenderCSV renders the response as CSV.
func (resp QueryMeterResponse) RenderCSV(w http.ResponseWriter, r *http.Request, groupByKeys []string, meterIDOrSlug string) {
	records := [][]string{}

	// CSV headers
	headers := []string{"window_start", "window_end", "subject"}
	if len(groupByKeys) > 0 {
		headers = append(headers, groupByKeys...)
	}
	headers = append(headers, "value")
	records = append(records, headers)

	// CSV data
	for _, row := range resp.Data {
		data := []string{row.WindowStart.Format(time.RFC3339), row.WindowEnd.Format(time.RFC3339)}
		if row.Subject != nil {
			data = append(data, *row.Subject)
		} else {
			data = append(data, "")
		}
		for _, k := range groupByKeys {
			data = append(data, row.GroupBy[k])
		}
		data = append(data, fmt.Sprintf("%f", row.Value))
		records = append(records, data)
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", meterIDOrSlug))

	// Write response
	writer := csv.NewWriter(w)
	err := writer.WriteAll(records)
	if err != nil {
		slog.Error("writing record to csv", "error", err)
	}

	if err := writer.Error(); err != nil {
		slog.Error("writing csv", "error", err)
	}
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
