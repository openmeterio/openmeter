package invoicesync

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundToAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   alpacadecimal.Decimal
		currency string
		want     int64
	}{
		{"USD whole dollars", alpacadecimal.NewFromFloat(100.00), "USD", 10000},
		{"USD with cents", alpacadecimal.NewFromFloat(1.50), "USD", 150},
		{"USD sub-cent rounds", alpacadecimal.NewFromFloat(0.005), "USD", 1},
		{"USD zero", alpacadecimal.NewFromFloat(0), "USD", 0},
		{"USD negative", alpacadecimal.NewFromFloat(-10.50), "USD", -1050},
		{"JPY zero-decimal", alpacadecimal.NewFromFloat(100), "JPY", 100},
		{"JPY with decimals rounds", alpacadecimal.NewFromFloat(99.7), "JPY", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RoundToAmount(tt.amount, tt.currency)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("invalid currency returns error", func(t *testing.T) {
		_, err := RoundToAmount(alpacadecimal.NewFromFloat(42.99), "INVALID")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid currency")
	})
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   alpacadecimal.Decimal
		currency string
		contains string // GOBL formatting includes currency symbols; check substring
	}{
		{"USD integer", alpacadecimal.NewFromInt(100), "USD", "100"},
		{"USD decimal", alpacadecimal.NewFromFloat(99.95), "USD", "99.95"},
		{"USD zero", alpacadecimal.NewFromInt(0), "USD", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatAmount(tt.amount, tt.currency)
			require.NoError(t, err)
			assert.Contains(t, got, tt.contains)
		})
	}

	t.Run("invalid currency returns error", func(t *testing.T) {
		_, err := FormatAmount(alpacadecimal.NewFromFloat(42.5), "INVALID")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid currency")
	})
}

func TestFormatQuantity(t *testing.T) {
	tests := []struct {
		name     string
		quantity alpacadecimal.Decimal
		want     string
	}{
		{"integer", alpacadecimal.NewFromInt(5), "5"},
		{"decimal", alpacadecimal.NewFromFloat(2.50), "2.50"},
		{"large integer with comma", alpacadecimal.NewFromInt(1000), "1,000"},
		{"zero", alpacadecimal.NewFromInt(0), "0"},
		{"small decimal", alpacadecimal.NewFromFloat(0.33), "0.33"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatQuantity(tt.quantity, "USD")
			assert.Equal(t, tt.want, got)
		})
	}
}
