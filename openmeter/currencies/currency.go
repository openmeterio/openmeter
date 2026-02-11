package currencies

import (
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
)

type CurrencyService interface {
	ListCurrencies() ([]*currency.Def, error)
}

func ListCurrencies() ([]*currency.Def, error) {
	defs := currency.Definitions()
	return lo.Map(lo.Filter(
		defs,
		func(def *currency.Def, _ int) bool {
			// NOTE: this filters out non-iso currencies such as crypto
			return def.ISONumeric != ""
		},
	), func(def *currency.Def, _ int) *currency.Def {
		return def
	}), nil
}
