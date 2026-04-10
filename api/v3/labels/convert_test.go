package labels

import (
	"encoding"
	"testing"

	"github.com/stretchr/testify/assert"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/models"
)

// stringerValue implements fmt.Stringer for testing annotation conversion.
type stringerValue struct{ s string }

func (v stringerValue) String() string { return v.s }

// textMarshalerValue implements encoding.TextMarshaler for testing annotation conversion.
type textMarshalerValue struct{ s string }

func (v textMarshalerValue) MarshalText() ([]byte, error) { return []byte(v.s), nil }

// failingTextMarshaler always returns an error from MarshalText.
type failingTextMarshaler struct{}

func (failingTextMarshaler) MarshalText() ([]byte, error) {
	return nil, assert.AnError
}

// Ensure interface compliance (compile-time check).
var (
	_ encoding.TextMarshaler = textMarshalerValue{}
	_ encoding.TextMarshaler = failingTextMarshaler{}
)

func TestToMetadataAnnotations(t *testing.T) {
	tests := []struct {
		name            string
		labels          *api.Labels
		wantMetadata    models.Metadata
		wantAnnotations models.Annotations
		wantErr         bool
	}{
		{
			name:            "nil labels",
			labels:          nil,
			wantMetadata:    nil,
			wantAnnotations: nil,
		},
		{
			name:            "empty labels",
			labels:          &api.Labels{},
			wantMetadata:    nil,
			wantAnnotations: nil,
		},
		{
			name: "metadata only",
			labels: &api.Labels{
				"env":  "production",
				"team": "platform",
			},
			wantMetadata: models.Metadata{
				"env":  "production",
				"team": "platform",
			},
			wantAnnotations: nil,
		},
		{
			name: "reserved openmeter prefix is rejected",
			labels: &api.Labels{
				"openmeter_region": "us-east-1",
				"openmeter_tier":   "standard",
			},
			wantMetadata:    nil,
			wantAnnotations: nil,
			wantErr:         true,
		},
		{
			name: "mixed metadata and reserved prefix",
			labels: &api.Labels{
				"env":              "production",
				"openmeter_region": "us-east-1",
			},
			wantMetadata: models.Metadata{
				"env": "production",
			},
			wantAnnotations: nil,
			wantErr:         true,
		},
		{
			name: "reserved prefix label is rejected",
			labels: &api.Labels{
				"openmeter_key": "value",
			},
			wantMetadata:    nil,
			wantAnnotations: nil,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToMetadataAnnotations(tt.labels)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantMetadata, result.Metadata)
			assert.Equal(t, tt.wantAnnotations, result.Annotations)
		})
	}
}

func TestFromMetadataAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		metadata    models.Metadata
		annotations models.Annotations
		wantLabels  *api.Labels
	}{
		{
			name:        "nil inputs",
			metadata:    nil,
			annotations: nil,
			wantLabels:  &api.Labels{},
		},
		{
			name:        "empty inputs",
			metadata:    models.Metadata{},
			annotations: models.Annotations{},
			wantLabels:  &api.Labels{},
		},
		{
			name: "metadata only",
			metadata: models.Metadata{
				"env":  "production",
				"team": "platform",
			},
			annotations: nil,
			wantLabels: &api.Labels{
				"env":  "production",
				"team": "platform",
			},
		},
		{
			name:     "annotation string value",
			metadata: nil,
			annotations: models.Annotations{
				"region": "us-east-1",
			},
			wantLabels: &api.Labels{
				"openmeter_region": "us-east-1",
			},
		},
		{
			name:     "annotation already has prefix is skipped due to reserved prefix validation",
			metadata: nil,
			annotations: models.Annotations{
				"openmeter_region": "us-east-1",
			},
			wantLabels: &api.Labels{},
		},
		{
			name:     "annotation stringer value",
			metadata: nil,
			annotations: models.Annotations{
				"tier": stringerValue{"standard"},
			},
			wantLabels: &api.Labels{
				"openmeter_tier": "standard",
			},
		},
		{
			name:     "annotation text marshaler value",
			metadata: nil,
			annotations: models.Annotations{
				"tier": textMarshalerValue{"enterprise"},
			},
			wantLabels: &api.Labels{
				"openmeter_tier": "enterprise",
			},
		},
		{
			name:     "annotation failing text marshaler is skipped",
			metadata: nil,
			annotations: models.Annotations{
				"tier": failingTextMarshaler{},
			},
			wantLabels: &api.Labels{},
		},
		{
			name: "invalid metadata key is skipped",
			metadata: models.Metadata{
				"_invalid": "value",
				"valid":    "value",
			},
			annotations: nil,
			wantLabels: &api.Labels{
				"valid": "value",
			},
		},
		{
			name:     "invalid annotation key is skipped",
			metadata: nil,
			annotations: models.Annotations{
				"_invalid": "value",
				"valid":    "ok",
			},
			wantLabels: &api.Labels{
				"openmeter_valid": "ok",
			},
		},
		{
			name: "mixed metadata and annotations",
			metadata: models.Metadata{
				"env": "production",
			},
			annotations: models.Annotations{
				"region": "us-east-1",
			},
			wantLabels: &api.Labels{
				"env":              "production",
				"openmeter_region": "us-east-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels := FromMetadataAnnotations(tt.metadata, tt.annotations)
			assert.Equal(t, tt.wantLabels, labels)
		})
	}
}
