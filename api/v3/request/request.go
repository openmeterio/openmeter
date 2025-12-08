package request

import "net/http"

type QueryAttributes struct {
	Pagination CursorPagination
	Filters    []Filter
	Sorts      []SortBy
}

// GetAttributes return the Attributes found in the request query string
func GetAttributes(r *http.Request) (*QueryAttributes, error) {
	a := &AipAttributes{}

	conf := newConfig()
	for _, v := range opts {
		v(conf)
	}

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

	sort, err := extractSort(queryValues, conf)
	if err != nil {
		return nil, err
	}

	a.Sorts = sort

	return a, nil
}
