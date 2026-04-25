package meterevent_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestListEventsV2Params_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		params  meterevent.ListEventsV2Params
		wantErr bool
	}{
		{
			name:    "namespace required",
			params:  meterevent.ListEventsV2Params{},
			wantErr: true,
		},
		{
			name: "valid default sort and no filters",
			params: meterevent.ListEventsV2Params{
				Namespace: "my_namespace",
			},
		},
		{
			name: "stored_at filter is accepted",
			params: meterevent.ListEventsV2Params{
				Namespace: "my_namespace",
				StoredAt: &filter.FilterTime{
					Gte: &now,
				},
			},
		},
		{
			name: "ingested_at sort is accepted",
			params: meterevent.ListEventsV2Params{
				Namespace: "my_namespace",
				SortBy:    streaming.EventSortFieldIngestedAt,
			},
		},
		{
			name: "stored_at sort is accepted",
			params: meterevent.ListEventsV2Params{
				Namespace: "my_namespace",
				SortBy:    streaming.EventSortFieldStoredAt,
			},
		},
		{
			name: "unknown sort rejected",
			params: meterevent.ListEventsV2Params{
				Namespace: "my_namespace",
				SortBy:    streaming.EventSortField("created_at"),
			},
			wantErr: true,
		},
		{
			name: "customer_id without In is rejected",
			params: meterevent.ListEventsV2Params{
				Namespace: "my_namespace",
				CustomerID: &filter.FilterString{
					Eq: lo.ToPtr("01G65Z755AFWAKHE12NY0CQ9FH"),
				},
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

func TestEvent_Cursor(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ingested := now.Add(5 * time.Second)
	stored := now.Add(10 * time.Second)

	e := meterevent.Event{
		ID:         "evt-1",
		Time:       now,
		IngestedAt: ingested,
		StoredAt:   stored,
		StoreRowID: "01J000000000000000000000AA",
	}

	t.Run("defaults to time and uses store_row_id as tiebreak", func(t *testing.T) {
		c := e.Cursor()
		require.Equal(t, now, c.Time)
		require.Equal(t, "01J000000000000000000000AA", c.ID)
	})

	t.Run("switches to ingested_at", func(t *testing.T) {
		ev := e
		ev.SortBy = streaming.EventSortFieldIngestedAt
		c := ev.Cursor()
		require.Equal(t, ingested, c.Time)
		require.Equal(t, "01J000000000000000000000AA", c.ID)
	})

	t.Run("switches to stored_at", func(t *testing.T) {
		ev := e
		ev.SortBy = streaming.EventSortFieldStoredAt
		c := ev.Cursor()
		require.Equal(t, stored, c.Time)
		require.Equal(t, "01J000000000000000000000AA", c.ID)
	})
}
