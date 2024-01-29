package router

import (
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
			err := fmt.Errorf("meter not found: %w", err)
			models.NewStatusProblem(r.Context(), err, http.StatusNotFound).Respond(logger, w, r)
			return
		}

		err := fmt.Errorf("get meter: %w", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(logger, w, r)
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
			// Validate group by, `subject` is a special group by
			if ok := groupBy == "subject" || meter.GroupBy[groupBy] != ""; !ok {
				err := fmt.Errorf("invalid group by: %s", groupBy)
				models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(logger, w, r)
				return
			}

			queryParams.GroupBy = append(queryParams.GroupBy, groupBy)
		}
	}

	if params.WindowTimeZone != nil {
		tz, err := time.LoadLocation(*params.WindowTimeZone)
		if err != nil {
			err := fmt.Errorf("invalid time zone: %w", err)
			models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(logger, w, r)
			return
		}
		queryParams.WindowTimeZone = tz
	}

	if err := queryParams.Validate(meter.WindowSize); err != nil {
		err := fmt.Errorf("invalid query parameters: %w", err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(logger, w, r)
		return
	}

	// Query connector
	data, err := a.config.StreamingConnector.QueryMeter(r.Context(), meter.Namespace, meter.Slug, queryParams)
	if err != nil {
		err := fmt.Errorf("query meter: %w", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(logger, w, r)
		return
	}

	resp := &QueryMeterResponse{
		WindowSize: params.WindowSize,
		From:       params.From,
		To:         params.To,
		Data:       data,
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
