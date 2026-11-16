package query

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestConvertISO8601DurationToWindowSize(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		want     meter.WindowSize
		wantErr  bool
	}{
		{name: "minute", duration: "PT1M", want: meter.WindowSizeMinute},
		{name: "hour", duration: "PT1H", want: meter.WindowSizeHour},
		{name: "day", duration: "P1D", want: meter.WindowSizeDay},
		{name: "month", duration: "P1M", want: meter.WindowSizeMonth},
		{name: "invalid", duration: "P1Y", wantErr: true},
		{name: "empty", duration: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertISO8601DurationToWindowSize(tt.duration)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertWindowSizeToISO8601Duration(t *testing.T) {
	tests := []struct {
		name    string
		ws      meter.WindowSize
		want    string
		wantErr bool
	}{
		{name: "minute", ws: meter.WindowSizeMinute, want: "PT1M"},
		{name: "hour", ws: meter.WindowSizeHour, want: "PT1H"},
		{name: "day", ws: meter.WindowSizeDay, want: "P1D"},
		{name: "month", ws: meter.WindowSizeMonth, want: "P1M"},
		{name: "unknown", ws: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertWindowSizeToISO8601Duration(tt.ws)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractStringsFromQueryFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  *apiv3models.QueryFilterString
		want    []string
		wantErr bool
	}{
		{
			name:   "nil filter",
			filter: nil,
			want:   nil,
		},
		{
			name:   "eq operator",
			filter: &apiv3models.QueryFilterString{Eq: lo.ToPtr("hello")},
			want:   []string{"hello"},
		},
		{
			name:   "in operator",
			filter: &apiv3models.QueryFilterString{In: &[]string{"a", "b", "c"}},
			want:   []string{"a", "b", "c"},
		},
		{
			name: "eq and in together rejected",
			filter: &apiv3models.QueryFilterString{
				Eq: lo.ToPtr("x"),
				In: &[]string{"y"},
			},
			wantErr: true,
		},
		{
			name:    "neq rejected",
			filter:  &apiv3models.QueryFilterString{Neq: lo.ToPtr("x")},
			wantErr: true,
		},
		{
			name:    "nin rejected",
			filter:  &apiv3models.QueryFilterString{Nin: &[]string{"x"}},
			wantErr: true,
		},
		{
			name:    "contains rejected",
			filter:  &apiv3models.QueryFilterString{Contains: lo.ToPtr("x")},
			wantErr: true,
		},
		{
			name:    "ncontains rejected",
			filter:  &apiv3models.QueryFilterString{Ncontains: lo.ToPtr("x")},
			wantErr: true,
		},
		{
			name: "and rejected",
			filter: &apiv3models.QueryFilterString{
				And: &[]apiv3models.QueryFilterString{{Eq: lo.ToPtr("x")}},
			},
			wantErr: true,
		},
		{
			name: "or rejected",
			filter: &apiv3models.QueryFilterString{
				Or: &[]apiv3models.QueryFilterString{{Eq: lo.ToPtr("x")}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractStringsFromQueryFilter(tt.filter, "test")
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
