package productcatalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	PlanStatusDraft     PlanStatus = "draft"
	PlanStatusActive    PlanStatus = "active"
	PlanStatusArchived  PlanStatus = "archived"
	PlanStatusScheduled PlanStatus = "scheduled"
	PlanStatusInvalid   PlanStatus = "invalid"
)

type PlanStatus string

func (s PlanStatus) Values() []PlanStatus {
	return []PlanStatus{
		PlanStatusDraft,
		PlanStatusActive,
		PlanStatusArchived,
		PlanStatusScheduled,
	}
}

func (s PlanStatus) Validate() error {
	if !slices.Contains(s.Values(), s) {
		return fmt.Errorf("invalid plan status: %s", s)
	}
	return nil
}

var (
	_ models.Validator     = (*Plan)(nil)
	_ models.Equaler[Plan] = (*Plan)(nil)
)

type Plan struct {
	PlanMeta

	// Phases
	Phases []Phase `json:"phases"`
}

// ValidationErrors returns a list of possible validation errors for the plan.
// It returns nil if the plan has no validation issues.
func (p Plan) ValidationErrors() (models.ValidationIssues, error) {
	return models.AsValidationIssues(p.Validate())
}

func (p Plan) ValidateWith(validators ...models.ValidatorFunc[Plan]) error {
	return models.Validate(p, validators...)
}

// HasUnitConfig reports whether any rate card in any phase carries a unit_config
// conversion. The v1 API cannot represent unit_config, so v1 read and mutation
// surfaces use this to reject such plans instead of silently stripping the field.
func (p Plan) HasUnitConfig() bool {
	return lo.SomeBy(p.Phases, func(ph Phase) bool {
		return ph.RateCards.HasUnitConfig()
	})
}

// HasCurrencyOverrides reports whether any rate card explicitly overrides the
// plan currency. The v1 API cannot represent these overrides, so v1 read and
// mutation surfaces use this to avoid silently stripping them.
func (p Plan) HasCurrencyOverrides() bool {
	return lo.SomeBy(p.Phases, func(ph Phase) bool {
		return ph.RateCards.HasCurrencyOverride()
	})
}

// UsesCustomCurrency reports whether the plan or any of its rate cards uses a custom currency.
func (p Plan) UsesCustomCurrency() bool {
	if p.Currency != nil && p.Currency.IsCustom() {
		return true
	}

	return lo.SomeBy(p.Phases, func(ph Phase) bool {
		return ph.RateCards.HasCustomCurrency()
	})
}

func ValidatePlanMeta() models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		return p.PlanMeta.Validate()
	}
}

func ValidatePlanPhases() models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		var errs []error

		if len(p.Phases) == 0 {
			return ErrPlanWithNoPhases
		}

		phaseKeys := make(map[string]struct{}, len(p.Phases))

		lastPhaseIdx := len(p.Phases) - 1

		for idx, phase := range p.Phases {
			phaseFieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("phases").
					WithExpression(
						models.NewFieldAttrValue("key", phase.Key),
					),
			)

			if idx != lastPhaseIdx {
				if phase.Duration == nil {
					errs = append(errs, models.ErrorWithFieldPrefix(phaseFieldSelector, ErrPlanHasNonLastPhaseWithNoDuration))
				}
			} else {
				if phase.Duration != nil {
					errs = append(errs, models.ErrorWithFieldPrefix(phaseFieldSelector, ErrPlanHasLastPhaseWithDuration))
				}
			}

			// Check for duplicated phase keys
			if _, ok := phaseKeys[phase.Key]; ok {
				selector := models.NewFieldSelectorGroup(
					models.NewFieldSelector("phases").
						WithExpression(
							models.NewFieldAttrValue("key", phase.Key),
						),
					models.NewFieldSelector("key"),
				)

				errs = append(errs, models.ErrorWithFieldPrefix(selector, ErrPlanPhaseDuplicatedKey))
			}
			phaseKeys[phase.Key] = struct{}{}

			if err := phase.Validate(); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(phaseFieldSelector, err))
			}
		}

		return errors.Join(errs...)
	}
}

// ValidatePlanBillingCadenceLiteral validates that the billing cadence of the plan is at least a month.
func ValidatePlanBillingCadenceLiteral() models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		var errs []error

		isoString := p.BillingCadence.ISOString()

		if !lo.Contains(ErrPlanBillingCadenceAllowedValues, isoString) {
			errs = append(errs, ErrPlanBillingCadenceInvalid)
		}

		return errors.Join(errs...)
	}
}

