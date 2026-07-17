package currencies

import (
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

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

func NewFiatCurrency(code currencyx.Code) (Currency, error) {
	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
		WithCode(code).
		Build()
	if err != nil {
		return Currency{}, fmt.Errorf("build fiat currency: %w", err)
	}

	return Currency{Currency: currency}, nil
}

func (c Currency) Validate() error {
	var errs []error

	if c.Currency == nil {
		errs = append(errs, errors.New("currency is required"))
	} else {
		if err := c.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}

		if c.Currency.IsCustom() && c.ID == "" {
			errs = append(errs, errors.New("managed custom currency ID is required"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetCode returns the resolved currency code, or an empty code when the currency is missing.
func (c Currency) GetCode() currencyx.Code {
	if c.Currency == nil {
		return ""
	}

	return c.Currency.Details().Code
}

func (c Currency) IsFiat() bool {
	return c.Currency != nil && c.Currency.Type() == currencyx.CurrencyTypeFiat
}

func (c Currency) IsCustom() bool {
	return c.Currency != nil && c.Currency.Type() == currencyx.CurrencyTypeCustom
}

func (c Currency) GetID() string {
	return c.ID
}

// Equal compares fiat currencies by code and managed custom currencies by resource ID.
// Custom codes can be reused after archival, so their codes do not establish identity.
func (c Currency) Equal(other currencyx.CurrencyIdentity) bool {
	if c.Currency == nil || other == nil {
		return false
	}

	if c.Currency.IsFiat() {
		return c.Currency.Equal(other)
	}

	if !other.IsCustom() {
		return false
	}

	managed, ok := other.(currencyx.ManagedCurrency)
	return ok && c.ID != "" && c.ID == managed.GetID()
}

func (c Currency) Clone() Currency {
	if c.CostBasis != nil {
		c.CostBasis = lo.ToPtr(slices.Clone(*c.CostBasis))
	}

	return c
}

// Identity uniquely identifies a currency by its fiat code or custom currency ID.
func (c Currency) Identity() (string, error) {
	if c.IsFiat() {
		return fmt.Sprintf("FIAT:%s", c.GetCode()), nil
	}

	if c.IsCustom() {
		return fmt.Sprintf("CUSTOM:%s:%s", c.Namespace, c.ID), nil
	}

	return "", fmt.Errorf("currency is not fiat or custom")
}

var (
	_ models.Validator          = Currency{}
	_ currencyx.Currency        = (*Currency)(nil)
	_ currencyx.ManagedCurrency = (*Currency)(nil)
)
