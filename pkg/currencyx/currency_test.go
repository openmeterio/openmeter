package currencyx_test

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestRoundToPrecision(t *testing.T) {
	cases := []struct {
		def      string
		amount   float64
		expected float64
	}{
		// Subunits = 2, smallestDenomination = 1
		{"USD", 1.23456789, 1.23},
		{"USD", 1.23556789, 1.24},

		// Subunits = 0, smallestDenomination = 1
		{"JPY", 1.23456789, 1.0},
		{"JPY", 1.9556789, 2.0},
	}

	for _, c := range cases {
		calculator, err := currencyx.Code(c.def).Calculator()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		amount := alpacadecimal.NewFromFloat(c.amount)
		result := calculator.RoundToPrecision(amount).InexactFloat64()

		require.Equal(t, c.expected, result)
	}
}
