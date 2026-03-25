package charges

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Charge struct {
	t meta.ChargeType

	flatFee        *flatfee.Charge
	usageBased     *usagebased.Charge
	creditPurchase *creditpurchase.Charge
}

func (c Charge) Type() meta.ChargeType {
	return c.t
}

func NewCharge[T flatfee.Charge | usagebased.Charge | creditpurchase.Charge](ch T) Charge {
	switch v := any(ch).(type) {
	case flatfee.Charge:
		return Charge{
			t:       meta.ChargeTypeFlatFee,
			flatFee: &v,
		}
	case creditpurchase.Charge:
		return Charge{
			t:              meta.ChargeTypeCreditPurchase,
			creditPurchase: &v,
		}
	case usagebased.Charge:
		return Charge{
			t:          meta.ChargeTypeUsageBased,
			usageBased: &v,
		}
	}

	return Charge{}
}

func (c Charge) Validate() error {
	switch c.t {
	case meta.ChargeTypeFlatFee:
		if c.flatFee == nil {
			return fmt.Errorf("flat fee charge is nil")
		}

		return c.flatFee.Validate()
	case meta.ChargeTypeCreditPurchase:
		if c.creditPurchase == nil {
			return fmt.Errorf("credit purchase charge is nil")
		}

		return c.creditPurchase.Validate()
	case meta.ChargeTypeUsageBased:
		if c.usageBased == nil {
			return fmt.Errorf("usage based charge is nil")
		}

		return c.usageBased.Validate()
	}

	return fmt.Errorf("invalid charge type: %s", c.t)
}

func (c Charge) AsFlatFeeCharge() (flatfee.Charge, error) {
	if c.t != meta.ChargeTypeFlatFee {
		return flatfee.Charge{}, fmt.Errorf("charge is not a flat fee charge")
	}

	if c.flatFee == nil {
		return flatfee.Charge{}, fmt.Errorf("flat fee charge is nil")
	}

	return *c.flatFee, nil
}

func (c Charge) AsCreditPurchaseCharge() (creditpurchase.Charge, error) {
	if c.t != meta.ChargeTypeCreditPurchase {
		return creditpurchase.Charge{}, fmt.Errorf("charge is not a credit purchase charge")
	}

	if c.creditPurchase == nil {
		return creditpurchase.Charge{}, fmt.Errorf("credit purchase charge is nil")
	}

	return *c.creditPurchase, nil
}

func (c Charge) AsUsageBasedCharge() (usagebased.Charge, error) {
	if c.t != meta.ChargeTypeUsageBased {
		return usagebased.Charge{}, fmt.Errorf("charge is not a usage based charge")
	}

	if c.usageBased == nil {
		return usagebased.Charge{}, fmt.Errorf("usage based charge is nil")
	}

	return *c.usageBased, nil
}

func (c Charge) GetChargeID() (meta.ChargeID, error) {
	switch c.t {
	case meta.ChargeTypeFlatFee:
		if c.flatFee == nil {
			return meta.ChargeID{}, fmt.Errorf("flat fee charge is nil")
		}

		return c.flatFee.GetChargeID(), nil
	case meta.ChargeTypeCreditPurchase:
		if c.creditPurchase == nil {
			return meta.ChargeID{}, fmt.Errorf("credit purchase charge is nil")
		}

		return c.creditPurchase.GetChargeID(), nil
	case meta.ChargeTypeUsageBased:
		if c.usageBased == nil {
			return meta.ChargeID{}, fmt.Errorf("usage based charge is nil")
		}

		return c.usageBased.GetChargeID(), nil
	}

	return meta.ChargeID{}, fmt.Errorf("invalid charge type: %s", c.t)
}

var _ entutils.InIDOrderAccessor = (*Charge)(nil)

func (c Charge) GetID() string {
	id, err := c.GetChargeID()
	if err != nil {
		return ""
	}

	return id.ID
}

