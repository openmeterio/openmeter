package request

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

// ErrReturnEmptySet this error is used as underlying error as a signal to HandleAPIError that it should return an empty set.
var ErrReturnEmptySet = errors.New("should return an empty set")

var (
	filterMap = map[string]QueryFilterOp{
		"oeq":       QueryFilterOrEQ,
		"eq":        QueryFilterEQ,
		"neq":       QueryFilterNEQ,
		"gt":        QueryFilterGT,
		"gte":       QueryFilterGTE,
		"lt":        QueryFilterLT,
		"lte":       QueryFilterLTE,
		"contains":  QueryFilterContains,
		"ocontains": QueryFilterOrContains,
		"exists":    QueryFilterExists,
	}
	ErrUnallowedFilterColumn = errors.New("unallowed filtering column")
	ErrUnallowedFilterMethod = errors.New("unallowed filtering method")
)

func filterName(value QueryFilterOp) string {
	for k, v := range filterMap {
		if v == value {
			return k
		}
	}
	return ""
}

const (
	FilterQuery = "filter"

	// filter[field][eq]
	QueryFilterEQ QueryFilterOp = iota
	// filter[field][neq]
	QueryFilterNEQ
	// filter[field][gt]
	QueryFilterGT
	// filter[field][gte]
	QueryFilterGTE
	// filter[field][lt]
	QueryFilterLT
	// filter[field][lte]
	QueryFilterLTE
	// filter[field][contains]
	QueryFilterContains
	// filter[field]
	QueryFilterExists
	// filter[field][oeq]
	QueryFilterOrEQ
	// filter[field][ocontains]
	QueryFilterOrContains
)

var (
	// lookup to only focus filter[foo] and not filterfoo[bar]
	prefixLookup = FilterQuery + "["
)

// QueryFilter column filter
type QueryFilter struct {
	Name   string
	Path   *string
	Value  string
	Values []string
	Filter QueryFilterOp
}

type QueryFilterOp int

func extractFilter(ctx context.Context, qs url.Values, c *config) ([]QueryFilter, *apierrors.BaseAPIError) {
	var out []QueryFilter

	for i, v := range qs {
		if !strings.HasPrefix(i, prefixLookup) {
			continue
		}
		for _, filter := range v {
			o, err := parseFilterQs(ctx, filter, i)
			if err != nil {
				return nil, err
			}

			// no field name provided is an invalid query filter
			if o.Name == "" {
				continue
			}

			// if there is value that means we're falling back on
			// EXIST query filter
			if filter == "" {
				o.Filter = QueryFilterExists
			}

			o.Value = filter

			checkFilters := c.authorizedFilters != nil
			var ok bool
			var filters AIPFilterOption

			if checkFilters && strings.ContainsRune(o.Name, '.') {
				parts := strings.SplitN(o.Name, ".", 2) // allow filters[known_custom_field.unknown_key]
				filters, ok = c.authorizedFilters[parts[0]]
				if !ok {
					filters, ok = c.authorizedFilters[o.Name] // specific case where only 1 field is allowed
				}
				ok = ok && filters.DotFilter
			} else if checkFilters {
				filters, ok = c.authorizedFilters[o.Name]
				ok = ok && !filters.DotFilter // forbid using whole field for dot filters
			}

			if checkFilters {
				if !ok {
					if c.strictMode {
						return nil, apierrors.NewBadRequestError(ctx, ErrUnallowedFilterMethod,
							apierrors.InvalidParameters{
								apierrors.InvalidParameter{
									Field:  o.Name,
									Reason: "unauthorized filter",
									Source: apierrors.InvalidParamSourceQuery,
									Rule:   "unknown_property",
								},
							})
					}
					continue
				}
				if !slices.Contains(filters.Filters, o.Filter) {
					if c.strictMode {
						return nil, apierrors.NewBadRequestError(ctx, ErrUnallowedFilterColumn,
							apierrors.InvalidParameters{
								apierrors.InvalidParameter{
									Field:  filterName(o.Filter),
									Reason: "unauthorized filter on column",
									Source: apierrors.InvalidParamSourceQuery,
									Rule:   "unknown_property",
								},
							})
					}
					continue
				}
				if filters.ValidationFunc != nil {
					if err := filters.ValidationFunc(o.Name, o.Value); err != nil {
						if errors.Is(err, ErrReturnEmptySet) {
							// for errors in uuid format, we want to handle it by returning an empty list.
							return nil, apierrors.NewEmptySetResponse(ctx, c.paginationKind == paginationKindCursor)
						}
						return nil, apierrors.NewBadRequestError(ctx, ErrUnallowedFilterColumn,
							apierrors.InvalidParameters{
								apierrors.InvalidParameter{
									Field:  filter,
									Reason: err.Error(),
									Source: apierrors.InvalidParamSourceQuery,
									Rule:   "unauthorized filter on column",
								},
							})
					}
				}
			}

			if o.Filter == QueryFilterOrEQ || o.Filter == QueryFilterOrContains {
				o.Values = parseMultipleStringValues(o.Value)
			}
			out = append(out, o)
		}
	}

	return out, nil
}

func parseMultipleStringValues(strValue string) []string {
	var out []string
	for _, v := range strings.Split(strValue, ",") {
		out = append(out, strings.TrimSpace(v))
	}
	return out
}

func parseFilterQs(ctx context.Context, filter, qs string) (QueryFilter, *apierrors.BaseAPIError) {
	o := QueryFilter{}
	i := strings.IndexRune(qs, '[')
	if i == -1 {
		return o, nil
	}

	endFirst := strings.IndexRune(qs, ']')
	if endFirst == -1 {
		return o, nil
	}
	o.Filter = QueryFilterEQ
	o.Name = qs[i+1 : endFirst]

	qsRest := qs[endFirst+1:]

	if len(qsRest) > 0 {
		start := strings.IndexRune(qsRest, '[')
		end := strings.IndexRune(qsRest, ']')
		op := qsRest[start+1 : end]
		if len(op) > 0 {
			if queryOp, ok := filterMap[op]; ok {
				o.Filter = queryOp
			} else {
				return QueryFilter{}, apierrors.NewBadRequestError(ctx, ErrUnallowedFilterColumn,
					apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  filter,
							Reason: fmt.Sprintf("invalid operation '%s' on filter", op),
							Source: apierrors.InvalidParamSourceQuery,
							Rule:   "unauthorized filter on column",
						},
					})
			}
		}
	}

	return o, nil
}