// ValidatePlanHasAlignedBillingCadences validates that the billing cadence of the plan is aligned with the billing cadence of the rate cards.
func ValidatePlanHasAlignedBillingCadences() models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		var errs []error

		for _, phase := range p.Phases {
			for _, rateCard := range phase.RateCards.Billables() {
				rateCardFieldSelector := models.NewFieldSelectorGroup(
					models.NewFieldSelector("phases").
						WithExpression(
							models.NewFieldAttrValue("key", phase.Key),
						),
					models.NewFieldSelector("rateCards").
						WithExpression(
							models.NewFieldAttrValue("key", rateCard.Key()),
						),
				)

				if rateCard.GetBillingCadence() != nil {
					rateCardBillingCadence := lo.FromPtr(rateCard.GetBillingCadence())

					if err := ValidateBillingCadencesAlign(p.BillingCadence, rateCardBillingCadence); err != nil {
						errs = append(errs, models.ErrorWithFieldPrefix(rateCardFieldSelector, err))
					}
				}
			}
		}

		return errors.Join(errs...)
	}
}

// ValidatePlanRateCardCurrencies enforces the allowed relationship between the
// plan's default currency and rate card overrides. Managed-resource existence
// and cost-basis availability are validated separately by the plan service.
func ValidatePlanRateCardCurrencies() models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		if p.Currency == nil {
			return ErrCurrencyInvalid
		}

		var errs []error

		for _, phase := range p.Phases {
			for _, rateCard := range phase.RateCards {
				override := rateCard.AsMeta().Currency
				if override == nil {
					continue
				}

				fieldSelector := models.NewFieldSelectorGroup(
					models.NewFieldSelector("phases").
						WithExpression(models.NewFieldAttrValue("key", phase.Key)),
					models.NewFieldSelector("rateCards").
						WithExpression(models.NewFieldAttrValue("key", rateCard.Key())),
					models.NewFieldSelector("currency"),
				)

				switch {
				case !p.Currency.IsFiat():
					errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardCurrencyOverrideNotAllowed))
				case override.Equal(p.Currency):
					errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardCurrencyOverrideRedundant))
				case override.IsFiat():
					errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrPlanMultipleFiatCurrencies))
				}
			}
		}

		return errors.Join(errs...)
	}
}

// ValidatePlanWithCurrencies validates managed currency references and ensures
// custom rate card currencies under a fiat plan have a configured cost-basis
// pair. It intentionally checks pair availability, not a time-specific rate;
// charges select the effective rate at their full service period start.
func ValidatePlanWithCurrencies(ctx context.Context, namespace string, resolver CurrencyResolver) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		if resolver == nil {
			return errors.New("currency resolver is required")
		}

		var errs []error

		planCurrency, err := existingOrResolveCurrency(
			ctx,
			namespace,
			resolver,
			p.Currency,
			models.NewFieldSelectorGroup(models.NewFieldSelector("currency")),
		)
		if err != nil {
			return err
		}

		costBasisAvailable := map[costBasisPairKey]bool{}

		for _, phase := range p.Phases {
			for _, rateCard := range phase.RateCards {
				meta := rateCard.AsMeta()
				override := meta.Currency
				if override == nil {
					continue
				}

				fieldSelector := models.NewFieldSelectorGroup(
					models.NewFieldSelector("phases").
						WithExpression(models.NewFieldAttrValue("key", phase.Key)),
					models.NewFieldSelector("rateCards").
						WithExpression(models.NewFieldAttrValue("key", rateCard.Key())),
					models.NewFieldSelector("currency"),
				)

				resolvedOverride, err := existingOrResolveCurrency(ctx, namespace, resolver, override, fieldSelector)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				if planCurrency.Type() != currencyx.CurrencyTypeFiat || resolvedOverride.Type() != currencyx.CurrencyTypeCustom {
					continue
				}

				overrideCode := resolvedOverride.GetCode()
				managedCustomCurrency, ok := resolvedOverride.(currencyx.ManagedCurrency)
				if !ok {
					return fmt.Errorf("resolved custom currency %q has no managed resource identity", overrideCode)
				}

				pairKey := costBasisPairKey{
					customCurrencyID: managedCustomCurrency.GetID(),
					fiatCurrencyCode: planCurrency.GetCode(),
				}
				hasCostBasis, ok := costBasisAvailable[pairKey]
				if !ok {
					hasCostBasis, err = resolver.HasCostBasis(ctx, namespace, managedCustomCurrency, planCurrency)
					if err != nil {
						return fmt.Errorf("checking cost basis for currency %q: %w", overrideCode, err)
					}

					costBasisAvailable[pairKey] = hasCostBasis
				}

				if !hasCostBasis {
					errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrCurrencyCostBasisNotFound))
				}
			}
		}

		return errors.Join(errs...)
	}
}

func (p Plan) Validate() error {
	return p.ValidateWith(
		ValidatePlanMeta(),
		ValidatePlanPhases(),
		ValidatePlanRateCardCurrencies(),
		ValidatePlanBillingCadenceLiteral(),
		ValidatePlanHasAlignedBillingCadences(),
	)
}

