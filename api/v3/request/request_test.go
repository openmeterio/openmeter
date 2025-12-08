package request

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/samber/lo"
)

const (
	baseUrl = "http://konghq.com/metergin"
)

func TestGetAttributes(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected *QueryAttributes
		wantErr  bool
	}{
		{"page size", "page[size]=10", &QueryAttributes{
			Pagination: CursorPagination{
				Size: 10,
			},
		}, false},
		{"complex", "page[size]=10&&sort[field]=category&sort[order]=asc&filter[category][eq]=api&filter[name][contains]=peter", &QueryAttributes{
			Pagination: CursorPagination{
				Size: 10,
			},
			Sorts: []SortBy{
				{
					Field: "category",
					Order: "asc",
				},
			},
			Filters: map[string]FilterOperator{
				"category": {
					Eq: lo.ToPtr("api"),
				},
				"name": {
					Contains: lo.ToPtr("peter"),
				},
			},
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", fmt.Sprintf("%s?%s", baseUrl, tt.query), nil)
			a, err := GetAttributes(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAttributes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(a, tt.expected) {
				t.Errorf("GetAttributes() = %+v, want %+v", a, tt.expected)
			}
		})
	}
}
