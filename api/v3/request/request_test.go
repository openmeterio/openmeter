package request

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestGetAttributes(t *testing.T) {
	beforeCursor := pagination.NewCursor(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "before-id")
	afterCursor := pagination.NewCursor(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), "after-id")

	tests := []struct {
		name    string
		input   string
		opts    []AttributesOption
		want    *QueryAttributes
		wantErr bool
	}{
		// offset pagination
		{
			name:  "offset pagination default",
			input: "",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindOffset,
					Size: DefaultPaginationSize,
				},
			},
		},
		{
			name:  "offset pagination with size and number",
			opts:  []AttributesOption{WithOffsetPagination()},
			input: "page[size]=10&page[number]=3",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind:   paginationKindOffset,
					Size:   10,
					Number: 3,
				},
			},
		},
		// cursor pagination
		{
			name:  "cursor pagination with size",
			opts:  []AttributesOption{WithCursorPagination()},
			input: "page[size]=10",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindCursor,
					Size: 10,
				},
			},
		},
		{
			name:  "cursor pagination before",
			opts:  []AttributesOption{WithCursorPagination()},
			input: fmt.Sprintf("page[size]=5&page[before]=%s", beforeCursor.Encode()),
			want: &QueryAttributes{
				Pagination: Pagination{
					kind:   paginationKindCursor,
					Size:   5,
					Before: &beforeCursor,
				},
			},
		},
		{
			name:  "cursor pagination after",
			opts:  []AttributesOption{WithCursorPagination()},
			input: fmt.Sprintf("page[size]=7&page[after]=%s", afterCursor.Encode()),
			want: &QueryAttributes{
				Pagination: Pagination{
					kind:  paginationKindCursor,
					Size:  7,
					After: &afterCursor,
				},
			},
		},
		{
			name:    "invalid query",
			opts:    []AttributesOption{WithCursorPagination()},
			input:   "page[size]=lookatmyhorse",
			wantErr: true,
		},
		{
			name:    "cursor pagination range not supported",
			opts:    []AttributesOption{WithCursorPagination()},
			input:   fmt.Sprintf("page[size]=5&page[before]=%s&page[after]=%s", beforeCursor.Encode(), afterCursor.Encode()),
			wantErr: true,
		},
		{
			name:    "page size above maximum",
			input:   "page[size]=200",
			wantErr: true,
		},
		// sort by
		{
			name:  "single sort default order",
			input: "sort=id",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindOffset,
					Size: DefaultPaginationSize,
				},
				Sorts: []SortBy{
					{
						Field: "id",
						Order: SortOrderAsc,
					},
				},
			},
		},
		{
			name:  "single sort desc order",
			input: "sort=id desc",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindOffset,
					Size: DefaultPaginationSize,
				},
				Sorts: []SortBy{
					{
						Field: "id",
						Order: SortOrderDesc,
					},
				},
			},
		},
		{
			name:    "invalid sort order",
			input:   "sort=id invalid",
			wantErr: true,
		},
		// filters
		{
			name:  "single filter equals",
			input: "filter[id][eq]=123",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindOffset,
					Size: DefaultPaginationSize,
				},
				Filters: map[string]Filter{
					"id": {
						Eq: lo.ToPtr("123"),
					},
				},
			},
		},
		{
			name:  "single filter not equals",
			input: "filter[id][neq]=123",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindOffset,
					Size: DefaultPaginationSize,
				},
				Filters: map[string]Filter{
					"id": {
						Neq: lo.ToPtr("123"),
					},
				},
			},
		},
		{
			name:  "multiple filters",
			input: "filter[a][gt]=10&filter[b][gte]=11&filter[c][oeq]=foo,bar",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindOffset,
					Size: DefaultPaginationSize,
				},
				Filters: map[string]Filter{
					"a": {
						Gt: lo.ToPtr("10"),
					},
					"b": {
						Gte: lo.ToPtr("11"),
					},
					"c": {
						OrEq: lo.ToPtr([]string{"foo", "bar"}),
					},
				},
			},
		},
		{
			name:  "exists filter",
			input: "filter[active][exists]=true",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindOffset,
					Size: DefaultPaginationSize,
				},
				Filters: map[string]Filter{
					"active": {
						Exists: lo.ToPtr(true),
					},
				},
			},
		},
		{
			name:  "or equals filter",
			input: "filter[id][oeq]=foo,bar",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindOffset,
					Size: DefaultPaginationSize,
				},
				Filters: map[string]Filter{
					"id": {
						OrEq: lo.ToPtr([]string{"foo", "bar"}),
					},
				},
			},
		},
		// pagination, sort by, filters
		{
			name:  "pagination sort and filter combined",
			input: "page[size]=5&page[number]=2&sort=id desc&filter[name][eq]=kong",
			opts:  []AttributesOption{WithOffsetPagination()},
			want: &QueryAttributes{
				Pagination: Pagination{
					kind:   paginationKindOffset,
					Size:   5,
					Number: 2,
				},
				Filters: map[string]Filter{
					"name": {
						Eq: lo.ToPtr("kong"),
					},
				},
				Sorts: []SortBy{
					{
						Field: "id",
						Order: SortOrderDesc,
					},
				},
			},
		},
		// edge cases
		{
			name:  "unrelated query fields",
			opts:  []AttributesOption{WithCursorPagination()},
			input: "page[size]=10&foo=1&sort=id",
			want: &QueryAttributes{
				Pagination: Pagination{
					kind: paginationKindCursor,
					Size: 10,
				},
				Sorts: []SortBy{
					{
						Field: "id",
						Order: SortOrderAsc,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:8080?%s", tt.input), nil)
			require.NoError(t, err)

			attributes, err := GetAttributes(req, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, attributes)
		})
	}
}
