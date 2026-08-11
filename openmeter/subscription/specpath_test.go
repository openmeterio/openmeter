package subscription_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
)

func TestSpecPathJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		path     subscription.SpecPath
		wantJSON string
	}{
		{
			name:     "phase",
			path:     subscription.NewPhasePath("phase"),
			wantJSON: `"/phases/phase"`,
		},
		{
			name:     "item",
			path:     subscription.NewItemPath("phase", "item"),
			wantJSON: `"/phases/phase/items/item"`,
		},
		{
			name:     "item version",
			path:     subscription.NewItemVersionPath("phase", "item", 1),
			wantJSON: `"/phases/phase/items/item/idx/1"`,
		},
		{
			name:     "opaque key content",
			path:     subscription.NewPhasePath(`phase"key\suffix`),
			wantJSON: `"/phases/phase\"key\\suffix"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.path.Validate())

			serialized, err := json.Marshal(tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.wantJSON, string(serialized))

			var deserialized subscription.SpecPath
			err = json.Unmarshal(serialized, &deserialized)
			require.NoError(t, err)
			require.Equal(t, tt.path, deserialized)
		})
	}
}

func TestSpecPathJSONUnmarshalRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name           string
		data           string
		wantValidation bool
	}{
		{
			name: "malformed JSON",
			data: `"/phases/phase`,
		},
		{
			name: "non-string JSON",
			data: `42`,
		},
		{
			name:           "null",
			data:           `null`,
			wantValidation: true,
		},
		{
			name:           "wrong prefix",
			data:           `"/items/item"`,
			wantValidation: true,
		},
		{
			name:           "wrong segment count",
			data:           `"/phases/phase/items"`,
			wantValidation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := subscription.NewPhasePath("original")
			path := original

			err := json.Unmarshal([]byte(tt.data), &path)

			require.Error(t, err)
			require.Equal(t, original, path)

			var validationError *subscription.PatchValidationError
			if tt.wantValidation {
				require.ErrorAs(t, err, &validationError)
			} else {
				require.NotErrorAs(t, err, &validationError)
			}
		})
	}
}
