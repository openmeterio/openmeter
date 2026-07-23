package currencyresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ResolveCurrency replaces a code-only authoring currency with its resolved
// identity. Already resolved currencies are returned unchanged.
func ResolveCurrency(ctx context.Context, resolver currencies.CurrencyResolver, namespace string, identity currencyx.CurrencyIdentity) (currencyx.CurrencyIdentity, error) {
	if resolver == nil {
		return nil, errors.New("currency resolver is required")
	}

	ref, shouldResolve, err := currencyRefForResolution(identity)
	if err != nil {
		return nil, err
	}

	if !shouldResolve {
		return identity, nil
	}

	resolved, err := resolver.ResolveCurrency(ctx, namespace, ref)
	if err == nil {
		return resolved, nil
	}

	if models.IsGenericNotFoundError(err) {
		return nil, productcatalog.ErrCurrencyNotFound
	}

	return nil, fmt.Errorf("resolving currency %q: %w", identity.GetCode(), err)
}

func currencyRefForResolution(identity currencyx.CurrencyIdentity) (currencies.CurrencyRef, bool, error) {
	if identity == nil {
		return currencies.CurrencyRef{}, false, productcatalog.ErrCurrencyInvalid
	}

	if err := identity.Validate(); err != nil {
		return currencies.CurrencyRef{}, false, err
	}

	if identity.IsCustom() {
		if managed, ok := identity.(currencyx.ManagedCurrency); ok && managed.GetID() != "" {
			return currencies.CurrencyRef{}, false, nil
		}
	} else if _, ok := identity.(currencyx.Currency); ok {
		return currencies.CurrencyRef{}, false, nil
	}

	return currencies.CurrencyRef{Code: identity.GetCode()}, true, nil
}
