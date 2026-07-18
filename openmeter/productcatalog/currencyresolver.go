package productcatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CurrencyResolver interface {
	Resolve(ctx context.Context, namespace string, code currencyx.Code) (currencyx.CurrencyIdentity, error)
	HasCostBasis(ctx context.Context, namespace string, customCurrency currencyx.ManagedCurrency, fiatCurrency currencyx.CurrencyIdentity) (bool, error)
}

// ResolvePlanCurrencies replaces code-only authoring currencies with resolved
// currency identities before the plan crosses the persistence boundary.
func ResolvePlanCurrencies(ctx context.Context, namespace string, resolver CurrencyResolver, plan *Plan) error {
	if plan == nil {
		return errors.New("plan is required")
	}

	if resolver == nil {
		return errors.New("currency resolver is required")
	}

	planCurrency, err := existingOrResolveCurrency(
		ctx,
		namespace,
		resolver,
		plan.Currency,
		models.NewFieldSelectorGroup(models.NewFieldSelector("currency")),
	)
	if err != nil {
		return err
	}

	plan.Currency = planCurrency

	var errs []error
	for _, phase := range plan.Phases {
		for _, rateCard := range phase.RateCards {
			meta := rateCard.AsMeta()
			if meta.Currency == nil {
				continue
			}

			fieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("phases").
					WithExpression(models.NewFieldAttrValue("key", phase.Key)),
				models.NewFieldSelector("rateCards").
					WithExpression(models.NewFieldAttrValue("key", rateCard.Key())),
				models.NewFieldSelector("currency"),
			)

			rateCardCurrency, err := existingOrResolveCurrency(ctx, namespace, resolver, meta.Currency, fieldSelector)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if err = setRateCardCurrency(rateCard, rateCardCurrency); err != nil {
				return fmt.Errorf("setting rate card currency [phase.key=%s ratecard.key=%s]: %w", phase.Key, rateCard.Key(), err)
			}
		}
	}

	return errors.Join(errs...)
}

// ResolveAddonCurrencies replaces code-only authoring currencies with resolved
// currency identities before the add-on crosses the persistence boundary.
func ResolveAddonCurrencies(ctx context.Context, namespace string, resolver CurrencyResolver, addon *Addon) error {
	if addon == nil {
		return errors.New("add-on is required")
	}

	if resolver == nil {
		return errors.New("currency resolver is required")
	}

	addonCurrency, err := existingOrResolveCurrency(
		ctx,
		namespace,
		resolver,
		addon.Currency,
		models.NewFieldSelectorGroup(models.NewFieldSelector("currency")),
	)
	if err != nil {
		return err
	}

	addon.Currency = addonCurrency

	var errs []error
	for _, rateCard := range addon.RateCards {
		meta := rateCard.AsMeta()
		if meta.Currency == nil {
			continue
		}

		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("rateCards").
				WithExpression(models.NewFieldAttrValue("key", rateCard.Key())),
			models.NewFieldSelector("currency"),
		)

		rateCardCurrency, err := existingOrResolveCurrency(ctx, namespace, resolver, meta.Currency, fieldSelector)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err = setRateCardCurrency(rateCard, rateCardCurrency); err != nil {
			return fmt.Errorf("setting add-on rate card currency [ratecard.key=%s]: %w", rateCard.Key(), err)
		}
	}

	return errors.Join(errs...)
}

func resolveCurrency(ctx context.Context, namespace string, resolver CurrencyResolver, code currencyx.Code, fieldSelector *models.FieldDescriptor) (currencyx.CurrencyIdentity, error) {
	identity, err := resolver.Resolve(ctx, namespace, code)
	if err == nil {
		return identity, nil
	}

	if models.IsGenericNotFoundError(err) {
		return nil, models.ErrorWithFieldPrefix(fieldSelector, ErrCurrencyNotFound)
	}

	return nil, fmt.Errorf("resolving currency %q: %w", code, err)
}

type costBasisPairKey struct {
	customCurrencyID string
	fiatCurrencyCode currencyx.Code
}

func existingOrResolveCurrency(ctx context.Context, namespace string, resolver CurrencyResolver, existing currencyx.CurrencyIdentity, fieldSelector *models.FieldDescriptor) (currencyx.CurrencyIdentity, error) {
	if existing == nil {
		return nil, models.ErrorWithFieldPrefix(fieldSelector, ErrCurrencyInvalid)
	}

	if err := existing.Validate(); err != nil {
		return nil, models.ErrorWithFieldPrefix(fieldSelector, err)
	}

	if existing.IsCustom() {
		if managed, ok := existing.(currencyx.ManagedCurrency); ok && managed.GetID() != "" {
			return existing, nil
		}
	} else if _, ok := existing.(currencyx.Currency); ok {
		return existing, nil
	}

	return resolveCurrency(ctx, namespace, resolver, existing.GetCode(), fieldSelector)
}

func setRateCardCurrency(rateCard RateCard, identity currencyx.CurrencyIdentity) error {
	return rateCard.ChangeMeta(func(meta RateCardMeta) (RateCardMeta, error) {
		meta.Currency = identity
		return meta, nil
	})
}
