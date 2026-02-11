package currencies

import (
	"github.com/invopop/gobl/currency"
)

type CurrencyService interface {
	ListCurrencies() ([]*currency.Def, error)
}
