package currencyx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodeValidate(t *testing.T) {
	tests := []struct {
		name    string
		code    Code
		wantErr bool
	}{
		{name: "fiat", code: "USD"},
		{name: "custom", code: "CREDITS"},
		{name: "empty", wantErr: true},
		{name: "custom too short", code: "CC", wantErr: true},
		{name: "custom too long", code: "CUSTOM_CURRENCY_CODE_TOO_LONG", wantErr: true},
		{name: "surrounding whitespace", code: " CREDITS ", wantErr: true},
		{name: "route delimiter", code: "CREDITS|USD", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.code.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCodeIsFiat(t *testing.T) {
	require.True(t, Code("USD").IsFiat())
	require.False(t, Code("CREDITS").IsFiat())
}
