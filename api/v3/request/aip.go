package request

import (
	"net/http"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/filters"
)

// GetAipAttributes returns the AipAttributes parsed from the request query string.
// If strict mode is enabled via WithAipStrictMode and an invalid parameter is
// encountered, it returns a *apierrors.BaseAPIError.
func GetAipAttributes(r *http.Request, opts ...AipParseOption) (*AipAttributes, error) {
	a := &AipAttributes{}

	conf := newConfig()
	lo.ForEach(opts, func(v AipParseOption, _ int) { v(conf) })

	queryValues := r.URL.Query()

	pagination, err := extractPagination(r.Context(), queryValues, conf)
	if err != nil {
		return nil, err
	}
	a.Pagination = pagination

	filters, err := extractFilter(r.Context(), queryValues, conf)
	if err != nil {
		return nil, err
	}
	a.Filters = filters

	sort, sortErr := extractSort(queryValues, conf)
	if sortErr != nil {
		return nil, sortErr
	}

	a.Sorts = sort

	return a, nil
}

// RemapAipAttributes remaps the filters and sorts to another name
// this is used when API is not inlined with the database entities
func RemapAipAttributes(attrs *AipAttributes, mappedAttributes map[string]string) {
	lo.ForEach(attrs.Filters, func(f QueryFilter, k int) {
		attrs.Filters[k].Name = remapName(f.Name, mappedAttributes)
	})
	lo.ForEach(attrs.Sorts, func(s SortBy, k int) {
		attrs.Sorts[k].Field = remapName(s.Field, mappedAttributes)
	})
}

// remapName remaps a field name using the provided mapping.
// Supports dot-notation: "labels.env" is remapped as "mapped_labels.env" if "labels" is in the map.
func remapName(name string, mappedAttributes map[string]string) string {
	if mapped, ok := mappedAttributes[name]; ok {
		return mapped
	}
	parts := strings.SplitN(name, ".", 2) // allow known_custom_field.unknown_key
	mapped, ok := mappedAttributes[parts[0]]
	if ok && len(parts) == 2 {
		return mapped + "." + parts[1]
	}
	return name
}

type AipAttributes struct {
	Pagination Pagination
	Filters    []QueryFilter
	Sorts      []SortBy
}

type paginationKind int

const (
	paginationKindOffset paginationKind = iota
	paginationKindCursor
)

const (
	defaultPaginationMaxSize = 100
)

type config struct {
	strictMode          bool
	defaultPageSize     int
	maxPageSize         int
	paginationKind      paginationKind
	cursorValidateUUIDs bool
	cursorCipherKey     string
	defaultSort         *defaultSort
	authorizedFilters   AuthorizedFilters
	authorizedSorts     []string
	authorizedDotSorts  []string
}

func newConfig() *config {
	return &config{
		maxPageSize:         defaultPaginationMaxSize,
		cursorValidateUUIDs: false,
		cursorCipherKey:     DefaultCipherKey,
		strictMode:          false,
		defaultPageSize:     DefaultPaginationSize,
		paginationKind:      paginationKindOffset,
	}
}

type AipParseOption func(*config)

// WithAipStrictMode sets the parser a Strict, which means when some fallbackable
// arguments like page[size] or page[number] are invalid, the parser will return
// a 400 baseApiError instead of processing the request with default pagination size.
func WithAipStrictMode() AipParseOption {
	return func(c *config) {
		c.strictMode = true
	}
}

// WithCursorPagination sets the AIP request parser to only take the cursor AIP
// attributes in consideration and will ignore other kinds of paginations
func WithCursorPagination() AipParseOption {
	return func(c *config) {
		c.paginationKind = paginationKindCursor
	}
}

// WithOffsetPagination sets the AIP request parser to only take the offset AIP
// attributes (page[number], page[size]) in consideration and will ignore other
// kinds of paginations.
//
// This is the default parser behavior.
func WithOffsetPagination() AipParseOption {
	return func(c *config) {
		c.paginationKind = paginationKindOffset
	}
}

// WithDefaultPageSize sets the AIP request parser default page size.
// This value is used when the client is not setting the page[size] querystring
// or when the page[size] attribute is not valid and the parser is not using
// strict mode. Non-positive values are ignored.
//
// Default value is 20.
func WithDefaultPageSize(value int) AipParseOption {
	return func(c *config) {
		if value > 0 {
			c.defaultPageSize = value
		}
	}
}

