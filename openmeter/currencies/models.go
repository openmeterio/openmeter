package currencies

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
)

type Currency struct {
	ID                   string
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Symbol               string `json:"symbol,omitempty"`
	SmallestDenomination int8   `json:"smallest_denomination,omitempty"`
	IsCustom             bool
	DisambiguateSymbol   string
	Subunits             uint32
}

type CreateCurrencyInput struct {
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Symbol               string `json:"symbol"`
	SmallestDenomination int8   `json:"smallest_denomination"`
}

type CostBasis struct {
	ID            string
	CurrencyID    string                `json:"currency_id"`
	FiatCode      string                `json:"fiat_code"`
	Rate          alpacadecimal.Decimal `json:"rate"`
	EffectiveFrom time.Time             `json:"effective_from"`
	CreatedAt     time.Time
}

type CreateCostBasisInput struct {
	CurrencyID    string     `json:"currency_id"`
	FiatCode      string     `json:"fiat_code"`
	Rate          float32    `json:"rate"`
	EffectiveFrom *time.Time `json:"effective_from"`
}

type GetCostBasisInput struct {
	ID string
}

type CostBases = []CostBasis
