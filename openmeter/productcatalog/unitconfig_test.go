package productcatalog

import (
	"encoding/json"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  UnitConfig
		wantErr bool
	}{
		{
			name: "valid divide config",
			config: UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1e9),
				Rounding:         RoundingModeCeiling,
				Precision:        0,
			},
		},
		{
			name: "valid multiply config",
			config: UnitConfig{
				Operation:        ConversionOperationMultiply,
				ConversionFactor: decimal.NewFromFloat(1.2),
				Rounding:         RoundingModeNone,
				DisplayUnit:      lo.ToPtr("credits"),
			},
		},
		{
			name: "valid with display unit",
			config: UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(3600),
				Rounding:         RoundingModeHalfUp,
				Precision:        2,
				DisplayUnit:      lo.ToPtr("hours"),
			},
		},
		{
			name: "invalid operation",
			config: UnitConfig{
				Operation:        "invalid",
				ConversionFactor: decimal.NewFromInt(1),
			},
			wantErr: true,
		},
		{
			name: "zero factor",
			config: UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.Zero,
			},
			wantErr: true,
		},
		{
			name: "negative factor",
			config: UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromInt(-1),
			},
			wantErr: true,
		},
		{
			name: "invalid rounding mode",
			config: UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromInt(1),
				Rounding:         "invalid",
			},
			wantErr: true,
		},
		{
			name: "negative precision",
			config: UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromInt(1),
				Precision:        -1,
			},
			wantErr: true,
		},
		{
			name: "empty rounding defaults to none (valid)",
			config: UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromInt(1),
				Rounding:         "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnitConfig_Convert(t *testing.T) {
	tests := []struct {
		name   string
		config *UnitConfig
		input  decimal.Decimal
		want   string
	}{
		{
			name:   "nil config returns input",
			config: nil,
			input:  decimal.NewFromFloat(1000),
			want:   "1000",
		},
		{
			name: "divide bytes to GB",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1e9),
			},
			input: decimal.NewFromFloat(5e9),
			want:  "5",
		},
		{
			name: "divide seconds to hours",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(3600),
			},
			input: decimal.NewFromFloat(7200),
			want:  "2",
		},
		{
			name: "divide with fractional result",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1000),
			},
			input: decimal.NewFromFloat(1500),
			want:  "1.5",
		},
		{
			name: "multiply for margin",
			config: &UnitConfig{
				Operation:        ConversionOperationMultiply,
				ConversionFactor: decimal.NewFromFloat(1.2),
			},
			input: decimal.NewFromFloat(100),
			want:  "120",
		},
		{
			name: "multiply by 1 (identity)",
			config: &UnitConfig{
				Operation:        ConversionOperationMultiply,
				ConversionFactor: decimal.NewFromInt(1),
			},
			input: decimal.NewFromFloat(42),
			want:  "42",
		},
		{
			name: "zero input quantity",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1e6),
			},
			input: decimal.Zero,
			want:  "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Convert(tt.input)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestUnitConfig_ConvertAndRound(t *testing.T) {
	tests := []struct {
		name   string
		config *UnitConfig
		input  decimal.Decimal
		want   string
	}{
		{
			name:   "nil config returns input",
			config: nil,
			input:  decimal.NewFromFloat(1500),
			want:   "1500",
		},
		{
			name: "package pricing: divide and ceil",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1e6),
				Rounding:         RoundingModeCeiling,
				Precision:        0,
			},
			input: decimal.NewFromFloat(1500001),
			want:  "2",
		},
		{
			name: "package pricing: exact division no rounding needed",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1e6),
				Rounding:         RoundingModeCeiling,
				Precision:        0,
			},
			input: decimal.NewFromFloat(2e6),
			want:  "2",
		},
		{
			name: "divide and floor",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1000),
				Rounding:         RoundingModeFloor,
				Precision:        0,
			},
			input: decimal.NewFromFloat(1999),
			want:  "1",
		},
		{
			name: "divide and half_up",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1000),
				Rounding:         RoundingModeHalfUp,
				Precision:        0,
			},
			input: decimal.NewFromFloat(1500),
			want:  "2",
		},
		{
			name: "divide and half_up rounds down",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1000),
				Rounding:         RoundingModeHalfUp,
				Precision:        0,
			},
			input: decimal.NewFromFloat(1499),
			want:  "1",
		},
		{
			name: "precision 2 with ceil",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(3600),
				Rounding:         RoundingModeCeiling,
				Precision:        2,
			},
			input: decimal.NewFromFloat(5000),
			want:  "1.39",
		},
		{
			name: "no rounding mode",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(3),
				Rounding:         RoundingModeNone,
			},
			input: decimal.NewFromFloat(10),
			want:  "3.3333333333333333",
		},
		{
			name: "multiply with ceil for margin",
			config: &UnitConfig{
				Operation:        ConversionOperationMultiply,
				ConversionFactor: decimal.NewFromFloat(1.2),
				Rounding:         RoundingModeCeiling,
				Precision:        0,
			},
			input: decimal.NewFromFloat(10),
			want:  "12",
		},
		{
			name: "empty rounding behaves like none",
			config: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(3),
				Rounding:         "",
			},
			input: decimal.NewFromFloat(10),
			want:  "3.3333333333333333",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ConvertAndRound(tt.input)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestUnitConfig_Equal(t *testing.T) {
	tests := []struct {
		name string
		a    *UnitConfig
		b    *UnitConfig
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "one nil",
			a:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1)},
			b:    nil,
			want: false,
		},
		{
			name: "equal",
			a: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1e9),
				Rounding:         RoundingModeCeiling,
				Precision:        0,
				DisplayUnit:      lo.ToPtr("GB"),
			},
			b: &UnitConfig{
				Operation:        ConversionOperationDivide,
				ConversionFactor: decimal.NewFromFloat(1e9),
				Rounding:         RoundingModeCeiling,
				Precision:        0,
				DisplayUnit:      lo.ToPtr("GB"),
			},
			want: true,
		},
		{
			name: "different operation",
			a:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1)},
			b:    &UnitConfig{Operation: ConversionOperationMultiply, ConversionFactor: decimal.NewFromInt(1)},
			want: false,
		},
		{
			name: "different factor",
			a:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1)},
			b:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(2)},
			want: false,
		},
		{
			name: "different display unit",
			a:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1), DisplayUnit: lo.ToPtr("GB")},
			b:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1), DisplayUnit: lo.ToPtr("TB")},
			want: false,
		},
		{
			name: "different precision",
			a:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1), Precision: 0},
			b:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1), Precision: 2},
			want: false,
		},
		{
			name: "different rounding mode",
			a:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1), Rounding: RoundingModeCeiling},
			b:    &UnitConfig{Operation: ConversionOperationDivide, ConversionFactor: decimal.NewFromInt(1), Rounding: RoundingModeFloor},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Equal(tt.b))
		})
	}
}

