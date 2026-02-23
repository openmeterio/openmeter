package currencies

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
)

type Currency struct {
	ID       string
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol,omitempty"`
	IsCustom bool
}

type CreateCurrencyInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

type CostBasis struct {
	ID            string                `json:"id"`
	CurrencyID    string                `json:"currency_id"`
	FiatCode      string                `json:"fiat_code"`
	Rate          alpacadecimal.Decimal `json:"rate"`
	EffectiveFrom time.Time             `json:"effective_from"`
}

type CreateCostBasisInput struct {
	CurrencyID    string
	FiatCode      string                `json:"fiat_code"`
	Rate          alpacadecimal.Decimal `json:"rate"`
	EffectiveFrom *time.Time            `json:"effective_from"`
}

type GetCostBasisInput struct {
	CurrencyID string `json:"currency_id"`
}

type CostBases = []CostBasis
