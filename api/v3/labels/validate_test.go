package labels

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
)

func TestValidateLabel(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		wantErr      bool
		wantKeyErr   bool
		wantValueErr bool
	}{
		{
			name:  "valid alphanumeric",
			key:   "key",
			value: "value",
		},
		{
			name:  "valid with underscore",
			key:   "openmeter_good",
			value: "openmeter_good",
		},
		{
			name:  "valid with dot",
			key:   "openmeter.good",
			value: "openmeter.good",
		},
		{
			name:  "valid with hyphen",
			key:   "openmeter-good",
			value: "openmeter-good",
		},
		{
			name:  "valid single character",
			key:   "a",
			value: "b",
		},
		{
			name:  "valid two characters",
			key:   "ab",
			value: "cd",
		},
		{
			name:  "valid uppercase",
			key:   "MyKey",
			value: "MyValue",
		},
		{
			name:  "valid digits",
			key:   "key123",
			value: "value456",
		},
		{
			name:  "valid starts and ends with digit",
			key:   "1key1",
			value: "2value2",
		},
		{
			name:         "invalid key trailing underscore",
			key:          "openmeter_bad_",
			value:        "good",
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: false,
		},
		{
			name:         "invalid key trailing hyphen",
			key:          "openmeter-bad-",
			value:        "good",
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: false,
		},
		{
			name:         "invalid key trailing dot",
			key:          "openmeter.bad.",
			value:        "good",
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: false,
		},
		{
			name:         "invalid key leading underscore",
			key:          "_openmeter-bad",
			value:        "good",
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: false,
		},
		{
			name:         "invalid key leading hyphen",
			key:          "-openmeter-bad",
			value:        "good",
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: false,
		},
		{
			name:         "invalid key leading dot",
			key:          ".openmeter.bad",
			value:        "good",
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: false,
		},
		{
			name:         "invalid key empty",
			key:          "",
			value:        "good",
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: false,
		},
		{
			name:         "invalid value trailing underscore",
			key:          "good",
			value:        "openmeter_bad_",
			wantErr:      true,
			wantKeyErr:   false,
			wantValueErr: true,
		},
		{
			name:         "invalid value leading dot",
			key:          "good",
			value:        ".openmeter.bad",
			wantErr:      true,
			wantKeyErr:   false,
			wantValueErr: true,
		},
		{
			name:         "invalid value empty",
			key:          "good",
			value:        "",
			wantErr:      true,
			wantKeyErr:   false,
			wantValueErr: true,
		},
		{
			name:         "both key and value invalid",
			key:          "_bad",
			value:        "bad_",
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLabel(tt.key, tt.value)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Equal(t, tt.wantKeyErr, errors.Is(err, ErrInvalidLabelKey))
			assert.Equal(t, tt.wantValueErr, errors.Is(err, ErrInvalidLabelValue))
		})
	}
}

func TestValidateLabels(t *testing.T) {
	tests := []struct {
		name          string
		labels        api.Labels
		wantErr       bool
		wantKeyErr    bool
		wantValueErr  bool
		wantErrSubstr string
	}{
		{
			name:    "nil labels",
			labels:  nil,
			wantErr: false,
		},
		{
			name:    "empty labels",
			labels:  api.Labels{},
			wantErr: false,
		},
		{
			name: "all valid",
			labels: api.Labels{
				"env":     "production",
				"team":    "platform",
				"version": "v1",
			},
			wantErr: false,
		},
		{
			name: "one invalid key",
			labels: api.Labels{
				"_bad": "value",
				"good": "value",
			},
			wantErr:       true,
			wantKeyErr:    true,
			wantErrSubstr: `"_bad"`,
		},
		{
			name: "one invalid value",
			labels: api.Labels{
				"good": "bad_",
			},
			wantErr:       true,
			wantValueErr:  true,
			wantErrSubstr: `"bad_"`,
		},
		{
			name: "both key and value invalid in same entry",
			labels: api.Labels{
				"_bad": "bad_",
			},
			wantErr:      true,
			wantKeyErr:   true,
			wantValueErr: true,
		},
		{
			name: "multiple invalid keys",
			labels: api.Labels{
				"_bad1": "good",
				"_bad2": "good",
			},
			wantErr:    true,
			wantKeyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLabels(tt.labels)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Equal(t, tt.wantKeyErr, errors.Is(err, ErrInvalidLabelKey))
			assert.Equal(t, tt.wantValueErr, errors.Is(err, ErrInvalidLabelValue))
			if tt.wantErrSubstr != "" {
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
			}
		})
	}
}
