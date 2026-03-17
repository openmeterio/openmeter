package request

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/samber/lo"

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
	key, _ := lo.FindKey(filterMap, value)
	return key
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

// lookup to only focus filter[foo] and not filterfoo[bar]
var prefixLookup = FilterQuery + "["

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
	type filterEntry struct{ key, value string }

	entries := lo.FlatMap(lo.Entries(qs), func(e lo.Entry[string, []string], _ int) []filterEntry {
		if !strings.HasPrefix(e.Key, prefixLookup) {
			return nil
		}
		return lo.Map(e.Value, func(v string, _ int) filterEntry { return filterEntry{e.Key, v} })
	})

	var out []QueryFilter
	for _, e := range entries {
		qf, skip, err := processFilter(ctx, e.value, e.key, c)
		if err != nil {
			return nil, err
		}
		if !skip {
			out = append(out, qf)
		}
	}
	return out, nil
}

func processFilter(ctx context.Context, filter, key string, c *config) (QueryFilter, bool, *apierrors.BaseAPIError) {
	o, err := parseFilterQs(ctx, filter, key)
	if err != nil {
		return QueryFilter{}, false, err
	}
	if o.Name == "" {
		return QueryFilter{}, true, nil
	}
	if filter == "" {
		o.Filter = QueryFilterExists
	}
	o.Value = filter

	skip, apiErr := checkFilterAuthorization(ctx, o, filter, c)
	if apiErr != nil {
		return QueryFilter{}, false, apiErr
	}
	if skip {
		return QueryFilter{}, true, nil
	}

	if o.Filter == QueryFilterOrEQ || o.Filter == QueryFilterOrContains {
		o.Values = parseMultipleStringValues(o.Value)
	}
	return o, false, nil
}

func resolveAuthorizedFilter(name string, authorizedFilters AuthorizedFilters) (AIPFilterOption, bool) {
	if !strings.ContainsRune(name, '.') {
		opt, ok := authorizedFilters[name]
		return opt, ok && !opt.DotFilter
	}
	parts := strings.SplitN(name, ".", 2) // allow filters[known_custom_field.unknown_key]
	if opt, ok := authorizedFilters[parts[0]]; ok && opt.DotFilter {
		return opt, true
	}
	opt, ok := authorizedFilters[name] // specific case where only 1 field is allowed
	return opt, ok && opt.DotFilter
}

func checkFilterAuthorization(ctx context.Context, o QueryFilter, rawFilter string, c *config) (bool, *apierrors.BaseAPIError) {
	if c.authorizedFilters == nil {
		return false, nil
	}

	authorizedOpt, ok := resolveAuthorizedFilter(o.Name, c.authorizedFilters)
	if !ok && c.strictMode {
		return false, apierrors.NewBadRequestError(ctx, ErrUnallowedFilterMethod,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Field:  o.Name,
					Reason: "unauthorized filter",
					Source: apierrors.InvalidParamSourceQuery,
					Rule:   "unknown_property",
				},
			})
	}
	if !ok {
		return true, nil
	}

	filterAllowed := slices.Contains(authorizedOpt.Filters, o.Filter)
	if !filterAllowed && c.strictMode {
		return false, apierrors.NewBadRequestError(ctx, ErrUnallowedFilterColumn,
			apierrors.InvalidParameters{
				apierrors.InvalidParameter{
					Field:  filterName(o.Filter),
					Reason: "unauthorized filter on column",
					Source: apierrors.InvalidParamSourceQuery,
					Rule:   "unknown_property",
				},
			})
	}
	if !filterAllowed {
		return true, nil
	}

	if authorizedOpt.ValidationFunc == nil {
		return false, nil
	}

	err := authorizedOpt.ValidationFunc(o.Name, o.Value)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, ErrReturnEmptySet) {
		// for errors in uuid format, we want to handle it by returning an empty list.
		return false, apierrors.NewEmptySetResponse(ctx, c.paginationKind == paginationKindCursor)
	}
	return false, apierrors.NewBadRequestError(ctx, ErrUnallowedFilterColumn,
		apierrors.InvalidParameters{
			apierrors.InvalidParameter{
				Field:  rawFilter,
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
				Rule:   "unauthorized filter on column",
			},
		})
}

func parseMultipleStringValues(strValue string) []string {
	return lo.Map(strings.Split(strValue, ","), func(v string, _ int) string { return strings.TrimSpace(v) })
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
	if len(qsRest) == 0 {
		return o, nil
	}

	start := strings.IndexRune(qsRest, '[')
	end := strings.IndexRune(qsRest, ']')
	if start == -1 || end == -1 || end <= start {
		return o, nil
	}

	op := qsRest[start+1 : end]
	if len(op) == 0 {
		return o, nil
	}

	queryOp, ok := filterMap[op]
	if !ok {
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

	o.Filter = queryOp
	return o, nil
}
