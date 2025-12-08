package request

import (
	"net/http"
)

const (
	defaultPaginationMaxSize = 100
)

type config struct {
	defaultPageSize int
	maxPageSize     int
}

func newConfig() *config {
	return &config{
		defaultPageSize: DefaultPaginationSize,
		maxPageSize:     defaultPaginationMaxSize,
	}
}

type QueryAttributes struct {
	Pagination CursorPagination          `query:"page"`
	Filters    map[string]FilterOperator `query:"filter"`
	Sorts      []SortBy                  `query:"sort"`
}

// GetAttributes return the Attributes found in the request query string
func GetAttributes(r *http.Request) (*QueryAttributes, error) {
	// This does not work:
	attributes := &QueryAttributes{}
	// This works:
	//attributes := &map[string]interface{}{}

	err := Unmarshal(r.Context(), r.URL.RawQuery, &attributes)
	if err != nil {
		return nil, err
	}

	//pagination, err := extractPagination(r.Context(), queryValues, conf)
	//if err != nil {
	//	return nil, err
	//}
	//a.Pagination = pagination
	//
	//filters, err := extractFilter(r.Context(), queryValues, conf)
	//if err != nil {
	//	return nil, err
	//}
	//a.Filters = filters
	//
	//sort, err := extractSort(queryValues, conf)
	//if err != nil {
	//	return nil, err
	//}
	//
	//a.Sorts = sort

	return attributes, nil
}
