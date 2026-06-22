package currencies_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
)

func TestCreateCurrencyInputValidate(t *testing.T) {
	valid := currencies.CreateCurrencyInput{
		Namespace: "ns",
		Code:      "CREDITS",
		Name:      "Credits",
		Symbol:    "cr",
	}

	tests := []struct {
		name    string
		input   currencies.CreateCurrencyInput
		wantErr string
	}{
		{
			name:  "valid",
			input: valid,
		},
		{
			name: "fiat code collision",
			input: currencies.CreateCurrencyInput{
				Namespace: "ns",
				Code:      "USD",
				Name:      "Credits",
				Symbol:    "cr",
			},
			wantErr: "custom currency code cannot conflict with fiat currency code",
		},
		{
			name: "invalid structural code",
			input: currencies.CreateCurrencyInput{
				Namespace: "ns",
				Code:      "BAD|CODE",
				Name:      "Credits",
				Symbol:    "cr",
			},
			wantErr: "currency code cannot contain route delimiter",
		},
		{
			name: "missing code",
			input: currencies.CreateCurrencyInput{
				Namespace: "ns",
				Name:      "Credits",
				Symbol:    "cr",
			},
			wantErr: "code is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
