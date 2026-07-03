package openmeter

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const metersBasePath = "/openmeter/meters"

// MetersService groups the meter operations. Access it via Client.Meters.
type MetersService struct {
	client *Client
}

// StringFilter expresses a comparison on a string field. Set exactly one
// operator; unset fields are omitted from the request.
type StringFilter struct {
	Eq       *string `json:"eq,omitempty"`
	Neq      *string `json:"neq,omitempty"`
	Contains *string `json:"contains,omitempty"`
	Gt       *string `json:"gt,omitempty"`
	Gte      *string `json:"gte,omitempty"`
	Lt       *string `json:"lt,omitempty"`
	Lte      *string `json:"lte,omitempty"`
	Exists   *bool   `json:"exists,omitempty"`
}

// MeterFilter narrows a meter listing by key and/or name.
type MeterFilter struct {
	Key  *StringFilter
	Name *StringFilter
}

// PageParams selects a page of a paginated listing.
type PageParams struct {
	// Size is the number of items per page.
	Size *int
	// Number is the 1-based page number.
	Number *int
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
func (p MeterListParams) values() url.Values {
	q := url.Values{}

	if p.Page != nil {
		if p.Page.Size != nil {
			setDeepObjectString(q, "page", "size", strconv.Itoa(*p.Page.Size))
		}
		if p.Page.Number != nil {
			setDeepObjectString(q, "page", "number", strconv.Itoa(*p.Page.Number))
		}
	}

	if len(p.Sort) > 0 {
		q.Set("sort", strings.Join(p.Sort, ","))
	}

	if p.Filter != nil {
		addStringFilter(q, "filter[key]", p.Filter.Key)
		addStringFilter(q, "filter[name]", p.Filter.Name)
	}

	return q
}

// Get retrieves a single meter by its ULID.
func (s *MetersService) Get(ctx context.Context, meterID string) (*Meter, error) {
	req, err := s.client.newRequest(ctx, http.MethodGet, metersBasePath+"/"+url.PathEscape(meterID), nil, nil, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out Meter
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns a page of meters, applying the pagination, sort, and filter
// parameters as query-string arguments.
func (s *MetersService) List(ctx context.Context, params MeterListParams) (*MeterPagePaginatedResponse, error) {
	req, err := s.client.newRequest(ctx, http.MethodGet, metersBasePath, params.values(), nil, contentTypeJSON)
	if err != nil {
		return nil, err
	}

	var out MeterPagePaginatedResponse
	if err := s.client.doJSON(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Query runs a usage query against a meter and returns the structured JSON
// result. Use QueryCSV for the CSV representation of the same data.
func (s *MetersService) Query(ctx context.Context, meterID string, request MeterQueryRequest) (*MeterQueryResult, error) {
	req, err := s.client.newRequest(ctx, http.MethodPost, metersBasePath+"/"+url.PathEscape(meterID)+"/query", nil, request, contentTypeJSON)
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
// representation (Accept: text/csv) and returns the raw CSV bytes.
func (s *MetersService) QueryCSV(ctx context.Context, meterID string, request MeterQueryRequest) ([]byte, error) {
	req, err := s.client.newRequest(ctx, http.MethodPost, metersBasePath+"/"+url.PathEscape(meterID)+"/query", nil, request, contentTypeCSV)
	if err != nil {
		return nil, err
	}
	return s.client.doRaw(req)
}
