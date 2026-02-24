package charges

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type GenericCharge interface {
	Validate() error
	AsCharge() Charge
	Type() ChargeType

	GetIntentMeta() IntentMeta
	GetManagedResource() models.ManagedResource
	GetStatus() ChargeStatus
}

type ChargeType string

const (
	ChargeTypeFlatFee        ChargeType = "flat_fee"
	ChargeTypeUsageBased     ChargeType = "usage_based"
	ChargeTypeCreditPurchase ChargeType = "credit_purchase"
)

func (t ChargeType) Values() []string {
	return []string{
		string(ChargeTypeFlatFee),
		string(ChargeTypeUsageBased),
		string(ChargeTypeCreditPurchase),
	}
}

func (t ChargeType) Validate() error {
	if !slices.Contains(t.Values(), string(t)) {
		return fmt.Errorf("invalid charge type: %s", t)
	}

	return nil
}

type Charge struct {
	t ChargeType

	flatFee        *FlatFeeCharge
	usageBased     *UsageBasedCharge
	creditPurchase *CreditPurchaseCharge
}

func (c Charge) AsGenericCharge() (GenericCharge, error) {
	switch c.t {
	case ChargeTypeFlatFee:
		if c.flatFee == nil {
			return nil, fmt.Errorf("flat fee charge is nil")
		}

		return c.flatFee, nil
	case ChargeTypeUsageBased:
		if c.usageBased == nil {
			return nil, fmt.Errorf("usage based charge is nil")
		}

		return c.usageBased, nil
	case ChargeTypeCreditPurchase:
		if c.creditPurchase == nil {
			return nil, fmt.Errorf("credit purchase charge is nil")
		}

		return c.creditPurchase, nil
	default:
		return nil, fmt.Errorf("invalid charge type: %s", c.t)
	}
}

func (c Charge) Type() ChargeType {
	return c.t
}

func NewCharge[T FlatFeeCharge | UsageBasedCharge | CreditPurchaseCharge](ch T) Charge {
	switch v := any(ch).(type) {
	case FlatFeeCharge:
		return Charge{
			t:       ChargeTypeFlatFee,
			flatFee: &v,
		}
	case UsageBasedCharge:
		return Charge{
			t:          ChargeTypeUsageBased,
			usageBased: &v,
		}
	case CreditPurchaseCharge:
		return Charge{
			t:              ChargeTypeCreditPurchase,
			creditPurchase: &v,
		}
	}

	return Charge{}
}

func (c Charge) Validate() error {
	genericCharge, err := c.AsGenericCharge()
	if err != nil {
		return fmt.Errorf("failed to convert charge to generic charge: %w", err)
	}

	return genericCharge.Validate()
}

func (c Charge) AsFlatFeeCharge() (FlatFeeCharge, error) {
	if c.t != ChargeTypeFlatFee {
		return FlatFeeCharge{}, fmt.Errorf("charge is not a flat fee charge")
	}

	if c.flatFee == nil {
		return FlatFeeCharge{}, fmt.Errorf("flat fee charge is nil")
	}

	return *c.flatFee, nil
}

func (c Charge) AsUsageBasedCharge() (UsageBasedCharge, error) {
	if c.t != ChargeTypeUsageBased {
		return UsageBasedCharge{}, fmt.Errorf("charge is not a usage based charge")
	}

	if c.usageBased == nil {
		return UsageBasedCharge{}, fmt.Errorf("usage based charge is nil")
	}

	return *c.usageBased, nil
}

func (c Charge) AsCreditPurchase() (CreditPurchaseCharge, error) {
	if c.t != ChargeTypeCreditPurchase {
		return CreditPurchaseCharge{}, fmt.Errorf("charge is not a credit purchase charge")
	}

	if c.creditPurchase == nil {
		return CreditPurchaseCharge{}, fmt.Errorf("credit purchase charge is nil")
	}

	return *c.creditPurchase, nil
}

type ChargeStatus string

const (
	// ChargeStatusActive is the status of a charge that is active and is not yet fully settled for the service period.
	ChargeStatusActive ChargeStatus = "active"
	// ChargeStatusSettled is the status of a charge that is settled and is fully settled for the service period. The charge might receive additional
	// late events in the future.
	ChargeStatusSettled ChargeStatus = "settled"
	// ChargeStatusFinal is the status of a charge that is final and is fully settled for the service period. The charge will not receive any additional
	// late events in the future.
	ChargeStatusFinal ChargeStatus = "final"
)

func (s ChargeStatus) Values() []string {
	return []string{
		string(ChargeStatusActive),
		string(ChargeStatusSettled),
		string(ChargeStatusFinal),
	}
}

func (s ChargeStatus) Validate() error {
	if !slices.Contains(s.Values(), string(s)) {
		return fmt.Errorf("invalid charge status: %s", s)
	}

	return nil
}

type Charges []Charge

func (c Charges) Validate() error {
	var errs []error

	for i, ch := range c {
		if err := ch.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("charge [%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

type ProRatingModeAdapterEnum string

const (
	ProratePricesProratingAdapterMode ProRatingModeAdapterEnum = ProRatingModeAdapterEnum(productcatalog.ProRatingModeProratePrices)
	NoProratingAdapterMode            ProRatingModeAdapterEnum = "no_prorate"
)

func (e ProRatingModeAdapterEnum) Values() []string {
	return []string{
		string(ProratePricesProratingAdapterMode),
		string(NoProratingAdapterMode),
	}
}
