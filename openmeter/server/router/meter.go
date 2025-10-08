package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	httpdriver "github.com/openmeterio/openmeter/openmeter/meter/httphandler"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
)

// GET /api/v1/meters
func (a *Router) ListMeters(w http.ResponseWriter, r *http.Request, params api.ListMetersParams) {
	a.meterHandler.ListMeters().With(params).ServeHTTP(w, r)
}

// GET /api/v1/meters/{meterIdOrSlug}
func (a *Router) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	a.meterHandler.GetMeter().With(meterIdOrSlug).ServeHTTP(w, r)
}

// POST /api/v1/meters/
func (a *Router) CreateMeter(w http.ResponseWriter, r *http.Request) {
	a.meterHandler.CreateMeter().ServeHTTP(w, r)
}

// PUT /api/v1/meters/{meterIdOrSlug}
func (a *Router) UpdateMeter(w http.ResponseWriter, r *http.Request, meterID string) {
	a.meterHandler.UpdateMeter().With(meterID).ServeHTTP(w, r)
}

// DELETE /api/v1/meters/{meterIdOrSlug}
func (a *Router) DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	a.meterHandler.DeleteMeter().With(meterIdOrSlug).ServeHTTP(w, r)
}

// GET /api/v1/meters/{meterIdOrSlug}/query
func (a *Router) QueryMeter(w http.ResponseWriter, r *http.Request, meterIDOrSlug string, params api.QueryMeterParams) {
	// Construct handler params
	handlerParams := httpdriver.QueryMeterParams{
		IdOrSlug:         meterIDOrSlug,
		QueryMeterParams: params,
	}

	// Get media type
	mediatype, err := commonhttp.GetMediaType(r)
	if err != nil {
		a.config.Logger.DebugContext(r.Context(), "invalid media type", "error", err)
	}

	// CSV
	if mediatype == "text/csv" {
		a.meterHandler.QueryMeterCSV().With(handlerParams).ServeHTTP(w, r)
		return
	}

	// JSON is the default
	a.meterHandler.QueryMeter().With(handlerParams).ServeHTTP(w, r)
}

// POST /api/v1/meters/{meterIdOrSlug}/query
func (a *Router) QueryMeterPost(w http.ResponseWriter, r *http.Request, meterIDOrSlug string) {
	a.meterHandler.QueryMeterPost().With(meterIDOrSlug).ServeHTTP(w, r)
}

// GET /api/v1/meters/{meterIdOrSlug}/subjects
func (a *Router) ListMeterSubjects(w http.ResponseWriter, r *http.Request, meterIDOrSlug string, params api.ListMeterSubjectsParams) {
	a.meterHandler.ListSubjects().With(httpdriver.ListSubjectsParams{
		IdOrSlug: meterIDOrSlug,
		From:     params.From,
		To:       params.To,
	}).ServeHTTP(w, r)
}

// GET /api/v1/meters/{meterIdOrSlug}/group-by/{groupByKey}/values
func (a *Router) ListMeterGroupByValues(w http.ResponseWriter, r *http.Request, meterIDOrSlug string, groupByKey string, params api.ListMeterGroupByValuesParams) {
	a.meterHandler.ListGroupByValues().With(httpdriver.ListGroupByValuesParams{
		IdOrSlug:   meterIDOrSlug,
		GroupByKey: groupByKey,
		From:       params.From,
		To:         params.To,
	}).ServeHTTP(w, r)
}
