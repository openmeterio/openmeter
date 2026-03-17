package request

import (
	"net/url"
	"slices"
	"strings"

	"github.com/samber/lo"
)

const SortQuery = "sort"

type defaultSort struct {
	field string
	order SortOrder
}

func extractSort(qs url.Values, c *config) ([]SortBy, error) {
	if !qs.Has(SortQuery) {
		if c.defaultSort == nil {
			return nil, nil
		}
		return []SortBy{{Field: c.defaultSort.field, Order: c.defaultSort.order}}, nil
	}

	segments := strings.Split(qs.Get(SortQuery), ",")
	out := lo.FilterMap(segments, func(v string, _ int) (SortBy, bool) {
		parts := strings.Fields(strings.TrimSpace(v))
		if len(parts) == 0 {
			return SortBy{}, false
		}
		sortBy := SortBy{Field: parts[0], Order: SortOrderAsc}
		if len(parts) > 1 {
			order := SortOrder(parts[1])
			if order == SortOrderAsc || order == SortOrderDesc {
				sortBy.Order = order
			}
		}
		return sortBy, isAuthorizedSort(sortBy.Field, c)
	})
	return out, nil
}

func isAuthorizedSort(field string, c *config) bool {
	checkSorts := len(c.authorizedSorts) != 0
	checkDotSorts := len(c.authorizedDotSorts) != 0
	switch {
	case !checkDotSorts && !checkSorts:
		return true
	case checkDotSorts && strings.ContainsRune(field, '.'):
		parts := strings.SplitN(field, ".", 2)
		return slices.Contains(c.authorizedDotSorts, parts[0]) || slices.Contains(c.authorizedDotSorts, field)
	case checkSorts:
		return slices.Contains(c.authorizedSorts, field)
	}
	return false
}
