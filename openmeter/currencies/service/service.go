package service

import (
	"github.com/invopop/gobl/currency"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/samber/lo"
)

var _ currencies.CurrencyService = (*Service)(nil)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) ListCurrencies() ([]*currency.Def, error) {
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
