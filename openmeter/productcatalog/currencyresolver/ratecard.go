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

// ResolveCurrenciesForRateCards resolves every code-only rate-card currency in
// one batch while retaining already managed custom-currency identities.
func ResolveCurrenciesForRateCards(ctx context.Context, resolver currencies.CurrencyResolver, namespace string, rateCards *productcatalog.RateCards) error {
	if rateCards == nil || len(*rateCards) == 0 {
		return nil
	}

	if resolver == nil {
		return errors.New("currency resolver is required")
	}

	type pendingCurrency struct {
		rateCard      productcatalog.RateCard
		ref           currencies.CurrencyRef
		fieldSelector *models.FieldDescriptor
	}

	pending := make([]pendingCurrency, 0, len(*rateCards))
	refs := make([]currencies.CurrencyRef, 0, len(*rateCards))
	var errs []error

	for _, rateCard := range *rateCards {
		identity := rateCard.AsMeta().Currency
		if identity == nil {
			continue
		}

		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("rateCards").
				WithExpression(models.NewFieldAttrValue("key", rateCard.Key())),
			models.NewFieldSelector("currency"),
		)

		ref, shouldResolve, err := currencyRefForResolution(identity)
		if err != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, err))
			continue
		}

		if !shouldResolve {
			continue
		}

		pending = append(pending, pendingCurrency{
			rateCard:      rateCard,
			ref:           ref,
			fieldSelector: fieldSelector,
		})
		refs = append(refs, ref)
	}

	if len(refs) == 0 {
		return errors.Join(errs...)
	}

	resolved, err := resolver.BatchResolveCurrencies(ctx, namespace, refs...)
	if err != nil {
		return fmt.Errorf("failed to resolve currencies: %w", err)
	}

	for _, item := range pending {
		currency := resolved[item.ref]
		if currency == nil {
			errs = append(errs, models.ErrorWithFieldPrefix(item.fieldSelector, productcatalog.ErrCurrencyNotFound))
			continue
		}

		if err := setRateCardCurrency(item.rateCard, currency); err != nil {
			return fmt.Errorf("setting rate card currency [ratecard.key=%s]: %w", item.rateCard.Key(), err)
		}
	}

	return errors.Join(errs...)
}

func setRateCardCurrency(rateCard productcatalog.RateCard, identity currencyx.CurrencyIdentity) error {
	return rateCard.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Currency = identity
		return meta, nil
	})
}
