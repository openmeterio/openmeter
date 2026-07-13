package currencyx

import (
	"errors"
	"fmt"

	"github.com/invopop/gobl/currency"
)

var _ fmt.Stringer = (*Code)(nil)

// Code represents a fiat or custom currency code. Code values used directly as
// Currency values are treated as fiat currencies for backwards compatibility.
type Code currency.Code

func (c Code) String() string {
	return string(c)
}

func (c Code) Validate() error {
	if c == "" {
		return errors.New("currency code is required")
	}

	return currency.Code(c).Validate()
}