// WithDefaultSort sets the default sort parameter if none is declared
// in the incoming request
func WithDefaultSort(field string, order SortOrder) AipParseOption {
	return func(c *config) {
		c.defaultSort = &defaultSort{
			field: field,
			order: order,
		}
	}
}

// WithCursorValidateUUIDs makes the AIP request parser to validate every UUID
// passed within a cursor in page[before] or page[after].
func WithCursorValidateUUIDs() AipParseOption {
	return func(c *config) {
		c.cursorValidateUUIDs = true
	}
}

// WithCursorCipherKey sets the cipher key used with the cursor pagination encoding
// and decoding methods
//
// by default the aip request parser uses the request.DefaultCipherKey value
func WithCursorCipherKey(key string) AipParseOption {
	return func(c *config) {
		c.cursorCipherKey = key
	}
}

// WithAuthorizedFilters defines the set of filters that the parser should parse
// other filters are ignored
//
// by default the parser takes all the filters that are passed to it
// Use the DotFilter parameter for filters that have unknown sub-attributes (filters[labels.key_1]=true)
func WithAuthorizedFilters(fields map[string]AIPFilterOption) AipParseOption {
	return func(c *config) {
		c.authorizedFilters = fields
	}
}

// WithAuthorizedSorts defines the set of allowed sort fields. Sorts on any
// field not in the provided list are ignored.
//
// By default the parser accepts all sort fields.
// Do not use dot notation (field.subfield) with this method; use WithAuthorizedDotSorts instead.
func WithAuthorizedSorts(fields []string) AipParseOption {
	return func(c *config) {
		c.authorizedSorts = fields
	}
}

// WithAuthorizedDotSorts is equivalent to WithAuthorizedSorts but allows
// sorting on user-defined sub-attributes.
//
// examples:
// "foo" allows ?sort=foo.bar or ?sort=foo.baz.
// "foo.bar" only allows ?sort=foo.bar.
// "foo" rejects ?sort=foo because it doesn't have a sub-attribute.
func WithAuthorizedDotSorts(fields []string) AipParseOption {
	return func(c *config) {
		c.authorizedDotSorts = fields
	}
}

// WithMaxPageSize defines the maximum size of the pagination.
// Non-positive values are ignored.
//
// Default value is 100.
func WithMaxPageSize(size int) AipParseOption {
	return func(c *config) {
		if size > 0 {
			c.maxPageSize = size
		}
	}
}

// ValidationFunc is a field-level validation callback. A non-nil error return
// indicates validation failure; the error message is included in the API error response.
type ValidationFunc func(field, value string) error

// AuthorizedFilters reprensents the map of fields that are authorized to be
// filtered on
type AuthorizedFilters map[string]AIPFilterOption

// AIPFilterOption defines the list of available filters for a giving field
// and its optional validation function
type AIPFilterOption struct {
	Filters        []QueryFilterOp
	ValidationFunc ValidationFunc
	DotFilter      bool
}

// FilterStringFromAip extracts a *filters.StringFilter for the named field from AIP query filters.
// Returns nil if no matching filters are found.
func FilterStringFromAip(queryFilters []QueryFilter, field string) *filters.StringFilter {
	matching := lo.Filter(queryFilters, func(qf QueryFilter, _ int) bool { return qf.Name == field })
	if len(matching) == 0 {
		return nil
	}

	var f filters.StringFilter
	lo.ForEach(matching, func(qf QueryFilter, _ int) {
		v := qf.Value
		switch qf.Filter {
		case QueryFilterEQ:
			f.Eq = &v
		case QueryFilterNEQ:
			f.Neq = &v
		case QueryFilterGT:
			f.Gt = &v
		case QueryFilterGTE:
			f.Gte = &v
		case QueryFilterLT:
			f.Lt = &v
		case QueryFilterLTE:
			f.Lte = &v
		case QueryFilterContains:
			f.Contains = &v
		case QueryFilterOrEQ:
			f.Oeq = &v
		case QueryFilterOrContains:
			f.Ocontains = &v
		case QueryFilterExists:
			t := true
			f.Exists = &t
		}
	})

	if f.IsEmpty() {
		return nil
	}
	return &f
}
