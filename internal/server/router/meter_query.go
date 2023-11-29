package router

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

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

	// Query meter
	resp, problem := a.QueryMeterWithMeter(r.Context(), logger, meter, params)
	if problem != nil {
		problem.Respond(w, r)
		return
	}

	// Render response
	err = a.QueryMeterRenderByAcceptHeader(w, r, logger, resp)
	if err != nil {
		logger.Error("rendering response", "error", err)
	}
}

// QueryMeter queries the values stored for a meter.
func (a *Router) QueryMeterWithMeter(ctx context.Context, logger *slog.Logger, meter models.Meter, params api.QueryMeterParams) (*QueryMeterResponse, models.Problem) {
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
			// Validate group by, `subject` is a special group by
			if ok := groupBy == "subject" || meter.GroupBy[groupBy] != ""; !ok {
				err := fmt.Errorf("invalid group by: %s", groupBy)
				logger.Warn("invalid group by", "error", err)
				return nil, models.NewStatusProblem(ctx, err, http.StatusBadRequest)
			}

			queryParams.GroupBy = append(queryParams.GroupBy, groupBy)
		}
	}

	if params.WindowTimeZone != nil {
		tz, err := time.LoadLocation(*params.WindowTimeZone)
		if err != nil {
			logger.Warn("invalid time zone", "error", err)
			return nil, models.NewStatusProblem(ctx, err, http.StatusBadRequest)
		}
		queryParams.WindowTimeZone = tz
	}

	if err := queryParams.Validate(meter.WindowSize); err != nil {
		logger.Warn("invalid parameters", "error", err)
		return nil, models.NewStatusProblem(ctx, err, http.StatusBadRequest)
	}

	// Query connector
	data, err := a.config.StreamingConnector.QueryMeter(ctx, meter.Namespace, meter.Slug, queryParams)
	if err != nil {
		logger.Error("connector", "error", err)
		return nil, models.NewStatusProblem(ctx, err, http.StatusInternalServerError)
	}

	resp := &QueryMeterResponse{
		WindowSize: params.WindowSize,
		From:       params.From,
		To:         params.To,
		Data:       data,
		// Fields Required for CSV Render but not present in response
		meterSlug:   meter.Slug,
		groupByKeys: queryParams.GroupBy,
	}

	// If total data is queried for a period, replace the window start and end with the period for each row
	if params.WindowSize == nil {
		for i := range resp.Data {
			if params.From != nil {
				resp.Data[i].WindowStart = *params.From
			}
			if params.To != nil {
				resp.Data[i].WindowEnd = *params.To
			}
		}
	}

	return resp, nil
}

type IQueryMeterResponse interface {
	RenderCSV(w http.ResponseWriter, r *http.Request)
	RenderJSON(w http.ResponseWriter, r *http.Request) error
	Render(w http.ResponseWriter, r *http.Request) error
}

// QueryMeterResponse is returned by the QueryMeter endpoint.
type QueryMeterResponse struct {
	WindowSize  *models.WindowSize     `json:"windowSize,omitempty"`
	From        *time.Time             `json:"from,omitempty"`
	To          *time.Time             `json:"to,omitempty"`
	Data        []models.MeterQueryRow `json:"data"`
	meterSlug   string                 `json:"-"`
	groupByKeys []string               `json:"-"`
}

// Render renders content based on the Accept header.
func (a *Router) QueryMeterRenderByAcceptHeader(w http.ResponseWriter, r *http.Request, logger *slog.Logger, resp IQueryMeterResponse) error {
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
		resp.RenderCSV(w, r)
		return nil
	}
	return resp.RenderJSON(w, r)
}

// Render implements the chi renderer interface.
func (resp QueryMeterResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// RenderJSON renders the response as CSV.
func (resp QueryMeterResponse) RenderJSON(w http.ResponseWriter, r *http.Request) error {
	return render.Render(w, r, resp)
}

// RenderCSV renders the response as CSV.
func (resp QueryMeterResponse) RenderCSV(w http.ResponseWriter, r *http.Request) {
	records := [][]string{}

	// CSV headers
	headers := []string{"window_start", "window_end", "subject"}
	if len(resp.groupByKeys) > 0 {
		headers = append(headers, resp.groupByKeys...)
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
		for _, k := range resp.groupByKeys {
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
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", resp.meterSlug))

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
