package meter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func TestListMetersParams_Validate_Sort(t *testing.T) {
	namespace := "test-ns"

	tests := []struct {
		name    string
		params  ListMetersParams
		wantErr bool
	}{
		{
			name: "no sort",
			params: ListMetersParams{
				Namespace: namespace,
				Page:      pagination.Page{PageSize: 10, PageNumber: 1},
			},
		},
		{
			name: "sort by key asc",
			params: ListMetersParams{
				Namespace: namespace,
				Page:      pagination.Page{PageSize: 10, PageNumber: 1},
				OrderBy:   OrderByKey,
				Order:     sortx.OrderAsc,
			},
		},
		{
			name: "sort by name desc",
			params: ListMetersParams{
				Namespace: namespace,
				Page:      pagination.Page{PageSize: 10, PageNumber: 1},
				OrderBy:   OrderByName,
				Order:     sortx.OrderDesc,
			},
		},
		{
			name: "sort by aggregation",
			params: ListMetersParams{
				Namespace: namespace,
				Page:      pagination.Page{PageSize: 10, PageNumber: 1},
				OrderBy:   OrderByAggregation,
				Order:     sortx.OrderAsc,
			},
		},
		{
			name: "sort by createdAt",
			params: ListMetersParams{
				Namespace: namespace,
				Page:      pagination.Page{PageSize: 10, PageNumber: 1},
				OrderBy:   OrderByCreatedAt,
				Order:     sortx.OrderAsc,
			},
		},
		{
			name: "sort by updatedAt",
			params: ListMetersParams{
				Namespace: namespace,
				Page:      pagination.Page{PageSize: 10, PageNumber: 1},
				OrderBy:   OrderByUpdatedAt,
				Order:     sortx.OrderDesc,
			},
		},
		{
			name: "invalid order by field",
			params: ListMetersParams{
				Namespace: namespace,
				Page:      pagination.Page{PageSize: 10, PageNumber: 1},
				OrderBy:   OrderBy("invalid_field"),
				Order:     sortx.OrderAsc,
			},
			wantErr: true,
		},
		{
			name: "invalid order direction",
			params: ListMetersParams{
				Namespace: namespace,
				Page:      pagination.Page{PageSize: 10, PageNumber: 1},
				OrderBy:   OrderByKey,
				Order:     sortx.Order("SIDEWAYS"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
