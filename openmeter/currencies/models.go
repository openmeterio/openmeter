package currencies

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type Currency struct {
	models.ManagedModel
	models.NamespacedID
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

// OrderBy specifies the field to sort currencies by.
type OrderBy string

const (
	OrderByCode OrderBy = "code"
	OrderByName OrderBy = "name"
)

func (o OrderBy) Validate() error {
	switch o {
	case OrderByCode, OrderByName, "":
		return nil
	}
	return fmt.Errorf("invalid order by: %s", o)
}

var _ models.Validator = (*ListCurrenciesInput)(nil)

type ListCurrenciesInput struct {
	pagination.Page

	Namespace string `json:"namespace"`

	// FilterType filters currencies by type: "custom" or "fiat". Nil means no filter.
	FilterType *CurrencyType `json:"filter_type,omitempty"`
	// Code filters currencies by code field. Nil means no filter.
	Code *filter.FilterString `json:"code,omitempty"`

	OrderBy OrderBy
	Order   sortx.Order
}

func (i ListCurrenciesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.FilterType != nil {
		if err := i.FilterType.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("filter_type: %w", err))
		}
	}

	if i.Code != nil {
		if err := i.Code.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("code: %w", err))
		}
	}

	if err := i.OrderBy.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

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

const (
	CurrencyTypeCustom CurrencyType = "custom"
	CurrencyTypeFiat   CurrencyType = "fiat"
)

var _ models.Validator = (*CreateCurrencyInput)(nil)

type CreateCurrencyInput struct {
	Namespace string `json:"namespace"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
}

func (i CreateCurrencyInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.Code == "" {
		errs = append(errs, errors.New("code is required"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if i.Symbol == "" {
		errs = append(errs, errors.New("symbol is required"))
	}

	return errors.Join(errs...)
}

type CostBasis struct {
	models.ManagedModel
	models.NamespacedID
	CurrencyID    string                `json:"currency_id"`
	FiatCode      string                `json:"fiat_code"`
	Rate          alpacadecimal.Decimal `json:"rate"`
	EffectiveFrom time.Time             `json:"effective_from"`
}

var _ models.Validator = (*CreateCostBasisInput)(nil)

type CreateCostBasisInput struct {
	Namespace     string                `json:"namespace"`
	CurrencyID    string                `json:"currency_id"`
	FiatCode      string                `json:"fiat_code"`
	Rate          alpacadecimal.Decimal `json:"rate"`
	EffectiveFrom *time.Time            `json:"effective_from,omitempty"`
}

func (i CreateCostBasisInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CurrencyID == "" {
		errs = append(errs, errors.New("currency_id is required"))
	}

	if i.FiatCode == "" {
		errs = append(errs, errors.New("fiat_code is required"))
	}

	if !i.Rate.IsPositive() {
		errs = append(errs, errors.New("rate must be positive"))
	}

	return errors.Join(errs...)
}

type ListCostBasesInput struct {
	pagination.Page

	Namespace  string `json:"namespace"`
	CurrencyID string `json:"currency_id"`

	// FilterFiatCode filters cost bases by fiat currency code. Nil means no filter.
	FilterFiatCode *string `json:"filter_fiat_code,omitempty"`
}

func (i ListCostBasesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CurrencyID == "" {
		errs = append(errs, errors.New("currency_id is required"))
	}

	return errors.Join(errs...)
}
