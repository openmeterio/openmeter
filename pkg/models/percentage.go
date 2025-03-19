package models

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
)

type Percentage struct {
	alpacadecimal.Decimal
}

// NewPercentage creates a new Percentage from a numeric value, representation:
// 50% is represented as 50
func NewPercentage[T float64 | int | alpacadecimal.Decimal](value T) Percentage {
	p := Percentage{}

	switch v := any(value).(type) {
	case int:
		p = Percentage{Decimal: alpacadecimal.NewFromInt(int64(v))}
	case float64:
		p = Percentage{Decimal: alpacadecimal.NewFromFloat(v)}
	case alpacadecimal.Decimal:
		p = Percentage{Decimal: v}
	}

	return p
}

func (p Percentage) MarshalJSON() ([]byte, error) {
	// alpacadecimal by default marshals to a string with quotes
	return []byte(p.Decimal.String()), nil
}

func (p *Percentage) UnmarshalJSON(data []byte) error {
	// alpacadecimal supports unmarshaling both from quoted and non-quoted strings
	return p.Decimal.UnmarshalJSON(data)
}

func (p Percentage) String() string {
	return fmt.Sprintf("%s%%", p.Decimal.String())
}

// ApplyTo applies the percentage to a value, e.g:
// NewPercentage(50).ApplyTo(100) = 50
func (p Percentage) ApplyTo(value alpacadecimal.Decimal) alpacadecimal.Decimal {
	return value.Mul(p.Decimal).Div(alpacadecimal.NewFromInt(100))
}

// ApplyMarkupTo applies the percentage to a value as a markup, e.g:
// NewPercentage(50).ApplyMarkupTo(100) = 150
func (p Percentage) ApplyMarkupTo(value alpacadecimal.Decimal) alpacadecimal.Decimal {
	return value.Mul(p.Decimal.Add(alpacadecimal.NewFromInt(100))).Div(alpacadecimal.NewFromInt(100))
}
