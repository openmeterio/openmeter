package currencies

import (
	"errors"
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
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Symbol *string `json:"symbol,omitempty"`
}

func (c Currency) String() string {
	return c.GetCode().String()
}

func (c Currency) GetCode() currencyx.Code {
	return currencyx.Code(c.Code)
}

func (c Currency) GetID() string {
	return c.ID
}

func (c Currency) Type() currencyx.CurrencyType {
	return c.GetCode().Type()
}

func (c Currency) IsFiat() bool {
	return c.GetCode().IsFiat()
}

func (c Currency) IsCustom() bool {
	return c.GetCode().IsCustom()
}

func (c Currency) Equal(other currencyx.CurrencyIdentity) bool {
	if other == nil || !c.IsCustom() || !other.IsCustom() {
		return false
	}

	managed, ok := other.(currencyx.ManagedCurrency)
	return ok && c.ID != "" && c.ID == managed.GetID()
}

func (c Currency) Validate() error {
	var errs []error

	if err := c.GetCode().Validate(); err != nil {
		errs = append(errs, fmt.Errorf("code: %w", err))
	}

	if c.IsCustom() && c.ID == "" {
		errs = append(errs, errors.New("managed custom currency ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var (
	_ currencyx.CurrencyIdentity = (*Currency)(nil)
	_ currencyx.ManagedCurrency  = (*Currency)(nil)
)
