package currencies

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListCurrenciesInput struct {
	pagination.Page

	// FilterType filters currencies by type: "custom" or "fiat". Nil means no filter.
	FilterType *CurrencyType
}

// CurrencyType distinguishes custom currencies from ISO/fiat ones.
type CurrencyType string

const (
	CurrencyTypeCustom CurrencyType = "custom"
	CurrencyTypeFiat   CurrencyType = "fiat"
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

type ListCostBasesInput struct {
	pagination.Page

	CurrencyID string

	// FilterFiatCode filters cost bases by fiat currency code. Nil means no filter.
	FilterFiatCode *string
}

type CostBases = []CostBasis
