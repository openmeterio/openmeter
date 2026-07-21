package meterevent_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func TestListSubjectsParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  meterevent.ListSubjectsParams
		wantErr bool
	}{
		{
			name:    "namespace required",
			params:  meterevent.ListSubjectsParams{},
			wantErr: true,
		},
		{
			name: "valid with no filters",
			params: meterevent.ListSubjectsParams{
				Namespace: "my_namespace",
			},
		},
		{
			name: "key filter is accepted",
			params: meterevent.ListSubjectsParams{
				Namespace: "my_namespace",
				Key: &filter.FilterString{
					Contains: lo.ToPtr("customer"),
				},
			},
		},
		{
			name: "cursor without id is rejected",
			params: meterevent.ListSubjectsParams{
				Namespace: "my_namespace",
				Cursor:    &pagination.Cursor{},
			},
			wantErr: true,
		},
		{
			name: "valid cursor is accepted",
			params: meterevent.ListSubjectsParams{
				Namespace: "my_namespace",
				Cursor:    lo.ToPtr(meterevent.Subject{Key: "customer-1"}.Cursor()),
			},
		},
		{
			name: "limit over maximum is rejected",
			params: meterevent.ListSubjectsParams{
				Namespace: "my_namespace",
				Limit:     lo.ToPtr(meterevent.MaximumLimit + 1),
			},
			wantErr: true,
		},
		{
			name: "zero limit is rejected",
			params: meterevent.ListSubjectsParams{
				Namespace: "my_namespace",
				Limit:     lo.ToPtr(0),
			},
			wantErr: true,
		},
		{
			name: "negative limit is rejected",
			params: meterevent.ListSubjectsParams{
				Namespace: "my_namespace",
				Limit:     lo.ToPtr(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestSubject_Cursor(t *testing.T) {
	cursor := meterevent.Subject{Key: "customer-1"}.Cursor()

	// The cursor must round-trip through its encoded form so keyset pagination
	// can resume from the subject key.
	decoded, err := pagination.DecodeCursor(cursor.Encode())
	require.NoError(t, err)
	require.Equal(t, "customer-1", decoded.ID)
	require.True(t, decoded.Time.Equal(time.Time{}))
}
