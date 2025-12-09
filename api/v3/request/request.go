package request

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request/query"
)

type config struct {
	defaultPageSize int
	maxPageSize     int
	paginationKind  paginationKind
	defaultSort     *SortBy
	parseOptions    *query.ParseOptions
}

func newConfig() *config {
	return &config{
		paginationKind:  paginationKindOffset,
		maxPageSize:     DefaultPaginationMaxSize,
		defaultPageSize: DefaultPaginationSize,
		parseOptions: &query.ParseOptions{
			Comma: true,
		},
	}
}

type QueryAttributes struct {
	Pagination Pagination        `query:"page"`
	Filters    map[string]Filter `query:"filter"`
	Sorts      []SortBy          `query:"sort"`
}

type AttributesOption func(*config)

// WithCursorPagination sets the attributes parser to only take the cursor
// attributes in consideration and will ignore other kinds of paginations.
//
// This is the default behavior.
func WithCursorPagination() AttributesOption {
	return func(c *config) {
		c.paginationKind = paginationKindCursor
	}
}

// WithCursorPagination sets the  request parser to only take the offset
// attributes in consideration and will ignore other kinds of paginations.
func WithOffsetPagination() AttributesOption {
	return func(c *config) {
		c.paginationKind = paginationKindOffset
	}
}

// WithDefaultPageSizeDefault sets the  request parser default page size.
// This value is used when the client is not setting the page[size] querystring
// or when the page[size] attribute is not valid.
//
// Default value is 20
func WithDefaultPageSizeDefault(value int) AttributesOption {
	return func(c *config) {
		c.defaultPageSize = value
	}
}

// WithDefaultSort sets the default sort order for the attributes parser.
// This value is used when the client is not setting the sort querystring
// or when the sort attribute is not valid.
//
// Default value is nil
func WithDefaultSort(sort *SortBy) AttributesOption {
	return func(c *config) {
		c.defaultSort = sort
	}
}

// WithQueryParseOptions overrides the default query parse options used when
// parsing request attributes.
func WithQueryParseOptions(opts *query.ParseOptions) AttributesOption {
	return func(c *config) {
		c.parseOptions = opts
	}
}

// GetAttributes return the Attributes found in the request query string
func GetAttributes(r *http.Request, opts ...AttributesOption) (*QueryAttributes, error) {
	conf := newConfig()
	for _, v := range opts {
		v(conf)
	}

	a := &QueryAttributes{
		Pagination: Pagination{
			kind:   conf.paginationKind,
			Size:   conf.defaultPageSize,
			Number: 1,
		},
	}

	if conf.defaultSort != nil {
		a.Sorts = []SortBy{*conf.defaultSort}
	}

	err := query.Unmarshal(r.Context(), r.URL.RawQuery, a, conf.parseOptions)
	if err != nil {
		return nil, err
	}

	if err := a.Pagination.Validate(); err != nil {
		if errors.Is(err, ErrCursorPaginationSizeInvalid) {
			return nil, apierrors.NewBadRequestError(r.Context(), err,
				apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page[size]",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				},
			)
		}

		if errors.Is(err, ErrCursorPaginationRange) {
			return nil, apierrors.NewBadRequestError(r.Context(), err,
				apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page[after]",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
					apierrors.InvalidParameter{
						Field:  "page[before]",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				},
			)
		}

		if errors.Is(err, ErrCursorPaginationAfterInvalid) {
			return nil, apierrors.NewBadRequestError(r.Context(), err,
				apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page[after]",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				},
			)
		}

		if errors.Is(err, ErrCursorPaginationBeforeInvalid) {
			return nil, apierrors.NewBadRequestError(r.Context(), err,
				apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page[before]",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				},
			)
		}

		return nil, apierrors.NewBadRequestError(r.Context(), err,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Field:  "page",
					Reason: "unable to parse query field",
					Source: apierrors.InvalidParamSourceQuery,
				},
			},
		)
	}

	if a.Pagination.Size > conf.maxPageSize {
		return nil, apierrors.NewBadRequestError(r.Context(), err,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Field:  "page[size]",
					Reason: fmt.Sprintf("page size must be less than or equal to %d", conf.maxPageSize),
					Source: apierrors.InvalidParamSourceQuery,
				},
			},
		)
	}

	for _, sort := range a.Sorts {
		if err := sort.Validate(); err != nil {
			if errors.Is(err, ErrSortFieldRequired) || errors.Is(err, ErrSortOrderInvalid) || errors.Is(err, ErrSortByInvalid) {
				return nil, apierrors.NewBadRequestError(r.Context(), err,
					apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					},
				)
			}
		}
	}

	for key, filter := range a.Filters {
		if err := filter.Validate(); err != nil {
			return nil, apierrors.NewBadRequestError(r.Context(), err,
				apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  fmt.Sprintf("filter[%s]", key),
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				},
			)
		}
	}

	return a, nil
}
