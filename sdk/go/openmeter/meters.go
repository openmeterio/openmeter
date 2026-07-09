package openmeter

import (
	"context"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const metersBasePath = "/openmeter/meters"

// MeterAggregation is the aggregation type a meter applies to matched events.
//
// It is a plain string so it stays forward compatible: a value the server
// introduces after this SDK was built decodes into the string unchanged instead
// of erroring, and re-encodes as-is. Compare against the constants below and
// handle unrecognized values with a default case rather than assuming the set is
// closed.
type MeterAggregation string

const (
	MeterAggregationSum         MeterAggregation = "sum"
	MeterAggregationCount       MeterAggregation = "count"
	MeterAggregationUniqueCount MeterAggregation = "unique_count"
	MeterAggregationAvg         MeterAggregation = "avg"
	MeterAggregationMin         MeterAggregation = "min"
	MeterAggregationMax         MeterAggregation = "max"
	MeterAggregationLatest      MeterAggregation = "latest"
)

// Meter is a configuration that defines how to match and aggregate events.
type Meter struct {
	ID          string           `json:"id"`
	Key         string           `json:"key"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Aggregation MeterAggregation `json:"aggregation"`
	EventType   string           `json:"event_type"`
	// EventsFrom, when set, is the date from which the meter includes events.
	EventsFrom *time.Time `json:"events_from,omitempty"`
	// ValueProperty is a JSONPath expression extracting the aggregated value from
	// the event data. Ignored for count aggregation.
	ValueProperty *string `json:"value_property,omitempty"`
	// Dimensions maps group-by dimension names to JSONPath expressions.
	Dimensions map[string]string `json:"dimensions,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	DeletedAt  *time.Time        `json:"deleted_at,omitempty"`
}

// MeterPagePaginatedResponse is a page of meters plus pagination metadata.
type MeterPagePaginatedResponse struct {
	Data []Meter       `json:"data"`
	Meta PaginatedMeta `json:"meta"`
}

// MeterQueryGranularity is the size of the time buckets a query groups usage into.
//
// Like MeterAggregation it is a plain string for forward compatibility: an
// unrecognized value round-trips through decode and encode unchanged rather than
// erroring. Compare against the constants below and handle unknown values with a
// default case.
type MeterQueryGranularity string

const (
	MeterQueryGranularityMinute MeterQueryGranularity = "PT1M"
	MeterQueryGranularityHour   MeterQueryGranularity = "PT1H"
	MeterQueryGranularityDay    MeterQueryGranularity = "P1D"
	MeterQueryGranularityMonth  MeterQueryGranularity = "P1M"
)

// QueryFilterStringMapItem is a per-dimension filter in a meter query. For the
// reserved subject and customer_id dimensions only Eq/In are supported.
type QueryFilterStringMapItem struct {
	Eq        *string  `json:"eq,omitempty"`
	Neq       *string  `json:"neq,omitempty"`
	In        []string `json:"in,omitempty"`
	Nin       []string `json:"nin,omitempty"`
	Contains  *string  `json:"contains,omitempty"`
	Ncontains *string  `json:"ncontains,omitempty"`
	Exists    *bool    `json:"exists,omitempty"`
}

// MeterQueryFilters filters a meter query by dimension values.
type MeterQueryFilters struct {
	Dimensions map[string]QueryFilterStringMapItem `json:"dimensions,omitempty"`
}

// MeterQueryRequest is the POST body for querying a meter for usage.
type MeterQueryRequest struct {
	From        *time.Time             `json:"from,omitempty"`
	To          *time.Time             `json:"to,omitempty"`
	Granularity *MeterQueryGranularity `json:"granularity,omitempty"`
	// TimeZone is an IANA Time Zone Database name used to align time buckets.
	// Defaults to UTC when unset.
	TimeZone          *string            `json:"time_zone,omitempty"`
	GroupByDimensions []string           `json:"group_by_dimensions,omitempty"`
	Filters           *MeterQueryFilters `json:"filters,omitempty"`
}

// MeterQueryRow is one aggregated bucket of a meter query result.
type MeterQueryRow struct {
	Value Numeric   `json:"value"`
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	// Dimensions holds the group-by values for this row. subject and customer_id
	// are reserved dimension keys.
	Dimensions map[string]string `json:"dimensions"`
}

// MeterQueryResult is the JSON result of a meter query.
type MeterQueryResult struct {
	From *time.Time      `json:"from,omitempty"`
	To   *time.Time      `json:"to,omitempty"`
	Data []MeterQueryRow `json:"data"`
}

// CreateMeterRequest is the body for creating a meter.
type CreateMeterRequest struct {
	Name        string            `json:"name"`
	Key         string            `json:"key"`
	Aggregation MeterAggregation  `json:"aggregation"`
	EventType   string            `json:"event_type"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	// EventsFrom, when set, is the date from which the meter includes events.
	EventsFrom *time.Time `json:"events_from,omitempty"`
	// ValueProperty is a JSONPath expression extracting the aggregated value from
	// the event data. Ignored for count aggregation.
	ValueProperty *string `json:"value_property,omitempty"`
	// Dimensions maps group-by dimension names to JSONPath expressions.
	Dimensions map[string]string `json:"dimensions,omitempty"`
}

// UpdateMeterRequest is the body for updating a meter. All fields are optional;
// only the set ones are sent.
type UpdateMeterRequest struct {
	Name        *string           `json:"name,omitempty"`
	Description *string           `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Dimensions  map[string]string `json:"dimensions,omitempty"`
}

// MetersService groups the meter operations. Access it via Client.Meters.
type MetersService struct {
	client *Client
}

// MeterFilter narrows a meter listing by key and/or name.
type MeterFilter struct {
	Key  *StringFilter
	Name *StringFilter
}

// MeterListParams are the optional query parameters for listing meters. The
// zero value lists the first default page unfiltered and unsorted.
type MeterListParams struct {
	Page *PageParams
	// Sort holds one or more sort attributes (e.g. "name", "created_at desc").
	// They are joined into a single comma-separated `sort` query parameter.
	Sort   []string
	Filter *MeterFilter
}

// values serializes the params into a query string using the three styles the
// list-meters endpoint declares (deepObject page, form sort, deepObject filter).
// It returns an error if a filter value cannot be represented in the query
// encoding (see joinFilterList).
func (p MeterListParams) values() (url.Values, error) {
	q := url.Values{}

	addPageParams(q, p.Page)

	if len(p.Sort) > 0 {
		q.Set("sort", strings.Join(p.Sort, ","))
	}

	if p.Filter != nil {
		if err := addStringFilter(q, "filter[key]", p.Filter.Key); err != nil {
			return nil, err
		}
		if err := addStringFilter(q, "filter[name]", p.Filter.Name); err != nil {
			return nil, err
		}
	}

	return q, nil
}

// Get retrieves a single meter by its ULID.
func (s *MetersService) Get(ctx context.Context, meterID string) (*Meter, error) {
	path, err := resourcePath(metersBasePath, meterID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil, nil, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out Meter
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Create creates a meter and returns the created resource (HTTP 201).
func (s *MetersService) Create(ctx context.Context, request CreateMeterRequest) (*Meter, error) {
	req, err := s.client.newRequest(ctx, http.MethodPost, metersBasePath, nil, request, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out Meter
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Update replaces a meter by ID and returns the updated resource.
func (s *MetersService) Update(ctx context.Context, meterID string, request UpdateMeterRequest) (*Meter, error) {
	path, err := resourcePath(metersBasePath, meterID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodPut, path, nil, request, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out Meter
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Delete removes a meter by ID. It returns nil on success (HTTP 204 No Content).
func (s *MetersService) Delete(ctx context.Context, meterID string) error {
	path, err := resourcePath(metersBasePath, meterID)
	if err != nil {
		return err
	}

	req, err := s.client.newRequest(ctx, http.MethodDelete, path, nil, nil, contentTypeJSON)
	if err != nil {
		return err
	}

	_, err = s.client.doRaw(req)
	return err
}

// List returns a page of meters, applying the pagination, sort, and filter
// parameters as query-string arguments.
func (s *MetersService) List(ctx context.Context, params MeterListParams) (*MeterPagePaginatedResponse, error) {
	query, err := params.values()
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, metersBasePath, query, nil, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out MeterPagePaginatedResponse
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// ListAll returns an iterator over every meter matching params, transparently
// fetching successive pages. Range over it with Go 1.23+ range-over-func:
//
//	for meter, err := range client.Meters.ListAll(ctx, params) {
//		if err != nil {
//			return err
//		}
//		// use meter
//	}
//
// On a page fetch error the iterator yields one (zero-value, err) pair and
// stops. Any Page in params seeds the starting page and page size; ListAll then
// advances the page number itself. Breaking out of the loop stops paging.
func (s *MetersService) ListAll(ctx context.Context, params MeterListParams) iter.Seq2[Meter, error] {
	return paginate(params.Page, func(page, size int) ([]Meter, int, error) {
		pageParams := params
		pageParams.Page = &PageParams{Size: Int(size), Number: Int(page)}

		resp, err := s.List(ctx, pageParams)
		if err != nil {
			return nil, 0, err
		}

		return resp.Data, resp.Meta.Page.Total, nil
	})
}

// Query runs a usage query against a meter and returns the structured JSON
// result. Use QueryCSV for the CSV representation of the same data.
func (s *MetersService) Query(ctx context.Context, meterID string, request MeterQueryRequest) (*MeterQueryResult, error) {
	path, err := resourcePath(metersBasePath, meterID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodPost, path+"/query", nil, request, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out MeterQueryResult
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// QueryCSV runs the same usage query as Query but negotiates the CSV
// representation (Accept: text/csv) and returns the CSV bytes. The response is
// buffered in memory and capped; for large exports use QueryCSVStream instead.
func (s *MetersService) QueryCSV(ctx context.Context, meterID string, request MeterQueryRequest) ([]byte, error) {
	path, err := resourcePath(metersBasePath, meterID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodPost, path+"/query", nil, request, contentTypeCSV)
	if err != nil {
		return nil, err
	}

	return s.client.doRaw(req)
}

// QueryCSVStream is like QueryCSV but returns the CSV response as a stream,
// letting the caller process large exports without buffering the whole payload
// in memory. The caller must close the returned reader.
func (s *MetersService) QueryCSVStream(ctx context.Context, meterID string, request MeterQueryRequest) (io.ReadCloser, error) {
	path, err := resourcePath(metersBasePath, meterID)
	if err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodPost, path+"/query", nil, request, contentTypeCSV)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.doStream(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}
