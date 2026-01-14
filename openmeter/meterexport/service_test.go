package meterexport

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDataExportConfig_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		config   DataExportConfig
		wantJSON string
	}{
		{
			name: "UTC timezone",
			config: DataExportConfig{
				ExportWindowSize:     meter.WindowSizeMinute,
				ExportWindowTimeZone: time.UTC,
				MeterID: models.NamespacedID{
					Namespace: "test-ns",
					ID:        "meter-1",
				},
			},
			wantJSON: `{"exportWindowSize":"MINUTE","exportWindowTimeZone":"UTC","meterId":{"namespace":"test-ns","id":"meter-1"}}`,
		},
		{
			name: "America/New_York timezone",
			config: DataExportConfig{
				ExportWindowSize:     meter.WindowSizeHour,
				ExportWindowTimeZone: mustLoadLocation(t, "America/New_York"),
				MeterID: models.NamespacedID{
					Namespace: "prod",
					ID:        "api-calls",
				},
			},
			wantJSON: `{"exportWindowSize":"HOUR","exportWindowTimeZone":"America/New_York","meterId":{"namespace":"prod","id":"api-calls"}}`,
		},
		{
			name: "nil timezone marshals to empty string",
			config: DataExportConfig{
				ExportWindowSize:     meter.WindowSizeDay,
				ExportWindowTimeZone: nil,
				MeterID: models.NamespacedID{
					Namespace: "test",
					ID:        "test",
				},
			},
			wantJSON: `{"exportWindowSize":"DAY","exportWindowTimeZone":"","meterId":{"namespace":"test","id":"test"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.config)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(data))

			// Test round-trip (skip if timezone is nil since it won't round-trip)
			if tt.config.ExportWindowTimeZone != nil {
				var decoded DataExportConfig
				err = json.Unmarshal(data, &decoded)
				require.NoError(t, err)

				assert.Equal(t, tt.config.ExportWindowSize, decoded.ExportWindowSize)
				assert.Equal(t, tt.config.MeterID, decoded.MeterID)
				require.NotNil(t, decoded.ExportWindowTimeZone)
				assert.Equal(t, tt.config.ExportWindowTimeZone.String(), decoded.ExportWindowTimeZone.String())
			}
		})
	}
}

func TestDataExportConfig_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		jsonInput  string
		wantTZ     string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "valid UTC",
			jsonInput: `{"exportWindowSize":"MINUTE","exportWindowTimeZone":"UTC","meterId":{"namespace":"ns","id":"id"}}`,
			wantTZ:    "UTC",
		},
		{
			name:      "valid Europe/London",
			jsonInput: `{"exportWindowSize":"HOUR","exportWindowTimeZone":"Europe/London","meterId":{"namespace":"ns","id":"id"}}`,
			wantTZ:    "Europe/London",
		},
		{
			name:      "empty timezone results in nil",
			jsonInput: `{"exportWindowSize":"MINUTE","exportWindowTimeZone":"","meterId":{"namespace":"ns","id":"id"}}`,
			wantTZ:    "",
		},
		{
			name:       "invalid timezone",
			jsonInput:  `{"exportWindowSize":"MINUTE","exportWindowTimeZone":"Invalid/Timezone","meterId":{"namespace":"ns","id":"id"}}`,
			wantErr:    true,
			wantErrMsg: "invalid timezone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config DataExportConfig
			err := json.Unmarshal([]byte(tt.jsonInput), &config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}

			require.NoError(t, err)

			if tt.wantTZ == "" {
				assert.Nil(t, config.ExportWindowTimeZone)
			} else {
				require.NotNil(t, config.ExportWindowTimeZone)
				assert.Equal(t, tt.wantTZ, config.ExportWindowTimeZone.String())
			}
		})
	}
}

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	require.NoError(t, err)
	return loc
}