// Equal returns true if the two Plans are equal.
func (p Plan) Equal(o Plan) bool {
	if !p.PlanMeta.Equal(o.PlanMeta) {
		return false
	}

	if len(p.Phases) != len(o.Phases) {
		return false
	}

	for i, phase := range p.Phases {
		if !phase.Equal(o.Phases[i]) {
			return false
		}
	}

	return true
}

var (
	_ models.Validator             = (*PlanMeta)(nil)
	_ models.CustomValidator[Plan] = (*Plan)(nil)
	_ models.Equaler[PlanMeta]     = (*PlanMeta)(nil)
)

type PlanMeta struct {
	EffectivePeriod

	// Key is the unique key for Plan.
	Key string `json:"key"`

	// Version
	Version int `json:"version"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Currency
	Currency currencyx.CurrencyIdentity `json:"currency"`

	// BillingCadence is the default billing cadence for subscriptions using this plan.
	BillingCadence datetime.ISODuration `json:"billing_cadence"`

	// ProRatingConfig is the default pro-rating configuration for subscriptions using this plan.
	ProRatingConfig ProRatingConfig `json:"pro_rating_config"`

	// SettlementMode is the settlement mode for subscriptions using this plan.
	SettlementMode SettlementMode `json:"settlement_mode"`

	// Metadata
	Metadata models.Metadata `json:"metadata,omitempty"`
}

// Validate validates the PlanMeta.
func (p PlanMeta) Validate() error {
	var errs []error

	if p.Currency == nil {
		errs = append(errs, ErrCurrencyInvalid)
	} else if err := p.Currency.Validate(); err != nil {
		errs = append(errs, models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(models.NewFieldSelector("currency")),
			ErrCurrencyInvalid,
		))
	}

	if err := p.EffectivePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid effective period: %w", err))
	}

	if p.Key == "" {
		errs = append(errs, ErrResourceKeyEmpty)
	}

	if p.Name == "" {
		errs = append(errs, ErrResourceNameEmpty)
	}

	if p.BillingCadence.IsZero() {
		errs = append(errs, fmt.Errorf("invalid BillingCadence: must not be empty"))
	}

	if err := p.ProRatingConfig.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid ProRatingConfig: %s", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Equal returns true if the two PlanMetas are equal.
func (p PlanMeta) Equal(o PlanMeta) bool {
	if p.Key != o.Key {
		return false
	}

	if p.Version != o.Version {
		return false
	}

	if p.Name != o.Name {
		return false
	}

	if p.Description != o.Description {
		return false
	}

	if (p.Currency == nil) != (o.Currency == nil) {
		return false
	}

	if p.Currency != nil && !p.Currency.Equal(o.Currency) {
		return false
	}

	if p.BillingCadence != o.BillingCadence {
		return false
	}

	if !p.ProRatingConfig.Equal(o.ProRatingConfig) {
		return false
	}

	if !p.EffectivePeriod.Equal(o.EffectivePeriod) {
		return false
	}

	if !p.Metadata.Equal(o.Metadata) {
		return false
	}

	if p.SettlementMode != o.SettlementMode {
		return false
	}

	return true
}

// Status returns the current status of the Plan
func (p PlanMeta) Status() PlanStatus {
	return p.StatusAt(clock.Now())
}

// StatusAt returns the plan status relative to time t.
func (p PlanMeta) StatusAt(t time.Time) PlanStatus {
	from := lo.FromPtr(p.EffectiveFrom)
	to := lo.FromPtr(p.EffectiveTo)

	// Plan has DraftStatus if neither the EffectiveFrom nor EffectiveTo are set
	if from.IsZero() && to.IsZero() {
		return PlanStatusDraft
	}

	// Plan has ArchivedStatus if EffectiveTo is in the past relative to time t.
	if from.Before(t) && (to.Before(t) && from.Before(to)) {
		return PlanStatusArchived
	}

	// Plan has ActiveStatus if EffectiveFrom is set in the past relative to time t and EffectiveTo is not set
	// or in the future relative to time t.
	if from.Before(t) && (to.IsZero() || to.After(t)) {
		return PlanStatusActive
	}

	// Plan is ScheduledForActiveStatus if EffectiveFrom is set in the future relative to time t and EffectiveTo is not set
	// or in the future relative to time t.
	if from.After(t) && (to.IsZero() || to.After(from)) {
		return PlanStatusScheduled
	}

	return PlanStatusInvalid
}

func ValidatePlanWithFeatures(ctx context.Context, resolver NamespacedFeatureResolver) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		var errs []error

		for _, phase := range p.Phases {
			phaseFieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("phases").
					WithExpression(
						models.NewFieldAttrValue("key", phase.Key),
					),
			)

			if err := ValidateRateCardsWithFeatures(ctx, resolver)(phase.RateCards); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(phaseFieldSelector, err))
			}
		}

		return errors.Join(errs...)
	}
}
