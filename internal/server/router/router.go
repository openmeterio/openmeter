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

func (a *Router) GetMeters(w http.ResponseWriter, r *http.Request) {
	if err := render.RenderList(w, r, NewMeterListResponse(a.config.Meters)); err != nil {
		_ = render.Render(w, r, api.ErrUnprocessableEntity(err))
		return
	}
}

func NewMeterListResponse(meters []*models.Meter) []render.Renderer {
	list := make([]render.Renderer, 0, len(meters))
	for _, m := range meters {
		list = append(list, m)
	}
	return list
}

func (a *Router) GetMetersById(w http.ResponseWriter, r *http.Request, meterID string) {
	println(meterID)
	for _, meter := range a.config.Meters {
		if meter.ID == meterID {
			if err := render.Render(w, r, meter); err != nil {
				_ = render.Render(w, r, api.ErrUnprocessableEntity(err))
			}
			return
		}
	}

	_ = render.Render(w, r, api.ErrNotFound)
}

type GetValuesByMeterIdResponse struct {
	WindowSize *models.WindowSize   `json:"windowSize"`
	Data       []*models.MeterValue `json:"data"`
}

func (rd *GetValuesByMeterIdResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func ValidateGetValuesByMeterIdParams(meter *models.Meter, params api.GetValuesByMeterIdParams) error {
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
		if (meter.Aggregation == models.MeterAggregationCountDistinct) && *params.WindowSize != meter.WindowSize {
			return fmt.Errorf("expected window size to be %s, but got %s", meter.WindowSize, *params.WindowSize)
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
		if meter.Aggregation == models.MeterAggregationCountDistinct {
			return fmt.Errorf("expected window size to be %s", meter.WindowSize)
		}
	}

	return nil
}

func (a *Router) GetValuesByMeterId(w http.ResponseWriter, r *http.Request, meterId string, params api.GetValuesByMeterIdParams) {
	for _, meter := range a.config.Meters {
		if meter.ID == meterId {
			if err := ValidateGetValuesByMeterIdParams(meter, params); err != nil {
				_ = render.Render(w, r, api.ErrBadRequest(err))
				return
			}

			values, err := a.config.StreamingConnector.GetValues(meter, &streaming.GetValuesParams{
				From:       params.From,
				To:         params.To,
				Subject:    params.Subject,
				WindowSize: params.WindowSize,
			})
			if err != nil {
				_ = render.Render(w, r, api.ErrInternalServerError(err))
				return
			}

			windowSize := params.WindowSize
			resp := &GetValuesByMeterIdResponse{
				WindowSize: windowSize,
				Data:       values,
			}

			if err := render.Render(w, r, resp); err != nil {
				_ = render.Render(w, r, api.ErrUnprocessableEntity(err))
				return
			}

			return
		}
	}

	_ = render.Render(w, r, api.ErrNotFound)
}
