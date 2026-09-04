package billingcommon

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func FromAPIChargeCostBasis(in *api.BillingChargeCostBasis) (*costbasis.Intent, error) {
	if in == nil {
		return nil, nil
	}

	costBasisType, err := in.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("invalid cost basis: %w", err)
	}

	var fiatCurrencyCode *api.CurrencyCode
	var input costbasis.NewIntentFromFieldsInput

	switch costBasisType {
	case string(api.BillingChargeCostBasisDynamicTypeDynamic):
		dynamic, err := in.AsBillingChargeCostBasisDynamic()
		if err != nil {
			return nil, fmt.Errorf("invalid dynamic cost basis: %w", err)
		}

		input.Mode = costbasis.ModeDynamic
		fiatCurrencyCode = &dynamic.FiatCurrency
	case string(api.BillingChargeCostBasisManualTypeManual):
		manual, err := in.AsBillingChargeCostBasisManual()
		if err != nil {
			return nil, fmt.Errorf("invalid manual cost basis: %w", err)
		}

		rate, err := alpacadecimal.NewFromString(manual.Rate)
		if err != nil {
			return nil, fmt.Errorf("invalid cost basis rate: %w", err)
		}

		input.Mode = costbasis.ModeManual
		input.Rate = &rate
		fiatCurrencyCode = manual.FiatCurrency
	case string(api.BillingChargeCostBasisPinnedTypePinned):
		pinned, err := in.AsBillingChargeCostBasisPinned()
		if err != nil {
			return nil, fmt.Errorf("invalid pinned cost basis: %w", err)
		}

		input.Mode = costbasis.ModePinned
		input.CurrencyCostBasisID = &pinned.CostBasisId
		fiatCurrencyCode = &pinned.FiatCurrency
	default:
		return nil, fmt.Errorf("invalid cost basis type: %s", costBasisType)
	}

	if fiatCurrencyCode != nil {
		input.FiatCurrency, err = currencyx.NewFiatCurrency(*fiatCurrencyCode)
		if err != nil {
			return nil, fmt.Errorf("invalid cost basis fiat currency: %w", err)
		}
	}

	intent, err := costbasis.NewIntentFromFields(input)
	if err != nil {
		return nil, fmt.Errorf("invalid cost basis: %w", err)
	}

	return &intent, nil
}

// ToAPIChargeResolvedCostBasis pairs the resolved state with the fiat currency
// it converts into, which is not part of the state itself.
func ToAPIChargeResolvedCostBasis(fiat *currencyx.FiatCurrency, state *costbasis.State) (*api.BillingChargeResolvedCostBasis, error) {
	if state == nil {
		return nil, nil
	}

	if fiat == nil {
		return nil, errors.New("resolved cost basis without a fiat currency")
	}

	return &api.BillingChargeResolvedCostBasis{
		FiatCurrency: api.CurrencyCode(fiat.Details().Code),
		Rate:         state.CostBasis.String(),
		CostBasisId:  state.CostBasisID,
		ResolvedAt:   state.ResolvedAt,
	}, nil
}
