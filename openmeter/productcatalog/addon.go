package productcatalog

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

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
		errs = append(errs, fmt.Errorf("invalid EffectivePeriod: %s", err))
	}

	if m.Key == "" {
		errs = append(errs, fmt.Errorf("invalid Key: must not be empty"))
	}

	if m.Name == "" {
		errs = append(errs, fmt.Errorf("invalid Name: must not be empty"))
	}

	if err := m.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid Currency: %s", err))
	}

	if err := m.InstanceType.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid InstanceType: %s", err))
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
	return m.StatusAt(time.Now())
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

func (a Addon) Validate() error {
	var errs []error

	if err := a.AddonMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	// Check for
	// * duplicated rate card keys
	// * namespace mismatch
	// * invalid RateCards
	rateCardKeys := make(map[string]RateCard)
	for _, rateCard := range a.RateCards {
		if _, ok := rateCardKeys[rateCard.Key()]; ok {
			errs = append(errs, fmt.Errorf("duplicated ratecard: %s", rateCard.Key()))
		} else {
			rateCardKeys[rateCard.Key()] = rateCard
		}

		if err := rateCard.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid ratecard: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Publishable validates the Addon to ensure that it meets all requirements needed for being published.
// It is a stricter version of Validate. It is the caller responsibility to handle managed resource specific parameters
// to ensure the Addon is eligible for publishing. E.g. checking the DeletedAt attribute of the addon.Addon.
func (a Addon) Publishable() error {
	return a.ValidateWith(
		AddonWithAllowedStatus(AddonStatusDraft),
		AddonWithBillingCadenceAligned(),
		AddonWithCompatiblePrices(),
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
		return fmt.Errorf("invalid AddonInstanceType: %s", a)
	}
}

func (a AddonInstanceType) Values() []string {
	return []string{
		string(AddonInstanceTypeSingle),
		string(AddonInstanceTypeMultiple),
	}
}

func AddonWithAllowedStatus(allowed ...AddonStatus) models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		status := a.Status()
		if lo.Contains(allowed, status) {
			return nil
		}

		return fmt.Errorf("addon status %s is not valid, must be one of %+v", status, allowed)
	}
}

func AddonWithBillingCadenceAligned() models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		if a.RateCards.BillingCadenceAligned() {
			return nil
		}

		return errors.New("the billing cadence of the ratecards in add-on must be aligned")
	}
}

func AddonWithCompatiblePrices() models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		switch a.InstanceType {
		case AddonInstanceTypeSingle:
			return nil
		case AddonInstanceTypeMultiple:
			for _, rc := range a.RateCards {
				if price := rc.AsMeta().Price; price != nil && price.Type() != FlatPriceType {
					return fmt.Errorf(
						"invalid ratecard for add-on with multiple instance type [ratecard.key=%s]: no price or flat price are allowed, got: %s",
						rc.Key(), price.Type())
				}
			}

			return nil
		default:
			return fmt.Errorf("invalid add-on instance type: %s", a.InstanceType)
		}
	}
}

// Determines if an Addon RateCard will effect a given Plan RateCard
// Right now we only support a single RateCard per addon effecting a single plan RateCard and we match them by key.
func AddonRateCardMatcherForAGivenPlanRateCard(planRateCard RateCard) func(addonRateCard RateCard) bool {
	return func(addonRateCard RateCard) bool {
		return addonRateCard.Key() == planRateCard.Key()
	}
}
