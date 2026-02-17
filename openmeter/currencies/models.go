package currencies

import (
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Currency represents a currency.
type Currency struct {
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Symbol               string `json:"symbol,omitempty"`
	SmallestDenomination int8   `json:"smallest_denomination,omitempty"`
	IsCustom             bool
	DisambiguateSymbol   string
	Subunits             uint32
}

// CreateCurrencyInput represents the input for creating a currency.
type CreateCurrencyInput struct {
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	Symbol               string `json:"symbol"`
	SmallestDenomination int8   `json:"smallest_denomination"`
}

// ListCurrenciesInput represents the input for listing currencies.
type ListCurrenciesInput struct {
	pagination.Page
}
