package adapter

import (
	"errors"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// CurrencyReference is the persistence representation shared by product
// catalog resources that can reference either a fiat code or a managed custom
// currency. CustomCurrency must be eagerly loaded when CustomCurrencyID is set.
type CurrencyReference struct {
	FiatCurrencyCode *string
	CustomCurrencyID *string
	CustomCurrency   *entdb.CustomCurrency
}

// FromDBCurrencyReference restores either a fiat value identity or a managed
// custom identity. An optional empty reference represents inheritance.
func FromDBCurrencyReference(ref CurrencyReference, optional bool) (currencyx.CurrencyIdentity, error) {
	switch {
	case ref.FiatCurrencyCode != nil && ref.CustomCurrencyID != nil:
		return nil, errors.New("fiat currency code and custom currency ID are mutually exclusive")
	case ref.FiatCurrencyCode != nil:
		identity, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
			WithCode(currencyx.Code(*ref.FiatCurrencyCode)).
			Build()
		if err != nil {
			return nil, fmt.Errorf("invalid fiat currency code %q: %w", *ref.FiatCurrencyCode, err)
		}

		return identity, nil
	case ref.CustomCurrencyID != nil:
		if ref.CustomCurrency == nil {
			return nil, fmt.Errorf("custom currency %q is not loaded", *ref.CustomCurrencyID)
		}
		if ref.CustomCurrency.ID != *ref.CustomCurrencyID {
			return nil, fmt.Errorf("loaded custom currency %q does not match reference %q", ref.CustomCurrency.ID, *ref.CustomCurrencyID)
		}

		identity, err := FromDBCustomCurrency(ref.CustomCurrency)
		if err != nil {
			return nil, err
		}
		if err := identity.Validate(); err != nil {
			return nil, fmt.Errorf("invalid custom currency %q: %w", *ref.CustomCurrencyID, err)
		}

		return identity, nil
	case optional:
		return nil, nil
	default:
		return nil, errors.New("currency reference is required")
	}
}

// ToDBCurrencyReference keeps fiat currencies as codes and custom currencies as
// managed resource IDs. An optional nil identity represents inheritance.
func ToDBCurrencyReference(identity currencyx.CurrencyIdentity, optional bool) (CurrencyReference, error) {
	if identity == nil {
		if optional {
			return CurrencyReference{}, nil
		}

		return CurrencyReference{}, errors.New("currency reference is required")
	}

	if err := identity.Validate(); err != nil {
		return CurrencyReference{}, fmt.Errorf("invalid currency: %w", err)
	}

	if identity.IsFiat() {
		code := identity.GetCode().String()
		return CurrencyReference{FiatCurrencyCode: &code}, nil
	}

	managed, ok := identity.(currencyx.ManagedCurrency)
	if !ok || managed.GetID() == "" {
		return CurrencyReference{}, fmt.Errorf("custom currency %q has no managed resource identity", identity.GetCode())
	}

	id := managed.GetID()
	return CurrencyReference{CustomCurrencyID: &id}, nil
}