func TestUnitConfig_Clone(t *testing.T) {
	original := &UnitConfig{
		Operation:        ConversionOperationDivide,
		ConversionFactor: decimal.NewFromFloat(1e9),
		Rounding:         RoundingModeCeiling,
		Precision:        2,
		DisplayUnit:      lo.ToPtr("GB"),
	}

	clone := original.Clone()

	assert.True(t, original.Equal(&clone))

	// Mutate clone to ensure deep copy
	clone.DisplayUnit = lo.ToPtr("TB")
	assert.NotEqual(t, *original.DisplayUnit, *clone.DisplayUnit)
}

func TestUnitConfig_JSON(t *testing.T) {
	config := &UnitConfig{
		Operation:        ConversionOperationDivide,
		ConversionFactor: decimal.NewFromFloat(1e9),
		Rounding:         RoundingModeCeiling,
		Precision:        0,
		DisplayUnit:      lo.ToPtr("GB"),
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var decoded UnitConfig
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.True(t, config.Equal(&decoded))
}

func TestUnitConfig_Round(t *testing.T) {
	tests := []struct {
		name      string
		rounding  RoundingMode
		precision int
		input     string
		want      string
	}{
		{
			name:      "ceil precision 0",
			rounding:  RoundingModeCeiling,
			precision: 0,
			input:     "1.1",
			want:      "2",
		},
		{
			name:      "ceil exact value",
			rounding:  RoundingModeCeiling,
			precision: 0,
			input:     "2",
			want:      "2",
		},
		{
			name:      "floor precision 0",
			rounding:  RoundingModeFloor,
			precision: 0,
			input:     "1.9",
			want:      "1",
		},
		{
			name:      "floor exact value",
			rounding:  RoundingModeFloor,
			precision: 0,
			input:     "2",
			want:      "2",
		},
		{
			name:      "half_up at midpoint",
			rounding:  RoundingModeHalfUp,
			precision: 0,
			input:     "1.5",
			want:      "2",
		},
		{
			name:      "half_up below midpoint",
			rounding:  RoundingModeHalfUp,
			precision: 0,
			input:     "1.4",
			want:      "1",
		},
		{
			name:      "none preserves value",
			rounding:  RoundingModeNone,
			precision: 0,
			input:     "1.23456",
			want:      "1.23456",
		},
		{
			name:      "ceil precision 2",
			rounding:  RoundingModeCeiling,
			precision: 2,
			input:     "1.001",
			want:      "1.01",
		},
		{
			name:      "floor precision 2",
			rounding:  RoundingModeFloor,
			precision: 2,
			input:     "1.999",
			want:      "1.99",
		},
		{
			name:      "ceil zero value",
			rounding:  RoundingModeCeiling,
			precision: 0,
			input:     "0",
			want:      "0",
		},
		{
			name:      "floor zero value",
			rounding:  RoundingModeFloor,
			precision: 0,
			input:     "0",
			want:      "0",
		},
		{
			name:      "half_up zero value",
			rounding:  RoundingModeHalfUp,
			precision: 0,
			input:     "0",
			want:      "0",
		},
		{
			name:      "none zero value",
			rounding:  RoundingModeNone,
			precision: 0,
			input:     "0",
			want:      "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &UnitConfig{
				Rounding:  tt.rounding,
				Precision: tt.precision,
			}
			input, err := decimal.NewFromString(tt.input)
			require.NoError(t, err)
			result := uc.Round(input)
			assert.Equal(t, tt.want, result.String())
		})
	}
}