func (c Charge) GetNamespace() string {
	id, err := c.GetChargeID()
	if err != nil {
		return ""
	}

	return id.Namespace
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

type ChargeIntent struct {
	t meta.ChargeType

	flatFee        *flatfee.Intent
	creditPurchase *creditpurchase.Intent
	usageBased     *usagebased.Intent
}

func NewChargeIntent[T flatfee.Intent | usagebased.Intent | creditpurchase.Intent](ch T) ChargeIntent {
	switch v := any(ch).(type) {
	case flatfee.Intent:
		return ChargeIntent{
			t:       meta.ChargeTypeFlatFee,
			flatFee: &v,
		}
	case creditpurchase.Intent:
		return ChargeIntent{
			t:              meta.ChargeTypeCreditPurchase,
			creditPurchase: &v,
		}
	case usagebased.Intent:
		return ChargeIntent{
			t:          meta.ChargeTypeUsageBased,
			usageBased: &v,
		}
	}

	return ChargeIntent{}
}

func (i ChargeIntent) Type() meta.ChargeType {
	return i.t
}

func (i ChargeIntent) Validate() error {
	switch i.t {
	case meta.ChargeTypeFlatFee:
		if i.flatFee == nil {
			return fmt.Errorf("flat fee is nil")
		}

		return i.flatFee.Validate()
	case meta.ChargeTypeCreditPurchase:
		if i.creditPurchase == nil {
			return fmt.Errorf("credit purchase is nil")
		}

		return i.creditPurchase.Validate()
	case meta.ChargeTypeUsageBased:
		if i.usageBased == nil {
			return fmt.Errorf("usage based is nil")
		}

		return i.usageBased.Validate()
	}

	return fmt.Errorf("invalid charge type: %s", i.t)
}

func (i ChargeIntent) AsFlatFeeIntent() (flatfee.Intent, error) {
	if i.t != meta.ChargeTypeFlatFee {
		return flatfee.Intent{}, fmt.Errorf("charge is not a flat fee charge")
	}

	if i.flatFee == nil {
		return flatfee.Intent{}, fmt.Errorf("flat fee is nil")
	}

	return *i.flatFee, nil
}

func (i ChargeIntent) AsCreditPurchaseIntent() (creditpurchase.Intent, error) {
	if i.t != meta.ChargeTypeCreditPurchase {
		return creditpurchase.Intent{}, fmt.Errorf("charge is not a credit purchase charge")
	}

	if i.creditPurchase == nil {
		return creditpurchase.Intent{}, fmt.Errorf("credit purchase is nil")
	}

	return *i.creditPurchase, nil
}

func (i ChargeIntent) AsUsageBasedIntent() (usagebased.Intent, error) {
	if i.t != meta.ChargeTypeUsageBased {
		return usagebased.Intent{}, fmt.Errorf("charge is not a usage based charge")
	}

	if i.usageBased == nil {
		return usagebased.Intent{}, fmt.Errorf("usage based is nil")
	}

	return *i.usageBased, nil
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

type ChargeIntentsByType struct {
	FlatFee        []WithIndex[flatfee.Intent]
	CreditPurchase []WithIndex[creditpurchase.Intent]
	UsageBased     []WithIndex[usagebased.Intent]
}

func (i ChargeIntents) ByType() (ChargeIntentsByType, error) {
	out := ChargeIntentsByType{
		FlatFee:        make([]WithIndex[flatfee.Intent], 0, len(i)),
		CreditPurchase: make([]WithIndex[creditpurchase.Intent], 0, len(i)),
		UsageBased:     make([]WithIndex[usagebased.Intent], 0, len(i)),
	}

	for idx, ch := range i {
		switch ch.Type() {
		case meta.ChargeTypeFlatFee:
			flatFee, err := ch.AsFlatFeeIntent()
			if err != nil {
				return ChargeIntentsByType{}, fmt.Errorf("converting flat fee intent[%d]: %w", idx, err)
			}

			out.FlatFee = append(out.FlatFee, WithIndex[flatfee.Intent]{Index: idx, Value: flatFee})
		case meta.ChargeTypeCreditPurchase:
			creditPurchase, err := ch.AsCreditPurchaseIntent()
			if err != nil {
				return ChargeIntentsByType{}, fmt.Errorf("converting credit purchase intent[%d]: %w", idx, err)
			}

			out.CreditPurchase = append(out.CreditPurchase, WithIndex[creditpurchase.Intent]{Index: idx, Value: creditPurchase})
		case meta.ChargeTypeUsageBased:
			usageBased, err := ch.AsUsageBasedIntent()
			if err != nil {
				return ChargeIntentsByType{}, fmt.Errorf("converting usage based intent[%d]: %w", idx, err)
			}

			out.UsageBased = append(out.UsageBased, WithIndex[usagebased.Intent]{Index: idx, Value: usageBased})
		default:
			return ChargeIntentsByType{}, fmt.Errorf("unsupported charge type[%d]: %s", idx, ch.Type())
		}
	}

	return out, nil
}
