package creditpurchase

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// SchemaLevel identifies the persisted credit-purchase cost-basis representation.
// Zero is reserved for aggregates that have not been loaded from persistence.
type SchemaLevel int

const (
	SchemaLevelLegacy    SchemaLevel = 1
	SchemaLevelCostBasis SchemaLevel = 2
)

func (SchemaLevel) Values() []int {
	return []int{
		int(SchemaLevelLegacy),
		int(SchemaLevelCostBasis),
	}
}

func (s SchemaLevel) Validate() error {
	if !slices.Contains(s.Values(), int(s)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid credit purchase schema level: %d", s))
	}

	return nil
}

type CostBasisType string

const (
	CostBasisTypeFiat           CostBasisType = "fiat"
	CostBasisTypeCustomCurrency CostBasisType = "custom_currency"
)

func (CostBasisType) Values() []string {
	return []string{
		string(CostBasisTypeFiat),
		string(CostBasisTypeCustomCurrency),
	}
}

func (t CostBasisType) Validate() error {
	if !slices.Contains(t.Values(), string(t)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid credit purchase cost basis type: %s", t))
	}

	return nil
}

// FiatCostBasis is the fixed scalar cost basis of a fiat-denominated credit
// purchase. Its currency is always the charge currency.
type FiatCostBasis struct {
	Rate alpacadecimal.Decimal `json:"rate"`
}

func (c FiatCostBasis) Validate() error {
	if !c.Rate.IsPositive() {
		return models.NewGenericValidationError(errors.New("rate must be positive"))
	}

	return nil
}

// CostBasis keeps fiat purchase pricing separate from the custom-currency
// cost-basis intent, whose manual, pinned, and dynamic invariants remain owned
// by the shared costbasis package.
type CostBasis struct {
	kind CostBasisType

	fiat           *FiatCostBasis
	customCurrency *chargecostbasis.Intent
}

var _ models.Validator = (*CostBasis)(nil)

func NewCostBasis[T FiatCostBasis | chargecostbasis.Intent](in T) CostBasis {
	switch value := any(in).(type) {
	case FiatCostBasis:
		return CostBasis{
			kind: CostBasisTypeFiat,
			fiat: &value,
		}
	case chargecostbasis.Intent:
		return CostBasis{
			kind:           CostBasisTypeCustomCurrency,
			customCurrency: &value,
		}
	default:
		return CostBasis{}
	}
}

func (c CostBasis) Type() CostBasisType {
	return c.kind
}

func (c *CostBasis) Validate() error {
	if c == nil {
		return models.NewGenericValidationError(errors.New("cost basis is required"))
	}

	if err := c.kind.Validate(); err != nil {
		return err
	}

	switch c.kind {
	case CostBasisTypeFiat:
		if c.fiat == nil {
			return models.NewGenericValidationError(errors.New("fiat cost basis is required"))
		}

		if c.customCurrency != nil {
			return models.NewGenericValidationError(errors.New("custom currency cost basis must be absent for fiat cost basis"))
		}

		return c.fiat.Validate()
	case CostBasisTypeCustomCurrency:
		if c.customCurrency == nil {
			return models.NewGenericValidationError(errors.New("custom currency cost basis is required"))
		}

		if c.fiat != nil {
			return models.NewGenericValidationError(errors.New("fiat cost basis must be absent for custom currency cost basis"))
		}

		return c.customCurrency.Validate()
	default:
		return models.NewGenericValidationError(fmt.Errorf("invalid credit purchase cost basis type: %s", c.kind))
	}
}

func (c CostBasis) AsFiat() (FiatCostBasis, error) {
	if c.kind != CostBasisTypeFiat {
		return FiatCostBasis{}, models.NewGenericValidationError(errors.New("cost basis is not fiat"))
	}

	if c.fiat == nil {
		return FiatCostBasis{}, models.NewGenericValidationError(errors.New("fiat cost basis is required"))
	}

	return *c.fiat, nil
}

func (c CostBasis) AsCustomCurrency() (chargecostbasis.Intent, error) {
	if c.kind != CostBasisTypeCustomCurrency {
		return chargecostbasis.Intent{}, models.NewGenericValidationError(errors.New("cost basis is not custom currency"))
	}

	if c.customCurrency == nil {
		return chargecostbasis.Intent{}, models.NewGenericValidationError(errors.New("custom currency cost basis is required"))
	}

	return c.customCurrency.Clone(), nil
}

type ResolvedCostBasis struct {
	FiatCurrency *currencyx.FiatCurrency
	Rate         alpacadecimal.Decimal
}

func (c ResolvedCostBasis) Validate() error {
	var errs []error

	if c.FiatCurrency == nil {
		errs = append(errs, errors.New("fiat currency is required"))
	} else if err := c.FiatCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("fiat currency: %w", err))
	}

	if !c.Rate.IsPositive() {
		errs = append(errs, errors.New("rate must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetResolvedCostBasis returns the fiat currency and per-credit rate used by
// every monetary consumer. It never consults the legacy settlement shadow.
func (c ChargeBase) GetResolvedCostBasis() (ResolvedCostBasis, error) {
	if c.Intent.Settlement.Type() == SettlementTypePromotional {
		return ResolvedCostBasis{}, errors.New("promotional credit purchase does not have a cost basis")
	}

	if c.Intent.CostBasis == nil {
		return ResolvedCostBasis{}, errors.New("cost basis is required")
	}

	switch c.Intent.CostBasis.Type() {
	case CostBasisTypeFiat:
		costBasis, err := c.Intent.CostBasis.AsFiat()
		if err != nil {
			return ResolvedCostBasis{}, err
		}

		fiatCurrency, err := currencyx.NewFiatCurrency(c.Intent.Currency.GetCode())
		if err != nil {
			return ResolvedCostBasis{}, fmt.Errorf("mapping credit currency to fiat currency: %w", err)
		}

		resolved := ResolvedCostBasis{
			FiatCurrency: fiatCurrency,
			Rate:         costBasis.Rate,
		}

		return resolved, resolved.Validate()
	case CostBasisTypeCustomCurrency:
		intent, err := c.Intent.CostBasis.AsCustomCurrency()
		if err != nil {
			return ResolvedCostBasis{}, err
		}

		fiatCurrency, err := intent.GetFiatCurrency()
		if err != nil {
			return ResolvedCostBasis{}, fmt.Errorf("getting custom-currency fiat currency: %w", err)
		}
		if c.State.ResolvedCostBasis == nil {
			return ResolvedCostBasis{}, errors.New("custom-currency cost basis is unresolved")
		}

		resolved := ResolvedCostBasis{
			FiatCurrency: fiatCurrency,
			Rate:         c.State.ResolvedCostBasis.CostBasis,
		}

		return resolved, resolved.Validate()
	default:
		return ResolvedCostBasis{}, fmt.Errorf("unsupported credit purchase cost basis type: %s", c.Intent.CostBasis.Type())
	}
}
