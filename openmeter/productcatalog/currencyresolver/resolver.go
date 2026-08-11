package currencyresolver

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ResolveCurrenciesForAddon(ctx context.Context, resolver currencies.NamespacedCurrencyResolver, addon *productcatalog.Addon) error {
	if addon == nil {
		return errors.New("add-on is required")
	}

	if err := ResolveCurrency(ctx, resolver, &addon.Currency); err != nil {
		return models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(models.NewFieldSelector("currency")),
			err,
		)
	}

	if err := ResolveCurrenciesForRateCards(ctx, resolver, &addon.RateCards); err != nil {
		return fmt.Errorf("resolving add-on currencies: %w", err)
	}

	return nil
}

func ResolveCurrenciesForPlan(ctx context.Context, resolver currencies.NamespacedCurrencyResolver, plan *productcatalog.Plan) error {
	if plan == nil {
		return errors.New("plan is required")
	}

	if err := ResolveCurrency(ctx, resolver, &plan.Currency); err != nil {
		return models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(models.NewFieldSelector("currency")),
			err,
		)
	}

	var errs []error
	for idx := range plan.Phases {
		phase := &plan.Phases[idx]
		if err := ResolveCurrenciesForRateCards(ctx, resolver, &phase.RateCards); err != nil {
			fieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("phases").
					WithExpression(models.NewFieldAttrValue("key", phase.Key)),
			)
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, err))
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("resolving plan currencies: %w", err)
	}

	return nil
}

// ResolveCurrenciesForRateCards resolves every code-only rate-card currency in
// one batch while retaining already managed custom-currency identities.
func ResolveCurrenciesForRateCards(
	ctx context.Context,
	resolver currencies.NamespacedCurrencyResolver,
	rateCards *productcatalog.RateCards,
) error {
	if rateCards == nil || len(*rateCards) == 0 {
		return nil
	}

	if resolver == nil {
		return errors.New("currency resolver is required")
	}

	refsByRateCardIdx := make(map[int]currencies.CurrencyRef, len(*rateCards))

	var errs []error

	for idx, rateCard := range *rateCards {
		reference := rateCard.AsMeta().Currency

		if reference == nil {
			continue
		}

		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("rateCards").
				WithExpression(models.NewFieldAttrValue("key", rateCard.Key())),
			models.NewFieldSelector("currency"),
		)

		if reference.IsCostBasisResolved() {
			if err := reference.Validate(); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, err))
			} else if err := validateResolvedCurrencyNamespace(resolver, reference); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, err))
			}

			continue
		}

		refsByRateCardIdx[idx] = currencies.CurrencyRef{
			Code: reference.Code,
			ID:   lo.FromPtr(reference.CustomCurrencyID),
		}
	}

	if len(refsByRateCardIdx) == 0 {
		return errors.Join(errs...)
	}

	result, err := resolver.BatchResolveCurrencies(ctx, slices.Collect(maps.Values(refsByRateCardIdx))...)
	if err != nil {
		return fmt.Errorf("failed to resolve currencies: %w", err)
	}

	for idx, ref := range refsByRateCardIdx {
		rc := (*rateCards)[idx]

		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("rateCards").
				WithExpression(models.NewFieldAttrValue("key", rc.Key())),
			models.NewFieldSelector("currency"),
		)

		resolvedCurrency := result[ref]
		if resolvedCurrency == nil {
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, productcatalog.ErrCurrencyNotFound))

			continue
		}

		reference := lo.FromPtr(rc.AsMeta().Currency)

		if reference.Code != resolvedCurrency.GetCode() {
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, productcatalog.ErrCurrencyNotFound))
		}

		if reference.CustomCurrencyID != nil && lo.FromPtr(reference.CustomCurrencyID) != resolvedCurrency.ID {
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, productcatalog.ErrCurrencyNotFound))
		}

		if err = rc.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			resolvedReference, err := rc.AsMeta().Currency.WithCurrency(resolvedCurrency)
			if err != nil {
				return m, fmt.Errorf("invalid resolved currency reference [ratecard.key=%s]: %w", rc.Key(), err)
			}

			m.Currency = &resolvedReference

			return m, nil
		}); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// ResolveCurrency populates the runtime currency and stable custom-currency ID
// of an authoring reference. Already resolved references are returned unchanged.
func ResolveCurrency(ctx context.Context, resolver currencies.NamespacedCurrencyResolver, reference *currencies.CurrencyReference) error {
	if resolver == nil {
		return errors.New("currency resolver is required")
	}

	if reference == nil {
		return errors.New("currency reference is required")
	}

	if err := reference.Validate(); err != nil {
		return fmt.Errorf("invalid currency reference: %w", err)
	}

	if reference.IsCostBasisResolved() {
		return validateResolvedCurrencyNamespace(resolver, reference)
	}

	resolved, err := resolver.ResolveCurrency(ctx, currencies.CurrencyRef{
		Code: reference.Code,
		ID:   lo.FromPtr(reference.CustomCurrencyID),
	})
	if err != nil {
		if models.IsGenericNotFoundError(err) {
			return productcatalog.ErrCurrencyNotFound
		}

		return fmt.Errorf("resolving currency %q: %w", reference.Code, err)
	}

	r, err := reference.WithCurrency(resolved)
	if err != nil {
		return fmt.Errorf("invalid currency %q: %w", reference.Code, err)
	}

	*reference = r

	return nil
}

func validateResolvedCurrencyNamespace(resolver currencies.NamespacedCurrencyResolver, reference *currencies.CurrencyReference) error {
	if reference.IsFiat() || !reference.IsCostBasisResolved() {
		return nil
	}

	resolved, ok := reference.CustomCurrency()
	if !ok || resolved.Namespace != resolver.Namespace() {
		return productcatalog.ErrCurrencyNotFound
	}

	return nil
}
