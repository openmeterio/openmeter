package creditpurchase

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CurrentSchemaLevel identifies the persisted credit-purchase representation.
// Keep the marker so future persistence changes can introduce a new level.
const CurrentSchemaLevel = 2

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

type costBasisJSON struct {
	Type                CostBasisType          `json:"type"`
	Mode                chargecostbasis.Mode   `json:"mode,omitempty"`
	FiatCurrency        *currencyx.FiatCode    `json:"fiatCurrency,omitempty"`
	CurrencyCostBasisID *string                `json:"currencyCostBasisID,omitempty"`
	Rate                *alpacadecimal.Decimal `json:"rate,omitempty"`
}

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

func (c CostBasis) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("validating credit purchase cost basis: %w", err)
	}

	serde := costBasisJSON{Type: c.kind}

	switch c.kind {
	case CostBasisTypeFiat:
		serde.Rate = &c.fiat.Rate
	case CostBasisTypeCustomCurrency:
		serde.Mode = c.customCurrency.Kind()

		fiatCurrency, err := c.customCurrency.GetFiatCurrency()
		if err != nil {
			return nil, fmt.Errorf("getting custom-currency fiat currency: %w", err)
		}
		serde.FiatCurrency = lo.ToPtr(fiatCurrency.GetFiatCode())

		switch serde.Mode {
		case chargecostbasis.ModeDynamic:
		case chargecostbasis.ModePinned:
			intent, err := c.customCurrency.AsPinned()
			if err != nil {
				return nil, err
			}
			serde.CurrencyCostBasisID = &intent.CurrencyCostBasisID
		case chargecostbasis.ModeManual:
			intent, err := c.customCurrency.AsManual()
			if err != nil {
				return nil, err
			}
			serde.Rate = &intent.Rate
		default:
			return nil, fmt.Errorf("unsupported custom-currency cost basis mode: %s", serde.Mode)
		}
	default:
		return nil, fmt.Errorf("unsupported credit purchase cost basis type: %s", c.kind)
	}

	return json.Marshal(serde)
}

func (c *CostBasis) UnmarshalJSON(data []byte) error {
	var serde costBasisJSON
	if err := json.Unmarshal(data, &serde); err != nil {
		return fmt.Errorf("deserializing credit purchase cost basis: %w", err)
	}

	var decoded CostBasis

	switch serde.Type {
	case CostBasisTypeFiat:
		if serde.Rate == nil {
			return errors.New("fiat cost basis rate is required")
		}
		if serde.Mode != "" || serde.FiatCurrency != nil || serde.CurrencyCostBasisID != nil {
			return errors.New("fiat cost basis contains custom-currency fields")
		}

		decoded = NewCostBasis(FiatCostBasis{Rate: *serde.Rate})
	case CostBasisTypeCustomCurrency:
		if serde.FiatCurrency == nil {
			return errors.New("custom-currency fiat currency is required")
		}

		fiatCurrency, err := currencyx.NewFiatCurrency(*serde.FiatCurrency)
		if err != nil {
			return fmt.Errorf("mapping custom-currency fiat currency: %w", err)
		}

		switch serde.Mode {
		case chargecostbasis.ModeDynamic:
			if serde.Rate != nil || serde.CurrencyCostBasisID != nil {
				return errors.New("dynamic cost basis contains resolved fields")
			}
			decoded = NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.DynamicIntent{
				FiatCurrency: fiatCurrency,
			}))
		case chargecostbasis.ModePinned:
			if serde.CurrencyCostBasisID == nil {
				return errors.New("pinned currency cost basis ID is required")
			}
			if serde.Rate != nil {
				return errors.New("pinned cost basis contains a manual rate")
			}
			decoded = NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.PinnedIntent{
				FiatCurrency:        fiatCurrency,
				CurrencyCostBasisID: *serde.CurrencyCostBasisID,
			}))
		case chargecostbasis.ModeManual:
			if serde.Rate == nil {
				return errors.New("manual cost basis rate is required")
			}
			if serde.CurrencyCostBasisID != nil {
				return errors.New("manual cost basis contains a pinned cost basis ID")
			}
			decoded = NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
				FiatCurrency: fiatCurrency,
				Rate:         *serde.Rate,
			}))
		default:
			return fmt.Errorf("unsupported custom-currency cost basis mode: %s", serde.Mode)
		}
	default:
		return fmt.Errorf("unsupported credit purchase cost basis type: %s", serde.Type)
	}

	if err := decoded.Validate(); err != nil {
		return fmt.Errorf("validating credit purchase cost basis: %w", err)
	}

	*c = decoded

	return nil
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

func (c CostBasis) GetFiatCurrency(purchaseCurrency currencies.Currency) (*currencyx.FiatCurrency, error) {
	switch c.Type() {
	case CostBasisTypeFiat:
		if purchaseCurrency.IsCustom() {
			return nil, errors.New("fiat cost basis requires a fiat purchase currency")
		}

		fiatCurrency, err := currencyx.NewFiatCurrency(purchaseCurrency.GetCode())
		if err != nil {
			return nil, fmt.Errorf("mapping purchase currency to fiat currency: %w", err)
		}

		return fiatCurrency, nil
	case CostBasisTypeCustomCurrency:
		intent, err := c.AsCustomCurrency()
		if err != nil {
			return nil, err
		}

		fiatCurrency, err := intent.GetFiatCurrency()
		if err != nil {
			return nil, fmt.Errorf("getting custom-currency fiat currency: %w", err)
		}

		return fiatCurrency, nil
	default:
		return nil, fmt.Errorf("unsupported credit purchase cost basis type: %s", c.Type())
	}
}

type ResolvedCostBasis struct {
	FiatCurrency *currencyx.FiatCurrency
	Rate         alpacadecimal.Decimal
}

// FiatAmount converts a credit amount to fiat and rounds it at the settlement
// currency's precision.
func (c ResolvedCostBasis) FiatAmount(creditAmount alpacadecimal.Decimal) alpacadecimal.Decimal {
	return c.FiatCurrency.RoundToPrecision(creditAmount.Mul(c.Rate))
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
	if c.State.ResolvedCostBasis == nil {
		return ResolvedCostBasis{}, errors.New("cost basis is unresolved")
	}

	fiatCurrency, err := c.Intent.CostBasis.GetFiatCurrency(c.Intent.Currency)
	if err != nil {
		return ResolvedCostBasis{}, fmt.Errorf("getting fiat currency: %w", err)
	}

	resolved := ResolvedCostBasis{
		FiatCurrency: fiatCurrency,
		Rate:         c.State.ResolvedCostBasis.CostBasis,
	}

	return resolved, resolved.Validate()
}
