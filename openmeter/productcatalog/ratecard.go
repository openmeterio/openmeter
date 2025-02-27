package productcatalog

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	FlatFeeRateCardType    RateCardType = "flat_fee"
	UsageBasedRateCardType RateCardType = "usage_based"
)

type RateCardType string

func (s RateCardType) Values() []string {
	return []string{
		string(FlatFeeRateCardType),
		string(UsageBasedRateCardType),
	}
}

type RateCard interface {
	models.Validator
	models.Equaler[RateCard]

	Type() RateCardType
	AsMeta() RateCardMeta
	Feature() *feature.Feature
	Key() string
	Merge(RateCard) error
	GetBillingCadence() *isodate.Period
}

type RateCardSerde struct {
	Type RateCardType `json:"type"`
}

var (
	_ models.Validator             = (*RateCardMeta)(nil)
	_ models.Equaler[RateCardMeta] = (*RateCardMeta)(nil)
)

type RateCardMeta struct {
	// Key is the unique key for Plan.
	Key string `json:"key"`

	// Name of the RateCard
	Name string `json:"name"`

	// Description for the RateCard
	Description *string `json:"description,omitempty"`

	// Metadata a set of key/value pairs describing metadata for the RateCard
	Metadata models.Metadata `json:"metadata,omitempty"`

	// Feature defines optional Feature assigned to RateCard
	Feature *feature.Feature `json:"feature,omitempty"`

	// EntitlementTemplate defines the template used for instantiating entitlement.Entitlement.
	// If Feature is set then template must be provided as well.
	EntitlementTemplate *EntitlementTemplate `json:"entitlementTemplate,omitempty"`

	// TaxConfig defines provider specific tax information.
	TaxConfig *TaxConfig `json:"taxConfig,omitempty"`

	// Price defines the price for the RateCard
	Price *Price `json:"price"`
}

func (r RateCardMeta) Equal(v RateCardMeta) bool {
	if r.Key != v.Key {
		return false
	}

	if r.Name != v.Name {
		return false
	}

	if lo.FromPtr(r.Description) != lo.FromPtr(v.Description) {
		return false
	}

	rf := lo.FromPtr(r.Feature)
	vf := lo.FromPtr(v.Feature)

	if rf.ID != vf.ID {
		return false
	}

	if rf.Key != vf.Key {
		return false
	}

	if r.EntitlementTemplate.Equal(v.EntitlementTemplate) {
		return false
	}

	if r.TaxConfig.Equal(v.TaxConfig) {
		return false
	}

	if (r.Price != nil && v.Price == nil) ||
		(r.Price == nil && v.Price != nil) {
		return false
	}

	return r.Price.Equal(v.Price)
}

