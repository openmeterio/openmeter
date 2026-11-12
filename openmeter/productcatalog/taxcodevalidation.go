package productcatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateRateCardTaxCodes checks that every Tax Code the rate cards reference exists in the
// namespace and isn't deleted. Plan, add-on and subscription services all check Tax Code
// references here, so changing how soft-deleted codes are treated only touches this function.
//
// BatchResolve returns deleted codes too (old rows may still point at them), which is why the
// deleted check lives here rather than in the resolver.
//
// allowDeleted lists Tax Code IDs the caller accepts even when soft-deleted: references inherited
// from something already stored, not picked in this request. Missing IDs are rejected regardless.
func ValidateRateCardTaxCodes(ctx context.Context, resolver TaxCodeResolver, namespace string, rateCards RateCards, allowDeleted ...string) error {
	ids := rateCards.TaxCodeIDs()
	if len(ids) == 0 {
		return nil
	}

	taxCodes, err := resolver.BatchResolve(ctx, namespace, ids...)
	if err != nil {
		return fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	return checkRateCardTaxCodes(rateCards, taxCodes, allowDeleted)
}

// checkRateCardTaxCodes classifies the referenced Tax Codes against an already-fetched
// BatchResolve result: a missing ID is not found, a soft-deleted one is rejected unless allowed.
// Shared by the single-call and per-phase validators so both report the same issues with the same
// rate-card-key-prefixed field.
func checkRateCardTaxCodes(rateCards RateCards, taxCodes map[string]*taxcode.TaxCode, allowDeleted []string) error {
	var errs []error

	for _, rc := range rateCards {
		cfg := rc.AsMeta().TaxConfig
		if cfg == nil || cfg.TaxCodeID == "" {
			continue
		}

		tc := taxCodes[cfg.TaxCodeID]

		switch {
		case tc == nil:
			errs = append(errs, rateCardTaxCodeError(rc, cfg.TaxCodeID, ErrRateCardTaxCodeNotFound))
		case tc.IsDeleted() && !lo.Contains(allowDeleted, cfg.TaxCodeID):
			errs = append(errs, rateCardTaxCodeError(rc, cfg.TaxCodeID, ErrRateCardTaxCodeDeleted))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// rateCardTaxCodeError prefixes the tax code issue with the rate card key. Callers add the phase
// prefix on top where a phase context exists.
func rateCardTaxCodeError(rc RateCard, taxCodeID string, issue models.ValidationIssue) error {
	fieldSelector := models.NewFieldSelectorGroup(
		models.NewFieldSelector("ratecards").WithExpression(
			models.NewFieldAttrValue("key", rc.Key()),
		),
	)

	return models.ErrorWithFieldPrefix(fieldSelector,
		fmt.Errorf("%s [ratecard.key=%s tax_code.id=%s]: %w",
			issue.Message(), rc.Key(), taxCodeID, issue),
	)
}

// ValidatePlanWithTaxCodes verifies every phase's Tax Code references, rejecting missing and
// soft-deleted codes. It is the publish gate for plans: a draft may inherit deleted references,
// but a version cannot be published while it holds one. Issues are prefixed with the phase key.
func ValidatePlanWithTaxCodes(ctx context.Context, resolver TaxCodeResolver, namespace string) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		ids := p.TaxCodeIDs()
		if len(ids) == 0 {
			return nil
		}

		taxCodes, err := resolver.BatchResolve(ctx, namespace, ids...)
		if err != nil {
			return fmt.Errorf("failed to resolve tax codes: %w", err)
		}

		var errs []error

		for _, phase := range p.Phases {
			phaseErr := checkRateCardTaxCodes(phase.RateCards, taxCodes, nil)
			if phaseErr == nil {
				continue
			}

			phaseFieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("phases").
					WithExpression(
						models.NewFieldAttrValue("key", phase.Key),
					),
			)

			errs = append(errs, models.ErrorWithFieldPrefix(phaseFieldSelector, phaseErr))
		}

		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}
}

// ValidateAddonWithTaxCodes verifies the add-on's Tax Code references, rejecting missing and
// soft-deleted codes. It is the publish gate for add-ons: a draft may inherit deleted references,
// but a version cannot be published while it holds one.
func ValidateAddonWithTaxCodes(ctx context.Context, resolver TaxCodeResolver, namespace string) models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		ids := a.RateCards.TaxCodeIDs()
		if len(ids) == 0 {
			return nil
		}

		taxCodes, err := resolver.BatchResolve(ctx, namespace, ids...)
		if err != nil {
			return fmt.Errorf("failed to resolve tax codes: %w", err)
		}

		return checkRateCardTaxCodes(a.RateCards, taxCodes, nil)
	}
}
