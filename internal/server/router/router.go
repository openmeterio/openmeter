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
	ServeHTTP(w http.ResponseWriter, r *http.Request, namespace string)
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

func (a *Router) ListMeters(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("operation", "listMeters")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

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

func (a *Router) CreateMeter(w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("not implemented: manage meters via config or checkout OpenMeter Cloud")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	err := fmt.Errorf("not implemented: manage meters via config or checkout OpenMeter Cloud")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	logger := slog.With("operation", "getMeter", "id", meterIdOrSlug)
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

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

// QueryMeter queries the values stored for a meter.
func (a *Router) QueryMeter(w http.ResponseWriter, r *http.Request, meterIDOrSlug string, params api.QueryMeterParams) {
	logger := slog.With("operation", "queryMeter", "id", meterIDOrSlug, "params", params)
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get meter
	meter, err := a.config.Meters.GetMeterByIDOrSlug(r.Context(), namespace, meterIDOrSlug)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			logger.Warn("meter not found", "error", err)
			models.NewStatusProblem(r.Context(), err, http.StatusNotFound).Respond(w, r)
			return
		}

		logger.Error("get meter", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	a.QueryMeterWithMeter(w, r, logger, meter, params)
}

// QueryMeter queries the values stored for a meter.
func (a *Router) QueryMeterWithMeter(w http.ResponseWriter, r *http.Request, logger *slog.Logger, meter models.Meter, params api.QueryMeterParams) {
	// Query Params
	queryParams := &streaming.QueryParams{
		From:        params.From,
		To:          params.To,
		WindowSize:  params.WindowSize,
		Aggregation: meter.Aggregation,
	}

	if params.Subject != nil {
		queryParams.Subject = *params.Subject
	}

	if params.GroupBy != nil {
		for _, groupBy := range *params.GroupBy {
			// Subject is a special group by
			if groupBy == "subject" {
				queryParams.GroupBySubject = true
				continue
			}

			// Validate group by
			if ok := meter.GroupBy[groupBy] != ""; !ok {
				err := fmt.Errorf("invalid group by: %s", groupBy)
				logger.Warn("invalid group by", "error", err)
				models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
				return
			}

			queryParams.GroupBy = append(queryParams.GroupBy, groupBy)
		}
	}

	if params.WindowTimeZone != nil {
		tz, err := time.LoadLocation(*params.WindowTimeZone)
		if err != nil {
			logger.Warn("invalid time zone", "error", err)
			models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
			return
		}
		queryParams.WindowTimeZone = tz
	}

	if err := queryParams.Validate(meter.WindowSize); err != nil {
		logger.Warn("invalid parameters", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
		return
	}

	// Query connector
	data, err := a.config.StreamingConnector.QueryMeter(r.Context(), meter.Namespace, meter.Slug, queryParams)
	if err != nil {
		logger.Error("connector", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	resp := &QueryMeterResponse{
		WindowSize: params.WindowSize,
		From:       params.From,
		To:         params.To,
		Data:       data,
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
		resp.RenderCSV(w, r, queryParams.GroupBy, meter.Slug)
	} else {
		_ = render.Render(w, r, resp)
	}
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
			var groupByValue string

			if row.GroupBy[k] != nil {
				groupByValue = *row.GroupBy[k]
			}
			data = append(data, groupByValue)
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
func (a *Router) ListMeterSubjects(w http.ResponseWriter, r *http.Request, meterIDOrSlug string) {
	logger := slog.With("operation", "listMeterSubjects", "id", meterIDOrSlug)
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	subjects, err := a.config.StreamingConnector.ListMeterSubjects(r.Context(), namespace, meterIDOrSlug, nil, nil)
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
