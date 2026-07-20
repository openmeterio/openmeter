package currencies

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CurrencyType distinguishes custom currencies from ISO/fiat ones.
type CurrencyType string

func (t CurrencyType) Validate() error {
	switch t {
	case CurrencyTypeCustom, CurrencyTypeFiat:
		return nil
	default:
		return fmt.Errorf("currency type: %s", t)
	}
}

func (t CurrencyType) Values() []CurrencyType {
	return []CurrencyType{
		CurrencyTypeCustom,
		CurrencyTypeFiat,
	}
}

const (
	CurrencyTypeCustom CurrencyType = "custom"
	CurrencyTypeFiat   CurrencyType = "fiat"
)

type Currency struct {
	models.ManagedModel
	models.NamespacedID
	currencyx.Currency

	// CostBasis is included only if the Currency is expanded.
	CostBasis *[]CostBasis `json:"-"`
}
