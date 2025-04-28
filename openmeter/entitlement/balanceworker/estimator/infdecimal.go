package estimator

import (
	"math"

	"github.com/alpacahq/alpacadecimal"
)

type InfDecimal struct {
	value alpacadecimal.Decimal

	// infinite is true if the value is +infinite
	infinite bool
}

func NewInfDecimal[T int | float64](value T) InfDecimal {
	return InfDecimal{
		value:    alpacadecimal.NewFromFloat(float64(value)),
		infinite: false,
	}
}

func NewInfDecimalFromDecimal(value alpacadecimal.Decimal) InfDecimal {
	return InfDecimal{
		value:    value,
		infinite: false,
	}
}

func (d InfDecimal) Add(other InfDecimal) InfDecimal {
	return InfDecimal{
		value:    d.value.Add(other.value),
		infinite: d.infinite || other.infinite,
	}
}

func (d InfDecimal) IsNegative() bool {
	if d.infinite {
		return false
	}

	return d.value.IsNegative()
}

func (d InfDecimal) GreaterThanOrEqual(other InfDecimal) bool {
	if d.infinite && !other.infinite {
		return true
	}

	if !d.infinite && other.infinite {
		return false
	}

	// TODO: This is mathematically wrong, but it's ok for our use case
	if d.infinite && other.infinite {
		return true
	}

	return d.value.GreaterThanOrEqual(other.value)
}

func (d InfDecimal) GreaterThan(other InfDecimal) bool {
	if d.infinite && !other.infinite {
		return true
	}

	if !d.infinite && other.infinite {
		return false
	}

	// TODO: This is mathematically wrong, but it's ok for our use case
	if d.infinite && other.infinite {
		return true
	}

	return d.value.GreaterThan(other.value)
}

func (d InfDecimal) InexactFloat64() float64 {
	if d.infinite {
		return math.Inf(1)
	}

	return d.value.InexactFloat64()
}

func (d InfDecimal) MarshalJSON() ([]byte, error) {
	if d.infinite {
		return []byte("\"+inf\""), nil
	}

	return d.value.MarshalJSON()
}

func (d *InfDecimal) UnmarshalJSON(data []byte) error {
	if string(data) == "\"+inf\"" {
		d.infinite = true
		return nil
	}

	return d.value.UnmarshalJSON(data)
}

func (d InfDecimal) String() string {
	if d.infinite {
		return "+inf"
	}

	return d.value.String()
}

var infinite = InfDecimal{
	infinite: true,
}
