package openmeter

import (
	"context"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const metersBasePath = "/openmeter/meters"

// defaultListPageSize is the page size ListAll requests when the caller does not
// specify one, chosen to keep round-trips low without oversized responses.
const defaultListPageSize = 100

// MetersService groups the meter operations. Access it via Client.Meters.
type MetersService struct {
	client *Client
}

// StringFilter expresses a comparison on a string field. Set exactly one
// operator; unset fields are omitted from the request. It mirrors the API's
// FilterString type. (The spec also allows a bare string shorthand for equality
// on query-string filters, e.g. filter[key]=value; that is a server-side parse
// convenience, not a distinct JSON shape — use Eq for the same effect.)
type StringFilter struct {
	Eq        *string  `json:"eq,omitempty"`
	Neq       *string  `json:"neq,omitempty"`
	Gt        *string  `json:"gt,omitempty"`
	Gte       *string  `json:"gte,omitempty"`
	Lt        *string  `json:"lt,omitempty"`
	Lte       *string  `json:"lte,omitempty"`
	Contains  *string  `json:"contains,omitempty"`
	Oeq       []string `json:"oeq,omitempty"`
	Ocontains []string `json:"ocontains,omitempty"`
	Exists    *bool    `json:"$exists,omitempty"`
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
	return func(yield func(Meter, error) bool) {
		page, size := 1, defaultListPageSize
		if params.Page != nil {
			if params.Page.Number != nil {
				page = *params.Page.Number
			}
			if params.Page.Size != nil {
				size = *params.Page.Size
			}
		}

		seen := 0
		for {
			pageParams := params
			pageParams.Page = &PageParams{Size: Int(size), Number: Int(page)}

			resp, err := s.List(ctx, pageParams)
			if err != nil {
				yield(Meter{}, err)
				return
			}

			for _, m := range resp.Data {
				if !yield(m, nil) {
					return
				}
			}

			seen += len(resp.Data)

			// Stop on an empty page (nothing more to fetch) or once we have seen
			// the reported total. The total guard is skipped when the server
			// reports a non-positive total so a bad count can't end paging early.
			if len(resp.Data) == 0 {
				return
			}
			if resp.Meta.Page.Total > 0 && seen >= resp.Meta.Page.Total {
				return
			}

			page++
		}
	}
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
