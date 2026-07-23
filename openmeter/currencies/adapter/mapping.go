package adapter

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// CurrencyReference is the persistence representation shared by product
// catalog resources that can reference either a fiat code or a managed custom
// currency. CustomCurrency and its cost-basis edge must be eagerly loaded when
// CustomCurrencyID is set for the resulting domain reference to be resolved.
type CurrencyReference struct {
	FiatCurrencyCode *string
	CustomCurrencyID *string
	CustomCurrency   *entdb.CustomCurrency
}

// FromDBCurrencyReference restores a domain reference. An optional empty
// persistence reference represents inheritance.
func FromDBCurrencyReference(ref CurrencyReference, optional bool) (*currencies.CurrencyReference, error) {
	switch {
	case ref.FiatCurrencyCode != nil && ref.CustomCurrencyID != nil:
		return nil, errors.New("fiat currency code and custom currency ID are mutually exclusive")
	case ref.FiatCurrencyCode != nil:
		currency, err := currencies.NewFiatCurrency(currencyx.Code(*ref.FiatCurrencyCode))
		if err != nil {
			return nil, fmt.Errorf("invalid fiat currency code %q: %w", *ref.FiatCurrencyCode, err)
		}

		reference := currency.Reference()
		return &reference, nil
	case ref.CustomCurrencyID != nil:
		if ref.CustomCurrency == nil {
			return nil, fmt.Errorf("custom currency %q is not loaded", *ref.CustomCurrencyID)
		}
		if ref.CustomCurrency.ID != *ref.CustomCurrencyID {
			return nil, fmt.Errorf("loaded custom currency %q does not match reference %q", ref.CustomCurrency.ID, *ref.CustomCurrencyID)
		}

		currency, err := FromDBCustomCurrency(ref.CustomCurrency)
		if err != nil {
			return nil, err
		}
		if err := currency.Validate(); err != nil {
			return nil, fmt.Errorf("invalid custom currency %q: %w", *ref.CustomCurrencyID, err)
		}

		reference := currency.Reference()
		return &reference, nil
	case optional:
		return nil, nil
	default:
		return nil, errors.New("currency reference is required")
	}
}

// ToDBCurrencyReference keeps fiat currencies as codes and custom currencies
// as managed resource IDs. An optional nil reference represents inheritance.
func ToDBCurrencyReference(reference *currencies.CurrencyReference, optional bool) (CurrencyReference, error) {
	if reference == nil {
		if optional {
			return CurrencyReference{}, nil
		}

		return CurrencyReference{}, errors.New("currency reference is required")
	}

	if err := reference.Validate(); err != nil {
		return CurrencyReference{}, fmt.Errorf("invalid currency: %w", err)
	}

	if reference.IsFiat() {
		code := reference.Code.String()
		return CurrencyReference{FiatCurrencyCode: &code}, nil
	}

	if reference.CustomCurrencyID == nil {
		return CurrencyReference{}, fmt.Errorf("custom currency %q has no managed resource identity", reference.Code)
	}

	return CurrencyReference{CustomCurrencyID: reference.CustomCurrencyID}, nil
}
