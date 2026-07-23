package productcatalog

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateCurrency requires the runtime state needed by currency-aware
// validators to be resolved while preserving intrinsic reference validation.
func ValidateCurrency() models.ValidatorFunc[currencies.CurrencyReference] {
	return func(reference currencies.CurrencyReference) error {
		if !reference.IsResolved() {
			return fmt.Errorf("currency reference %q is not resolved", reference.Code)
		}

		if reference.IsCustom() {
			currency, _ := reference.CustomCurrency()
			if currency.DeletedAt != nil && !currency.DeletedAt.After(clock.Now()) {
				return ErrCurrencyNotFound
			}
		}

		return reference.Validate()
	}
}

func ValidateCurrencyWithOverride(reference currencies.CurrencyReference) models.ValidatorFunc[*currencies.CurrencyReference] {
	at := clock.Now()

	return func(override *currencies.CurrencyReference) error {
		if override == nil {
			return nil
		}

		if err := ValidateCurrency()(reference); err != nil {
			return fmt.Errorf("invalid currency: %w", err)
		}

		if err := ValidateCurrency()(*override); err != nil {
			return fmt.Errorf("invalid currency override: %w", err)
		}

		switch reference.Code.Type() {
		case currencyx.CurrencyTypeCustom:
			return fmt.Errorf("custom currency cannot be overridden: %w", ErrCurrencyInvalid)
		case currencyx.CurrencyTypeFiat:
			if override.IsFiat() {
				return fmt.Errorf("fiat currency cannot be overridden with another")
			}

			if override.IsCustom() {
				oc, _ := override.CustomCurrency()

				if oc.CostBasis == nil {
					return fmt.Errorf("cost basis was not eager loaded for resolved custom currency %q", override.Code)
				}

				for _, cb := range *oc.CostBasis {
					if cb.FiatCode == reference.Code && cb.IsEffectiveAt(at) {
						return nil
					}
				}

				return fmt.Errorf(
					"currency override has no matching cost-basis [reference.code=%s override.code=%s]: %w",
					reference.Code,
					override.Code,
					ErrCurrencyCostBasisNotFound,
				)
			}

			return fmt.Errorf("unknown currency override type: %q", override.Code)
		default:
			return fmt.Errorf("invalid currency type: %s", reference.Code)
		}
	}
}
