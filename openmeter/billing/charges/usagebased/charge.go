package usagebased

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ meta.ChargeAccessor = (*ChargeBase)(nil)

type ChargeBase struct {
	meta.ManagedResource

	Intent         Intent          `json:"intent"`
	IntentOverride *IntentOverride `json:"intentOverride,omitempty"`
	Status         Status          `json:"status"`

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

	if c.IntentOverride != nil {
		if err := c.IntentOverride.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("intent override: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c ChargeBase) GetChargeID() meta.ChargeID {
	return meta.ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

func (c ChargeBase) GetCustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: c.Namespace,
		ID:        c.Intent.CustomerID,
	}
}

func (c ChargeBase) GetCurrency() currencyx.Code {
	return c.Intent.Currency
}

func (c ChargeBase) GetIntentDeletedAt() *time.Time {
	if c.IntentOverride != nil {
		return c.IntentOverride.IntentDeletedAt
	}

	return c.Intent.IntentDeletedAt
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
	Expands      Expands         `json:"expands"`
}

type Charges []Charge

func (c Charge) GetStatus() Status {
	return c.Status
}

func (c Charge) WithStatus(status Status) Charge {
	c.Status = status
	return c
}

func (c Charge) GetBase() ChargeBase {
	return c.ChargeBase
}

func (c Charge) WithBase(base ChargeBase) Charge {
	c.ChargeBase = base
	return c
}

func (c Charge) Validate() error {
	var errs []error

	if err := c.ChargeBase.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge base: %w", err))
	}

	if err := c.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c Charge) GetCurrentRealizationRun() (RealizationRun, error) {
	if c.State.CurrentRealizationRunID == nil {
		return RealizationRun{}, fmt.Errorf("no current realization run")
	}

	return c.Realizations.GetByID(*c.State.CurrentRealizationRunID)
}

func (c Charge) GetCustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: c.Namespace,
		ID:        c.Intent.CustomerID,
	}
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

	// IntentDeletedAt marks the usage-based base/original intent as deleted.
	// Adapters derive the effective charge DeletedAt from this value when no intent override is present.
	IntentDeletedAt *time.Time `json:"intentDeletedAt,omitempty"`

	FeatureKey string `json:"featureKey"`

	Price productcatalog.Price `json:"price"`

	Discounts productcatalog.Discounts `json:"discounts"`
}

type IntentOverride struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Metadata    models.Metadata `json:"metadata,omitempty"`

	TaxBehavior *productcatalog.TaxBehavior `json:"taxBehavior,omitempty"`
	TaxCodeID   *string                     `json:"taxCodeID,omitempty"`

	// IntentDeletedAt marks the usage-based override intent as deleted.
	// When an override is present, adapters derive the effective charge DeletedAt from this value instead of the base intent.
	IntentDeletedAt *time.Time `json:"intentDeletedAt,omitempty"`

	ServicePeriod     timeutil.ClosedPeriod `json:"servicePeriod"`
	FullServicePeriod timeutil.ClosedPeriod `json:"fullServicePeriod"`
	BillingPeriod     timeutil.ClosedPeriod `json:"billingPeriod"`

	FeatureKey string                   `json:"featureKey"`
	Price      productcatalog.Price     `json:"price"`
	Discounts  productcatalog.Discounts `json:"discounts"`
}

func (o IntentOverride) Normalized() IntentOverride {
	o.ServicePeriod = meta.NormalizeClosedPeriod(o.ServicePeriod)
	o.FullServicePeriod = meta.NormalizeClosedPeriod(o.FullServicePeriod)
	o.BillingPeriod = meta.NormalizeClosedPeriod(o.BillingPeriod)

	return o
}

func (o IntentOverride) Validate() error {
	var errs []error

	if o.Name == "" {
		errs = append(errs, errors.New("name cannot be empty"))
	}

	if o.TaxBehavior != nil {
		if err := o.TaxBehavior.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax behavior: %w", err))
		}
	}

	if err := o.ServicePeriod.ValidateAsRequired(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := o.FullServicePeriod.ValidateAsRequired(); err != nil {
		errs = append(errs, fmt.Errorf("full service period: %w", err))
	}

	if err := o.BillingPeriod.ValidateAsRequired(); err != nil {
		errs = append(errs, fmt.Errorf("billing period: %w", err))
	}

	if o.FeatureKey == "" {
		errs = append(errs, errors.New("feature key is required"))
	}

	if err := o.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("price: %w", err))
	}

	if err := o.Discounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discounts: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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
	CurrentRealizationRunID *string      `json:"currentRealizationRunId"`
	AdvanceAfter            *time.Time   `json:"advanceAfter"`
	FeatureID               string       `json:"featureId"`
	RatingEngine            RatingEngine `json:"ratingEngine"`
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

	if err := s.RatingEngine.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("rating engine: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Expands struct {
	RealtimeUsage *totals.Totals `json:"realtimeUsage,omitempty"`
}

func (e Expands) Validate() error {
	var errs []error

	if e.RealtimeUsage != nil {
		if err := e.RealtimeUsage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("realtime usage: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
