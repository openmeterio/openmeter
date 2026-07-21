package currencies

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type Service interface {
	CurrencyService
	CostBasisService
}

type CurrencyService interface {
	ListCurrencies(ctx context.Context, params ListCurrenciesInput) (pagination.Result[Currency], error)
	CreateCurrency(ctx context.Context, params CreateCurrencyInput) (Currency, error)
	GetCurrency(ctx context.Context, params GetCurrencyInput) (Currency, error)
}

type CostBasisService interface {
	CreateCostBasis(ctx context.Context, params CreateCostBasisInput) (CostBasis, error)
	GetCostBasis(ctx context.Context, params GetCostBasisInput) (CostBasis, error)
	ListCostBases(ctx context.Context, params ListCostBasesInput) (pagination.Result[CostBasis], error)
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

// FilteringOptions controls how sibling currency filters are combined.
// The default behavior intersects filters; Union combines them with OR.
// This option is internal to the currencies service until cross-field filter
// composition is supported by pkg/filter.
type FilteringOptions struct {
	Union bool `json:"-"`
}

type ListCurrenciesInput struct {
	pagination.Page
	FilteringOptions

	Namespace string `json:"namespace"`

	// CurrencyType filters currencies by type: "custom" or "fiat". Nil means no filter.
	CurrencyType *CurrencyType `json:"currency_type,omitempty"`
	// ID filters currencies by managed resource ID. Fiat currencies have no ID.
	ID *filter.FilterString `json:"id,omitempty"`
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

	if i.CurrencyType != nil {
		if err := i.CurrencyType.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency_type: %w", err))
		}
	}

	if i.ID != nil {
		if err := i.ID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("id: %w", err))
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ models.Validator = (*CreateCurrencyInput)(nil)

type CreateCurrencyInput struct {
	currencyx.CurrencyDetails
	Namespace string `json:"namespace"`
}

func (i CreateCurrencyInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	_, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(i.Code).
		WithName(i.Name).
		WithPrecision(i.Precision).
		WithSymbol(i.Symbol).
		WithDecimalMark(i.DecimalMark).
		WithThousandsSeparator(i.ThousandsSeparator).
		Build()
	if err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ models.Validator = (*CreateCostBasisInput)(nil)

type CreateCostBasisInput struct {
	Namespace     string                `json:"namespace"`
	CurrencyID    string                `json:"currency_id"`
	FiatCode      currencyx.Code        `json:"fiat_code"`
	Rate          alpacadecimal.Decimal `json:"rate"`
	EffectiveFrom *time.Time            `json:"effective_from,omitempty"`
	EffectiveTo   *time.Time            `json:"effective_to,omitempty"`
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ models.Validator = (*GetCostBasisInput)(nil)

type GetCostBasisInput struct {
	models.NamespacedID
	CostBasisExpandOptions
}

func (i GetCostBasisInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ models.Validator = (*ListCostBasesInput)(nil)

type ListCostBasesInput struct {
	pagination.Page

	Namespace  string `json:"namespace"`
	CurrencyID string `json:"currency_id"`

	// FilterFiatCode filters cost bases by fiat currency code. Nil means no filter.
	FilterFiatCode *currencyx.Code `json:"filter_fiat_code,omitempty"`
}

func (i ListCostBasesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CurrencyID == "" {
		errs = append(errs, errors.New("currency_id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CostBasisExpandOptions struct {
	CustomCurrency bool
}

type CurrencyExpandOptions struct {
	CostBasis bool
}

type GetCurrencyInput struct {
	models.NamespacedID
	CurrencyExpandOptions
}

func (i GetCurrencyInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
