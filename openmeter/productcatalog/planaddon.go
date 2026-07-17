package productcatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanAddonMeta struct {
	models.Metadata
	models.Annotations

	PlanAddonConfig
}

type PlanAddonConfig struct {
	// FromPlanPhase
	FromPlanPhase string `json:"fromPlanPhase"`

	// MaxQuantity caps how many units a customer may assign for multiple-instance addons.
	// nil means unlimited; a positive integer enforces a hard cap; zero or negative is invalid.
	MaxQuantity *int `json:"maxQuantity"`
}

var (
	_ models.Validator                  = (*PlanAddon)(nil)
	_ models.CustomValidator[PlanAddon] = (*PlanAddon)(nil)
)

type PlanAddon struct {
	PlanAddonMeta

	// Plan
	Plan Plan `json:"plan"`

	// Addon
	Addon Addon `json:"addon"`
}

func (c PlanAddon) ValidateWith(validators ...models.ValidatorFunc[PlanAddon]) error {
	return models.Validate(c, validators...)
}

// ValidationErrors returns a list of possible validation error(s) regarding to compatibility of the plan and add-on in the assignment.
// It returns nil if the plan and add-on are compatible.
func (c PlanAddon) ValidationErrors() (models.ValidationIssues, error) {
	err := c.Validate()
	if err == nil {
		return nil, nil
	}

	return models.AsValidationIssues(err)
}

func (c PlanAddon) Validate() error {
	var errs []error

	// Validate plan

	// Check plan status
	allowedPlanStatuses := []PlanStatus{PlanStatusDraft, PlanStatusActive, PlanStatusScheduled}
	if !lo.Contains(allowedPlanStatuses, c.Plan.Status()) {
		errs = append(errs, models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(models.NewFieldSelector("plan")),
			ErrPlanAddonIncompatibleStatus,
		))
	}

	// Validate add-on

	addonPrefix := models.NewFieldSelectorGroup(models.NewFieldSelector("addon"))

	// Add-on must be active and the effective period of add-on must be open-ended
	// as we do not support scheduled changes for add-ons.
	if c.Addon.Status() != AddonStatusActive || c.Addon.EffectiveTo != nil {
		errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix, ErrPlanAddonIncompatibleStatus))
	}

	// Validate add-on assignment

	switch c.Addon.InstanceType {
	case AddonInstanceTypeMultiple:
		if c.MaxQuantity != nil && *c.MaxQuantity <= 0 {
			errs = append(errs, ErrPlanAddonMaxQuantityMustBeSet)
		}
	case AddonInstanceTypeSingle:
		if c.MaxQuantity != nil {
			errs = append(errs, ErrPlanAddonMaxQuantityMustNotBeSet)
		}
	}

	if err := c.Plan.ValidateWith(
		ValidateAddonBillingCadenceAreAlignedWithPlan(c.Addon.RateCards),
	); err != nil {
		errs = append(errs, err)
	}

	_, fromPhaseIdx, ok := lo.FindIndexOf(c.Plan.Phases, func(item Phase) bool {
		return item.Key == c.FromPlanPhase
	})

	if ok {
		// Validate ratecards from plan phases and addon.
		for _, phase := range c.Plan.Phases[fromPhaseIdx:] {
			if err := phase.ValidateWith(
				ValidatePlanPhaseAndAddonRateCardsAreCompatible(c.Addon.RateCards),
			); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix, err))
			}
		}

		if err := ValidatePlanAddonRateCardCurrencies()(c); err != nil {
			errs = append(errs, err)
		}
	} else {
		errs = append(errs, ErrPlanAddonUnknownPlanPhaseKey)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ValidatePlanAddonRateCardCurrencies validates effective rate card currencies
// starting at FromPlanPhase and for every subsequent phase where the add-on
// remains applied. Full assignment validation remains responsible for rejecting
// an unknown starting phase.
func ValidatePlanAddonRateCardCurrencies() models.ValidatorFunc[PlanAddon] {
	return func(pa PlanAddon) error {
		var errs []error
		planPrefix := models.NewFieldSelectorGroup(models.NewFieldSelector("plan"))
		addonPrefix := models.NewFieldSelectorGroup(models.NewFieldSelector("addon"))

		if pa.Plan.Currency == nil {
			errs = append(errs, models.ErrorWithFieldPrefix(planPrefix, ErrCurrencyInvalid))
		}
		if pa.Addon.Currency == nil {
			errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix, ErrCurrencyInvalid))
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}

		if pa.Plan.Currency.IsFiat() && pa.Addon.Currency.IsFiat() &&
			!pa.Plan.Currency.Equal(pa.Addon.Currency) {
			errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix, ErrPlanMultipleFiatCurrencies))
		}

		_, fromPhaseIdx, ok := lo.FindIndexOf(pa.Plan.Phases, func(item Phase) bool {
			return item.Key == pa.FromPlanPhase
		})
		if !ok {
			return errors.Join(errs...)
		}

		for _, phase := range pa.Plan.Phases[fromPhaseIdx:] {
			if err := phase.ValidateWith(
				ValidatePlanPhaseAndAddonRateCardCurrencies(pa.Plan.Currency, pa.Addon),
			); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(addonPrefix, err))
			}
		}

		return errors.Join(errs...)
	}
}

