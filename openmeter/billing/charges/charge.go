package charges

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

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
	switch c.t {
	case ChargeTypeFlatFee:
		if c.flatFee == nil {
			return fmt.Errorf("flat fee charge is nil")
		}

		return c.flatFee.Validate()
	case ChargeTypeUsageBased:
		if c.usageBased == nil {
			return fmt.Errorf("usage based charge is nil")
		}

		return c.usageBased.Validate()
	case ChargeTypeCreditPurchase:
		if c.creditPurchase == nil {
			return fmt.Errorf("credit purchase charge is nil")
		}

		return c.creditPurchase.Validate()
	}

	return fmt.Errorf("invalid charge type: %s", c.t)
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

func (c Charge) AsCreditPurchaseCharge() (CreditPurchaseCharge, error) {
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
	// ChargeStatusCreated is the status of a charge that is created and is not yet active.
	ChargeStatusCreated ChargeStatus = "created"
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

type ChargeIntent struct {
	t ChargeType

	flatFee        *FlatFeeIntent
	usageBased     *UsageBasedIntent
	creditPurchase *CreditPurchaseIntent
}

func NewChargeIntent[T FlatFeeIntent | UsageBasedIntent | CreditPurchaseIntent](ch T) ChargeIntent {
	switch v := any(ch).(type) {
	case FlatFeeCharge:
		return ChargeIntent{
			t:       ChargeTypeFlatFee,
			flatFee: &v.Intent,
		}
	case UsageBasedIntent:
		return ChargeIntent{
			t:          ChargeTypeUsageBased,
			usageBased: &v,
		}
	case CreditPurchaseIntent:
		return ChargeIntent{
			t:              ChargeTypeCreditPurchase,
			creditPurchase: &v,
		}
	}

	return ChargeIntent{}
}

func (i ChargeIntent) Type() ChargeType {
	return i.t
}

func (i ChargeIntent) Validate() error {
	switch i.t {
	case ChargeTypeFlatFee:
		if i.flatFee == nil {
			return fmt.Errorf("flat fee is nil")
		}

		return i.flatFee.Validate()
	case ChargeTypeUsageBased:
		if i.usageBased == nil {
			return fmt.Errorf("usage based is nil")
		}

		return i.usageBased.Validate()
	case ChargeTypeCreditPurchase:
		if i.creditPurchase == nil {
			return fmt.Errorf("credit purchase is nil")
		}

		return i.creditPurchase.Validate()
	}

	return fmt.Errorf("invalid charge type: %s", i.t)
}

func (i ChargeIntent) AsFlatFeeIntent() (FlatFeeIntent, error) {
	if i.t != ChargeTypeFlatFee {
		return FlatFeeIntent{}, fmt.Errorf("charge is not a flat fee charge")
	}

	if i.flatFee == nil {
		return FlatFeeIntent{}, fmt.Errorf("flat fee is nil")
	}

	return *i.flatFee, nil
}

func (i ChargeIntent) AsUsageBasedIntent() (UsageBasedIntent, error) {
	if i.t != ChargeTypeUsageBased {
		return UsageBasedIntent{}, fmt.Errorf("charge is not a usage based charge")
	}

	if i.usageBased == nil {
		return UsageBasedIntent{}, fmt.Errorf("usage based is nil")
	}

	return *i.usageBased, nil
}

func (i ChargeIntent) AsCreditPurchaseIntent() (CreditPurchaseIntent, error) {
	if i.t != ChargeTypeCreditPurchase {
		return CreditPurchaseIntent{}, fmt.Errorf("charge is not a credit purchase charge")
	}

	if i.creditPurchase == nil {
		return CreditPurchaseIntent{}, fmt.Errorf("credit purchase is nil")
	}

	return *i.creditPurchase, nil
}

type ChargeIntents []ChargeIntent

func (i ChargeIntents) Validate() error {
	var errs []error

	for idx, ch := range i {
		if err := ch.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

type CreateChargeInputs struct {
	Namespace string
	Intents   ChargeIntents
}

func (i CreateChargeInputs) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := i.Intents.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intents: %w", err))
	}

	return errors.Join(errs...)
}

type ManagedResource struct {
	models.NamespacedModel
	models.ManagedModel
	ID string `json:"id"`
}

type NewManagedResourceInput struct {
	Namespace string
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func NewManagedResource(input NewManagedResourceInput) ManagedResource {
	return ManagedResource{
		NamespacedModel: models.NamespacedModel{
			Namespace: input.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: input.CreatedAt,
			UpdatedAt: input.UpdatedAt,
			DeletedAt: input.DeletedAt,
		},
		ID: input.ID,
	}
}
