package model

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/datex"
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

type rateCarder interface {
	json.Marshaler
	json.Unmarshaler
	Validator

	Type() RateCardType
	AsFlatFee() (FlatFeeRateCard, error)
	AsUsageBased() (UsageBasedRateCard, error)
	AsMeta() (RateCardMeta, error)
	FromFlatFee(FlatFeeRateCard)
	FromUsageBased(UsageBasedRateCard)
}

var _ rateCarder = (*RateCard)(nil)

type RateCard struct {
	t          RateCardType
	flatFee    *FlatFeeRateCard
	usageBased *UsageBasedRateCard
}

func (r *RateCard) MarshalJSON() ([]byte, error) {
	var b []byte
	var err error

	switch r.t {
	case FlatFeeRateCardType:
		b, err = json.Marshal(r.flatFee)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal FlatFeeRateCard: %w", err)
		}
	case UsageBasedRateCardType:
		b, err = json.Marshal(r.usageBased)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal UsageBasedRateCard: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid entitlement type: %s", r.t)
	}

	return b, nil
}

func (r *RateCard) UnmarshalJSON(bytes []byte) error {
	meta := &RateCardMeta{}

	if err := json.Unmarshal(bytes, meta); err != nil {
		return fmt.Errorf("failed to json unmarshal type: %w", err)
	}

	switch meta.Type {
	case FlatFeeRateCardType:
		v := &FlatFeeRateCard{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal FlatFeeRateCard: %w", err)
		}

		r.flatFee = v
		r.t = FlatFeeRateCardType
	case UsageBasedRateCardType:
		v := &UsageBasedRateCard{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal UsageBasedRateCard: %w", err)
		}

		r.usageBased = v
		r.t = UsageBasedRateCardType
	default:
		return fmt.Errorf("invalid type: %s", meta.Type)
	}

	return nil
}

func (r *RateCard) Validate() error {
	switch r.t {
	case FlatFeeRateCardType:
		if r.flatFee != nil {
			return r.flatFee.Validate()
		}

		return errors.New("invalid RateCard: not initialized")
	case UsageBasedRateCardType:
		if r.usageBased != nil {
			return r.usageBased.Validate()
		}

		return errors.New("invalid RateCard: not initialized")
	default:
		return fmt.Errorf("invalid type: %s", r.t)
	}
}

func (r *RateCard) Type() RateCardType {
	return r.t
}

func (r *RateCard) Key() string {
	var key string

	switch r.t {
	case FlatFeeRateCardType:
		if r.flatFee != nil {
			key = r.flatFee.Key
		}
	case UsageBasedRateCardType:
		if r.usageBased != nil {
			key = r.usageBased.Key
		}
	}

	return key
}

func (r *RateCard) AsFlatFee() (FlatFeeRateCard, error) {
	if r.t == "" || r.flatFee == nil {
		return FlatFeeRateCard{}, errors.New("invalid RateCard: not initialized")
	}

	return *r.flatFee, nil
}

func (r *RateCard) AsUsageBased() (UsageBasedRateCard, error) {
	if r.t == "" || r.usageBased == nil {
		return UsageBasedRateCard{}, errors.New("invalid RateCard: not initialized")
	}

	if r.t != UsageBasedRateCardType {
		return UsageBasedRateCard{}, fmt.Errorf("type mismatch: %s", r.t)
	}

	return *r.usageBased, nil
}

func (r *RateCard) AsMeta() (RateCardMeta, error) {
	if r.t == "" {
		return RateCardMeta{}, errors.New("invalid RateCard: not initialized")
	}

	switch r.t {
	case FlatFeeRateCardType:
		return r.flatFee.RateCardMeta, nil
	case UsageBasedRateCardType:
		return r.usageBased.RateCardMeta, nil
	default:
		return RateCardMeta{}, fmt.Errorf("type mismatch: %s", r.t)
	}
}

func (r *RateCard) FromFlatFee(c FlatFeeRateCard) {
	r.flatFee = &c
	r.t = FlatFeeRateCardType
}

func (r *RateCard) FromUsageBased(c UsageBasedRateCard) {
	r.usageBased = &c
	r.t = UsageBasedRateCardType
}

func NewRateCardFrom[T FlatFeeRateCard | UsageBasedRateCard](c T) RateCard {
	r := &RateCard{}

	switch any(c).(type) {
	case FlatFeeRateCard:
		flatFee := any(c).(FlatFeeRateCard)
		r.FromFlatFee(flatFee)
	case UsageBasedRateCard:
		usageBased := any(c).(UsageBasedRateCard)
		r.FromUsageBased(usageBased)
	}

	return *r
}

var _ Validator = (*RateCardMeta)(nil)

type RateCardMeta struct {
	// Key is the unique key for Plan.
	Key string `json:"key"`

	// Type defines the type of the RateCard
	Type RateCardType `json:"type"`

	// Name of the RateCard
	Name string `json:"name"`

	// Description for the RateCard
	Description *string `json:"description,omitempty"`

	// Metadata a set of key/value pairs describing metadata for the RateCard
	Metadata map[string]string `json:"metadata,omitempty"`

	// Feature defines optional Feature assigned to RateCard
	Feature *feature.Feature `json:"feature,omitempty"`

	// EntitlementTemplate defines the template used for instantiating entitlement.Entitlement.
	// If Feature is set then template must be provided as well.
	EntitlementTemplate *EntitlementTemplate `json:"entitlementTemplate,omitempty"`

	// TaxConfig defines provider specific tax information.
	TaxConfig *TaxConfig `json:"taxConfig,omitempty"`

	// PhaseID is the ULID identifier of the Phase the RateCard belongs to.
	PhaseID string `json:"-"`
}

func (r *RateCardMeta) Validate() error {
	var errs []error

	if r.Feature != nil && r.EntitlementTemplate == nil {
		errs = append(errs, errors.New("invalid EntitlementTemplate: must be provided if Feature is set"))
	}

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

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

var _ Validator = (*FlatFeeRateCard)(nil)

type FlatFeeRateCard struct {
	RateCardMeta

	// BillingCadence defines the billing cadence of the RateCard in ISO8601 format.
	// When nil (null) it means it is a one time fee.
	// Example: "P1D12H"
	BillingCadence *datex.Period `json:"billingCadence"`

	// Price defines the price for the RateCard
	Price Price `json:"price"`
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
	}

	if err := r.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid Price: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

var _ Validator = (*UsageBasedRateCard)(nil)

type UsageBasedRateCard struct {
	RateCardMeta

	// BillingCadence defines the billing cadence of the RateCard in ISO8601 format.
	// Example: "P1D12H"
	BillingCadence datex.Period `json:"billingCadence"`

	// Price defines the price for the RateCard
	Price *Price `json:"price"`
}

func (r *UsageBasedRateCard) Validate() error {
	var errs []error

	if err := r.RateCardMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.BillingCadence.IsNegative() || r.BillingCadence.IsZero() {
		errs = append(errs, errors.New("invalid BillingCadence: must not be negative or zero"))
	}

	if err := r.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid Price: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