// ValidatePlanPhaseAndAddonRateCardCurrencies ensures an add-on cannot change
// the currency of an existing priced rate card and that newly priced rate
// cards preserve the plan's single-fiat constraint.
func ValidatePlanPhaseAndAddonRateCardCurrencies(planCurrency currencyx.CurrencyIdentity, addon Addon) models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		if planCurrency == nil || addon.Currency == nil {
			return ErrCurrencyInvalid
		}

		var errs []error

		phaseRateCardsByKey := lo.SliceToMap(p.RateCards, func(item RateCard) (string, RateCard) {
			return item.Key(), item
		})

		for _, addonRateCard := range addon.RateCards {
			addonMeta := addonRateCard.AsMeta()
			if addonMeta.Price == nil {
				continue
			}

			fieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("ratecards").
					WithExpression(models.NewFieldAttrValue("key", addonRateCard.Key())),
				models.NewFieldSelector("currency"),
			)
			addonCurrency := addonMeta.EffectiveCurrency(addon.Currency)

			planRateCard, found := phaseRateCardsByKey[addonRateCard.Key()]
			if found && planRateCard.AsMeta().Price != nil {
				planRateCardMeta := planRateCard.AsMeta()
				planRateCardCurrency := planRateCardMeta.EffectiveCurrency(planCurrency)
				if !addonCurrency.Equal(planRateCardCurrency) {
					errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrPlanAddonCurrencyMismatch))
				}

				continue
			}

			switch {
			case !planCurrency.IsFiat() && !addonCurrency.Equal(planCurrency):
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardCurrencyOverrideNotAllowed))
			case planCurrency.IsFiat() && addonCurrency.IsFiat() && !addonCurrency.Equal(planCurrency):
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrPlanMultipleFiatCurrencies))
			}
		}

		return errors.Join(errs...)
	}
}

// ValidatePlanAddonWithCurrencies validates the managed currencies introduced
// by an add-on against the plan's invoice fiat. Structural compatibility is
// handled by PlanAddon.Validate.
func ValidatePlanAddonWithCurrencies(ctx context.Context, namespace string, resolver CurrencyResolver) models.ValidatorFunc[PlanAddon] {
	return func(pa PlanAddon) error {
		if resolver == nil {
			return errors.New("currency resolver is required")
		}

		planCurrency, err := existingOrResolveCurrency(
			ctx,
			namespace,
			resolver,
			pa.Plan.Currency,
			models.NewFieldSelectorGroup(
				models.NewFieldSelector("plan"),
				models.NewFieldSelector("currency"),
			),
		)
		if err != nil {
			return err
		}

		costBasisAvailable := map[costBasisPairKey]bool{}
		var errs []error

		for _, rateCard := range pa.Addon.RateCards {
			meta := rateCard.AsMeta()
			if meta.Price == nil {
				continue
			}

			identity := meta.EffectiveCurrency(pa.Addon.Currency)
			fieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("addon"),
				models.NewFieldSelector("ratecards").
					WithExpression(models.NewFieldAttrValue("key", rateCard.Key())),
				models.NewFieldSelector("currency"),
			)

			resolvedCurrency, err := existingOrResolveCurrency(ctx, namespace, resolver, identity, fieldSelector)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if planCurrency.Type() != currencyx.CurrencyTypeFiat || resolvedCurrency.Type() != currencyx.CurrencyTypeCustom {
				continue
			}

			code := resolvedCurrency.GetCode()
			managedCustomCurrency, ok := resolvedCurrency.(currencyx.ManagedCurrency)
			if !ok {
				return fmt.Errorf("resolved custom currency %q has no managed resource identity", code)
			}

			pairKey := costBasisPairKey{
				customCurrencyID: managedCustomCurrency.GetID(),
				fiatCurrencyCode: planCurrency.GetCode(),
			}
			hasCostBasis, ok := costBasisAvailable[pairKey]
			if !ok {
				hasCostBasis, err = resolver.HasCostBasis(ctx, namespace, managedCustomCurrency, planCurrency)
				if err != nil {
					return fmt.Errorf("checking cost basis for currency %q: %w", code, err)
				}

				costBasisAvailable[pairKey] = hasCostBasis
			}

			if !hasCostBasis {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrCurrencyCostBasisNotFound))
			}
		}

		return errors.Join(errs...)
	}
}

func ValidateAddonBillingCadenceAreAlignedWithPlan(addonRateCards RateCards) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		for _, rc := range addonRateCards {
			cad := rc.GetBillingCadence()
			if cad == nil {
				continue
			}

			if err := ValidateBillingCadencesAlign(p.BillingCadence, *cad); err != nil {
				return err
			}
		}

		return nil
	}
}

func ValidatePlanPhaseAndAddonRateCardsAreCompatible(addonRateCards RateCards) models.ValidatorFunc[Phase] {
	return func(p Phase) error {
		var errs []error

		phaseRateCardsByKey := lo.SliceToMap(p.RateCards, func(item RateCard) (string, RateCard) {
			return item.Key(), item
		})

		for _, addonRateCard := range addonRateCards {
			phaseRateCard, ok := phaseRateCardsByKey[addonRateCard.Key()]

			// Add-on ratecard is not present in plan phase ratecards, it is safe to skip.
			if !ok {
				continue
			}

			if err := NewRateCardWithOverlay(phaseRateCard, addonRateCard).Validate(); err != nil {
				errs = append(errs, fmt.Errorf("plan ratecard is not compatible with add-on ratecard: %w", err))
			}
		}

		return errors.Join(errs...)
	}
}
