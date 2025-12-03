package productcatalog

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	AddonStatusDraft    AddonStatus = "draft"
	AddonStatusActive   AddonStatus = "active"
	AddonStatusArchived AddonStatus = "archived"
	AddonStatusInvalid  AddonStatus = "invalid"
)

type AddonStatus string

func (s AddonStatus) Values() []string {
	return []string{
		string(AddonStatusDraft),
		string(AddonStatusActive),
		string(AddonStatusArchived),
	}
}

var (
	_ models.Validator          = (*AddonMeta)(nil)
	_ models.Equaler[AddonMeta] = (*AddonMeta)(nil)
)

type AddonMeta struct {
	EffectivePeriod

	// Key is the unique key for Add-on.
	Key string `json:"key"`

	// Version
	Version int `json:"version"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Currency
	Currency currency.Code `json:"currency"`

	// InstanceType
	InstanceType AddonInstanceType `json:"instanceType"`

	// Metadata
	Metadata models.Metadata `json:"metadata,omitempty"`

	// Annotations
	Annotations models.Annotations `json:"annotations,omitempty"`
}

func (m AddonMeta) Validate() error {
	var errs []error

	if err := m.EffectivePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid effective period: %w", err))
	}

	if m.Key == "" {
		errs = append(errs, ErrAddonKeyEmpty)
	}

	if m.Name == "" {
		errs = append(errs, ErrAddonNameEmpty)
	}

	if err := m.Currency.Validate(); err != nil {
		errs = append(errs, ErrCurrencyInvalid)
	}

	if err := m.InstanceType.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (m AddonMeta) Equal(v AddonMeta) bool {
	if m.Key != v.Key {
		return false
	}

	if m.Version != v.Version {
		return false
	}

	if m.Name != v.Name {
		return false
	}

	if lo.FromPtr(m.Description) != lo.FromPtr(v.Description) {
		return false
	}

	if m.Currency != v.Currency {
		return false
	}

	if m.InstanceType != v.InstanceType {
		return false
	}

	if !m.EffectivePeriod.Equal(v.EffectivePeriod) {
		return false
	}

	if !m.Metadata.Equal(v.Metadata) {
		return false
	}

	if !maps.Equal(m.Annotations, v.Annotations) {
		return false
	}

	return true
}

// Status returns the current status of the Addons
func (m AddonMeta) Status() AddonStatus {
	return m.StatusAt(clock.Now())
}

// StatusAt returns the Addon status relative to time t.
func (m AddonMeta) StatusAt(t time.Time) AddonStatus {
	from := lo.FromPtr(m.EffectiveFrom)
	to := lo.FromPtr(m.EffectiveTo)

	// Add-on has DraftStatus if neither the EffectiveFrom nor EffectiveTo are set
	if from.IsZero() && to.IsZero() {
		return AddonStatusDraft
	}

	// Add-on has ArchivedStatus if EffectiveTo is in the past relative to time t.
	if from.Before(t) && (to.Before(t) && from.Before(to)) {
		return AddonStatusArchived
	}

	// Add-on has ActiveStatus if EffectiveFrom is set in the past relative to time t and EffectiveTo is not set
	// or in the future relative to time t.
	if from.Before(t) && (to.IsZero() || to.After(t)) {
		return AddonStatusActive
	}

	return AddonStatusInvalid
}

var (
	_ models.Validator              = (*Addon)(nil)
	_ models.CustomValidator[Addon] = (*Addon)(nil)
	_ models.Equaler[Addon]         = (*Addon)(nil)
)

type Addon struct {
	AddonMeta

	// RateCards
	RateCards RateCards `json:"rateCards"`
}

func (a Addon) ValidateWith(validators ...models.ValidatorFunc[Addon]) error {
	return models.Validate(a, validators...)
}

// ValidationErrors returns a list of possible validation errors for the add-on.
// It returns nil if the add-on has no validation issues.
func (a Addon) ValidationErrors() (models.ValidationIssues, error) {
	return models.AsValidationIssues(a.Validate())
}

func (a Addon) Validate() error {
	return a.ValidateWith(
		ValidateAddonMeta(),
		ValidateAddonRateCards(),
	)
}

// Publishable validates the Addon to ensure that it meets all requirements needed for being published.
// It is a stricter version of Validate. It is the caller's responsibility to handle managed resource-specific parameters
// to ensure the Addon is eligible for publishing. E.g. checking the DeletedAt attribute of the addon.Addon.
func (a Addon) Publishable() error {
	return a.ValidateWith(
		ValidateAddonMeta(),
		ValidateAddonRateCards(),
		ValidateAddonStatusPublishable(),
		ValidateAddonHasSingleBillingCadence(),
		ValidateAddonHasCompatiblePrices(),
	)
}

func (a Addon) Equal(v Addon) bool {
	if !a.AddonMeta.Equal(v.AddonMeta) {
		return false
	}

	return a.RateCards.Equal(v.RateCards)
}

type AddonInstanceType string

const (
	AddonInstanceTypeSingle   AddonInstanceType = "single"
	AddonInstanceTypeMultiple AddonInstanceType = "multiple"
)

func (a AddonInstanceType) Validate() error {
	switch a {
	case AddonInstanceTypeSingle, AddonInstanceTypeMultiple:
		return nil
	default:
		return ErrAddonInvalidInstanceType
	}
}

func (a AddonInstanceType) Values() []string {
	return []string{
		string(AddonInstanceTypeSingle),
		string(AddonInstanceTypeMultiple),
	}
}

// ValidateAddonMeta returns a validation function can be passed to the object
// which implements models.CustomValidator interface. It validates attributes in AddonMeta of Addon.
func ValidateAddonMeta() models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		return a.AddonMeta.Validate()
	}
}

// ValidateAddonRateCards returns a validation function can be passed to the object
// which implements models.CustomValidator interface. It checks for invalid and duplicated ratecards.
func ValidateAddonRateCards() models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		if len(a.RateCards) == 0 {
			return ErrAddonHasNoRateCards
		}

		return ValidateRateCards()(a.RateCards)
	}
}

func ValidateAddonStatusPublishable() models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		if err := ValidateAddonWithStatus(AddonStatusDraft)(a); err != nil {
			return ErrAddonInvalidStatusForPublish
		}

		return nil
	}
}

func ValidateAddonWithStatus(allowed ...AddonStatus) models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		status := a.Status()
		if lo.Contains(allowed, status) {
			return nil
		}

		return ErrAddonInvalidStatus
	}
}

func ValidateAddonHasSingleBillingCadence() models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		if a.RateCards.SingleBillingCadence() {
			return nil
		}

		return models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").WithExpression(models.WildCard)),
			ErrRateCardMultipleBillingCadence,
		)
	}
}

func ValidateAddonHasCompatiblePrices() models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		switch a.InstanceType {
		case AddonInstanceTypeSingle:
			return nil
		case AddonInstanceTypeMultiple:
			for _, rc := range a.RateCards {
				if price := rc.AsMeta().Price; price != nil && price.Type() != FlatPriceType {
					return models.ErrorWithFieldPrefix(
						models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").
							WithExpression(models.NewFieldAttrValue("key", rc.Key()))),
						ErrAddonInvalidPriceForMultiInstance,
					)
				}
			}

			return nil
		default:
			return ErrAddonInvalidInstanceType
		}
	}
}

// Determines if an Addon RateCard will effect a given Plan RateCard
// Right now we only support a single RateCard per addon effecting a single plan RateCard and we match them by key.
// FIXME(galexi): matching like this is unwieldy as sometimes we'd want to match productcatalog.RateCard, sometimes addon.RateCard, or subscriptionaddon.RateCard...
func AddonRateCardMatcherForAGivenPlanRateCard(planRateCard RateCard) func(addonRateCard RateCard) bool {
	return func(addonRateCard RateCard) bool {
		return addonRateCard.Key() == planRateCard.Key()
	}
}

func ValidateAddonWithFeatures(ctx context.Context, resolver NamespacedFeatureResolver) models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		return ValidateRateCardsWithFeatures(ctx, resolver)(a.RateCards)
	}
}
