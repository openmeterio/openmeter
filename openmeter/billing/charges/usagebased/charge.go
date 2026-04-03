package usagebased

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
)

var _ meta.ChargeAccessor = (*ChargeBase)(nil)

type ChargeBase struct {
	meta.ManagedResource

	Intent Intent `json:"intent"`
	Status Status `json:"status"`

	State State `json:"state"`
}

func (c ChargeBase) Validate() error {
	var errs []error

	if err := c.ManagedResource.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed resource: %w", err))
	}

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := c.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	if err := c.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c ChargeBase) GetChargeID() meta.ChargeID {
	return meta.ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

func (c ChargeBase) ErrorAttributes() models.Attributes {
	return models.Attributes{
		"charge_id":   c.ID,
		"namespace":   c.Namespace,
		"charge_type": string(meta.ChargeTypeUsageBased),
	}
}

var _ meta.ChargeAccessor = (*Charge)(nil)

type Charge struct {
	ChargeBase

	Realizations RealizationRuns `json:"realizations"`
}

type Charges []Charge

func (c Charge) Validate() error {
	var errs []error

	if err := c.ChargeBase.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge base: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c Charge) GetCurrentRealizationRun() (RealizationRun, error) {
	if c.State.CurrentRealizationRunID == nil {
		return RealizationRun{}, fmt.Errorf("no current realization run")
	}

	return c.Realizations.GetByID(*c.State.CurrentRealizationRunID)
}

func (c Charge) GetFeatureKeyOrID() ref.IDOrKey {
	switch c.Status {
	case StatusCreated:
		return ref.IDOrKey{
			Key: c.Intent.FeatureKey,
		}
	case StatusDeleted:
		if c.State.FeatureID != "" {
			return ref.IDOrKey{
				ID: c.State.FeatureID,
			}
		}

		return ref.IDOrKey{
			Key: c.Intent.FeatureKey,
		}
	default:
		return ref.IDOrKey{
			ID: c.State.FeatureID,
		}
	}
}

func (c Charge) ResolveFeatureMeter(featureMeters feature.FeatureMeters) (feature.FeatureMeter, error) {
	const requireMeter = true

	featureRef := c.GetFeatureKeyOrID()
	if featureRef.ID != "" {
		return featureMeters.GetByID(featureRef.ID, requireMeter)
	}

	featureMeter, err := featureMeters.Get(featureRef.Key, requireMeter)
	if err != nil {
		return feature.FeatureMeter{}, fmt.Errorf("get feature meter: %w", err)
	}

	return featureMeter, nil
}

// GetFeatureKeysOrIDs returns the unique state-aware feature references for the charges.
// Each charge contributes the ref returned by GetFeatureKeyOrID, so created charges use keys,
// deleted charges prefer IDs and fall back to keys, and all other states use IDs.
func (c Charges) GetFeatureKeysOrIDs() []ref.IDOrKey {
	return lo.Uniq(lo.Map(c, func(charge Charge, _ int) ref.IDOrKey {
		return charge.GetFeatureKeyOrID()
	}))
}

type Intent struct {
	meta.Intent

	InvoiceAt      time.Time                     `json:"invoiceAt"`
	SettlementMode productcatalog.SettlementMode `json:"settlementMode"`

	FeatureKey string `json:"featureKey"`

	Price productcatalog.Price `json:"price"`

	Discounts productcatalog.Discounts `json:"discounts"`
}

func (i Intent) Normalized() Intent {
	i.Intent = i.Intent.Normalized()
	i.InvoiceAt = meta.NormalizeTimestamp(i.InvoiceAt)

	return i
}

func (i Intent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := i.SettlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
	}

	if err := i.Discounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discounts: %w", err))
	}

	if i.InvoiceAt.IsZero() {
		errs = append(errs, fmt.Errorf("invoice at is required"))
	}

	if err := i.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("price: %w", err))
	}

	if i.FeatureKey == "" {
		errs = append(errs, fmt.Errorf("feature key is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type State struct {
	CurrentRealizationRunID *string    `json:"currentRealizationRunId"`
	AdvanceAfter            *time.Time `json:"advanceAfter"`
	FeatureID               string     `json:"featureId"`
}

func (s State) Normalized() State {
	s.AdvanceAfter = meta.NormalizeOptionalTimestamp(s.AdvanceAfter)

	return s
}

func (s State) Validate() error {
	var errs []error

	if s.CurrentRealizationRunID != nil && *s.CurrentRealizationRunID == "" {
		errs = append(errs, fmt.Errorf("current realization run ID must be non-empty"))
	}

	if s.FeatureID == "" {
		errs = append(errs, fmt.Errorf("feature id must be set"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
