package clickhouse_connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateEventsTable(t *testing.T) {
	tests := []struct {
		data createEventsTableData
		want string
	}{
		{
			data: createEventsTableData{
				Database:        "openmeter",
				EventsTableName: "meter_events",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := templateQuery(createEventsTableTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCreateMeterView(t *testing.T) {
	tests := []struct {
		data createMeterViewData
		want string
	}{
		{
			data: createMeterViewData{
				Database:        "openmeter",
				EventsTableName: "meter_events",
				MeterViewName:   "meter_meter1",
				ValueProperty:   "$.duration_ms",
				GroupBy:         map[string]string{"group1": "$.group1", "group2": "$.group2"},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("", func(t *testing.T) {
			got, err := templateQuery(createMeterViewTemplate, tt.data)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
