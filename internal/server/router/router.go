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

	"github.com/openmeterio/openmeter/api"
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

type Config struct {
	StreamingConnector streaming.Connector
	IngestHandler      http.Handler
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

func (a *Router) IngestEvents(w http.ResponseWriter, r *http.Request) {
	a.config.IngestHandler.ServeHTTP(w, r)
}

func (a *Router) ListMeters(w http.ResponseWriter, r *http.Request) {
	list := make([]render.Renderer, 0, len(a.config.Meters))
	for _, m := range a.config.Meters {
		list = append(list, m)
	}

	_ = render.RenderList(w, r, list)
}

func (a *Router) CreateMeter(w http.ResponseWriter, r *http.Request) {
	_ = render.Render(w, r, models.NewStatusProblem(r.Context(), nil, http.StatusMethodNotAllowed))
}

func (a *Router) DeleteMeter(w http.ResponseWriter, r *http.Request, meterSlug string) {
	_ = render.Render(w, r, models.NewStatusProblem(r.Context(), nil, http.StatusMethodNotAllowed))
}

func (a *Router) GetMeter(w http.ResponseWriter, r *http.Request, meterSlug string) {
	for _, meter := range a.config.Meters {
		if meter.Slug == meterSlug {
			_ = render.Render(w, r, meter)
			return
		}
	}

	_ = render.Render(w, r, models.NewStatusProblem(r.Context(), fmt.Errorf("meter is not found with slug %s", meterSlug), http.StatusNotFound))
}

type GetMeterValuesResponse struct {
	WindowSize *models.WindowSize   `json:"windowSize"`
	Data       []*models.MeterValue `json:"data"`
}

func (rd *GetMeterValuesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func ValidateGetMeterValuesParams(meter *models.Meter, params api.GetMeterValuesParams) error {
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
		if (meter.WindowSize == models.WindowSizeDay && *params.WindowSize != models.WindowSizeDay) ||
			(meter.WindowSize == models.WindowSizeHour && *params.WindowSize == models.WindowSizeMinute) {
			return fmt.Errorf("expected window size to be less than or equal to %s, but got %s", meter.WindowSize, *params.WindowSize)
		}
	} else {
		windowDuration := meter.WindowSize.Duration()
		if params.From != nil && params.From.Truncate(windowDuration) != *params.From {
			return fmt.Errorf("from must be aligned to the meter's window size of %s", meter.WindowSize)
		}
		if params.To != nil && params.To.Truncate(windowDuration) != *params.To {
			return fmt.Errorf("to must be aligned to the meter's window size of %s", meter.WindowSize)
		}
	}

	return nil
}

func (a *Router) GetMeterValues(w http.ResponseWriter, r *http.Request, meterSlug string, params api.GetMeterValuesParams) {
	for _, meter := range a.config.Meters {
		if meter.Slug == meterSlug {
			if err := ValidateGetMeterValuesParams(meter, params); err != nil {
				_ = render.Render(w, r, models.NewStatusProblem(r.Context(), err, http.StatusBadRequest))
				return
			}

			values, err := a.config.StreamingConnector.GetValues(meter, &streaming.GetValuesParams{
				From:       params.From,
				To:         params.To,
				Subject:    params.Subject,
				WindowSize: params.WindowSize,
			})
			if err != nil {
				_ = render.Render(w, r, models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError))
				return
			}

			windowSize := params.WindowSize
			resp := &GetMeterValuesResponse{
				WindowSize: windowSize,
				Data:       values,
			}

			_ = render.Render(w, r, resp)
			return
		}
	}

	_ = render.Render(w, r, models.NewStatusProblem(r.Context(), fmt.Errorf("meter is not found with slug %s", meterSlug), http.StatusNotFound))
}
