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
func (a AddonMeta) Status() AddonStatus {
	return a.StatusAt(time.Now())
}

// StatusAt returns the Addon status relative to time t.
func (a AddonMeta) StatusAt(t time.Time) AddonStatus {
	from := lo.FromPtrOr(a.EffectiveFrom, time.Time{})
	to := lo.FromPtrOr(a.EffectiveTo, time.Time{})

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

type Addon struct {
	AddonMeta

	// RateCards
	RateCards RateCards `json:"rateCards"`
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
