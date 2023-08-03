package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

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

func (a *Router) CreateNamespace(w http.ResponseWriter, r *http.Request) {
	namespace := &models.Namespace{}

	if a.config.NamespaceManager.IsManagementDisabled() {
		models.NewStatusProblem(r.Context(), errors.New("namespace management is disabled"), http.StatusForbidden).Respond(w, r)
		return
	}

	if err := render.DecodeJSON(r.Body, namespace); err != nil {
		models.NewStatusProblem(r.Context(), fmt.Errorf("cannot parse request body"), http.StatusBadRequest).Respond(w, r)
	}

	err := a.config.NamespaceManager.CreateNamespace(r.Context(), namespace.Namespace)
	if err != nil {
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, namespace)
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
	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}
	meter := &models.Meter{}

	if err := render.DecodeJSON(r.Body, meter); err != nil {
		models.NewStatusProblem(r.Context(), fmt.Errorf("cannot parse request body"), http.StatusBadRequest).Respond(w, r)
	}

	err := a.config.StreamingConnector.CreateMeter(r.Context(), namespace, meter)
	if err != nil {
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, meter)
}

func (a *Router) DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string, params api.DeleteMeterParams) {
	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}
	err := a.config.StreamingConnector.DeleteMeter(r.Context(), namespace, meterIdOrSlug)
	if err != nil {
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *Router) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string, params api.GetMeterParams) {
	for _, meter := range a.config.Meters {
		if meter.ID == meterIdOrSlug || meter.Slug == meterIdOrSlug {
			_ = render.Render(w, r, meter)
			return
		}
	}

	models.NewStatusProblem(r.Context(), fmt.Errorf("meter is not found with ID or slug %s", meterIdOrSlug), http.StatusNotFound).Respond(w, r)
}

type GetMeterValuesResponse struct {
	WindowSize *models.WindowSize   `json:"windowSize"`
	Data       []*models.MeterValue `json:"data"`
}

func (rd *GetMeterValuesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func ValidateGetMeterValuesParams(windowSizeIn models.WindowSize, params api.GetMeterValuesParams) error {
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
		if (windowSizeIn == models.WindowSizeDay && *params.WindowSize != models.WindowSizeDay) ||
			(windowSizeIn == models.WindowSizeHour && *params.WindowSize == models.WindowSizeMinute) {
			return fmt.Errorf("expected window size to be less than or equal to %s, but got %s", windowSizeIn, *params.WindowSize)
		}
	} else {
		windowDuration := windowSizeIn.Duration()
		if params.From != nil && params.From.Truncate(windowDuration) != *params.From {
			return fmt.Errorf("from must be aligned to the meter's window size of %s", windowSizeIn)
		}
		if params.To != nil && params.To.Truncate(windowDuration) != *params.To {
			return fmt.Errorf("to must be aligned to the meter's window size of %s", windowSizeIn)
		}
	}

	return nil
}

func (a *Router) GetMeterValues(w http.ResponseWriter, r *http.Request, meterIdOrSlug string, params api.GetMeterValuesParams) {
	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	if params.NamespaceInput != nil {
		namespace = *params.NamespaceInput
	}

	windowSizeIn := models.WindowSizeMinute

	// Set default window size if meter exists in confing
	for _, meter := range a.config.Meters {
		if meter.ID == meterIdOrSlug || meter.Slug == meterIdOrSlug {
			windowSizeIn = meter.WindowSize
		}
	}

	// Validate parameters
	if err := ValidateGetMeterValuesParams(windowSizeIn, params); err != nil {
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)
		return
	}

	values, err := a.config.StreamingConnector.QueryMeter(
		r.Context(),
		namespace,
		meterIdOrSlug,
		&streaming.GetValuesParams{
			From:       params.From,
			To:         params.To,
			Subject:    params.Subject,
			WindowSize: params.WindowSize,
		},
	)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			models.NewStatusProblem(r.Context(), err, http.StatusNotFound).Respond(w, r)
			return
		}

		slog.Error("error getting values", "err", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	resp := &GetMeterValuesResponse{
		WindowSize: params.WindowSize,
		Data:       values,
	}

	_ = render.Render(w, r, resp)
}
