package labels

import (
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
)

func TestValidateLabel(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:  "valid alphanumeric",
			key:   "key",
			value: "value",
		},
		{
			name:  "valid with underscore",
			key:   "my_good",
			value: "my_good",
		},
		{
			name:  "valid with dot",
			key:   "my.good",
			value: "my.good",
		},
		{
			name:  "valid with hyphen",
			key:   "my-good",
			value: "my-good",
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
			name:    "invalid key trailing underscore",
			key:     "bad_",
			value:   "good",
			wantErr: true,
		},
		{
			name:    "invalid key trailing hyphen",
			key:     "bad-",
			value:   "good",
			wantErr: true,
		},
		{
			name:    "invalid key trailing dot",
			key:     "bad.",
			value:   "good",
			wantErr: true,
		},
		{
			name:    "invalid key leading underscore",
			key:     "_bad",
			value:   "good",
			wantErr: true,
		},
		{
			name:    "invalid key leading hyphen",
			key:     "-bad",
			value:   "good",
			wantErr: true,
		},
		{
			name:    "invalid key leading dot",
			key:     ".bad",
			value:   "good",
			wantErr: true,
		},
		{
			name:    "invalid key empty",
			key:     "",
			value:   "good",
			wantErr: true,
		},
		{
			name:    "invalid value trailing underscore",
			key:     "good",
			value:   "bad_",
			wantErr: true,
		},
		{
			name:    "invalid value leading dot",
			key:     "good",
			value:   ".bad",
			wantErr: true,
		},
		{
			name:    "invalid value empty",
			key:     "good",
			value:   "",
			wantErr: true,
		},
		{
			name:    "both key and value invalid",
			key:     "_bad",
			value:   "bad_",
			wantErr: true,
		},
		{
			name:    "reserved openmeter prefix",
			key:     "openmeter_key",
			value:   "value",
			wantErr: true,
		},
		{
			name:    "reserved kong prefix",
			key:     "kong_key",
			value:   "value",
			wantErr: true,
		},
		{
			name:    "reserved konnect prefix",
			key:     "konnect_key",
			value:   "value",
			wantErr: true,
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
		})
	}
}

func TestValidateLabels(t *testing.T) {
	tests := []struct {
		name    string
		labels  api.Labels
		wantErr bool
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
			wantErr: true,
		},
		{
			name: "one invalid value",
			labels: api.Labels{
				"good": "bad_",
			},
			wantErr: true,
		},
		{
			name: "both key and value invalid in same entry",
			labels: api.Labels{
				"_bad": "bad_",
			},
			wantErr: true,
		},
		{
			name: "multiple invalid keys",
			labels: api.Labels{
				"_bad1": "good",
				"_bad2": "good",
			},
			wantErr: true,
		},
		{
			name: "reserved prefix key",
			labels: api.Labels{
				"openmeter_key": "value",
			},
			wantErr: true,
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
		})
	}
}
