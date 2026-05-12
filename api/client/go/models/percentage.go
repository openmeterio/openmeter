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
	return []byte(p.Decimal.String()), nil
}

func (p *Percentage) UnmarshalJSON(data []byte) error {
	return p.Decimal.UnmarshalJSON(data)
}

func (p Percentage) String() string {
	return fmt.Sprintf("%s%%", p.Decimal.String())
}