func (r RateCardMeta) Validate() error {
	var errs []error

	if r.EntitlementTemplate != nil {
		if err := r.EntitlementTemplate.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid EntitlementTemplate: %w", err))
		}
	}

	if r.TaxConfig != nil {
		if err := r.TaxConfig.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid TaxConfig: %w", err))
		}
	}

	if r.Price != nil {
		if err := r.Price.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid Price: %w", err))
		}
	}

	if r.Feature != nil {
		if r.Key != r.Feature.Key {
			errs = append(errs, errors.New("invalid Feature: key mismatch"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ RateCard = (*FlatFeeRateCard)(nil)

type FlatFeeRateCard struct {
	RateCardMeta

	// BillingCadence defines the billing cadence of the RateCard in ISO8601 format.
	// When nil (null) it means it is a one time fee.
	// Example: "P1D12H"
	BillingCadence *isodate.Period `json:"billingCadence"`
}

func (r *FlatFeeRateCard) GetBillingCadence() *isodate.Period {
	return r.BillingCadence
}

func (r *FlatFeeRateCard) Merge(v RateCard) error {
	if r.Type() != v.Type() {
		return errors.New("type mismatch")
	}

	vv, ok := v.(*FlatFeeRateCard)
	if !ok {
		return errors.New("failed to cast to FlatFeeRateCard")
	}

	r.RateCardMeta = vv.RateCardMeta
	r.BillingCadence = vv.BillingCadence

	return nil
}

func (r *FlatFeeRateCard) Feature() *feature.Feature {
	return r.RateCardMeta.Feature
}

func (r *FlatFeeRateCard) Type() RateCardType {
	return FlatFeeRateCardType
}

func (r *FlatFeeRateCard) Key() string {
	return r.RateCardMeta.Key
}

func (r *FlatFeeRateCard) Equal(v RateCard) bool {
	if r.Type() != v.Type() {
		return false
	}

	vv, ok := v.(*FlatFeeRateCard)
	if !ok {
		return false
	}

	if r.RateCardMeta.Equal(vv.RateCardMeta) {
		return false
	}

	if lo.FromPtr(r.BillingCadence).ISOString() != lo.FromPtr(vv.BillingCadence).ISOString() {
		return false
	}

	return true
}

func (r *FlatFeeRateCard) AsMeta() RateCardMeta {
	return r.RateCardMeta
}

func (r *FlatFeeRateCard) Validate() error {
	var errs []error

	if err := r.RateCardMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.BillingCadence != nil {
		if r.BillingCadence.IsNegative() || r.BillingCadence.IsZero() {
			errs = append(errs, errors.New("invalid BillingCadence: must not be negative or zero"))
		}

		// Billing Cadence has to be at least 1 ohur
		if per, err := r.BillingCadence.Subtract(isodate.NewPeriod(0, 0, 0, 0, 1, 0, 0)); err == nil && per.Sign() == -1 {
			errs = append(errs, errors.New("invalid BillingCadence: must be at least 1 hour"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ RateCard = (*UsageBasedRateCard)(nil)

type UsageBasedRateCard struct {
	RateCardMeta

	// BillingCadence defines the billing cadence of the RateCard in ISO8601 format.
	// Example: "P1D12H"
	BillingCadence isodate.Period `json:"billingCadence"`
}

func (r *UsageBasedRateCard) GetBillingCadence() *isodate.Period {
	return &r.BillingCadence
}

func (r *UsageBasedRateCard) Merge(v RateCard) error {
	if r.Type() != v.Type() {
		return errors.New("type mismatch")
	}

	vv, ok := v.(*UsageBasedRateCard)
	if !ok {
		return errors.New("failed to cast to UsageBasedRateCard")
	}

	r.RateCardMeta = vv.RateCardMeta
	r.BillingCadence = vv.BillingCadence

	return nil
}

func (r *UsageBasedRateCard) Feature() *feature.Feature {
	return r.RateCardMeta.Feature
}

func (r *UsageBasedRateCard) Type() RateCardType {
	return UsageBasedRateCardType
}

func (r *UsageBasedRateCard) Key() string {
	return r.RateCardMeta.Key
}

func (r *UsageBasedRateCard) Equal(v RateCard) bool {
	if r.Type() != v.Type() {
		return false
	}

	vv, ok := v.(*UsageBasedRateCard)
	if !ok {
		return false
	}

	if r.RateCardMeta.Equal(vv.RateCardMeta) {
		return false
	}

	if r.BillingCadence.ISOString() != vv.BillingCadence.ISOString() {
		return false
	}

	return true
}

func (r *UsageBasedRateCard) AsMeta() RateCardMeta {
	return r.RateCardMeta
}

func (r *UsageBasedRateCard) Validate() error {
	var errs []error

	if err := r.RateCardMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.BillingCadence.IsNegative() || r.BillingCadence.IsZero() {
		errs = append(errs, errors.New("invalid BillingCadence: must not be negative or zero"))
	}

	// Billing Cadence has to be at least 1 ohur
	if per, err := r.BillingCadence.Subtract(isodate.NewPeriod(0, 0, 0, 0, 1, 0, 0)); err == nil && per.Sign() == -1 {
		errs = append(errs, errors.New("invalid BillingCadence: must be at least 1 hour"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ models.Equaler[RateCards] = (*RateCards)(nil)

type RateCards []RateCard

func (c RateCards) At(idx int) RateCard {
	return c[idx]
}

func (c RateCards) Billables() RateCards {
	var billables RateCards
	for _, rc := range c {
		// An effective price of 0 is still counted as a billable item
		if rc.AsMeta().Price != nil {
			billables = append(billables, rc)
		}
	}

	return billables
}

func (c RateCards) Equal(v RateCards) bool {
	if len(c) != len(v) {
		return false
	}

	leftSet := make(map[string]RateCard)
	for _, rc := range c {
		leftSet[rc.Key()] = rc
	}

	rightSet := make(map[string]RateCard)
	for _, rc := range v {
		rightSet[rc.Key()] = rc
	}

	if len(leftSet) != len(rightSet) {
		return false
	}

	var visited int
	for key, left := range leftSet {
		right, ok := rightSet[key]
		if !ok {
			return false
		}

		if !left.Equal(right) {
			return false
		}

		visited++
	}

	return visited == len(rightSet)
}
