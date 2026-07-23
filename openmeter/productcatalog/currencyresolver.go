package productcatalog

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CostBasisChecker verifies whether a custom-to-fiat conversion pair has at
// least one configured cost basis.
type CostBasisChecker interface {
	HasCostBasis(ctx context.Context, namespace, customCurrencyID string, fiatCurrencyCode currencyx.Code) (bool, error)
}

type costBasisPairKey struct {
	customCurrencyID string
	fiatCurrencyCode currencyx.Code
}

func validateResolvedCurrency(identity currencyx.CurrencyIdentity, fieldSelector *models.FieldDescriptor) (currencyx.CurrencyIdentity, error) {
	if identity == nil {
		return nil, models.ErrorWithFieldPrefix(fieldSelector, ErrCurrencyInvalid)
	}

	if err := identity.Validate(); err != nil {
		return nil, models.ErrorWithFieldPrefix(fieldSelector, err)
	}

	if identity.IsCustom() {
		managed, ok := identity.(currencyx.ManagedCurrency)
		if !ok || managed.GetID() == "" {
			return nil, models.ErrorWithFieldPrefix(fieldSelector, ErrCurrencyInvalid)
		}
	}

	return identity, nil
}

func validateCostBasisChecker(checker CostBasisChecker) error {
	if checker == nil {
		return errors.New("cost basis checker is required")
	}

	return nil
}
